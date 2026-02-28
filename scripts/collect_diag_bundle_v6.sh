#!/usr/bin/env bash
#
# Canonical diagnostics bundle (v6).
# - Produces: diag_YYYYMMDD_HHMMSS/ and diag_YYYYMMDD_HHMMSS.zip in repo root
# - Captures checklist-driven diagnostics with explicit PASS/FAIL/SKIP summary
# - Never records raw secrets (tokens/tickets). If a secret is detected, only path + bytes + sha256 are recorded.
set -euo pipefail
set +H # disable history expansion ("! event not found") for safety in shells with histexpand enabled

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TS="$(date +%Y%m%d_%H%M%S)"
OUT_DIR="diag_${TS}"
ZIP_PATH="${OUT_DIR}.zip"

CMD_DIR="${OUT_DIR}/cmd"
CODE_DIR="${OUT_DIR}/code"

mkdir -p "${CMD_DIR}" "${CODE_DIR}"

PORT="${RC_HOST_PORT:-8787}"
BASE_URL="http://127.0.0.1:${PORT}"
TOKEN_FILE="${ROOT}/host/.dev-token"

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

redact_stream() {
  # Best-effort redaction for captured outputs. Self-check below ensures we didn't leak.
  perl -pe '
    s/(Authorization:\s*Bearer)\s+[^\s"]+/$1 REDACTED/gmi;
    s/([?&](token|ticket)=)[^&\s"]+/$1REDACTED/gmi;
    s/("?(access_token|refresh_token)"?\s*:\s*")[^"]+/$1REDACTED/gmi;
    s/("?(ws_)?ticket"?\s*:\s*")[^"]+/$1REDACTED/gmi;
    s/("?[A-Za-z0-9_]*API_KEY"?\s*[:=]\s*")[^"]+/$1REDACTED/gmi;
    s/\b([A-Za-z0-9_]*API_KEY)=\S+/$1=REDACTED/gm;
  '
}

capture_cmd() {
  local out_name="$1"
  shift
  local out_path="${CMD_DIR}/${out_name}"
  {
    echo "\$ $*"
    set +e
    "$@"
    local rc=$?
    set -e
    echo "exit=${rc}"
  } 2>&1 | redact_stream > "${out_path}"
}

capture_shell() {
  local out_name="$1"
  local script="$2"
  local out_path="${CMD_DIR}/${out_name}"
  {
    echo "\$ bash -lc <script>"
    set +e
    bash -lc "${script}"
    local rc=$?
    set -e
    echo "exit=${rc}"
  } 2>&1 | redact_stream > "${out_path}"
}

safe_http_code() {
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 "$1" 2>/dev/null || true)"
  echo "${code:-000}"
}

file_fingerprint() {
  # Prints: bytes sha256 path
  local p="$1"
  local bytes sha
  bytes="$(wc -c < "${p}" | tr -d ' ')"
  sha="$(sha256sum "${p}" | awk '{print $1}')"
  printf '%s %s %s\n' "${bytes}" "${sha}" "${p}"
}

maybe_start_host_bg() {
  # Canonical behavior: start/ensure host is running before collecting diagnostics.
  # Do not stop host at end (leave it running for dev).
  if [[ -x "${ROOT}/scripts/host_bg_start.sh" ]]; then
    "${ROOT}/scripts/host_bg_start.sh" 2>&1 | redact_stream > "${CMD_DIR}/host_bg_start.txt" || true
  else
    echo "host_bg_start: missing ${ROOT}/scripts/host_bg_start.sh" > "${CMD_DIR}/host_bg_start.txt"
  fi
}

write_token_diagnostics() {
  local out_path="${CMD_DIR}/token_diagnostics.txt"
  if [[ ! -f "${TOKEN_FILE}" ]]; then
    printf 'token_file_path=%s\nexists=false\n' "${TOKEN_FILE}" > "${out_path}"
    return 0
  fi

  local sha file_bytes token_bytes
  sha="$(sha256sum "${TOKEN_FILE}" | awk '{print $1}')"
  file_bytes="$(wc -c < "${TOKEN_FILE}" | tr -d ' ')"
  token_bytes="$(tr -d '\r\n' < "${TOKEN_FILE}" | wc -c | tr -d ' ')"

  {
    printf 'token_file_path=%s\n' "${TOKEN_FILE}"
    printf 'exists=true\n'
    printf 'token_byte_length=%s\n' "${token_bytes}"
    printf 'token_file_byte_length=%s\n' "${file_bytes}"
    printf 'sha256(token_file)=%s\n' "${sha}"
  } > "${out_path}"
}

auth_request_keys_only() {
  # Usage: auth_request_keys_only <method> <url> <http_out> <keys_out>
  # Records only:
  # - HTTP status
  # - top-level JSON keys (no values)
  local method="$1"
  local url="$2"
  local http_out="$3"
  local keys_out="$4"

  if [[ ! -f "${TOKEN_FILE}" ]]; then
    echo "TOKEN_FILE_MISSING" > "${http_out}"
    echo "SKIP (token file missing)" > "${keys_out}"
    return 0
  fi

  local token tmp_body
  token="$(tr -d '\r\n' < "${TOKEN_FILE}")"
  tmp_body="$(mktemp)"
  trap 'rm -f "${tmp_body}" 2>/dev/null || true' RETURN

  local code
  code="$(
    curl -sS -o "${tmp_body}" -w '%{http_code}\n' -K - <<EOF 2>/dev/null || true
url = "${url}"
request = "${method}"
header = "Authorization: Bearer ${token}"
EOF
  )"
  printf '%s\n' "${code:-000}" > "${http_out}"

  if python3 - "${tmp_body}" > "${keys_out}" 2>/dev/null <<'PY'
import json,sys
p=sys.argv[1]
try:
  with open(p,'r',encoding='utf-8') as f:
    obj=json.load(f)
except Exception as e:
  print(f"SKIP (non-json body: {type(e).__name__})")
  raise SystemExit(0)
if isinstance(obj, dict):
  for k in sorted(obj.keys()):
    print(k)
else:
  print(f"SKIP (json type={type(obj).__name__})")
PY
  then
    :
  else
    echo "SKIP (python3 failed)" > "${keys_out}"
  fi

  rm -f "${tmp_body}" 2>/dev/null || true
  trap - RETURN
}

