#!/usr/bin/env bash
#
# Convenience wrapper for local development:
# - starts the host via the canonical background entrypoint
# - runs the web dev server (hot reload) in the foreground
#
# Canonical host lifecycle scripts:
# - ./scripts/host_bg_start.sh
# - ./scripts/host_bg_status.sh
# - ./scripts/host_bg_stop.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "NOTE: scripts/dev.sh is a convenience wrapper; host_bg_* scripts are canonical." >&2

./scripts/host_bg_start.sh

echo "Starting web dev server (Ctrl-C to stop the web dev server)..." >&2
(cd web && npm run dev)

echo "Web dev server stopped." >&2
echo "Host is still running in the background. Stop it with: ./scripts/host_bg_stop.sh" >&2
