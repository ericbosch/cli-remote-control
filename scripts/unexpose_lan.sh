#!/usr/bin/env bash
#
# Undo optional LAN exposure configured by expose_lan.sh:
# - Removes ufw allowlist rule(s) added for rc-host
# - Restarts rc-host bound back to 127.0.0.1:8787 (safe default)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RULE_FILE="${ROOT}/host/.run/ufw_rc_host_lan_rule.txt"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

if ! command -v ufw >/dev/null 2>&1; then
  log "ufw: not found."
  exit 1
fi

cidr=""
port="8787"
if [[ -f "${RULE_FILE}" ]]; then
  cidr="$(rg -n '^cidr=' "${RULE_FILE}" | head -n 1 | cut -d= -f2- || true)"
  port="$(rg -n '^port=' "${RULE_FILE}" | head -n 1 | cut -d= -f2- || true)"
fi
if [[ -z "${cidr}" ]]; then
  log "no rule marker found at ${RULE_FILE}; will not delete any ufw rules automatically."
else
  log "removing ufw rule(s) for CIDR=${cidr} port=${port}..."
  # Delete matching numbered rules (reverse order).
  mapfile -t nums < <(sudo -n ufw status numbered 2>/dev/null | rg -n "\\b${port}/tcp\\b" | rg -n "\\bALLOW IN\\b" | rg -n "\\b${cidr}\\b" | perl -ne 'if(/\\[(\\d+)\\]/){print "$1\\n"}' | sort -nr)
  if [[ "${#nums[@]}" -eq 0 ]]; then
    log "no matching numbered rules found; you may need to delete manually: sudo ufw status numbered"
  else
    for n in "${nums[@]}"; do
      yes | sudo -n ufw delete "${n}" >/dev/null || true
    done
    log "removed ${#nums[@]} ufw rule(s)"
  fi
  rm -f "${RULE_FILE}" 2>/dev/null || true
fi

log "restarting host bound back to 127.0.0.1:8787..."
"${ROOT}/scripts/host_bg_stop.sh" || true
RC_HOST_BIND=127.0.0.1 RC_HOST_PORT=8787 "${ROOT}/scripts/host_bg_start.sh"

log "LAN exposure disabled."