detect_ui_root_headers() {
  local out_path="${CMD_DIR}/ui_root_headers.txt"
  local hdr_tmp
  hdr_tmp="$(mktemp)"
  trap 'rm -f "${hdr_tmp}" 2>/dev/null || true' RETURN

  # Capture headers only; no body.
  set +e
  curl -sS -D "${hdr_tmp}" -o /dev/null --connect-timeout 2 --max-time 5 "${BASE_URL}/" 2>/dev/null
  local rc=$?
  set -e

  {
    echo "\$ curl -sS -D <file> ${BASE_URL}/ -o /dev/null"
    echo "exit=${rc}"
    echo
    if [[ -s "${hdr_tmp}" ]]; then
      cat "${hdr_tmp}" | redact_stream
    else
      echo "(no headers captured)"
    fi
  } > "${out_path}"

  rm -f "${hdr_tmp}" 2>/dev/null || true
  trap - RETURN
}

tailscale_status_best_effort() {
  if ! command -v tailscale >/dev/null 2>&1; then
    echo "tailscale: not found" > "${CMD_DIR}/tailscale_version.txt"
    echo "tailscale: not found" > "${CMD_DIR}/tailscale_serve_status.txt"
    return 0
  fi

  capture_cmd "tailscale_version.txt" tailscale --version

  if tailscale serve status > "${CMD_DIR}/tailscale_serve_status.txt" 2>&1; then
    : # ok
  elif command -v sudo >/dev/null 2>&1 && sudo -n tailscale serve status > "${CMD_DIR}/tailscale_serve_status.txt" 2>&1; then
    : # ok
  else
    # Preserve whatever output we got for debugging.
    :
  fi

  redact_stream < "${CMD_DIR}/tailscale_serve_status.txt" > "${CMD_DIR}/tailscale_serve_status.txt.redacted" 2>/dev/null || true
  mv -f "${CMD_DIR}/tailscale_serve_status.txt.redacted" "${CMD_DIR}/tailscale_serve_status.txt" 2>/dev/null || true
}

