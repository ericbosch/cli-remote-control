import React, { useCallback, useEffect, useState } from 'react'
import {
  createSession,
  listSessions,
  terminateSession,
  getToken,
  setToken,
  getBaseUrl,
  setBaseUrl,
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

  const loadSessions = useCallback(async () => {
    if (!getToken()) return
    setError(null)
    try {
      const list = await listSessions()
      setSessions(list)
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
      const s = await createSession('shell')
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
              placeholder="http://127.0.0.1:8765"
            />
          </label>
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
        <Terminal sessionId={attachedId} onClose={() => setAttachedId(null)} />
      ) : (
        <div className="sessions-panel">
          <div className="sessions-header">
            <h2>Sessions</h2>
            <button type="button" onClick={handleCreate} disabled={loading}>
              New session
            </button>
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
