#!/usr/bin/env bash
#
# Cleanup known local runtime/build artifacts that commonly dirty the worktree.
# Safe-by-default: removes only a small allowlist of paths.
#
# Usage:
#   ./scripts/cleanup_local_artifacts.sh
#   ./scripts/cleanup_local_artifacts.sh --deep
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

DEEP=false
if [[ "${#}" -gt 0 ]]; then
  case "${1:-}" in
    --deep) DEEP=true ;;
    -h|--help)
      cat <<'EOF'
cleanup_local_artifacts.sh

Removes known local runtime/build artifacts.

Default removes:
  - diag_* (directories)
  - diag_*.zip
  - host/.run
  - host/*.log

--deep also removes:
  - web/node_modules
  - web/dist
  - android/.gradle
  - android/**/build
EOF
      exit 0
      ;;
    *)
      echo "unknown arg: ${1}" >&2
      exit 2
      ;;
  esac
fi

log() { printf '%s %s\n' "$(date -Is)" "$*" >&2; }

rm_if_exists() {
  local p="$1"
  if [[ -e "${p}" ]]; then
    log "delete: ${p}"
    rm -rf -- "${p}"
  else
    log "skip:   ${p} (missing)"
  fi
}

rm_glob_if_any() {
  local g="$1"
  local found=0
  shopt -s nullglob
  local matches=( ${g} )
  shopt -u nullglob
  if [[ "${#matches[@]}" -eq 0 ]]; then
    log "skip:   ${g} (no matches)"
    return 0
  fi
  for p in "${matches[@]}"; do
    found=1
    rm_if_exists "${p}"
  done
  [[ "${found}" -eq 1 ]]
}

log "cleanup: safe allowlist (deep=${DEEP})"

rm_glob_if_any "diag_*/" || true
rm_glob_if_any "diag_*.zip" || true
rm_if_exists "host/.run"
rm_glob_if_any "host/"'*.log' || true

if [[ "${DEEP}" == "true" ]]; then
  rm_if_exists "web/node_modules"
  rm_if_exists "web/dist"
  rm_if_exists "android/.gradle"

  # Only remove build dirs under android/, never outside the repo.
  while IFS= read -r p; do
    [[ -z "${p}" ]] && continue
    rm_if_exists "${p}"
  done < <(find android -type d -name build -print 2>/dev/null || true)
fi

log "done"