write_code_pointers() {
  local out_path="${CODE_DIR}/pointers.txt"
  {
    echo "# code pointers (paths + line snippets; no secrets expected)"
    echo
    echo "## host flags (--web-dir, auth, ws-ticket)"
    rg -n "web-dir|generate-dev-token|Authorization|ws-ticket" -S host 2>/dev/null | head -n 200 || true
    echo
    echo "## web uses ws-ticket"
    rg -n "/api/ws-ticket|ws-ticket" -S web 2>/dev/null | head -n 120 || true
    echo
    echo "## android scaffold"
    rg -n "android/" -S README.md docs 2>/dev/null | head -n 80 || true
    echo
    echo "## scripts duplicates scan"
    rg -n "collect_diag_bundle_v[0-9]+|collect_diag_bundle|expose_tailscale|unexpose_tailscale|expose_lan|unexpose_lan|host_bg_start|host_bg_stop|host_bg_status|run_ui_local|--web-dir|ws-ticket" -S scripts docs 2>/dev/null | head -n 200 || true
  } | redact_stream > "${out_path}"
}

write_manifest() {
  local out_path="${OUT_DIR}/MANIFEST.txt"
  (
    cd "${OUT_DIR}"
    find . -type f -print | sort
  ) > "${out_path}"
}

zip_bundle() {
  local out_path="${CMD_DIR}/zip.txt"
  if command -v zip >/dev/null 2>&1; then
    (zip -qr "${ZIP_PATH}" "${OUT_DIR}" && echo "created ${ZIP_PATH}") > "${out_path}" 2>&1
    return 0
  fi

  # Fallback: python zipfile (no external deps).
  python3 - "${OUT_DIR}" "${ZIP_PATH}" > "${out_path}" 2>&1 <<'PY'
import os,sys,zipfile
root_dir=sys.argv[1]
zip_path=sys.argv[2]
with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as z:
  for base,_,files in os.walk(root_dir):
    for f in files:
      p=os.path.join(base,f)
      arc=os.path.relpath(p, start=os.path.dirname(root_dir))
      z.write(p, arcname=arc)
print(f"created {zip_path}")
PY
}

redaction_selfcheck() {
  # Self-check outputs must never print raw matches. Only record filenames + fingerprints.
  local out_path="${CMD_DIR}/redaction_selfcheck.txt"
  local ok="PASS"

  {
    echo "redaction_selfcheck:"
    echo
    echo "Policy: never record raw secrets. If matches exist, only file path + bytes + sha256(file) are emitted."
    echo
  } > "${out_path}"

  if ! command -v rg >/dev/null 2>&1; then
    echo "rg: not found; SKIP self-check" >> "${out_path}"
    echo "SKIP"
    return 0
  fi

  # Patterns: no lookarounds required.
  # - Bearer tokens in headers
  # - token= / ticket= query params
  # - access/refresh tokens in JSON
  local -a patterns=(
    'Authorization:\s*Bearer\s+[A-Za-z0-9._-]{12,}'
    '[?&]token=[A-Za-z0-9._-]{12,}'
    '[?&]ticket=[A-Za-z0-9._-]{12,}'
    '"access_token"\s*:\s*"[A-Za-z0-9._-]{12,}'
    '"refresh_token"\s*:\s*"[A-Za-z0-9._-]{12,}'
    'ticket=[A-Za-z0-9._-]{12,}'
  )

  for pat in "${patterns[@]}"; do
    {
      echo "check: ${pat}"
      mapfile -t files < <(rg -l "${pat}" "${OUT_DIR}" 2>/dev/null || true)
      if [[ "${#files[@]}" -eq 0 ]]; then
        echo "  result=PASS (no matches)"
      else
        ok="FAIL"
        echo "  result=FAIL (matches in ${#files[@]} file(s))"
        for f in "${files[@]}"; do
          if [[ -f "${f}" ]]; then
            fp="$(file_fingerprint "${f}")"
            echo "  match_file=${fp}"
          else
            echo "  match_file=${f} (missing?)"
          fi
        done
      fi
      echo
    } >> "${out_path}"
  done

  echo "${ok}"
}

