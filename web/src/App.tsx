import React, { useCallback, useEffect, useState } from 'react'
import {
  createSession,
  listSessions,
  terminateSession,
  listEngines,
  getToken,
  setToken,
  getBaseUrl,
  setBaseUrl,
  clearBaseUrl,
  type SessionInfo,
} from './api'
import Terminal from './Terminal'
import './App.css'

function getAttachIdFromUrl(): string | null {
  const params = new URLSearchParams(window.location.search)
  const attach = params.get('attach')
  if (attach) return attach
  const hash = window.location.hash
  const m = /#?\/?terminal\/([a-z0-9]+)/i.exec(hash)
  return m ? m[1] : null
}

export default function App() {
  const [sessions, setSessions] = useState<SessionInfo[]>([])
  const [attachedId, setAttachedId] = useState<string | null>(() => getAttachIdFromUrl())
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [token, setTokenState] = useState(getToken())
  const [baseUrl, setBaseUrlState] = useState(getBaseUrl())
  const [showSettings, setShowSettings] = useState(!getToken())
  const [workspacePath, setWorkspacePath] = useState('')
  const [prompt, setPrompt] = useState('')
  const [mode, setMode] = useState<'structured' | 'pty'>('pty')
  const [engines, setEngines] = useState<string[]>(['shell'])
  const [engine, setEngine] = useState<string>('shell')
  const [name, setName] = useState('')
  const [connStatus, setConnStatus] = useState<'connecting' | 'connected' | 'reconnecting' | 'closed'>('closed')

  const attachedSession = attachedId ? sessions.find((s) => s.id === attachedId) || null : null

  const loadSessions = useCallback(async () => {
    if (!getToken()) return
    setError(null)
    try {
      const list = await listSessions()
      setSessions(list)
      if (attachedId && !list.find((s) => s.id === attachedId)) {
        setAttachedId(null)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load sessions')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadSessions()
    const t = setInterval(loadSessions, 5000)
    return () => clearInterval(t)
  }, [loadSessions])

  useEffect(() => {
    const t = getToken()
    if (!t) return
    let cancelled = false
    ;(async () => {
      try {
        const list = await listEngines()
        if (cancelled) return
        const cleaned = list.map((v) => v.trim()).filter(Boolean)
        const next = cleaned.length ? cleaned : ['shell']
        setEngines(next)
        setEngine((prev) => (next.includes(prev) ? prev : next[0] || 'shell'))
      } catch {
        if (cancelled) return
        setEngines(['shell'])
        setEngine('shell')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [token, baseUrl])

  const handleCreate = async () => {
    setError(null)
    try {
      const s = await createSession({
        engine,
        workspacePath,
        prompt,
        mode,
        name: name || undefined,
      })
      setSessions((prev) => [...prev, s])
      setAttachedId(s.id)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create')
    }
  }

  const handleTerminate = async (id: string) => {
    setError(null)
    try {
      await terminateSession(id)
      setSessions((prev) => prev.filter((s) => s.id !== id))
      if (attachedId === id) setAttachedId(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to terminate')
    }
  }

  const handleSaveSettings = () => {
    setToken(token)
    setBaseUrl(baseUrl)
    setShowSettings(false)
    loadSessions()
  }

  if (showSettings) {
    const origin = typeof window !== 'undefined' ? window.location.origin : ''
    const baseLooksLoopback = /^https?:\/\/(127\.0\.0\.1|localhost|\[::1\]|::1)(:|\/|$)/i.test(baseUrl.trim())
    const originLooksLoopback = /^https?:\/\/(127\.0\.0\.1|localhost|\[::1\]|::1)(:|\/|$)/i.test(origin)
    const likelyWrongOnPhone = origin && !originLooksLoopback && baseLooksLoopback
    const likelyMixedContent = origin.startsWith('https://') && baseUrl.trim().startsWith('http://') && origin !== baseUrl.trim()
    return (
      <div className="app">
        <div className="settings-panel">
          <h2>Settings</h2>
          <label>
            Base URL
            <input
              type="text"
              value={baseUrl}
              onChange={(e) => setBaseUrlState(e.target.value)}
              placeholder="http://127.0.0.1:8787"
            />
          </label>
          {(likelyWrongOnPhone || likelyMixedContent) && (
            <div className="error" style={{ marginTop: 8 }}>
              This page is loaded from <code>{origin}</code>, but the Base URL is <code>{baseUrl || '(empty)'}</code>. That often
              causes “Failed to fetch” (phone/Tailscale: <code>127.0.0.1</code> points to the phone, and HTTPS pages can’t fetch HTTP).
              <div style={{ marginTop: 8 }}>
                <button
                  type="button"
                  onClick={() => {
                    setBaseUrlState(origin)
                    clearBaseUrl()
                  }}
                >
                  Use page origin
                </button>
              </div>
            </div>
          )}
          <label>
            Auth token
            <input
              type="password"
              value={token}
              onChange={(e) => setTokenState(e.target.value)}
              placeholder="Bearer token"
            />
          </label>
          <button type="button" onClick={handleSaveSettings}>
            Save & connect
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="rc-shell">
      <header className="rc-topbar">
        <div className="rc-brand">
          <div className="rc-title">Remote Control</div>
          <div className="rc-subtitle">localhost-only host · tailnet UI</div>
        </div>
        <div className="rc-status">
          <span className={`status-dot ${connStatus}`} />
          <span className="rc-status-text">{attachedId ? connStatus : 'idle'}</span>
          {attachedSession && <span className="rc-badge">{attachedSession.engine + ' · ' + attachedSession.state}</span>}
        </div>
        <div className="rc-actions">
          {attachedId && (
            <button type="button" className="settings-btn" onClick={() => setAttachedId(null)}>
              Detach
            </button>
          )}
          <button type="button" className="settings-btn" onClick={() => setShowSettings(true)}>
            Settings
          </button>
        </div>
      </header>

      {error && <div className="error">{error}</div>}

      <div className="rc-body">
        <aside className="rc-sidebar">
          <div className="rc-sidebar-header">
            <h2>Sessions</h2>
            <button type="button" onClick={handleCreate} disabled={loading}>
              New
            </button>
          </div>

          <div className="rc-create">
            <label>
              Engine
              <select value={engine} onChange={(e) => setEngine(e.target.value)}>
                {engines.map((e) => (
                  <option key={e} value={e}>
                    {e}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Name
              <input type="text" value={name} onChange={(e) => setName(e.target.value)} placeholder="optional" />
            </label>
            <label>
              Workspace
              <input
                type="text"
                value={workspacePath}
                onChange={(e) => setWorkspacePath(e.target.value)}
                placeholder="/home/you/project"
              />
            </label>
            <label>
              Prompt
              <input type="text" value={prompt} onChange={(e) => setPrompt(e.target.value)} placeholder="optional" />
            </label>
            {engine === 'cursor' && (
              <label>
                Mode
                <select value={mode} onChange={(e) => setMode(e.target.value as 'structured' | 'pty')}>
                  <option value="pty">PTY</option>
                  <option value="structured">Structured</option>
                </select>
              </label>
            )}
          </div>

          <div className="rc-session-list">
            {loading && sessions.length === 0 ? (
              <div className="rc-empty">Loading…</div>
            ) : sessions.length === 0 ? (
              <div className="rc-empty">No sessions yet. Create one.</div>
            ) : (
              <ul className="session-list">
                {sessions.map((s) => (
                  <li key={s.id} className={attachedId === s.id ? 'active' : ''}>
                    <button type="button" className="rc-session-row" onClick={() => setAttachedId(s.id)}>
                      <span className={`rc-dot ${s.state}`} />
                      <span className="session-name">{s.name || s.id}</span>
                      <span className="session-engine">{s.engine}</span>
                    </button>
                    <button
                      type="button"
                      className="danger"
                      onClick={() => handleTerminate(s.id)}
                      disabled={s.state === 'exited'}
                      title="Terminate session"
                    >
                      ✕
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </aside>

        <main className="rc-main">
          {attachedId ? (
            <Terminal
              sessionId={attachedId}
              session={attachedSession}
              onStatusChange={setConnStatus}
            />
          ) : (
            <div className="rc-hero">
              <h2>Cloud Remote Control</h2>
              <p>Create a session on the left, then attach to stream output and send commands.</p>
              <div className="rc-hero-actions">
                <button type="button" onClick={handleCreate} disabled={loading}>
                  New session
                </button>
                <button type="button" className="settings-btn" onClick={() => setShowSettings(true)}>
                  Settings
                </button>
              </div>
            </div>
          )}
        </main>
      </div>
    </div>
  )
}
