#!/usr/bin/env bash
#
# Installs and starts the system-level systemd service (requires sudo).
# This script intentionally does NOT attempt to pass sudo passwords.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

UNIT_SRC="${ROOT}/deploy/systemd/cli-remote-control.service"
UNIT_DST="/etc/systemd/system/cli-remote-control.service"

if ! command -v sudo >/dev/null 2>&1; then
  echo "sudo: not found; cannot install system service" >&2
  exit 1
fi

if ! sudo -n true 2>/dev/null; then
  echo "sudo requires an interactive password. Run the following in your terminal:" >&2
  echo "  sudo install -m 0644 \"${UNIT_SRC}\" \"${UNIT_DST}\"" >&2
  echo "  sudo systemctl daemon-reload" >&2
  echo "  sudo systemctl enable --now cli-remote-control.service" >&2
  exit 2
fi

sudo install -m 0644 "${UNIT_SRC}" "${UNIT_DST}"
sudo systemctl daemon-reload
sudo systemctl enable --now cli-remote-control.service
sudo systemctl --no-pager --full status cli-remote-control.service || true

