import React, { useCallback, useEffect, useState } from 'react'
import {
  createSession,
  listSessions,
  terminateSession,
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
  // Default to shell for responsiveness (cursor engines may require separate auth/login).
  const [engine, setEngine] = useState<'shell' | 'codex' | 'cursor'>('shell')
  const [name, setName] = useState('')

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
    <div className="app">
      <header className="header">
        <h1>CLI Remote Control</h1>
        <button type="button" className="settings-btn" onClick={() => setShowSettings(true)}>
          Settings
        </button>
      </header>
      {error && <div className="error">{error}</div>}
      {attachedId ? (
        <Terminal
          sessionId={attachedId}
          session={sessions.find((s) => s.id === attachedId) || null}
          onClose={() => setAttachedId(null)}
        />
      ) : (
        <div className="sessions-panel">
          <div className="sessions-header">
            <h2>Sessions</h2>
            <button type="button" onClick={handleCreate} disabled={loading}>
              New session
            </button>
          </div>
          <div className="cursor-create">
            <label>
              Engine
              <select value={engine} onChange={(e) => setEngine(e.target.value as 'shell' | 'codex' | 'cursor')}>
                <option value="shell">shell</option>
                <option value="codex">codex</option>
                <option value="cursor">cursor</option>
              </select>
            </label>
            <label>
              Session name (optional)
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. auth-hardening"
              />
            </label>
            <label>
              Workspace path
              <input
                type="text"
                value={workspacePath}
                onChange={(e) => setWorkspacePath(e.target.value)}
                placeholder="/home/you/project"
              />
            </label>
            <label>
              Initial prompt (optional)
              <input
                type="text"
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="e.g. set up tests for auth module"
              />
            </label>
            {engine === 'cursor' && (
              <label>
                Mode
                <select value={mode} onChange={(e) => setMode(e.target.value as 'structured' | 'pty')}>
                  <option value="pty">PTY (terminal)</option>
                  <option value="structured">Structured (experimental)</option>
                </select>
              </label>
            )}
          </div>
          {loading && sessions.length === 0 ? (
            <p>Loading…</p>
          ) : (
            <ul className="session-list">
              {sessions.map((s) => (
                <li key={s.id}>
                  <span className="session-name">{s.name || s.id}</span>
                  <span className="session-engine">{s.engine}</span>
                  <span className="session-state">{s.state}</span>
                  <span className="session-seq">seq {s.last_seq ?? 0}</span>
                  <button type="button" onClick={() => setAttachedId(s.id)}>
                    Attach
                  </button>
                  <button
                    type="button"
                    className="danger"
                    onClick={() => handleTerminate(s.id)}
                    disabled={s.state === 'exited'}
                  >
                    Terminate
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  )
}
