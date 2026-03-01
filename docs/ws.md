# WebSocket transport (shell-first)

This project uses **HTTP REST** for control-plane operations and a **WebSocket event stream** per session for realtime output + status.

## Endpoints

- Health: `GET /healthz`
- Sessions:
  - `GET /api/sessions`
  - `POST /api/sessions` body: `{ "engine": "shell", "name": "...", "workspacePath": "...", "prompt": "..." }`
- Engines (allowed values for UI selectors): `GET /api/engines`
- WS ticket (browser auth): `POST /api/ws-ticket` → `{ "ticket": "..." }`

### WebSocket event stream

- Path: `GET /ws/events/{sessionId}`
- Auth (choose one):
  - **Browser**: `?ticket=<ws-ticket>` query param (short-lived; single-use)
  - **Non-browser clients**: `Authorization: Bearer <token>` header
- Replay params:
  - `from_seq=<n>` replay from an event sequence number
  - `last_n=<n>` replay last N events (default used by server if omitted)

### Client → server messages (JSON)

Sent over the same WS connection:

- Input:
  - `{ "type": "input", "data": "echo hi\\n" }`
- Resize (optional):
  - `{ "type": "resize", "cols": 120, "rows": 30 }`

## URL rules (LAN + Tailscale)

- **Secure default**: `rc-host` binds `127.0.0.1:8787`.
- Remote access is expected to go through **Tailscale Serve** (e.g. `https://<taildns>:8443` → `http://127.0.0.1:8787`).
- Because of the localhost bind, **tailnet-direct** access to `http://<tail-ip>:8787` will be unreachable unless the bind address is explicitly changed (not recommended by default).

## Keepalive / reconnect

- The server sends periodic WebSocket **Ping** frames to keep proxies from dropping idle connections.
- Clients should reconnect with backoff, and resume by passing `from_seq` (last seen seq).

## Diagnostics

- Run the matrix probe: `./scripts/ws_matrix_check.sh`
  - Probes:
    - local `ws://127.0.0.1:8787/ws/events/{id}`
    - Serve `wss://...:8443/ws/events/{id}` (if detected)
    - tailnet-direct `:8787` reachability (expected SKIP under localhost bind)

