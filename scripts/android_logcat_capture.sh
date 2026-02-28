#!/usr/bin/env bash
set -euo pipefail
set +H

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

PKG="${RC_ANDROID_PKG:-com.ericbosch.rcclient}"
OUT="${1:-/tmp/rc_android_logcat.txt}"

if ! command -v adb >/dev/null 2>&1; then
  echo "adb: not found" > "${OUT}"
  exit 0
fi

if ! adb get-state >/dev/null 2>&1; then
  echo "adb: no device" > "${OUT}"
  exit 0
fi

redact() {
  perl -pe '
    s/(Authorization:\s*Bearer)\s+[^\s"]+/$1 REDACTED/gmi;
    s/([?&](token|ticket)=)[^&\s"]+/$1REDACTED/gmi;
    s/("?(access_token|refresh_token)"?\s*:\s*")[^"]+/$1REDACTED/gmi;
    s/\b([A-Za-z0-9_]*API_KEY)=\S+/$1=REDACTED/gm;
  '
}

# Capture last ~4000 lines and filter for relevant tags.
adb logcat -d -v time | tail -n 4000 | rg -n "(AndroidRuntime|OkHttp|kotlinx\\.serialization|${PKG})" -S 2>/dev/null | redact > "${OUT}" || true
echo "wrote ${OUT}"

