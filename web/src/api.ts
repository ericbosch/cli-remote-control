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
  if (typeof window !== 'undefined') {
    try {
      const u = new URL(window.location.origin)
      // Common dev setup: Vite UI on :5173 (or preview on :4173) but host API on :8787.
      // Do not rewrite for production/tailnet ports (e.g. :8443 behind Tailscale Serve).
      if (u.port === '5173' || u.port === '4173') {
        u.port = '8787'
        return u.toString().replace(/\/$/, '')
      }
      return window.location.origin
    } catch {
      // ignore
    }
  }
  return 'http://127.0.0.1:8787'
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
  last_seq?: number
  created: string
}

export async function listSessions(): Promise<SessionInfo[]> {
  const r = await fetch(`${getBaseUrl()}/api/sessions`, { headers: headers() })
  if (!r.ok) throw new Error(r.status === 401 ? 'Unauthorized' : `HTTP ${r.status}`)
  return r.json()
}

export interface CreateSessionBody {
  engine: 'shell' | 'codex' | 'cursor'
  workspacePath?: string
  prompt?: string
  mode?: 'structured' | 'pty'
  name?: string
}

export async function createSession(body: CreateSessionBody): Promise<SessionInfo> {
  const payload: Record<string, unknown> = {
    engine: body.engine,
    name: body.name,
    workspacePath: body.workspacePath || '',
    prompt: body.prompt || '',
    mode: body.mode || '',
    args: {},
  }
  const r = await fetch(`${getBaseUrl()}/api/sessions`, {
    method: 'POST',
    headers: headers(),
    body: JSON.stringify(payload),
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

export async function issueWSTicket(): Promise<string> {
  const r = await fetch(`${getBaseUrl()}/api/ws-ticket`, {
    method: 'POST',
    headers: headers(),
  })
  if (!r.ok) throw new Error(r.status === 401 ? 'Unauthorized' : `HTTP ${r.status}`)
  const obj = (await r.json()) as { ticket?: string }
  if (!obj || typeof obj.ticket !== 'string' || !obj.ticket) throw new Error('Missing ws ticket')
  return obj.ticket
}

export function wsEventsUrl(sessionId: string, ticket: string, params?: Record<string, string>): string {
  const base = getBaseUrl().replace(/^http/, 'ws')
  const url = new URL(`${base}/ws/events/${sessionId}`)
  url.searchParams.set('ticket', ticket)
  if (params) {
    for (const [k, v] of Object.entries(params)) url.searchParams.set(k, v)
  }
  return url.toString()
}
