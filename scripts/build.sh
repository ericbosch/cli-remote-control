#!/usr/bin/env bash
# Build web then host; host will serve web at / in production.
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "Building web..."
(cd web && npm ci && npm run build)

echo "Building host..."
(cd host && go build -o rc-host ./cmd/rc-host)

echo "Done. Run with web UI: host/rc-host serve --web-dir=web/dist"
