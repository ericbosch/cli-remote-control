# Troubleshooting

## Diagnostics

- Generate an audit bundle: `./scripts/collect_diag_bundle_v6.sh`
  - Output: `diag_YYYYMMDD_HHMMSS/` and `diag_YYYYMMDD_HHMMSS.zip` in repo root.
  - Start with `diag_.../SUMMARY.txt` for GO/NO-GO and PASS/FAIL/SKIP.
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
- Ensure **base URL** points to the host (e.g. `http://127.0.0.1:8787` for local, or your PC’s IP and port when using the phone).

## Android: can’t connect to host

- **Emulator:** Use base URL `http://10.0.2.2:8787` (emulator’s alias to host’s loopback).
- **Physical device:** Use your PC’s LAN IP (e.g. `http://192.168.1.100:8787`). Host must be started with `--bind=0.0.0.0`.
- **Cleartext:** The app has `android:usesCleartextTraffic="true"` for HTTP. For HTTPS use a proper URL.

## Terminal: no output or duplicate output

- **Reconnect:** Closing and re-opening the terminal reattaches and sends a **replay** of the last 64 KB, then live output. If you see duplication, the client may be rendering both; the protocol sends `replay` once then `output` for new data.
- **Session exited:** If the shell process exited, you’ll see “exited” and no new output; create a new session.

## Session list empty or stale

- Refresh the list (e.g. “New session” or pull-to-refresh if implemented). The list is fetched from the host; ensure the host is running and the token is correct.

## Logs

- Session logs are in the host’s `--log-dir` (default `logs/`). They are rotated by size if configured; otherwise ensure disk space. Logs do not contain the auth token.
