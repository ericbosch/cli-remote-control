# Troubleshooting

## Diagnostics

- Generate an audit bundle: `./scripts/collect_diag_bundle_v6.sh`
  - Output: `diag_YYYYMMDD_HHMMSS/` and `diag_YYYYMMDD_HHMMSS.zip` in repo root.
  - Start with `diag_.../SUMMARY.txt` for GO/NO-GO and PASS/FAIL/SKIP.
- Reproduce the realtime pipeline locally (WS ticket → WS connect → input → streamed output):
  - `./scripts/e2e_smoke_ws.sh`
  - Prints only safe metadata (token file fingerprint, HTTP status codes, session id).
- Safe auth debug (does not print the token):
  - Unauth sessions should be `401`: `curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8787/api/sessions`
  - Health should be `200`: `curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8787/healthz`

## Start host in background (tmux recommended)

- Start (idempotent): `./scripts/host_bg_start.sh`
  - Prefers tmux session `rc-host` if `tmux` is installed.
  - Otherwise falls back to `nohup` with `host/.run/rc-host.pid` and `host/.run/rc-host.log`.
- Status: `./scripts/host_bg_status.sh`
- Stop (idempotent): `./scripts/host_bg_stop.sh`
- Attach (tmux): `tmux attach -t rc-host`

## Host won’t start

- **“No auth token set”** — Pass `--token=YOUR_TOKEN`, set `RC_TOKEN`, or use `--generate-dev-token`.
- **“address already in use”** — Another process is using the port. Change port with `--port=8766` or stop the other process.
- **Go not found** — Install Go (see [setup-linux.md](setup-linux.md)).

## Web / Android: “Unauthorized” or 401

- Ensure the **token** in Settings matches the host’s token (Bearer prefix optional).
- Ensure **base URL** points to the host:
  - Local laptop browser: `http://127.0.0.1:8787`
  - Phone over Tailscale Serve (recommended): use the **same origin** you loaded the UI from (e.g. `https://<device>.ts.net:8443`)
- If the UI loads but API calls show “Failed to fetch”, check for **mixed content**:
  - An `https://...` page cannot fetch `http://...` URLs.
  - If you open the UI on your phone and Base URL is `http://127.0.0.1:8787`, that points to the phone (wrong host) and will fail.

## Mobile: “New session” opens a blank page

- This is usually either a **client crash** or a **missing SPA deep-link fallback**.
- Verify deep-link serving on the host:
  - `curl -sS -H 'Accept: text/html' -o /dev/null -w '%{http_code} %{content_type}\n' http://127.0.0.1:8787/__rc_deeplink_test__/sessions/abc`
  - Expected: `200 text/html...`
- If it still happens:
  - Refresh once (cached JS).
  - Run `./scripts/collect_diag_bundle_v6.sh` and check `ui_deeplink_serves_index=PASS`.

## Android: can’t connect to host

- **Preferred (secure):** Use Tailscale Serve URL (HTTPS) and keep host bound to `127.0.0.1`.
- **Emulator:** Use base URL `http://10.0.2.2:8787` (emulator’s alias to host’s loopback).
- **LAN IP:** Only works if host is explicitly started with `--bind=0.0.0.0` (not the default).

## Terminal: no output or duplicate output

- **Reconnect:** Closing and re-opening the terminal reattaches and sends a **replay** of the last 64 KB, then live output. If you see duplication, the client may be rendering both; the protocol sends `replay` once then `output` for new data.
- **Session exited:** If the shell process exited, you’ll see “exited” and no new output; create a new session.
  - If the UI renders but you still see no streaming, run `./scripts/e2e_smoke_ws.sh` to determine if the backend pipeline is healthy.

## Session list empty or stale

- Refresh the list (e.g. “New session” or pull-to-refresh if implemented). The list is fetched from the host; ensure the host is running and the token is correct.

## Logs

- Session logs are in the host’s `--log-dir` (default `logs/`). They are rotated by size if configured; otherwise ensure disk space. Logs do not contain the auth token.
