const TOKEN_KEY = 'rc-token'
const BASE_KEY = 'rc-base'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(t: string): void {
  localStorage.setItem(TOKEN_KEY, t)
}

export function getBaseUrl(): string {
  const b = localStorage.getItem(BASE_KEY)
  if (b) return b
  if (typeof window !== 'undefined') return window.location.origin
  return 'http://127.0.0.1:8765'
}

export function setBaseUrl(u: string): void {
  localStorage.setItem(BASE_KEY, u)
}

function headers(): HeadersInit {
  const t = getToken()
  return {
    'Content-Type': 'application/json',
    ...(t ? { Authorization: `Bearer ${t}` } : {}),
  }
}

export interface SessionInfo {
  id: string
  name: string
  engine: string
  state: string
  exit_code?: number
  created: string
}

export async function listSessions(): Promise<SessionInfo[]> {
  const r = await fetch(`${getBaseUrl()}/api/sessions`, { headers: headers() })
  if (!r.ok) throw new Error(r.status === 401 ? 'Unauthorized' : `HTTP ${r.status}`)
  return r.json()
}

export async function createSession(engine = 'shell', name?: string): Promise<SessionInfo> {
  const r = await fetch(`${getBaseUrl()}/api/sessions`, {
    method: 'POST',
    headers: headers(),
    body: JSON.stringify({ engine, name, args: {} }),
  })
  if (!r.ok) throw new Error(`HTTP ${r.status}`)
  return r.json()
}

export async function terminateSession(id: string): Promise<void> {
  const r = await fetch(`${getBaseUrl()}/api/sessions/${id}/terminate`, {
    method: 'POST',
    headers: headers(),
  })
  if (!r.ok && r.status !== 404) throw new Error(`HTTP ${r.status}`)
}

export function wsUrl(sessionId: string): string {
  const base = getBaseUrl().replace(/^http/, 'ws')
  const t = getToken()
  const sep = base.includes('?') ? '&' : '?'
  return `${base}/ws/sessions/${sessionId}${t ? `${sep}token=${encodeURIComponent(t)}` : ''}`
}
