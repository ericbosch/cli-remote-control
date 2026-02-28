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

If you're running the always-on systemd service, use the “live” helper scripts (does not rebuild/start host):

- Enable: `./scripts/live_expose_tailscale.sh`
- Disable: `./scripts/live_unexpose_tailscale.sh`

3) Open the printed `serve_url` on your phone (while connected to Tailscale).

Shared HTTPS:443 note:

- If another app already owns `https:443` on this node, `live_expose_tailscale.sh` will add rc-host under `/rc` (path mapping) instead of replacing `/`.

4) Disable Serve when done:

- `./scripts/unexpose_tailscale.sh`

Notes:

- The UI will prompt for the Bearer token; read it from `host/.dev-token` on the computer (do not paste it into issues/logs).
- Diagnostics v5 never prints raw tokens; only path + byte length + sha256(file).
- Warning: do not run `tailscale serve reset` unless you intend to wipe all Serve config on this device.

## SSH port-forward (fallback)

On your phone (e.g. Termux):

- `ssh -N -L 8787:127.0.0.1:8787 <user>@<your-computer>`

Then open on the phone:

- `http://127.0.0.1:8787`

## Optional: LAN exposure (opt-in, allowlist)

Prefer Tailscale Serve. If you intentionally want direct LAN access:

- Enable: `./scripts/expose_lan.sh`
  - Adds a `ufw` allow rule from your detected LAN CIDR → `tcp/8787`
  - Restarts the host bound to `0.0.0.0:8787`
- Disable: `./scripts/unexpose_lan.sh`
  - Removes the `ufw` rule(s) added by the script
  - Restarts the host bound back to `127.0.0.1:8787`

Warnings:

- This exposes the service to devices on your LAN. Use only on trusted networks.
- Keep the Bearer token private; never paste it into logs/issues.
