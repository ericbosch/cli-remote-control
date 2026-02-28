#!/usr/bin/env bash
#
# Live remote access (Tailscale Serve) for an already-running host.
# Keeps rc-host bound to localhost; exposes via tailnet WITHOUT clobbering
# existing HTTPS:443 Serve config (e.g. OpenClaw). Prefer a dedicated path.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PORT="${RC_HOST_PORT:-8787}"
BASE_URL="http://127.0.0.1:${PORT}"
STATE_FILE="${ROOT}/host/.run/serve-mode.json"

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

serve_status_json() {
  tailscale serve status -json 2>/dev/null || (command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve status -json 2>/dev/null) || echo '{}'
}

choose_path() {
  local json="$1"
  # Prefer /rc. If it already exists and points to our backend, keep using it.
  if python3 -c 'import json,sys; obj=json.loads(sys.stdin.read() or "{}"); \
handlers={}; [handlers.update((site.get("Handlers") or {})) for site in (obj.get("Web") or {}).values()]; \
h=handlers.get("/rc"); sys.exit(0 if isinstance(h,dict) and h.get("Proxy")=="http://127.0.0.1:8787" else 1)' <<<"${json}" 2>/dev/null; then
    echo "/rc"
    return 0
  fi
  # If /rc is taken (points elsewhere), use an alternative.
  if python3 -c 'import json,sys; obj=json.loads(sys.stdin.read() or "{}"); \
handlers={}; [handlers.update((site.get("Handlers") or {})) for site in (obj.get("Web") or {}).values()]; \
sys.exit(0 if "/rc" in handlers else 1)' <<<"${json}" 2>/dev/null; then
    echo "/cli-remote"
    return 0
  fi
  echo "/rc"
}

remediate_root_if_it_points_to_rc_host() {
  # If HTTPS:443 "/" currently proxies to rc-host (8787) and 8080 looks like OpenClaw,
  # restore "/" to 8080 before adding our dedicated path mapping.
  local json="$1"
  if ! python3 -c 'import json,sys; obj=json.loads(sys.stdin.read() or "{}"); \
root=None; \
for site in (obj.get("Web") or {}).values(): \
  h=(site.get("Handlers") or {}).get("/"); \
  root=(h.get("Proxy") if isinstance(h,dict) else None) or root; \
  if root is not None: break; \
sys.exit(0 if root=="http://127.0.0.1:8787" else 1)' <<<"${json}" 2>/dev/null; then
    return 0
  fi

  # Heuristic check: 8080 responds with 200 and text/html.
  local hdr
  hdr="$(curl -sS -D - http://127.0.0.1:8080/ -o /dev/null --connect-timeout 1 --max-time 2 2>/dev/null || true)"
  if ! printf '%s\n' "${hdr}" | rg -q '^HTTP/.* 200\\b'; then
    return 0
  fi
  if ! printf '%s\n' "${hdr}" | rg -qi '^content-type:\\s*text/html\\b'; then
    return 0
  fi

  log "detected https:443 '/' mapping points to rc-host; restoring '/' to localhost:8080 (OpenClaw heuristic)"
  if ! serve_cmd --bg --https=443 --set-path=/ http://127.0.0.1:8080; then
    log "failed to restore '/' mapping on https:443 (non-fatal; will still attempt /rc mapping)"
  fi
}

json="$(serve_status_json)"
remediate_root_if_it_points_to_rc_host "${json}"
json="$(serve_status_json)"
path="$(choose_path "${json}")"

log "configuring tailscale serve path mapping: https:443 ${path} -> ${BASE_URL}"
if serve_cmd --bg --https=443 --set-path="${path}" "${BASE_URL}"; then
  mode="path"
  https_port=443
else
  log "failed to add https:443 path mapping; falling back to dedicated port https:8443 -> ${BASE_URL}"
  if serve_cmd --bg --https=8443 "${BASE_URL}"; then
    mode="port"
    https_port=8443
    path=""
  else
    log "failed to configure tailscale serve (may need interactive sudo). Try manually:"
    log "  sudo tailscale serve --bg --https=443 --set-path=${path} ${BASE_URL}"
    log "  # or fallback:"
    log "  sudo tailscale serve --bg --https=8443 ${BASE_URL}"
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

# Best-effort: print the first URL token (may be a ts.net hostname).
url="$(printf '%s\n' "${status_out}" | awk 'NR==1{print $1"/"}')"
if [[ -n "${url}" && "${url}" != "/" ]]; then
  echo "serve_url=${url}"
fi
