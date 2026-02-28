#!/usr/bin/env bash
#
# Production-ish web build (deterministic):
# - installs deps with npm ci
# - builds to web/dist
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}/web"

npm ci
npm run build

