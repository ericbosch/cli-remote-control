# Preflight Recon (Phase 0)

Date: 2026-02-27

Scope: Verify dev run paths, auth invariants, and local engine availability without leaking secrets.

## Verified facts (with evidence)

### How to run (dev)

- **Host + web dev:** `./scripts/dev.sh`
  - Starts host via: `go run ./cmd/rc-host serve --generate-dev-token`
  - Starts web via: `npm run dev` in `web/`
- **Token file (deterministic):** `host/.dev-token`
  - Host logs token diagnostics as: `file=... len=<bytes> sha256=<hex>` (no raw token printed).

Evidence (repo file read):
- `scripts/dev.sh`
- `host/cmd/rc-host/main.go` (`defaultTokenFilePath()` + token load/generate path)

### Engine availability on this machine

Evidence (commands run):
- Codex:
  - `codex --version` → `codex-cli 0.106.0`
  - `codex app-server --help` works
- Cursor:
  - `cursor --version` → `cursor: not found` (no Cursor IDE install)
- Agent:
  - `agent --help` works (Cursor Agent CLI is present)

### Auth invariants (must not regress)

These were verified against a locally-started dev host (same command invocation) at `http://127.0.0.1:8787`:

- `GET /api/sessions` **without** `Authorization` → **401**
- `GET /api/sessions` **with** `Authorization: Bearer $(cat host/.dev-token)` → **200**
- `GET /healthz` → **200**

Evidence (commands run; token not echoed):
- `curl -w '%{http_code}\n' http://127.0.0.1:8787/api/sessions`
- `curl -K -` with header sourced from `host/.dev-token` (keeps token out of argv)
- `curl -w '%{http_code}\n' http://127.0.0.1:8787/healthz`

## Assumptions (only where not quickly verifiable)

- None for Phase 0. (Cursor IDE is not present; agent is present but may require a normal login flow to function.)

