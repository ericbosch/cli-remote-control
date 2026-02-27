#!/usr/bin/env bash
#
# Canonical diagnostics bundle (v5).
# - Produces: diag_YYYYMMDD_HHMMSS/ and diag_YYYYMMDD_HHMMSS.zip in repo root
# - Redacts tokens and *_API_KEY values from captured outputs
# - Never prints raw auth tokens (only path + byte length + sha256)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TS="$(date +%Y%m%d_%H%M%S)"
OUT_DIR="diag_${TS}"
ZIP_PATH="${OUT_DIR}.zip"

CMD_DIR="${OUT_DIR}/cmd"
FIX_DIR="${OUT_DIR}/fixtures"

mkdir -p "${CMD_DIR}" "${FIX_DIR}"

TOKEN_FILE="${ROOT}/host/.dev-token"
BASE_URL="http://127.0.0.1:8787"

maybe_start_host_bg() {
  # Canonical behavior: start/ensure host is running before collecting diagnostics.
  # Do not stop host at end (leave it running for dev).
  if [[ -x "${ROOT}/scripts/host_bg_start.sh" ]]; then
    "${ROOT}/scripts/host_bg_start.sh" 2>&1 | redact_stream > "${CMD_DIR}/host_bg_start.txt" || true
  else
    echo "host_bg_start: missing ${ROOT}/scripts/host_bg_start.sh" > "${CMD_DIR}/host_bg_start.txt"
  fi
}

