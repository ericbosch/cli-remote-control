#!/usr/bin/env bash
#
# Status for background rc-host.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RUN_DIR="${ROOT}/host/.run"
PID_FILE="${RUN_DIR}/rc-host.pid"
LOG_FILE="${RUN_DIR}/rc-host.log"
TMUX_SESSION="rc-host"
BASE_URL="http://127.0.0.1:8787"
TOKEN_FILE="${ROOT}/host/.dev-token"

http_code() {
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 1 --max-time 2 "$1" 2>/dev/null || true)"
  if [[ -z "${code}" ]]; then
    echo "000"
  else
    echo "${code}"
  fi
}

echo "healthz_http=$(http_code "${BASE_URL}/healthz")"
echo "sessions_unauth_http=$(http_code "${BASE_URL}/api/sessions")"

if [[ -f "${TOKEN_FILE}" ]]; then
  sha="$(sha256sum "${TOKEN_FILE}" | awk '{print $1}')"
  token_bytes="$(tr -d '\r\n' < "${TOKEN_FILE}" | wc -c | tr -d ' ')"
  echo "token_file=${TOKEN_FILE}"
  echo "token_len=${token_bytes}"
  echo "token_sha256=${sha}"
else
  echo "token_file=${TOKEN_FILE} (missing)"
fi

if command -v tmux >/dev/null 2>&1 && tmux has-session -t "${TMUX_SESSION}" 2>/dev/null; then
  echo "method=tmux session=${TMUX_SESSION}"
else
  echo "method=tmux (not running)"
fi

if [[ -f "${PID_FILE}" ]]; then
  pid="$(cat "${PID_FILE}" 2>/dev/null || true)"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    echo "method=nohup pid=${pid}"
  else
    echo "method=nohup pidfile_present_but_dead pid=${pid:-unknown}"
  fi
else
  echo "method=nohup (no pidfile)"
fi

if command -v lsof >/dev/null 2>&1; then
  echo "--- lsof :8787 ---"
  lsof -nP -iTCP:8787 -sTCP:LISTEN || true
fi

if [[ -f "${LOG_FILE}" ]]; then
  echo "--- tail ${LOG_FILE} ---"
  tail -n 30 "${LOG_FILE}" || true
fi
