#!/usr/bin/env bash
set -euo pipefail
set +H # disable history expansion for safety

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BASE_URL="${RC_BASE_URL:-http://127.0.0.1:8787}"
TOKEN_FILE="${RC_TOKEN_FILE:-${ROOT}/host/.dev-token}"
MARKER="${RC_E2E_MARKER:-__E2E_OK__}"
TIMEOUT_MS="${RC_E2E_TIMEOUT_MS:-5000}"

if [[ ! -f "${TOKEN_FILE}" ]]; then
  echo "e2e=SKIP (missing token file)"
  echo "token_file_path=${TOKEN_FILE}"
  exit 0
fi

# Only safe token metadata (never print the token).
TOKEN_BYTES="$(wc -c < "${TOKEN_FILE}" | tr -d ' ')"
TOKEN_SHA256="$(sha256sum "${TOKEN_FILE}" | awk '{print $1}')"
echo "token_file_path=${TOKEN_FILE}"
echo "token_file_byte_length=${TOKEN_BYTES}"
echo "sha256(token_file)=${TOKEN_SHA256}"
echo "base_url=${BASE_URL}"

RC_BASE_URL="${BASE_URL}" RC_TOKEN_FILE="${TOKEN_FILE}" RC_E2E_MARKER="${MARKER}" RC_E2E_TIMEOUT_MS="${TIMEOUT_MS}" node - <<'NODE'
import fs from 'node:fs'
import crypto from 'node:crypto'

const baseUrl = process.env.RC_BASE_URL || 'http://127.0.0.1:8787'
const tokenFile = process.env.RC_TOKEN_FILE
const marker = process.env.RC_E2E_MARKER || '__E2E_OK__'
const timeoutMs = Number(process.env.RC_E2E_TIMEOUT_MS || '5000')

function wsBase(u) {
  const url = new URL(u)
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  return url.toString().replace(/\/$/, '')
}

const token = fs.readFileSync(tokenFile, 'utf8').trim()

async function req(method, path, body) {
  const r = await fetch(baseUrl + path, {
    method,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    body: body ? JSON.stringify(body) : undefined,
  })
  const text = await r.text()
  return { status: r.status, text }
}

function fail(msg) {
  console.log(`e2e=FAIL (${msg})`)
  process.exit(1)
}

const ticketResp = await req('POST', '/api/ws-ticket')
console.log(`ws_ticket_http=${ticketResp.status}`)
if (ticketResp.status !== 200) fail('ws-ticket')
let ticket = ''
try {
  const obj = JSON.parse(ticketResp.text)
  ticket = typeof obj.ticket === 'string' ? obj.ticket : ''
} catch {
  // ignore
}
if (!ticket) fail('ws-ticket-json')

const sessResp = await req('POST', '/api/sessions', { engine: 'shell', name: '__e2e_smoke__', workspacePath: '', prompt: '', mode: '', args: {} })
console.log(`create_session_http=${sessResp.status}`)
if (sessResp.status !== 201) fail('create-session')
let sessionId = ''
try {
  const obj = JSON.parse(sessResp.text)
  sessionId = typeof obj.id === 'string' ? obj.id : ''
} catch {
  // ignore
}
if (!sessionId) fail('create-session-json')
console.log(`session_id=${sessionId}`)

const url = `${wsBase(baseUrl)}/ws/events/${encodeURIComponent(sessionId)}?ticket=${encodeURIComponent(ticket)}&last_n=32`

let wsConnected = false
let markerSeen = false
let lastEventSeq = 0
let terminateDone = false

async function terminate() {
  if (terminateDone) return
  terminateDone = true
  try {
    await req('POST', `/api/sessions/${encodeURIComponent(sessionId)}/terminate`)
  } catch {
    // ignore
  }
}

const ws = new WebSocket(url)

const timeout = setTimeout(async () => {
  console.log('e2e_ws_connect=' + (wsConnected ? 'PASS' : 'FAIL'))
  console.log('e2e_input_roundtrip=' + (wsConnected ? 'PASS' : 'FAIL'))
  console.log('e2e_output_marker=' + (markerSeen ? 'PASS' : 'FAIL'))
  await terminate()
  ws.close()
  process.exit(markerSeen ? 0 : 1)
}, timeoutMs)

ws.onopen = () => {
  wsConnected = true
  ws.send(JSON.stringify({ type: 'input', data: `echo ${marker}\n` }))
}

ws.onmessage = (ev) => {
  try {
    const msg = JSON.parse(ev.data)
    if (typeof msg?.seq === 'number') lastEventSeq = Math.max(lastEventSeq, msg.seq)
    if (msg?.kind === 'assistant') {
      const p = msg.payload || {}
      if (!markerSeen && typeof p.data === 'string' && p.data.includes(marker)) {
        markerSeen = true
        clearTimeout(timeout)
        console.log('e2e_ws_connect=PASS')
        console.log('e2e_input_roundtrip=PASS')
        console.log('e2e_output_marker=PASS')
        console.log(`last_event_seq=${lastEventSeq}`)
        terminate().finally(() => {
          ws.close()
          process.exit(0)
        })
      }
    }
  } catch {
    // ignore
  }
}

ws.onerror = async () => {
  clearTimeout(timeout)
  console.log('e2e_ws_connect=FAIL')
  console.log('e2e_input_roundtrip=FAIL')
  console.log('e2e_output_marker=FAIL')
  await terminate()
  process.exit(1)
}

ws.onclose = async () => {
  clearTimeout(timeout)
  if (!markerSeen) {
    console.log('e2e_ws_connect=' + (wsConnected ? 'PASS' : 'FAIL'))
    console.log('e2e_input_roundtrip=' + (wsConnected ? 'PASS' : 'FAIL'))
    console.log('e2e_output_marker=FAIL')
    await terminate()
    process.exit(1)
  }
}
NODE
