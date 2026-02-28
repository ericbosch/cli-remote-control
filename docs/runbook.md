# Runbook — cli-remote-control (always-on, Tailscale-first)

Service goals:

- **Secure defaults:** host binds `127.0.0.1`; `/api/*` requires Bearer; browser WS uses **ws-ticket** (no Bearer token in WS URL).
- **Always-on:** systemd keeps `rc-host` running 24/7 with auto-restart.
- **Remote access:** via **Tailscale Serve** (HTTPS inside your tailnet), without LAN exposure.
- **No secrets in logs:** do not paste token/tickets anywhere; scripts/diag record only path + byte length + sha256(file).

## LIVE STATUS (expected)

- Host base URL (local): `http://127.0.0.1:8787`
- UI served by host at `/` (requires `--web-dir web/dist`)
- Token file path: `host/.dev-token` (never print contents)

## Check status

User service (recommended for non-root install):

- Status: `systemctl --user status cli-remote-control.service`
- Logs: `journalctl --user -u cli-remote-control.service -n 200 --no-pager`

System service (optional; requires sudo install):

- Status: `systemctl status cli-remote-control.service`
- Logs: `journalctl -u cli-remote-control.service -n 200 --no-pager`

## Build + deploy (update)

From repo root:

1) Update code:
- `git pull --ff-only`

2) Build UI + host binary:
- `./scripts/prod_build_web.sh`
- `./scripts/prod_build_host.sh`

3) Restart service:
- User service: `systemctl --user restart cli-remote-control.service`
- System service: `sudo systemctl restart cli-remote-control.service`

## Install systemd service

### User service (no sudo)

1) Build artifacts:
- `./scripts/prod_build_web.sh`
- `./scripts/prod_build_host.sh`

2) Install + start:
- `./scripts/prod_install_systemd_user_service.sh`

Note: to keep the user service running even when logged out, enable linger (may require sudo depending on distro policy):
- `loginctl enable-linger "$USER"`

### System service (optional, root-owned)

1) Build artifacts:
- `./scripts/prod_build_web.sh`
- `./scripts/prod_build_host.sh`

2) Install + start (requires sudo in your terminal):
- `./scripts/prod_install_systemd_system_service.sh`

## Enable/disable Tailscale Serve (HTTPS in tailnet)

Prereq: host service must be healthy at `http://127.0.0.1:8787/healthz`.

- Enable: `./scripts/live_expose_tailscale.sh`
- Disable: `./scripts/live_unexpose_tailscale.sh`

Verify:
- `tailscale serve status`

Shared HTTPS:443 note:

- If another app (e.g. OpenClaw) already owns `https:443` on this node, **do not** replace `/`.
- This repo exposes rc-host on a dedicated port by default: `https:8443` (safe for SPAs and does not touch `https:443`).
- Do **not** run `tailscale serve reset` unless you intentionally want to wipe *all* Serve config on this device.

## Rotate token safely (no token printing)

1) Stop service:
- User: `systemctl --user stop cli-remote-control.service`
- System: `sudo systemctl stop cli-remote-control.service`

2) Rotate token file (delete old; host will re-generate on next start because service uses `--generate-dev-token`):
- `rm -f host/.dev-token`

3) Start service:
- User: `systemctl --user start cli-remote-control.service`
- System: `sudo systemctl start cli-remote-control.service`

4) Update the token on the phone UI/app by reading it locally from `host/.dev-token` (never paste it into logs/issues).

## Troubleshooting

- UI not served at `/`:
  - Ensure `web/dist/` exists: `./scripts/prod_build_web.sh`
  - Ensure service ExecStart has `--web-dir .../web/dist` (see `deploy/systemd/`)
- 401 Unauthorized:
  - Phone UI/app token must match `host/.dev-token`
- WS issues:
  - Browser uses ws-ticket (`POST /api/ws-ticket`) then `?ticket=...` (never use Bearer token in WS URL)
- Port conflict:
  - Check listener: `ss -lntp | rg ':8787\\b'`
  - Stop other service or change `--port`
- Diagnostics:
  - `./scripts/collect_diag_bundle_v6.sh`
  - Start with `diag_*/SUMMARY.txt`

## Repo hygiene

- `./scripts/cleanup_local_artifacts.sh` is safe-by-default and does **not** delete `host/.run` unless `--host-run` is provided.
