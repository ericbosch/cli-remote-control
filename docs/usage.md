# Usage (end-to-end)

This repo is designed to be safe-by-default:

- Host binds `127.0.0.1` by default (not LAN-exposed).
- `/api/*` requires a Bearer token.
- Browser WebSockets authenticate via a short-lived **WS ticket** (no Bearer token in the WS URL).

## Start the host (background)

- From repo root: `./scripts/host_bg_start.sh`
- Status: `./scripts/host_bg_status.sh`
- Stop: `./scripts/host_bg_stop.sh`

Token file (do not print it): `host/.dev-token`

## Run the web UI

### Dev (Vite)

- `cd web && npm ci && npm run dev`
- Open: `http://127.0.0.1:5173`

The UI will default the API base URL to `http://127.0.0.1:8787` (same host, different port).

### Prod (served by host)

- `./scripts/build.sh`
- `./host/rc-host serve --generate-dev-token --web-dir=web/dist`
- Open: `http://127.0.0.1:8787`

## Use from a phone without exposing LAN ports (SSH port-forward)

This keeps the host bound to `127.0.0.1` on your computer and tunnels ports to your phone.

On your **phone** (e.g. Termux), run:

- Host API/UI (recommended, prod mode): `ssh -N -L 8787:127.0.0.1:8787 <user>@<your-computer>`
- If using Vite dev UI too: `ssh -N -L 5173:127.0.0.1:5173 -L 8787:127.0.0.1:8787 <user>@<your-computer>`

Then on the phone browser:

- Prod UI: `http://127.0.0.1:8787`
- Dev UI: `http://127.0.0.1:5173` (API base URL should be `http://127.0.0.1:8787`)

## Remote access via Tailscale (recommended)

See `docs/remote_access.md` and use:

- `./scripts/expose_tailscale.sh`
- `./scripts/unexpose_tailscale.sh`

## Optional: LAN exposure (opt-in)
See `docs/remote_access.md`.

## Getting the token

The host writes a dev token to `host/.dev-token` when started with `--generate-dev-token`.

- Use the token value in the UI settings (Bearer prefix optional).
- Never paste the token into logs/issues; diagnostics only record path + length + sha256.
