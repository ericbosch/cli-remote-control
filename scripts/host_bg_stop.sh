#!/usr/bin/env bash
#
# Stop background rc-host idempotently.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

RUN_DIR="${ROOT}/host/.run"
PID_FILE="${RUN_DIR}/rc-host.pid"
LOG_FILE="${RUN_DIR}/rc-host.log"
TMUX_SESSION="rc-host"
BASE_URL="http://127.0.0.1:8787"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

http_code() {
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 1 --max-time 2 "$1" 2>/dev/null || true)"
  if [[ -z "${code}" ]]; then
    echo "000"
  else
    echo "${code}"
  fi
}

wait_down() {
  for _ in $(seq 1 100); do
    if [[ "$(http_code "${BASE_URL}/healthz")" != "200" ]]; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

stopped_any=false

if command -v tmux >/dev/null 2>&1 && tmux has-session -t "${TMUX_SESSION}" 2>/dev/null; then
  log "stopping tmux session: ${TMUX_SESSION}"
  tmux send-keys -t "${TMUX_SESSION}" C-c 2>/dev/null || true
  sleep 0.5
  tmux kill-session -t "${TMUX_SESSION}" 2>/dev/null || true
  stopped_any=true
fi

if [[ -f "${PID_FILE}" ]]; then
  pid="$(cat "${PID_FILE}" 2>/dev/null || true)"
  if [[ -n "${pid}" ]] && kill -0 "${pid}" 2>/dev/null; then
    log "stopping nohup pid=${pid}"
    kill "${pid}" 2>/dev/null || true
    stopped_any=true
  fi
  rm -f "${PID_FILE}" 2>/dev/null || true
fi

if wait_down; then
  log "host is stopped"
else
  log "host still appears up (healthz=200) after stop attempt"
  if [[ -f "${LOG_FILE}" ]]; then
    log "last nohup log lines: ${LOG_FILE}"
    tail -n 50 "${LOG_FILE}" || true
  fi
  exit 1
fi

if [[ "${stopped_any}" == "false" ]]; then
  log "already stopped (no tmux session, no live pidfile)"
fi