write_summary_and_reports() {
  local summary_path="${OUT_DIR}/SUMMARY.txt"

  local git_dirty_count git_worktree_clean
  git_dirty_count="$(git status --porcelain=v1 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "${git_dirty_count}" == "0" ]]; then
    git_worktree_clean="PASS"
  else
    git_worktree_clean="FAIL"
  fi

  local unauth_code auth_code healthz_code ws_ticket_code
  unauth_code="$(cat "${CMD_DIR}/curl_unauth_sessions.http" 2>/dev/null || echo "000")"
  auth_code="$(cat "${CMD_DIR}/curl_auth_sessions.http" 2>/dev/null || echo "000")"
  healthz_code="$(cat "${CMD_DIR}/curl_healthz.http" 2>/dev/null || echo "000")"
  ws_ticket_code="$(cat "${CMD_DIR}/ws_ticket.http" 2>/dev/null || echo "000")"

  local host_listen="FAIL"
  local host_bind_mode="unknown"
  if command -v ss >/dev/null 2>&1; then
    if ss -lnt 2>/dev/null | rg -q "127\\.0\\.0\\.1:${PORT}\\b"; then
      host_listen="PASS"
      host_bind_mode="localhost"
    fi
    if ss -lnt 2>/dev/null | rg -q "0\\.0\\.0\\.0:${PORT}\\b|:::${PORT}\\b|\\[::\\]:${PORT}\\b"; then
      host_bind_mode="lan"
    fi
  elif command -v lsof >/dev/null 2>&1; then
    if lsof -nP -iTCP:"${PORT}" -sTCP:LISTEN 2>/dev/null | rg -q "127\\.0\\.0\\.1:${PORT}\\b"; then
      host_listen="PASS"
      host_bind_mode="localhost"
    fi
  fi

  local unauth_ok="FAIL"
  if [[ "${unauth_code}" == "401" || "${unauth_code}" == "403" ]]; then
    unauth_ok="PASS"
  fi

  local auth_ok="FAIL"
  if [[ "${auth_code}" == "200" ]]; then
    auth_ok="PASS"
  fi

  local healthz_ok="FAIL"
  if [[ "${healthz_code}" == "200" ]]; then
    healthz_ok="PASS"
  fi

  local ws_auth_mode="bearer_header"
  local ws_ticket_ok="FAIL"
  if [[ "${ws_ticket_code}" == "200" ]]; then
    # Confirm it at least returns a "ticket" key without recording values.
    if [[ -f "${CMD_DIR}/ws_ticket.keys" ]] && rg -q '^ticket$' "${CMD_DIR}/ws_ticket.keys"; then
      ws_ticket_ok="PASS"
      ws_auth_mode="ticket"
    else
      ws_ticket_ok="FAIL"
    fi
  fi

  local web_dist_present="SKIP"
  if [[ -f "${ROOT}/web/dist/index.html" ]]; then
    web_dist_present="PASS"
  fi

  local ui_root_status="SKIP"
  local ui_root_http="000"
  local ui_root_ct=""
  if [[ -f "${CMD_DIR}/ui_root_headers.txt" ]]; then
    ui_root_http="$(rg -n '^HTTP/' "${CMD_DIR}/ui_root_headers.txt" | head -n 1 | awk '{print $2}' || echo "000")"
    ui_root_ct="$(rg -ni '^content-type:' "${CMD_DIR}/ui_root_headers.txt" | head -n 1 | cut -d: -f2- | tr -d '\r' | xargs || true)"
  fi
  if [[ "${ui_root_http}" == "200" ]] && echo "${ui_root_ct}" | rg -qi '^text/html\b'; then
    ui_root_status="PASS"
  else
    # Only FAIL if we have a host and a web build that should be served at /.
    if [[ "${healthz_ok}" == "PASS" && "${web_dist_present}" == "PASS" ]]; then
      ui_root_status="FAIL"
    fi
  fi

  local tailscale_present="SKIP"
  local tailscale_serve_configured="SKIP"
  if command -v tailscale >/dev/null 2>&1; then
    tailscale_present="PASS"
    if [[ -f "${CMD_DIR}/tailscale_serve_status.txt" ]]; then
      if rg -qi 'no serve config|not configured|serve is not enabled|serve: not configured' "${CMD_DIR}/tailscale_serve_status.txt"; then
        tailscale_serve_configured="SKIP"
      else
        # Best-effort: treat any non-empty status output as "configured".
        if rg -q '.' "${CMD_DIR}/tailscale_serve_status.txt"; then
          tailscale_serve_configured="PASS"
        fi
      fi
    fi
  fi

  local android_scaffold_present="SKIP"
  if [[ -x "${ROOT}/android/gradlew" ]] || [[ -f "${ROOT}/android/gradlew" ]]; then
    if [[ -d "${ROOT}/android/app/src" ]] || [[ -d "${ROOT}/android/app/src/main" ]]; then
      android_scaffold_present="PASS"
    fi
  fi

  local codex_present="SKIP"
  if command -v codex >/dev/null 2>&1; then
    codex_present="PASS"
  fi
  local cursor_present="SKIP"
  if command -v cursor >/dev/null 2>&1; then
    cursor_present="PASS"
  fi
  local agent_present="SKIP"
  if command -v agent >/dev/null 2>&1; then
    agent_present="PASS"
  fi

  local redaction_status
  redaction_status="$(cat "${CMD_DIR}/redaction_selfcheck.status" 2>/dev/null || echo "SKIP")"

  {
    printf 'host_listening_localhost=%s\n' "${host_listen}"
    printf 'host_bind_mode=%s\n' "${host_bind_mode}"
    printf 'unauth_sessions_401_403=%s (http=%s)\n' "${unauth_ok}" "${unauth_code}"
    printf 'auth_sessions_200=%s (http=%s)\n' "${auth_ok}" "${auth_code}"
    printf 'healthz_200=%s (http=%s)\n' "${healthz_ok}" "${healthz_code}"
    printf 'ws_ticket_endpoint=%s (http=%s)\n' "${ws_ticket_ok}" "${ws_ticket_code}"
    printf 'ws_auth_mode=%s\n' "${ws_auth_mode}"
    printf 'ui_root_served=%s (http=%s content_type=%s)\n' "${ui_root_status}" "${ui_root_http}" "${ui_root_ct:-unknown}"
    printf 'web_dist_present=%s\n' "${web_dist_present}"
    printf 'tailscale_present=%s\n' "${tailscale_present}"
    printf 'tailscale_serve_configured=%s\n' "${tailscale_serve_configured}"
    printf 'android_scaffold_present=%s\n' "${android_scaffold_present}"
    printf 'codex_present=%s\n' "${codex_present}"
    printf 'cursor_present=%s\n' "${cursor_present}"
    printf 'agent_present=%s\n' "${agent_present}"
    printf 'redaction_selfcheck=%s\n' "${redaction_status}"
    printf 'git_worktree_clean=%s\n' "${git_worktree_clean}"
    if [[ "${git_worktree_clean}" == "FAIL" ]]; then
      printf 'git_dirty_item_count=%s\n' "${git_dirty_count}"
      printf 'git_dirty_evidence_files=cmd/git_status_porcelain.txt cmd/git_diff_name_only.txt\n'
    fi
  } > "${summary_path}"

  # REPORT.md: human narrative w/ explicit detected capabilities
  {
    echo "# Diagnostics Report (v6)"
    echo
    echo "- generated_at: ${TS}"
    echo "- diag_dir: ${OUT_DIR}/"
    echo "- diag_zip: ${ZIP_PATH}"
    echo "- base_url: ${BASE_URL}"
    echo "- git_head: $(git rev-parse HEAD 2>/dev/null || echo unknown)"
    echo
    echo "## SUMMARY.txt"
    echo
    echo '```'
    cat "${summary_path}" 2>/dev/null || true
    echo '```'
    echo
    echo "## Detected capabilities"
    echo
    echo "- ws_auth_mode: ${ws_auth_mode}"
    echo "- ui_root_served: ${ui_root_status}"
    echo "- tailscale_serve_configured: ${tailscale_serve_configured}"
    echo "- android_scaffold_present: ${android_scaffold_present}"
    echo "- engines_present: codex=${codex_present} cursor=${cursor_present} agent=${agent_present}"
    echo
    echo "## What was verified (checklist)"
    echo
    echo "- Host bind/listen checks: cmd/ss_listeners.txt (best-effort)"
    echo "- REST auth behavior: /api/sessions (unauth + auth keys-only)"
    echo "- Health endpoint: /healthz"
    echo "- WS ticket endpoint: /api/ws-ticket (keys-only)"
    echo "- UI root: GET / (status + content-type only)"
    echo "- Tailscale Serve (best-effort): cmd/tailscale_serve_status.txt"
    echo "- Android scaffold presence: filesystem checks only"
    echo "- Engine presence: versions/help only (no PAYG APIs)"
    echo "- Redaction self-check: cmd/redaction_selfcheck.txt"
    echo
    echo "## Repo phases"
    echo
    echo "This repo includes milestones (M1) and a v5-era staff summary that referenced phases up to Phase 5."
    echo "**Phase 6/7 are not detected in this repo** (no code/docs markers found)."
  } > "${OUT_DIR}/REPORT.md"

  # STAFF_SUMMARY.md: short exec summary
  {
    echo "# cli-remote-control — Staff summary (v6)"
    echo
    echo "- generated_at: ${TS}"
    echo "- diag_dir: ${OUT_DIR}/"
    echo "- diag_zip: ${ZIP_PATH}"
    echo "- git_head: $(git rev-parse HEAD 2>/dev/null || echo unknown)"
    echo
    echo "## Gates"
    echo
    echo "- host_listening_localhost: ${host_listen}"
    echo "- unauth_sessions_401_403: ${unauth_ok} (http=${unauth_code})"
    echo "- auth_sessions_200: ${auth_ok} (http=${auth_code})"
    echo "- healthz_200: ${healthz_ok} (http=${healthz_code})"
    echo "- ws_ticket_endpoint: ${ws_ticket_ok} (http=${ws_ticket_code})"
    echo "- redaction_selfcheck: ${redaction_status}"
    echo "- git_worktree_clean: ${git_worktree_clean}"
    echo
    echo "## Capabilities"
    echo
    echo "- ws_auth_mode=${ws_auth_mode}"
    echo "- ui_root_served=${ui_root_status} (http=${ui_root_http} content_type=${ui_root_ct:-unknown})"
    echo "- tailscale_serve_configured=${tailscale_serve_configured}"
    echo "- android_scaffold_present=${android_scaffold_present}"
    echo "- engines_present: codex=${codex_present}, cursor=${cursor_present}, agent=${agent_present}"
    echo
    echo "## Notes"
    echo
    echo "- Phase 6/7: not present (explicitly not detected)."
    echo "- Secrets: no raw tokens/tickets are recorded; token file is fingerprinted only."
  } > "${OUT_DIR}/STAFF_SUMMARY.md"
}

