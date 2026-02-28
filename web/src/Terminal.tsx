import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { issueWSTicket, wsEventsUrl, type SessionInfo } from './api'

const WS_RECONNECT_DELAYS = [500, 1000, 2000, 5000]
const LAST_SEQ_KEY_PREFIX = 'rc-last-seq:'

interface TerminalProps {
  sessionId: string
  session: SessionInfo | null
  onClose?: () => void
}

type SessionEvent = {
  session_id: string
  engine: string
  ts_ms: number
  seq: number
  kind: string
  payload?: any
}

function safeText(v: unknown): string {
  if (typeof v === 'string') return v
  return ''
}

function formatTS(ms: number): string {
  try {
    return new Date(ms).toLocaleTimeString()
  } catch {
    return ''
  }
}

export default function Terminal({ sessionId, session, onClose }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const shouldReconnectRef = useRef(true)
  const connectNonceRef = useRef(0)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'reconnecting' | 'closed'>('connecting')
  const closedRef = useRef(false)
  const reconnectAttempt = useRef(0)
  const [lastSeq, setLastSeq] = useState<number>(() => {
    const v = localStorage.getItem(`${LAST_SEQ_KEY_PREFIX}${sessionId}`)
    const n = v ? Number(v) : 0
    return Number.isFinite(n) ? n : 0
  })
  const lastSeqRef = useRef<number>(lastSeq)
  const [thinkingOpen, setThinkingOpen] = useState(false)
  const [thinkingText, setThinkingText] = useState('')
  const thinkingRef = useRef('')
  const [thinkingHistory, setThinkingHistory] = useState<string[]>([])
  const [toolCalls, setToolCalls] = useState<Array<{ ts: number; seq: number; payload: any }>>([])
  const [toolOutputs, setToolOutputs] = useState<Array<{ ts: number; seq: number; payload: any }>>([])
  const [inputText, setInputText] = useState('')

  const statusColor = useMemo(() => {
    if (status === 'connected') return 'green'
    if (status === 'reconnecting' || status === 'connecting') return 'yellow'
    return 'red'
  }, [status])

  useEffect(() => {
    lastSeqRef.current = lastSeq
  }, [lastSeq])

  useEffect(() => {
    closedRef.current = status === 'closed'
  }, [status])

  const sendInput = useCallback(
    (text: string) => {
      const ws = wsRef.current
      if (!ws || ws.readyState !== WebSocket.OPEN) return
      ws.send(JSON.stringify({ type: 'input', data: text }))
    },
    [wsRef],
  )

  const writeLine = useCallback((line: string) => {
    const term = xtermRef.current
    if (!term) return
    term.write(line.replaceAll('\n', '\r\n'))
  }, [])

  const connect = useCallback(async () => {
    const myNonce = ++connectNonceRef.current
    setStatus(reconnectAttempt.current > 0 ? 'reconnecting' : 'connecting')
    const from = lastSeqRef.current > 0 ? String(lastSeqRef.current + 1) : ''
    try {
      const ticket = await issueWSTicket()
      if (!shouldReconnectRef.current || myNonce !== connectNonceRef.current) return
      const url = wsEventsUrl(sessionId, ticket, from ? { from_seq: from } : { last_n: '256' })
      const ws = new WebSocket(url)
      wsRef.current = ws

      ws.onopen = () => {
        setStatus('connected')
        reconnectAttempt.current = 0

        const term = xtermRef.current
        const fit = fitRef.current
        if (term && fit) {
          fit.fit()
          ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
        }
      }

      ws.onmessage = (ev) => {
        try {
          const msg = JSON.parse(ev.data as string) as SessionEvent
          if (!msg || typeof msg.kind !== 'string' || typeof msg.seq !== 'number') return

          if (msg.seq > lastSeqRef.current) {
            lastSeqRef.current = msg.seq
            setLastSeq(msg.seq)
            localStorage.setItem(`${LAST_SEQ_KEY_PREFIX}${sessionId}`, String(msg.seq))
          }

          switch (msg.kind) {
            case 'assistant': {
              const payload = msg.payload || {}
              const txt = safeText(payload.data)
              if (txt) writeLine(txt)
              return
            }
            case 'user': {
              const payload = msg.payload || {}
              const txt = safeText(payload.data)
              if (txt) writeLine(`\r\n[you ${formatTS(msg.ts_ms)}] ${txt}\r\n`)
              return
            }
            case 'thinking_delta': {
              const payload = msg.payload || {}
              const delta = safeText(payload.delta)
              if (delta) {
                thinkingRef.current += delta
                setThinkingText(thinkingRef.current)
              }
              return
            }
            case 'thinking_done': {
              const done = thinkingRef.current
              if (done) setThinkingHistory((prev) => [...prev, done])
              thinkingRef.current = ''
              setThinkingText('')
              return
            }
            case 'tool_call': {
              setToolCalls((prev) => [...prev, { ts: msg.ts_ms, seq: msg.seq, payload: msg.payload }])
              return
            }
            case 'tool_output': {
              setToolOutputs((prev) => [...prev, { ts: msg.ts_ms, seq: msg.seq, payload: msg.payload }])
              return
            }
            case 'error': {
              const payload = msg.payload || {}
              const m = safeText(payload.message) || safeText(payload.data)
              if (m) writeLine(`\r\n[error ${formatTS(msg.ts_ms)}] ${m}\r\n`)
              return
            }
            case 'status': {
              const payload = msg.payload || {}
              const state = safeText(payload.state)
              const code = typeof payload.exit_code === 'number' ? payload.exit_code : undefined
              if (state) writeLine(`\r\n[status ${formatTS(msg.ts_ms)}] ${state}${code != null ? ` (exit ${code})` : ''}\r\n`)
              if (state === 'exited') {
                setStatus('closed')
                ws.close()
              }
              return
            }
          }
        } catch {
          // ignore parse errors
        }
      }

      ws.onclose = (ev) => {
        wsRef.current = null
        writeLine(`\r\n[ws ${formatTS(Date.now())}] closed (code=${ev.code}${ev.reason ? ` reason=${ev.reason}` : ''})\r\n`)
        if (!shouldReconnectRef.current) return
        if (closedRef.current) return
        const delay = WS_RECONNECT_DELAYS[Math.min(reconnectAttempt.current, WS_RECONNECT_DELAYS.length - 1)]
        setStatus('reconnecting')
        reconnectAttempt.current += 1
        setTimeout(() => void connect(), delay)
      }

      ws.onerror = () => {
        writeLine(`\r\n[ws ${formatTS(Date.now())}] error\r\n`)
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to connect'
      writeLine(`\r\n[error ${formatTS(Date.now())}] ${msg}\r\n`)
      if (!shouldReconnectRef.current || closedRef.current) return
      const delay = WS_RECONNECT_DELAYS[Math.min(reconnectAttempt.current, WS_RECONNECT_DELAYS.length - 1)]
      setStatus('reconnecting')
      reconnectAttempt.current += 1
      setTimeout(() => void connect(), delay)
    }
  }, [sessionId, writeLine])

  useEffect(() => {
    if (!containerRef.current) return
    shouldReconnectRef.current = true
    const term = new XTerm({
      cursorBlink: true,
      theme: { background: '#1a1a2e', foreground: '#eee' },
      fontSize: 14,
      disableStdin: true,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(containerRef.current)
    fit.fit()
    xtermRef.current = term
    fitRef.current = fit

    const onResize = () => fit.fit()
    window.addEventListener('resize', onResize)

    connect()

    return () => {
      shouldReconnectRef.current = false
      window.removeEventListener('resize', onResize)
      wsRef.current?.close()
      wsRef.current = null
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
    }
  }, [sessionId, connect])

  return (
    <div className="terminal-container">
      <div className="terminal-status">
        <span className={`status-dot ${status}`} />
        <span className={`status-banner ${statusColor}`}>{status}</span>
        <span className="status-meta">
          {session?.engine || 'unknown'} · {session?.state || 'unknown'} · seq {lastSeq}
        </span>
        <button type="button" className="small-btn" onClick={() => setThinkingOpen((v) => !v)}>
          Thinking
        </button>
        {onClose && (
          <button type="button" className="close-btn" onClick={onClose}>
            Close
          </button>
        )}
      </div>

      <div className="session-panels">
        <div className="panel terminal-panel">
          <div ref={containerRef} className="terminal" />
          <div className="input-row">
            <textarea
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              placeholder="Send input (full line/message)…"
              rows={2}
            />
            <div className="input-actions">
              <button
                type="button"
                onClick={() => {
                  const txt = inputText.trimEnd()
                  if (!txt) return
                  sendInput(txt + '\n')
                  setInputText('')
                }}
                disabled={status !== 'connected'}
              >
                Send
              </button>
              <button type="button" onClick={() => sendInput('\u0003')} disabled={status !== 'connected'}>
                Ctrl+C
              </button>
            </div>
          </div>
        </div>

        <div className="panel side-panel">
          <details
            open={thinkingOpen}
            onToggle={(e) => setThinkingOpen((e.target as HTMLDetailsElement).open)}
            className="details"
          >
            <summary>Thinking ({thinkingHistory.length + (thinkingText ? 1 : 0)})</summary>
            {thinkingText && <pre className="mono pre-block">{thinkingText}</pre>}
            {thinkingHistory.slice().reverse().map((t, i) => (
              <pre key={i} className="mono pre-block">
                {t}
              </pre>
            ))}
          </details>

          <details className="details">
            <summary>Tool calls ({toolCalls.length})</summary>
            {toolCalls.slice().reverse().map((t) => (
              <pre key={t.seq} className="mono pre-block">
                [{formatTS(t.ts)} seq {t.seq}] {JSON.stringify(t.payload, null, 2)}
              </pre>
            ))}
          </details>

          <details className="details">
            <summary>Tool outputs ({toolOutputs.length})</summary>
            {toolOutputs.slice().reverse().map((t) => (
              <pre key={t.seq} className="mono pre-block">
                [{formatTS(t.ts)} seq {t.seq}] {JSON.stringify(t.payload, null, 2)}
              </pre>
            ))}
          </details>
        </div>
      </div>
    </div>
  )
}
