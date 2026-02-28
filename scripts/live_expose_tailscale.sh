#!/usr/bin/env bash
#
# Live remote access (Tailscale Serve) for an already-running host.
# Keeps rc-host bound to localhost; exposes via tailnet WITHOUT clobbering
# existing HTTPS:443 Serve config (e.g. OpenClaw/qBittorrent).
#
# IMPORTANT:
# - Do NOT use `tailscale serve reset` anywhere (would wipe existing config).
# - Path-based mappings like `/rc` often break SPAs built for `/` (assets fetched from `/assets/...`).
#   The safe default is to use a dedicated HTTPS port (8443) for rc-host.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${RC_HOST_PORT:-8787}"
BASE_URL="http://127.0.0.1:${PORT}"
STATE_FILE="${ROOT}/host/.run/serve-mode.json"
PREFER_HTTPS_PORT="${RC_TAILSCALE_HTTPS_PORT:-8443}"
LEGACY_PATH="${RC_TAILSCALE_PATH:-/rc}"

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

mkdir -p "${ROOT}/host/.run"
chmod 0700 "${ROOT}/host/.run" 2>/dev/null || true

serve_cmd() {
  # Prefer no-sudo; fall back to passwordless sudo when required.
  if tailscale serve "$@" 2>/dev/null; then
    return 0
  fi
  if command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve "$@"; then
    return 0
  fi
  return 1
}

log "configuring tailscale serve: https:${PREFER_HTTPS_PORT} -> ${BASE_URL} (does not touch https:443)"
if serve_cmd --bg --https="${PREFER_HTTPS_PORT}" "${BASE_URL}"; then
  mode="port"
  https_port="${PREFER_HTTPS_PORT}"
  path=""
else
  # Fallback: legacy path mapping (may break SPAs, but better than nothing).
  log "failed to configure https:${PREFER_HTTPS_PORT}; attempting legacy path mapping on https:443 ${LEGACY_PATH} -> ${BASE_URL}"
  if serve_cmd --bg --https=443 --set-path="${LEGACY_PATH}" "${BASE_URL}"; then
    mode="path"
    https_port=443
    path="${LEGACY_PATH}"
  else
    log "failed to configure tailscale serve (may need interactive sudo). Try manually:"
    log "  sudo tailscale serve --bg --https=${PREFER_HTTPS_PORT} ${BASE_URL}"
    log "  # or legacy path mapping:"
    log "  sudo tailscale serve --bg --https=443 --set-path=${LEGACY_PATH} ${BASE_URL}"
    exit 1
  fi
fi

python3 - "${STATE_FILE}" "${mode}" "${https_port}" "${path}" <<'PY'
import json,sys,os
path=sys.argv[1]
mode=sys.argv[2]
httpsPort=int(sys.argv[3])
p=sys.argv[4] if len(sys.argv)>4 else ""
obj={"mode":mode,"httpsPort":httpsPort}
if mode=="path":
  obj["path"]=p
tmp=path+".tmp"
with open(tmp,"w",encoding="utf-8") as f:
  json.dump(obj,f,indent=2,sort_keys=True)
  f.write("\n")
os.replace(tmp,path)
PY

log "tailscale serve status:"
status_out="$(tailscale serve status 2>/dev/null || (command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve status) || true)"
printf '%s\n' "${status_out}"

# If we successfully switched to dedicated port mode, best-effort remove legacy /rc mapping
# (path-based SPA hosting is commonly broken). Do not touch '/'.
if [[ "${mode}" == "port" ]]; then
  json="$(tailscale serve status -json 2>/dev/null || (command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve status -json 2>/dev/null) || echo '{}')"
  if python3 -c 'import json,sys; obj=json.loads(sys.stdin.read() or "{}"); \
handlers={}; [handlers.update((site.get("Handlers") or {})) for site in (obj.get("Web") or {}).values()]; \
h=handlers.get("/rc"); sys.exit(0 if isinstance(h,dict) and h.get("Proxy")=="http://127.0.0.1:8787" else 1)' <<<"${json}" 2>/dev/null; then
    log "removing legacy https:443 /rc mapping (now using dedicated port https:${https_port})"
    serve_cmd --https=443 --set-path=/rc off || true
  fi
fi

# Print user-facing URLs.
if [[ "${mode}" == "port" ]]; then
  host_url="$(printf '%s\n' "${status_out}" | awk 'NR==1{print $1}')"
  if [[ -n "${host_url}" ]]; then
    if [[ "${host_url}" == *":${https_port}" ]]; then
      echo "serve_url=${host_url}/"
    else
      echo "serve_url=${host_url}:${https_port}/"
    fi
  fi
else
  host_url="$(printf '%s\n' "${status_out}" | awk 'NR==1{print $1}')"
  if [[ -n "${host_url}" ]]; then
    echo "serve_url=${host_url}${path}"
  fi
fi
