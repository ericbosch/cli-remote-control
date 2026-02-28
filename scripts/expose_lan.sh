#!/usr/bin/env bash
#
# Optional LAN exposure (opt-in):
# - Adds a ufw allowlist rule to permit LAN CIDR -> tcp/8787
# - Restarts rc-host bound to 0.0.0.0:8787 so phones on Wi‑Fi can reach it
#
# WARNING: This exposes the host service to your LAN. Use only on a trusted network.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${RC_HOST_PORT:-8787}"
RULE_FILE="${ROOT}/host/.run/ufw_rc_host_lan_rule.txt"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

if ! command -v ufw >/dev/null 2>&1; then
  log "ufw: not found; cannot safely configure LAN allowlist."
  exit 1
fi

detect_lan_cidr() {
  local dev addr
  dev="$(ip -4 route show default 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="dev"){print $(i+1); exit}}')"
  if [[ -z "${dev}" ]]; then
    echo "192.168.0.0/16"
    return 0
  fi
  addr="$(ip -4 addr show dev "${dev}" 2>/dev/null | awk '/inet /{print $2; exit}')"
  if [[ -z "${addr}" ]]; then
    echo "192.168.0.0/16"
    return 0
  fi
  python3 - "${addr}" <<'PY'
import ipaddress,sys
iface=ipaddress.ip_interface(sys.argv[1])
print(str(iface.network))
PY
}

LAN_CIDR="$(detect_lan_cidr)"
log "LAN allowlist CIDR=${LAN_CIDR} port=${PORT}"
log "WARNING: exposing rc-host to LAN. This is opt-in; prefer Tailscale Serve."

if [[ -f "${ROOT}/web/package.json" ]] && [[ ! -f "${ROOT}/web/dist/index.html" ]]; then
  log "web/dist missing; building web UI so the host can serve it at / ..."
  (cd "${ROOT}/web" && npm ci && npm run build)
fi

log "adding ufw rule (allow from ${LAN_CIDR} to tcp/${PORT})..."
if sudo -n ufw allow from "${LAN_CIDR}" to any port "${PORT}" proto tcp >/dev/null; then
  :
else
  log "ufw allow failed (sudo may require a TTY/password). Try manually:"
  log "  sudo ufw allow from ${LAN_CIDR} to any port ${PORT} proto tcp"
  exit 1
fi

mkdir -p "${ROOT}/host/.run"
chmod 0700 "${ROOT}/host/.run" 2>/dev/null || true
printf 'cidr=%s\nport=%s\n' "${LAN_CIDR}" "${PORT}" > "${RULE_FILE}"

log "restarting host bound to 0.0.0.0:${PORT}..."
"${ROOT}/scripts/host_bg_stop.sh" || true
RC_HOST_BIND=0.0.0.0 RC_HOST_PORT="${PORT}" "${ROOT}/scripts/host_bg_start.sh"

log "done. Verify on another LAN device: http://<LAN_IP>:${PORT}/"

