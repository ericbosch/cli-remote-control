# Diagnostics (v6 bundle)

Canonical script: `scripts/collect_diag_bundle_v6.sh` (v5 is deprecated and delegates to v6 when present)

## What it produces

Running v6 creates two artifacts in the repo root:

- `diag_YYYYMMDD_HHMMSS/` (folder)
- `diag_YYYYMMDD_HHMMSS.zip` (zip of the folder)

## Bundle layout (v6)

- `SUMMARY.txt` — required PASS/FAIL/SKIP checks (machine-readable)
- `MANIFEST.txt` — file list for the bundle
- `cmd/` — captured command outputs (redacted)
- `code/` — code pointers (paths + snippets only; no secrets)

## Redaction guarantees

v6 redacts (best-effort) from all captured outputs and runs a redaction self-check:

- `Authorization: Bearer ...`
- `?token=...`
- `?ticket=...`
- `access_token` / `refresh_token`
- `*_API_KEY` values

It never prints raw auth tokens or WS tickets. Token diagnostics record only:

- token file path
- token byte length
- sha256 of the token file

## How to run

1. Ensure the host is running (dev default is `http://127.0.0.1:8787`).
2. Run: `./scripts/collect_diag_bundle_v6.sh`

If the host is not running, the bundle still gets created, but host-related checks will be FAIL/SKIP in `SUMMARY.txt`.
