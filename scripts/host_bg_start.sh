#!/usr/bin/env bash
#
# Start rc-host in the background, idempotently.
# Preference order:
#  1) tmux session "rc-host"
#  2) nohup + pidfile (host/.run/rc-host.pid)
#
# Security/policy:
# - Binds to 127.0.0.1 (default in host)
# - Never prints raw tokens
# - Any *_API_KEY env vars are logged as "ignored by policy" and cleared for the started process
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RUN_DIR="${ROOT}/host/.run"
mkdir -p "${RUN_DIR}"
chmod 0700 "${RUN_DIR}" 2>/dev/null || true

LOG_FILE="${RUN_DIR}/rc-host.log"
PID_FILE="${RUN_DIR}/rc-host.pid"
TMUX_SESSION="rc-host"
TOKEN_FILE="${ROOT}/host/.dev-token"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

BIND="${RC_HOST_BIND:-127.0.0.1}"
PORT="${RC_HOST_PORT:-8787}"
WEB_DIR="${RC_HOST_WEB_DIR:-}"
if [[ -z "${WEB_DIR}" ]] && [[ -f "${ROOT}/web/dist/index.html" ]]; then
  WEB_DIR="${ROOT}/web/dist"
fi
BASE_URL="http://127.0.0.1:${PORT}"

http_code() {
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 1 --max-time 2 "$1" 2>/dev/null || true)"
  if [[ -z "${code}" ]]; then
    echo "000"
  else
    echo "${code}"
  fi
}

token_diag() {
  if [[ ! -f "${TOKEN_FILE}" ]]; then
    log "token: missing file=${TOKEN_FILE}"
    return 0
  fi
  local sha token_bytes
  sha="$(sha256sum "${TOKEN_FILE}" | awk '{print $1}')"
  token_bytes="$(tr -d '\r\n' < "${TOKEN_FILE}" | wc -c | tr -d ' ')"
  log "token: file=${TOKEN_FILE} len=${token_bytes} sha256=${sha}"
}

policy_clear_api_keys() {
  # Clear in current shell so tmux/nohup child does not inherit.
  local keys
  keys="$(env | awk -F= '{print $1}' | rg -n '^[A-Za-z0-9_]+_API_KEY$' || true)"
  if [[ -n "${keys}" ]]; then
    while IFS= read -r k; do
      [[ -z "${k}" ]] && continue
      if [[ -n "${!k:-}" ]]; then
        log "${k}: ignored by policy"
        unset "${k}" || true
      fi
    done <<<"${keys}"
  fi
}

already_healthy() {
  [[ "$(http_code "${BASE_URL}/healthz")" == "200" ]]
}

wait_ready_or_dump_logs() {
  local deadline i
  deadline=100
  for i in $(seq 1 "${deadline}"); do
    if already_healthy; then
      return 0
    fi
    sleep 0.1
  done

  log "host did not become healthy within 10s: ${BASE_URL}/healthz"
  if tmux has-session -t "${TMUX_SESSION}" 2>/dev/null; then
    tmux capture-pane -pt "${TMUX_SESSION}" -S -200 > "${RUN_DIR}/rc-host.tmux.last200.log" 2>/dev/null || true
    log "last tmux output saved: ${RUN_DIR}/rc-host.tmux.last200.log"
    tail -n 50 "${RUN_DIR}/rc-host.tmux.last200.log" 2>/dev/null || true
  fi
  if [[ -f "${LOG_FILE}" ]]; then
    log "last nohup log lines: ${LOG_FILE}"
    tail -n 50 "${LOG_FILE}" || true
  fi
  return 1
}

start_tmux() {
  log "starting via tmux session: ${TMUX_SESSION}"
  rm -f "${LOG_FILE}" "${PID_FILE}" 2>/dev/null || true
  local web_flag=""
  if [[ -n "${WEB_DIR}" ]]; then
    web_flag="--web-dir \"${WEB_DIR}\""
  fi
  if ! tmux new-session -d -s "${TMUX_SESSION}" "cd \"${ROOT}/host\" && exec go run ./cmd/rc-host serve --generate-dev-token --log-dir \"${ROOT}/logs\" --bind \"${BIND}\" --port \"${PORT}\" ${web_flag}"; then
    log "tmux start failed (permission or environment). Falling back to nohup."
    return 1
  fi
  wait_ready_or_dump_logs
  token_diag
  log "host running (tmux). Attach with: tmux attach -t ${TMUX_SESSION}"
}

start_nohup() {
  log "starting via nohup (pidfile): ${PID_FILE}"
  rm -f "${LOG_FILE}" 2>/dev/null || true
  (
    cd "${ROOT}/host"
    args=(go run ./cmd/rc-host serve --generate-dev-token --log-dir "${ROOT}/logs" --bind "${BIND}" --port "${PORT}")
    if [[ -n "${WEB_DIR}" ]]; then
      args+=(--web-dir "${WEB_DIR}")
    fi
    nohup "${args[@]}" >> "${LOG_FILE}" 2>&1 &
    echo "$!" > "${PID_FILE}"
  )
  wait_ready_or_dump_logs
  token_diag
  log "host running (nohup). Logs: ${LOG_FILE}"
}

main() {
  policy_clear_api_keys

  if already_healthy; then
    log "already running: ${BASE_URL}"
    token_diag
    exit 0
  fi

  if tmux has-session -t "${TMUX_SESSION}" 2>/dev/null; then
    log "tmux session exists but host not healthy; restarting session: ${TMUX_SESSION}"
    tmux kill-session -t "${TMUX_SESSION}" || true
  fi

  if [[ -f "${PID_FILE}" ]]; then
    pid="$(cat "${PID_FILE}" 2>/dev/null || true)"
    if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
      log "pidfile process exists but host not healthy; stopping pid=${pid}"
      kill "${pid}" 2>/dev/null || true
      sleep 0.5
    fi
    rm -f "${PID_FILE}" 2>/dev/null || true
  fi

  if command -v tmux >/dev/null 2>&1; then
    if ! start_tmux; then
      start_nohup
    fi
  else
    start_nohup
  fi
}

main "$@"
