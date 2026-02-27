# CLI Remote Control

Remote control for your PC: run and attach to terminal sessions (shell and, in M2, AI CLIs) from an Android app. Sessions run locally on your machine; your phone is a client that sends input and receives streamed output.

## Quick start

1. **Install Go** (see [docs/setup-linux.md](docs/setup-linux.md)).
2. **Start the host:**
   ```bash
   cd host && go run ./cmd/rc-host serve --generate-dev-token
   ```
   Copy the printed token.
3. **Web UI:** Open http://127.0.0.1:8765 (or run `cd web && npm run dev` and use the Vite URL). Enter the token in Settings, create a session, attach.
4. **Android:** Install the app, set Server URL (e.g. `http://YOUR_PC_IP:8765`) and token in Settings, then list/create sessions and attach.

## Repo layout

- **host/** — Go daemon (PTY sessions, REST API, WebSocket, auth).
- **web/** — React + Vite + xterm.js UI (session list, terminal, reconnect).
- **android/** — Kotlin + Compose app (settings, sessions, WebView terminal).
- **docs/** — Setup, security, troubleshooting, invariants, decisions.
- **scripts/** — `dev.sh`, `build.sh`.

## Status and milestones

See [STATUS.md](STATUS.md) for current milestone (M1), acceptance criteria, and how to run (dev + prod).

## Security

Default bind is **127.0.0.1**; all endpoints require a **Bearer token**. See [docs/security.md](docs/security.md) for important warnings.

## License

See [LICENSE](LICENSE).
