#!/usr/bin/env bash
#
# Disable ONLY the Tailscale Serve exposure configured by live_expose_tailscale.sh.
# Never uses `tailscale serve reset`.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

STATE_FILE="${ROOT}/host/.run/serve-mode.json"

if ! command -v tailscale >/dev/null 2>&1; then
  log "tailscale: not found."
  exit 1
fi

serve_cmd() {
  if tailscale serve "$@" 2>/dev/null; then
    return 0
  fi
  if command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve "$@"; then
    return 0
  fi
  return 1
}

mode=""
https_port=""
path=""
if [[ -f "${STATE_FILE}" ]]; then
  mode="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["mode"])' "${STATE_FILE}" 2>/dev/null || true)"
  https_port="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["httpsPort"])' "${STATE_FILE}" 2>/dev/null || true)"
  path="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("path",""))' "${STATE_FILE}" 2>/dev/null || true)"
fi

if [[ "${mode}" == "path" && -n "${https_port}" && -n "${path}" ]]; then
  log "disabling tailscale serve mapping: https:${https_port} path=${path}"
  if ! serve_cmd --https="${https_port}" --set-path="${path}" off; then
    log "failed to disable mapping (may need interactive sudo). Try manually:"
    log "  sudo tailscale serve --https=${https_port} --set-path=${path} off"
    exit 1
  fi
elif [[ "${mode}" == "port" && -n "${https_port}" ]]; then
  log "disabling tailscale serve mapping: https:${https_port}"
  if ! serve_cmd --https="${https_port}" off; then
    log "failed to disable mapping (may need interactive sudo). Try manually:"
    log "  sudo tailscale serve --https=${https_port} off"
    exit 1
  fi
else
  log "serve mode unknown (missing ${STATE_FILE}); refusing to disable anything automatically."
  log "inspect current config: tailscale serve status"
  exit 2
fi

log "tailscale serve status (after unexpose):"
tailscale serve status 2>/dev/null || (command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve status) || true