redact_stream() {
  # Best-effort redaction for text/json/jsonl/ndjson.
  # Do NOT attempt to be perfect; also avoid leaking by not capturing secrets in argv where possible.
  perl -pe '
    s/(Authorization:\s*Bearer)\s+[^\s"]+/$1 REDACTED/gmi;
    s/([?&]token=)[^&\s"]+/$1REDACTED/gmi;
    s/("?(access_token|refresh_token)"?\s*:\s*")[^"]+/$1REDACTED/gmi;
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
  # Prints http code (or 000) without failing the script.
  local code
  code="$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 2 --max-time 5 "$1" 2>/dev/null || true)"
  if [[ -z "${code}" ]]; then
    echo "000"
  else
    echo "${code}"
  fi
}

auth_curl_to_file() {
  # Usage: auth_curl_to_file <url> <body_out_path> <http_code_out_path>
  local url="$1"
  local body_path="$2"
  local code_path="$3"
  if [[ ! -f "${TOKEN_FILE}" ]]; then
    echo "TOKEN_FILE_MISSING" > "${code_path}"
    return 0
  fi

  local token
  token="$(tr -d '\r\n' < "${TOKEN_FILE}")"

  # Keep token out of argv: provide header via curl config on stdin.
  local code
  code="$(
    curl -sS -o "${body_path}" -w '%{http_code}\n' -K - <<EOF 2>/dev/null || true
url = "${url}"
header = "Authorization: Bearer ${token}"
EOF
  )"
  printf '%s\n' "${code:-000}" > "${code_path}"

  # Redact any surprises from the body just in case.
  if [[ -f "${body_path}" ]]; then
    redact_stream < "${body_path}" > "${body_path}.redacted" || true
    mv -f "${body_path}.redacted" "${body_path}" || true
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

detect_api_keys_env() {
  local out_path="${CMD_DIR}/policy_env_api_keys.txt"
  {
    echo "Policy: API keys ignored by policy. Values are not recorded."
    echo
    echo "Detected env var NAMES matching *_API_KEY (may be empty):"
    env | awk -F= '{print $1}' | rg -n '^[A-Za-z0-9_]+_API_KEY$' || true
  } | redact_stream > "${out_path}"
}

host_pid_info() {
  local out_path="${CMD_DIR}/host_process_8787.txt"
  {
    echo "== ss (listeners) =="
    ss -lntp || true
    echo
    echo "== ss sport=:8787 (best-effort) =="
    ss -lntp 'sport = :8787' || true
    echo
    echo "== lsof :8787 (best-effort) =="
    if command -v lsof >/dev/null 2>&1; then
      lsof -nP -iTCP:8787 -sTCP:LISTEN || true
    else
      echo "lsof: not found"
    fi
    echo
    echo "== derived PIDs (best-effort) =="
    pids="$(
      ss -lntp 'sport = :8787' 2>/dev/null | perl -ne 'while(/pid=(\d+)/g){print "$1\n"}' | sort -u
    )"
    if [[ -z "${pids}" ]] && command -v lsof >/dev/null 2>&1; then
      pids="$(lsof -nP -t -iTCP:8787 -sTCP:LISTEN 2>/dev/null | sort -u || true)"
    fi
    if [[ -z "${pids}" ]]; then
      echo "No listener PID detected for :8787"
      exit 0
    fi
    echo "${pids}" | while read -r pid; do
      [[ -z "${pid}" ]] && continue
      echo "--- pid=${pid} ---"
      ps -p "${pid}" -o pid,ppid,user,etime,cmd || true
      if [[ -d "/proc/${pid}" ]]; then
        echo "cwd=$(readlink -f "/proc/${pid}/cwd" 2>/dev/null || echo '?')"
        echo "exe=$(readlink -f "/proc/${pid}/exe" 2>/dev/null || echo '?')"
        echo -n "cmdline="
        tr '\0' ' ' < "/proc/${pid}/cmdline" 2>/dev/null || true
        echo
      fi
    done
  } 2>&1 | redact_stream > "${out_path}"
}

write_summary() {
  local out_path="${OUT_DIR}/SUMMARY.txt"

  local unauth_code auth_code healthz_code
  unauth_code="$(cat "${CMD_DIR}/curl_unauth_sessions.http" 2>/dev/null || echo "000")"
  auth_code="$(cat "${CMD_DIR}/curl_auth_sessions.http" 2>/dev/null || echo "000")"
  healthz_code="$(cat "${CMD_DIR}/curl_healthz.http" 2>/dev/null || echo "000")"

  local host_listen="FAIL"
  if ss -lnt 2>/dev/null | rg -q '127\.0\.0\.1:8787\b'; then
    host_listen="PASS"
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

  local codex_schema="FAIL"
  if [[ -d "${FIX_DIR}/_codex_schema" ]] && find "${FIX_DIR}/_codex_schema" -type f | rg -q '.'; then
    codex_schema="PASS"
  fi

  local cursor_fixture="SKIP"
  if [[ -f "${FIX_DIR}/cursor-sample.full.ndjson" ]]; then
    if [[ -s "${FIX_DIR}/cursor-sample.full.ndjson" ]]; then
      # Validate that at least one line is JSON; avoid false PASS on plain-text error output.
      if python3 - <<'PY' "${FIX_DIR}/cursor-sample.full.ndjson" 2>/dev/null
import json,sys
p=sys.argv[1]
with open(p,'r',encoding='utf-8') as f:
  for line in f:
    line=line.strip()
    if not line:
      continue
    json.loads(line)
    print("ok")
    break
PY
      then
        cursor_fixture="PASS"
      else
        cursor_fixture="FAIL"
      fi
    else
      cursor_fixture="FAIL"
    fi
  fi

  local e2e="SKIP"
  if [[ -f "${CMD_DIR}/e2e_ws_first_message.json" ]]; then
    if [[ -s "${CMD_DIR}/e2e_ws_first_message.json" ]]; then
      e2e="PASS"
    else
      e2e="FAIL"
    fi
  fi

  local go="NO-GO"
  if [[ "${host_listen}" == "PASS" && "${unauth_ok}" == "PASS" && "${auth_ok}" == "PASS" && "${healthz_ok}" == "PASS" ]]; then
    if command -v codex >/dev/null 2>&1; then
      [[ "${codex_schema}" == "PASS" ]] && go="GO"
    else
      go="GO"
    fi
  fi

  {
    echo "${go}"
    echo
    printf 'host_listening_localhost=%s\n' "${host_listen}"
    printf 'unauth_sessions_401_403=%s (http=%s)\n' "${unauth_ok}" "${unauth_code}"
    printf 'auth_sessions_200=%s (http=%s)\n' "${auth_ok}" "${auth_code}"
    printf 'healthz_200=%s (http=%s)\n' "${healthz_ok}" "${healthz_code}"
    printf 'codex_schema_present=%s\n' "${codex_schema}"
    printf 'cursor_fixture_present=%s\n' "${cursor_fixture}"
    printf 'e2e_session_event=%s\n' "${e2e}"
  } > "${out_path}"
}

write_report_md() {
  # Human-friendly summary for sharing. Must be safe to publish: no raw tokens.
  local out_path="${OUT_DIR}/REPORT.md"
  local summary_path="${OUT_DIR}/SUMMARY.txt"

  local head sha status
  sha="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
  status="$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')"
  if [[ "${status}" == "0" ]]; then
    head="clean"
  else
    head="dirty(${status})"
  fi

  {
    echo "# Diagnostics Report (v5)"
    echo
    echo "- generated_at: ${TS}"
    echo "- diag_dir: ${OUT_DIR}/"
    echo "- diag_zip: ${ZIP_PATH}"
    echo "- git_head: ${sha}"
    echo "- git_status: ${head}"
    echo
    echo "## SUMMARY.txt"
    echo
    echo '```'
    if [[ -f "${summary_path}" ]]; then
      cat "${summary_path}"
    else
      echo "missing: ${summary_path}"
    fi
    echo '```'
    echo
    echo "## Token diagnostics (safe)"
    echo
    echo '```'
    if [[ -f "${CMD_DIR}/token_diagnostics.txt" ]]; then
      cat "${CMD_DIR}/token_diagnostics.txt"
    else
      echo "missing: ${CMD_DIR}/token_diagnostics.txt"
    fi
    echo '```'
    echo
    echo "## How to reproduce"
    echo
    echo "- Start host (idempotent): \`./scripts/host_bg_start.sh\`"
    echo "- Run diagnostics: \`./scripts/collect_diag_bundle_v5.sh\`"
    echo "- Read results: \`cat ${OUT_DIR}/SUMMARY.txt\`"
    echo
    echo "## Notes"
    echo
    echo "- Redaction: bearer tokens, query tokens, access/refresh tokens, and \`*_API_KEY\` values are redacted from captured outputs."
    echo "- This report is generated to be safe to share; it must not contain raw secrets."
  } > "${out_path}"
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
  if ! command -v zip >/dev/null 2>&1; then
    echo "zip: not found; skipping zip creation" > "${out_path}"
    return 0
  fi
  (zip -qr "${ZIP_PATH}" "${OUT_DIR}" && echo "created ${ZIP_PATH}") > "${out_path}" 2>&1
}

main() {
  echo "Writing bundle: ${OUT_DIR}/ and ${ZIP_PATH}"

  maybe_start_host_bg

  # Policy/environment inventory
  detect_api_keys_env
  write_token_diagnostics

  # Host observations
  host_pid_info
  capture_cmd "ss_listeners.txt" ss -lntp

  echo -n "$(safe_http_code "${BASE_URL}/api/sessions")" > "${CMD_DIR}/curl_unauth_sessions.http"
  echo -n "$(safe_http_code "${BASE_URL}/healthz")" > "${CMD_DIR}/curl_healthz.http"

  auth_curl_to_file "${BASE_URL}/api/sessions" "${CMD_DIR}/curl_auth_sessions.json" "${CMD_DIR}/curl_auth_sessions.http"

  # Code pointers (snippets/greps)
  capture_shell "snippet_server_go.txt" "nl -ba host/internal/server/server.go | sed -n '1,260p'"
  capture_shell "snippet_main_go.txt" "nl -ba host/cmd/rc-host/main.go | sed -n '1,220p'"
  capture_shell "snippet_ws_go.txt" "nl -ba host/internal/server/ws.go | sed -n '1,220p'"
  capture_shell "grep_pointers.txt" "rg -n \"generate-dev-token|\\.dev-token|cfg\\.Token|Authorization|ws-ticket\" -S host | head -n 200"

  # Codex fixtures
  capture_cmd "codex_version.txt" codex --version
  capture_shell "codex_app_server_help_head.txt" "codex app-server --help | head -n 80"
  if command -v codex >/dev/null 2>&1; then
    capture_shell "codex_schema_generate.txt" "mkdir -p \"${FIX_DIR}/_codex_schema\"; codex app-server generate-json-schema --out \"${FIX_DIR}/_codex_schema\""
  else
    echo "codex: not found" > "${CMD_DIR}/codex_schema_generate.txt"
  fi

  # Cursor/agent fixtures (best-effort, structured NDJSON first)
  capture_shell "cursor_detect.txt" "if command -v cursor >/dev/null 2>&1; then echo \"entrypoint=cursor agent\"; elif command -v agent >/dev/null 2>&1; then echo \"entrypoint=agent\"; else echo \"entrypoint=unavailable\"; fi"

  if command -v cursor >/dev/null 2>&1; then
    # Cursor IDE installs the `cursor` CLI; agent is the supported subcommand.
    (
      # Enforce policy: never use API key auth.
      for k in $(env | awk -F= '{print $1}' | rg '^[A-Za-z0-9_]+_API_KEY$' || true); do unset "${k}" || true; done
      timeout 20s cursor agent --print --output-format stream-json --stream-partial-output "Reply ONLY with OK" \
        > "${FIX_DIR}/cursor-sample.full.ndjson" 2> "${CMD_DIR}/cursor-sample.stderr.txt" || true
    )
  elif command -v agent >/dev/null 2>&1; then
    (
      for k in $(env | awk -F= '{print $1}' | rg '^[A-Za-z0-9_]+_API_KEY$' || true); do unset "${k}" || true; done
      timeout 20s agent --print --output-format stream-json --stream-partial-output "Reply ONLY with OK" \
        > "${FIX_DIR}/cursor-sample.full.ndjson" 2> "${CMD_DIR}/cursor-sample.stderr.txt" || true
    )
  fi

  if [[ -f "${FIX_DIR}/cursor-sample.full.ndjson" ]]; then
    redact_stream < "${FIX_DIR}/cursor-sample.full.ndjson" > "${FIX_DIR}/cursor-sample.full.ndjson.redacted" || true
    mv -f "${FIX_DIR}/cursor-sample.full.ndjson.redacted" "${FIX_DIR}/cursor-sample.full.ndjson" || true
    head -n 200 "${FIX_DIR}/cursor-sample.full.ndjson" > "${FIX_DIR}/cursor-sample.ndjson" || true
  fi
  if [[ -f "${CMD_DIR}/cursor-sample.stderr.txt" ]]; then
    redact_stream < "${CMD_DIR}/cursor-sample.stderr.txt" > "${CMD_DIR}/cursor-sample.stderr.txt.redacted" || true
    mv -f "${CMD_DIR}/cursor-sample.stderr.txt.redacted" "${CMD_DIR}/cursor-sample.stderr.txt" || true
  fi

  # E2E validation (best-effort): create a shell session and observe at least 1 WS message.
  if [[ -f "${TOKEN_FILE}" ]]; then
    session_json="${CMD_DIR}/e2e_create_session.json"
    session_http="${CMD_DIR}/e2e_create_session.http"
    token="$(tr -d '\r\n' < "${TOKEN_FILE}")"

    http_code="$(
      curl -sS -o "${session_json}" -w '%{http_code}\n' -K - <<EOF 2>/dev/null || true
url = "${BASE_URL}/api/sessions"
header = "Authorization: Bearer ${token}"
header = "Content-Type: application/json"
request = "POST"
data = "{\"engine\":\"shell\",\"name\":\"diag-e2e\"}"
EOF
    )"
    printf '%s\n' "${http_code:-000}" > "${session_http}"
    if [[ -f "${session_json}" ]]; then
      redact_stream < "${session_json}" > "${session_json}.redacted" || true
      mv -f "${session_json}.redacted" "${session_json}" || true
    fi

    if [[ "${http_code}" == "201" || "${http_code}" == "200" ]]; then
      session_id="$(
        python3 - "${session_json}" <<'PY'
import json,sys
p=sys.argv[1]
with open(p,'r',encoding='utf-8') as f:
  obj=json.load(f)
print(obj.get('id',''))
PY
        2>/dev/null || true
      )"

      if [[ -n "${session_id}" ]]; then
        node - <<'NODE' "${TOKEN_FILE}" "${session_id}" > "${CMD_DIR}/e2e_ws_first_message.json" 2> "${CMD_DIR}/e2e_ws_first_message.stderr.txt" || true
