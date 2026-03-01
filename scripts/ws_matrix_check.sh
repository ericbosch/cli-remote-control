#!/usr/bin/env bash
set -euo pipefail
set +H

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

PORT="${RC_HOST_PORT:-8787}"
LOCAL_BASE="${RC_LOCAL_BASE:-http://127.0.0.1:${PORT}}"
TOKEN_FILE="${RC_TOKEN_FILE:-${ROOT}/host/.dev-token}"

SERVE_BASE="${RC_SERVE_BASE:-}"
if [[ -z "${SERVE_BASE}" ]] && [[ -f "${ROOT}/host/.run/serve-mode.json" ]] && command -v python3 >/dev/null 2>&1; then
  serve_port="$(python3 -c 'import json,sys; obj=json.load(open(sys.argv[1],"r",encoding="utf-8")); print(obj.get("httpsPort",""))' "${ROOT}/host/.run/serve-mode.json" 2>/dev/null || true)"
  if [[ -n "${serve_port}" ]] && command -v tailscale >/dev/null 2>&1; then
    # Prefer the configured port (avoid accidentally selecting another service on :443).
    SERVE_BASE="$(tailscale serve status 2>/dev/null | rg -o "https://[^ ]+:${serve_port}" | head -n1 || true)"
  fi
fi
if [[ -z "${SERVE_BASE}" ]] && command -v tailscale >/dev/null 2>&1; then
  # Fallback: first https:// URL shown in `tailscale serve status`.
  SERVE_BASE="$(tailscale serve status 2>/dev/null | rg -o 'https://[^ ]+' | head -n1 || true)"
fi

TAIL_IP="${RC_TAIL_IP:-}"
if [[ -z "${TAIL_IP}" ]] && command -v tailscale >/dev/null 2>&1; then
  TAIL_IP="$(tailscale ip -4 2>/dev/null | head -n1 || true)"
fi

if [[ ! -f "${TOKEN_FILE}" ]]; then
  echo "ws_matrix_check=SKIP token file missing: ${TOKEN_FILE}"
  exit 0
fi

cd "${ROOT}/host"

args=(--token-file "${TOKEN_FILE}" --local-base "${LOCAL_BASE}")
if [[ -n "${SERVE_BASE}" ]]; then
  args+=(--serve-base "${SERVE_BASE}")
fi
if [[ -n "${TAIL_IP}" ]]; then
  args+=(--tail-ip "${TAIL_IP}")
fi

go run ./cmd/ws-matrix-check "${args[@]}"
