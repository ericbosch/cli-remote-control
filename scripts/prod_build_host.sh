#!/usr/bin/env bash
#
# Production-ish host build (on-disk binary):
# - builds to host/.run/rc-host
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

mkdir -p host/.run host/.run/logs
chmod 0700 host/.run 2>/dev/null || true

(cd host && go build -o .run/rc-host ./cmd/rc-host)

echo "built_host_bin=${ROOT}/host/.run/rc-host"

