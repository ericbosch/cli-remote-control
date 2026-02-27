# Diagnostics (v5 bundle)

Canonical script: `scripts/collect_diag_bundle_v5.sh`

## What it produces

Running v5 creates two artifacts in the repo root:

- `diag_YYYYMMDD_HHMMSS/` (folder)
- `diag_YYYYMMDD_HHMMSS.zip` (zip of the folder)

## Bundle layout

- `SUMMARY.txt` — GO/NO-GO plus required PASS/FAIL/SKIP checks
- `MANIFEST.txt` — file list for the bundle
- `cmd/` — captured command outputs and “code pointers” (snippets/greps)
- `fixtures/` — small, redacted engine fixtures (Codex schema bundle + best-effort Cursor/agent NDJSON)

## Redaction guarantees

v5 redacts (best-effort) from all captured outputs:

- `Authorization: Bearer ...`
- `?token=...`
- `?ticket=...`
- `access_token` / `refresh_token`
- `*_API_KEY` values

It never prints raw auth tokens. Token diagnostics record only:

- token file path
- token byte length
- sha256 of the token file

## How to run

1. Ensure the host is running (dev default is `http://127.0.0.1:8787`).
2. Run: `./scripts/collect_diag_bundle_v5.sh`

If the host is not running, the bundle still gets created, but host-related checks will be FAIL/SKIP in `SUMMARY.txt`.
