#!/usr/bin/env bash
#
# Live remote access (Tailscale Serve) for an already-running host.
# Keeps rc-host bound to localhost; exposes HTTPS via tailnet.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${RC_HOST_PORT:-8787}"
BASE_URL="http://127.0.0.1:${PORT}"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

if ! command -v tailscale >/dev/null 2>&1; then
  log "tailscale: not found. Install Tailscale first."
  exit 1
fi

if ! tailscale status >/dev/null 2>&1; then
  log "tailscale: not running/authenticated. Run: tailscale status"
  exit 1
fi

code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 "${BASE_URL}/healthz" 2>/dev/null || true)"
if [[ "${code:-000}" != "200" ]]; then
  log "host not healthy at ${BASE_URL}/healthz (http=${code:-000})"
  log "start it via systemd user service: systemctl --user start cli-remote-control.service"
  exit 1
fi

log "enabling tailscale serve (https=443 -> ${BASE_URL})..."
if tailscale serve --bg --https=443 "${BASE_URL}" 2>/dev/null; then
  :
elif command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve --bg --https=443 "${BASE_URL}"; then
  :
else
  log "failed to configure tailscale serve (may need interactive sudo). Run in a terminal:"
  log "  sudo tailscale serve --bg --https=443 ${BASE_URL}"
  exit 1
fi

log "tailscale serve status:"
status_out="$(tailscale serve status 2>/dev/null || (command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve status) || true)"
printf '%s\n' "${status_out}"

# Best-effort: print the first URL token (may be a ts.net hostname).
url="$(printf '%s\n' "${status_out}" | awk 'NR==1{print $1"/"}')"
if [[ -n "${url}" && "${url}" != "/" ]]; then
  echo "serve_url=${url}"
fi
