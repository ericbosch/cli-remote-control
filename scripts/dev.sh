#!/usr/bin/env bash
# Start host + web dev servers (host serves API; web dev server for frontend dev).
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# Start host in background (API; generates .dev-token on first run if needed)
(cd host && go run ./cmd/rc-host serve --generate-dev-token) &
HOST_PID=$!

# Start Vite dev server for web (hot reload)
(cd web && npm run dev) &
WEB_PID=$!

echo "Host PID: $HOST_PID  Web PID: $WEB_PID"
echo "Stop with: kill $HOST_PID $WEB_PID"
wait