const fs = require('fs');
const tokenFile = process.argv[2];
const sessionId = process.argv[3];
const token = fs.readFileSync(tokenFile, 'utf8').trim();
const wsUrl = `ws://127.0.0.1:8787/ws/sessions/${encodeURIComponent(sessionId)}?token=${encodeURIComponent(token)}`;
const ws = new WebSocket(wsUrl);
let done = false;
function finish(code) {
  if (done) return;
  done = true;
  try { ws.close(); } catch {}
  process.exit(code);
}
ws.onmessage = (ev) => {
  process.stdout.write(String(ev.data));
  finish(0);
};
ws.onerror = () => finish(2);
setTimeout(() => finish(3), 5000);
NODE
        redact_stream < "${CMD_DIR}/e2e_ws_first_message.json" > "${CMD_DIR}/e2e_ws_first_message.json.redacted" || true
        mv -f "${CMD_DIR}/e2e_ws_first_message.json.redacted" "${CMD_DIR}/e2e_ws_first_message.json" || true
        if [[ -f "${CMD_DIR}/e2e_ws_first_message.stderr.txt" ]]; then
          redact_stream < "${CMD_DIR}/e2e_ws_first_message.stderr.txt" > "${CMD_DIR}/e2e_ws_first_message.stderr.txt.redacted" || true
          mv -f "${CMD_DIR}/e2e_ws_first_message.stderr.txt.redacted" "${CMD_DIR}/e2e_ws_first_message.stderr.txt" || true
        fi
      fi
    fi
  fi

  write_summary
  write_report_md
  write_manifest
  zip_bundle
}

main "$@"
