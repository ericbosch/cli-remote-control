#!/usr/bin/env bash
#
# Installs and starts the user-level systemd service (no sudo required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

UNIT_SRC="${ROOT}/deploy/systemd/cli-remote-control.user.service"
UNIT_DIR="${HOME}/.config/systemd/user"
UNIT_DST="${UNIT_DIR}/cli-remote-control.service"

mkdir -p "${UNIT_DIR}"
cp -f "${UNIT_SRC}" "${UNIT_DST}"

systemctl --user daemon-reload
systemctl --user enable --now cli-remote-control.service

systemctl --user --no-pager --full status cli-remote-control.service || true

