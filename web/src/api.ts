const TOKEN_KEY = 'rc-token'
const BASE_KEY = 'rc-base'

function stripTrailingSlash(u: string): string {
  return u.replace(/\/$/, '')
}

function isLoopbackHost(host: string): boolean {
  const h = host.toLowerCase()
  return h === '127.0.0.1' || h === 'localhost' || h === '::1' || h === '[::1]'
}

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(t: string): void {
  localStorage.setItem(TOKEN_KEY, t)
}

export function getBaseUrl(): string {
  const b = localStorage.getItem(BASE_KEY)
  if (b) {
    // Guard against common footguns:
    // - Phone opens https://<tailnet>:8443 but baseUrl still points to http://127.0.0.1:8787 (mixed content + wrong host).
    // - Browser origin is non-loopback but baseUrl is loopback.
    try {
      const stored = new URL(b)
      if (typeof window !== 'undefined') {
        const origin = new URL(window.location.origin)
        const storedIsLoopback = isLoopbackHost(stored.hostname)
        const originIsLoopback = isLoopbackHost(origin.hostname)
        const mixedContent = origin.protocol === 'https:' && stored.protocol === 'http:' && stored.hostname !== origin.hostname
        if (mixedContent || (storedIsLoopback && !originIsLoopback)) {
          return stripTrailingSlash(window.location.origin)
        }
      }
      return stripTrailingSlash(stored.toString())
    } catch {
      // ignore invalid stored base
    }
  }
  if (typeof window !== 'undefined') {
    try {
      const u = new URL(window.location.origin)
      // Common dev setup: Vite UI on :5173 (or preview on :4173) but host API on :8787.
      // Do not rewrite for production/tailnet ports (e.g. :8443 behind Tailscale Serve).
      if (u.port === '5173' || u.port === '4173') {
        u.port = '8787'
        return stripTrailingSlash(u.toString())
      }
      return window.location.origin
    } catch {
      // ignore
    }
  }
  return 'http://127.0.0.1:8787'
}

export function setBaseUrl(u: string): void {
  const v = u.trim()
  if (!v) {
    localStorage.removeItem(BASE_KEY)
    return
  }
  localStorage.setItem(BASE_KEY, stripTrailingSlash(v))
}

export function clearBaseUrl(): void {
  localStorage.removeItem(BASE_KEY)
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
  engine: string
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
  if (!r.ok) {
    const text = await r.text().catch(() => '')
    try {
      const obj = JSON.parse(text) as any
      const e = obj?.error
      const msg = typeof e?.message === 'string' ? e.message : `HTTP ${r.status}`
      const hint = typeof e?.hint === 'string' ? e.hint : ''
      const rid = typeof e?.request_id === 'string' ? e.request_id : ''
      const suffix = [hint && `Hint: ${hint}`, rid && `request_id=${rid}`].filter(Boolean).join(' · ')
      throw new Error(suffix ? `${msg} (${r.status}) — ${suffix}` : `${msg} (${r.status})`)
    } catch {
      // ignore
    }
    throw new Error(text ? `HTTP ${r.status}: ${text.slice(0, 200)}` : `HTTP ${r.status}`)
  }
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

export async function listEngines(): Promise<string[]> {
  const r = await fetch(`${getBaseUrl()}/api/engines`, { headers: headers() })
  if (!r.ok) throw new Error(r.status === 401 ? 'Unauthorized' : `HTTP ${r.status}`)
  const arr = (await r.json()) as unknown
  if (!Array.isArray(arr)) throw new Error('Invalid engines response')
  const cleaned = arr.map((v) => String(v).trim()).filter(Boolean)
  return cleaned.length ? cleaned : ['shell']
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
