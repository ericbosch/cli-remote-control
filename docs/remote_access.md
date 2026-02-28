# Remote access (Tailscale-first)

Default posture:

- Host binds `127.0.0.1` by default (not LAN-exposed).
- `/api/*` requires a Bearer token.
- Browser WebSockets authenticate using **ws-ticket** (no Bearer token in the WS URL).

## Recommended: Tailscale Serve (no LAN exposure)

This keeps the host bound to localhost on your computer and exposes it safely over your Tailscale network.

1) Ensure Tailscale is installed and you are logged in:

- `tailscale status`

2) Start host + enable Serve:

- From repo root: `./scripts/expose_tailscale.sh`
- It configures: `https://<your-device>.<tailnet>.ts.net` → `http://127.0.0.1:8787`

3) Open the printed `serve_url` on your phone (while connected to Tailscale).

4) Disable Serve when done:

- `./scripts/unexpose_tailscale.sh`

Notes:

- The UI will prompt for the Bearer token; read it from `host/.dev-token` on the computer (do not paste it into issues/logs).
- Diagnostics v5 never prints raw tokens; only path + byte length + sha256(file).

## SSH port-forward (fallback)

On your phone (e.g. Termux):

- `ssh -N -L 8787:127.0.0.1:8787 <user>@<your-computer>`

Then open on the phone:

- `http://127.0.0.1:8787`
