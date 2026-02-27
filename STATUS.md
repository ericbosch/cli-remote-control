# Status

**Banner:** 🔴 Red — Codex sandbox blocks host bind; v5 cannot reach 127.0.0.1:8787

**Current milestone:** M1 — Core Remote Terminal (PTY) + Android Native App

## Checklist

- [ ] M1A: Host daemon (PTY, session manager, WS, REST, auth, logs)
- [ ] M1B: Web UI (React + Vite + xterm.js, session picker, reconnect)
- [ ] M1C: Android app (Compose, WebView terminal, settings, sessions)
- [ ] M1D: Docs + scripts + production build
- [ ] All M1 gates pass (see below)

## How to run

### Dev
- **Host only:** `cd host && go run ./cmd/rc-host serve`
- **Web only:** `cd web && npm run dev`
- **Both:** `./scripts/dev.sh` (host + web dev servers)

### Production
- Build: `./scripts/build.sh`
- Run: `./host/rc-host serve` (serves API + web at `/`)

### First run
- On first run with `--generate-dev-token`, host writes `host/.dev-token` (gitignored). Use it in web and Android as Bearer token.

## Acceptance criteria (M1 gates)

1. **Host:** `go run ./cmd/rc-host serve` binds 127.0.0.1, requires Bearer token, GET /api/sessions returns 200 with list, POST /api/sessions creates session, WS to /ws/sessions/{id} streams output and accepts input.
2. **Web:** Session list, create, attach (xterm), input and resize work; disconnect and reattach replays buffer.
3. **Android:** Settings (URL + token), session list, create, attach; terminal in WebView shows streamed output and sends input.
4. **Docs:** setup-linux.md, security.md, troubleshooting.md exist and are accurate.

## Manual test steps (M1)

1. Start host: `cd host && go run ./cmd/rc-host serve --generate-dev-token`; read token from `host/.dev-token`.
2. Open web: `cd web && npm run dev`; in UI set token, create session, attach, type `echo hello`, see output.
3. Resize terminal; disconnect and reattach; see replay then live output.
4. Terminate session from UI; verify session removed.
5. Android: set base URL (e.g. http://PC_IP:port) and token; list sessions, create, attach; same flow.

## Known issues

- (None yet)
