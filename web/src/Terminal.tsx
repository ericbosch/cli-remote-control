import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Terminal as XTerm } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { wsUrl } from './api'

const WS_RECONNECT_DELAYS = [500, 1000, 2000, 5000]

interface TerminalProps {
  sessionId: string
  onClose?: () => void
}

export default function Terminal({ sessionId, onClose }: TerminalProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const fitRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const [status, setStatus] = useState<'connecting' | 'connected' | 'reconnecting' | 'closed'>('connecting')
  const reconnectAttempt = useRef(0)

  const connect = useCallback(() => {
    const url = wsUrl(sessionId)
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      setStatus('connected')
      reconnectAttempt.current = 0
    }

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data as string)
        const term = xtermRef.current
        if (!term) return
        if (msg.type === 'replay') {
          term.write(msg.data || '')
          return
        }
        if (msg.type === 'output' && msg.data) {
          term.write(msg.data)
          return
        }
        if (msg.type === 'status') {
          if (msg.state === 'exited') {
            setStatus('closed')
            ws.close()
          }
        }
      } catch {
        // ignore parse errors
      }
    }

    ws.onclose = () => {
      wsRef.current = null
      if (status !== 'closed') {
        const delay = WS_RECONNECT_DELAYS[Math.min(reconnectAttempt.current, WS_RECONNECT_DELAYS.length - 1)]
        setStatus('reconnecting')
        reconnectAttempt.current += 1
        setTimeout(() => connect(), delay)
      }
    }

    ws.onerror = () => {}
  }, [sessionId, status])

  useEffect(() => {
    if (!containerRef.current) return
    const term = new XTerm({
      cursorBlink: true,
      theme: { background: '#1a1a2e', foreground: '#eee' },
      fontSize: 14,
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
      window.removeEventListener('resize', onResize)
      wsRef.current?.close()
      wsRef.current = null
      term.dispose()
      xtermRef.current = null
      fitRef.current = null
    }
  }, [sessionId, connect])

  useEffect(() => {
    const term = xtermRef.current
    const ws = wsRef.current
    if (!term || !ws || ws.readyState !== WebSocket.OPEN) return

    const disposable = term.onData((data) => {
      ws.send(JSON.stringify({ type: 'input', data }))
    })
    return () => disposable.dispose()
  }, [sessionId, status])

  useEffect(() => {
    const term = xtermRef.current
    const ws = wsRef.current
    const fit = fitRef.current
    if (!term || !ws || ws.readyState !== WebSocket.OPEN || !fit) return
    const cols = term.cols
    const rows = term.rows
    ws.send(JSON.stringify({ type: 'resize', cols, rows }))
  }, [sessionId, status])

  return (
    <div className="terminal-container">
      <div className="terminal-status">
        <span className={`status-dot ${status}`} />
        {status}
        {onClose && (
          <button type="button" className="close-btn" onClick={onClose}>
            Close
          </button>
        )}
      </div>
      <div ref={containerRef} className="terminal" />
    </div>
  )
}