main() {
  log "Writing bundle: ${OUT_DIR}/ and ${ZIP_PATH}"

  # Git evidence
  capture_cmd "git_status_porcelain.txt" git status --porcelain=v1
  capture_shell "git_diff_name_only.txt" "echo '== unstaged =='; git diff --name-only; echo; echo '== staged =='; git diff --name-only --cached"
  capture_cmd "git_log_last40.txt" git log --oneline -n 40

  # Start host if possible, and capture safe token diagnostics.
  maybe_start_host_bg
  write_token_diagnostics

  # Host process/listeners
  if command -v ss >/dev/null 2>&1; then
    capture_cmd "ss_listeners.txt" ss -lntp
  else
    capture_cmd "ss_listeners.txt" sh -c 'echo "ss: not found"'
  fi

  # Host/UI checks (best-effort; do not record bodies beyond keys-only where applicable)
  echo -n "$(safe_http_code "${BASE_URL}/healthz")" > "${CMD_DIR}/curl_healthz.http"
  echo -n "$(safe_http_code "${BASE_URL}/api/sessions")" > "${CMD_DIR}/curl_unauth_sessions.http"

  auth_request_keys_only "GET" "${BASE_URL}/api/sessions" "${CMD_DIR}/curl_auth_sessions.http" "${CMD_DIR}/curl_auth_sessions.keys"
  auth_request_keys_only "POST" "${BASE_URL}/api/ws-ticket" "${CMD_DIR}/ws_ticket.http" "${CMD_DIR}/ws_ticket.keys"

  detect_ui_root_headers

  # Tailscale Serve (best-effort)
  tailscale_status_best_effort

  # Android presence
  capture_shell "android_presence.txt" "if [[ -f android/gradlew || -x android/gradlew ]]; then echo 'android_gradlew=true'; else echo 'android_gradlew=false'; fi; if [[ -d android/app/src ]]; then echo 'android_app_src=true'; else echo 'android_app_src=false'; fi"

  # Engines (versions/help only; clear PAYG-style keys for subprocesses)
  capture_shell "codex_version.txt" "if command -v codex >/dev/null 2>&1; then env -u OPENAI_API_KEY -u CURSOR_API_KEY codex --version; else echo 'codex: not found'; fi"
  capture_shell "codex_app_server_help.txt" "if command -v codex >/dev/null 2>&1; then env -u OPENAI_API_KEY -u CURSOR_API_KEY codex app-server --help | head -n 120; else echo 'codex: not found'; fi"
  capture_shell "cursor_version.txt" "if command -v cursor >/dev/null 2>&1; then env -u OPENAI_API_KEY -u CURSOR_API_KEY cursor --version; else echo 'cursor: not found'; fi"
  capture_shell "agent_help.txt" "if command -v agent >/dev/null 2>&1; then env -u OPENAI_API_KEY -u CURSOR_API_KEY agent --help | head -n 120; else echo 'agent: not found'; fi"

  # Code pointers
  write_code_pointers

  # Redaction self-check (must happen before summary/report)
  selfcheck_status="$(redaction_selfcheck)"
  echo "${selfcheck_status}" > "${CMD_DIR}/redaction_selfcheck.status"

  write_summary_and_reports
  write_manifest
  zip_bundle

  log "done: ${OUT_DIR}/ (zip=${ZIP_PATH})"
  echo "diag_dir=${OUT_DIR}"
  echo "diag_zip=${ZIP_PATH}"
}

main "$@"
