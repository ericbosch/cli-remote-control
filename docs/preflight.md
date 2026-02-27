# Preflight Recon (Phase 0)

Date: 2026-02-27

Primary intent: run everything from Codex with a reliable background host + canonical diagnostics bundle (v5).

## Quick recon (commands run)

- `git status` → clean except untracked `logs/`
- `ls -la scripts` → `build.sh`, `dev.sh`, `collect_diag_bundle_v5.sh` present
- `ls -la scripts/collect_diag_bundle_v5.sh` → present + executable
- `ss -lntp | grep ':8787'` → empty at recon time (host not running)
- `which tmux` → `/usr/bin/tmux` (available)
- `which nohup` → `/usr/bin/nohup` (available)
- `go version` → `go1.22.2 linux/amd64`
- `codex --version` → `codex-cli 0.106.0`

## Verified facts (with evidence)

### How to run (dev) + deterministic token file

- **Host + web dev:** `./scripts/dev.sh`
  - Starts host via: `go run ./cmd/rc-host serve --generate-dev-token`
  - Starts web via: `npm run dev` in `web/`
- **Token file (deterministic):** `host/.dev-token` (must not move)
  - Host logs token diagnostics as: `file=... len=<bytes> sha256=<hex>` (no raw token printed).

Evidence (repo file read):
- `scripts/dev.sh`
- `host/cmd/rc-host/main.go` (`defaultTokenFilePath()` + token load/generate path)

### Engine availability on this machine

- Codex CLI is installed; `codex app-server` help works.
- `tmux` exists; background host can prefer tmux.
- Cursor IDE may be absent; Cursor Agent CLI (`agent`) may be present (depends on PATH).

## Assumptions (only if not quickly verifiable)

- None for Phase 0 beyond “host may not be running at time of recon”; Phase 1 scripts should start it reliably.

## Blockers observed (this Codex execution environment)

- Starting the host from here via `nohup go run ...` fails at bind time with:
  - `listen tcp 127.0.0.1:8787: socket: operation not permitted`
- `tmux` is installed on the machine, but connecting to the tmux socket fails from this environment with:
  - `error connecting to /tmp/tmux-1000/default (Operation not permitted)`

Impact: `scripts/host_bg_start.sh` falls back from tmux to nohup, but the host still cannot listen on `127.0.0.1:8787` when launched from this restricted environment. Diagnostics v5 will be `NO-GO` until the host can be started outside these restrictions.
