# WS diagnosis notes

This file is a human-friendly place to paste results from `./scripts/ws_matrix_check.sh` and correlate them with observed client behavior.

## Current deployment assumptions

- `rc-host` binds `127.0.0.1:8787`
- Tailscale Serve exposes HTTPS on `:8443` and proxies to `http://127.0.0.1:8787`

## Matrix results (paste)

Paste the output of:

`./scripts/ws_matrix_check.sh`

## Interpretation guide

- If `local_ws_*` fail:
  - The host WS handler is broken (not a Serve/proxy issue).
- If `local_ws_*` pass but `serve_wss_*` fail:
  - Serve/proxy or client-side WS compatibility issue (TLS/Upgrade/H2).
- If `serve_wss_*` pass but browsers/mobile still disconnect:
  - Likely client reconnection / framing / parsing issues (NDJSON splitting, buffering).
- `tailnet_ws_direct` is expected `SKIP` when binding to localhost.

