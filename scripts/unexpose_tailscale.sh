#!/usr/bin/env bash
#
# Disable Tailscale Serve exposure configured by expose_tailscale.sh.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

if ! command -v tailscale >/dev/null 2>&1; then
  log "tailscale: not found."
  exit 1
fi

log "resetting tailscale serve..."
if tailscale serve reset 2>/dev/null; then
  :
elif sudo -n tailscale serve reset; then
  :
else
  log "failed to reset tailscale serve (sudo may require a TTY/password)."
  log "try manually in an interactive terminal:"
  log "  sudo tailscale serve reset"
  exit 1
fi

log "tailscale serve status (after reset):"
tailscale serve status 2>/dev/null || sudo -n tailscale serve status || true
