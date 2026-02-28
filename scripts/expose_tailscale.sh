#!/usr/bin/env bash
#
# Tailscale-first remote access: expose the host UI/API via Tailscale Serve,
# while keeping the host bound to 127.0.0.1 locally.
#
# Requires:
# - tailscale installed + authenticated
# - sudo access for `tailscale serve` (common on Linux)
#
# Security:
# - Never prints raw Bearer tokens
# - Exposes only http://127.0.0.1:8787 as the backend
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

if ! command -v tailscale >/dev/null 2>&1; then
  log "tailscale: not found. Install Tailscale and sign in first."
  exit 1
fi

if [[ -f "${ROOT}/web/package.json" ]] && [[ ! -f "${ROOT}/web/dist/index.html" ]]; then
  log "web/dist missing; building web UI so the host can serve it at / ..."
  (cd "${ROOT}/web" && npm ci && npm run build)
fi

log "starting host (background)..."
"${ROOT}/scripts/host_bg_start.sh"

log "enabling tailscale serve (https=443 -> http://127.0.0.1:8787)..."
# Prefer running without sudo (works in some setups). Fall back to passwordless sudo.
if tailscale serve --bg --https=443 http://127.0.0.1:8787 2>/dev/null; then
  :
elif sudo -n tailscale serve --bg --https=443 http://127.0.0.1:8787; then
  :
else
  log "failed to configure tailscale serve (sudo may require a TTY/password)."
  log "try manually in an interactive terminal:"
  log "  sudo tailscale serve --bg --https=443 http://127.0.0.1:8787"
  exit 1
fi

log "tailscale serve status:"
tailscale serve status 2>/dev/null || sudo -n tailscale serve status || true

# Best-effort: print the first ts.net URL if present.
url="$( (tailscale serve status 2>/dev/null || sudo -n tailscale serve status 2>/dev/null || true) | awk 'NR==1{print $1"/"}' )"
if [[ -n "${url}" ]]; then
  echo "serve_url=${url}"
else
  echo "serve_url=UNKNOWN (see: tailscale serve status)"
fi
