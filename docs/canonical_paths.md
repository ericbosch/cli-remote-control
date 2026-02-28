# Canonical scripts & paths (consolidation checklist)

This repo intentionally provides **one canonical way** to do each common workflow. This avoids “3x implementations” drifting out of sync.

## Canonical entrypoints

### Host (rc-host) lifecycle (canonical)

- Start (idempotent, safe defaults): `./scripts/host_bg_start.sh`
- Status: `./scripts/host_bg_status.sh`
- Stop: `./scripts/host_bg_stop.sh`

Notes:
- Default secure posture is `--bind 127.0.0.1` unless explicitly opted in elsewhere (e.g. LAN exposure script).
- These scripts clear `*_API_KEY` env vars for the host process per repo policy.

### Remote access (canonical pair)

- Enable Tailscale Serve: `./scripts/expose_tailscale.sh`
- Disable Tailscale Serve: `./scripts/unexpose_tailscale.sh`

LAN exposure is intentionally **optional** and should be considered a local-network-only tool, not the default remote access path:
- Enable: `./scripts/expose_lan.sh`
- Disable: `./scripts/unexpose_lan.sh`

### Diagnostics bundle (canonical)

- Generate: `./scripts/collect_diag_bundle_v5.sh` (current)

If multiple `collect_diag_bundle_v*.sh` versions exist, only the newest is canonical; older versions are deprecated and may be removed.

## Deprecated or non-canonical helpers

- `./scripts/dev.sh` is a convenience wrapper (not the canonical host entrypoint). Prefer:
  - `./scripts/host_bg_start.sh`
  - then `cd web && npm run dev`

