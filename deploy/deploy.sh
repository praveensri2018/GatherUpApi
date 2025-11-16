#!/usr/bin/env bash
# Place: deploy/deploy.sh
# Usage: ./deploy/deploy.sh [environment]
# ... (header unchanged) ...

set -euo pipefail
IFS=$'\n\t'

### === CONFIG - change to suit your server ===
ENV="${1:-production}"

REPO_DIR="/srv/apps/gatherup"
BACKEND_SUBPATH="backend/go"
GO_BINARY_NAME="gatherup-server"
INSTALL_DIR="/opt/gatherup"
RELEASES_DIR="${INSTALL_DIR}/releases"
SYSTEMD_SERVICE="gatherup-server"
KEEP_RELEASES=5
GO_CMD="${GO_CMD:-go}"

log(){ printf "[%s] %s\n" "$(date '+%F %T')" "$*"; }
err(){ printf "[%s] ERROR: %s\n" "$(date '+%F %T')" "$*" >&2; }

# tell every time
log "=== tell every time: starting deploy.sh (env=${ENV}) ==="
log "Using REPO_DIR=${REPO_DIR}, INSTALL_DIR=${INSTALL_DIR}, RELEASES_DIR=${RELEASES_DIR}"

# sanity checks unchanged...
if [ -z "${REPO_DIR}" ] || [ -z "${BACKEND_SUBPATH}" ]; then
  err "REPO_DIR or BACKEND_SUBPATH not set"
  exit 2
fi

if ! command -v "${GO_CMD}" >/dev/null 2>&1; then
  err "go not found on PATH (set GO_CMD to the go binary if custom). Aborting."
  exit 2
fi

if [ "$(id -u)" -eq 0 ]; then
  log "Running as root; recommended to run as 'deploy' user. Script will still proceed."
fi

log "Starting deploy (env=${ENV})"

# fetch code...
# (keep the same fetch logic you have, but add tell-every-time logging)
log "Fetching latest from origin..."
git fetch --all --prune
git checkout main || git checkout -b main origin/main || true
git reset --hard origin/main

# build
log "Building Go backend..."
cd "${REPO_DIR}/${BACKEND_SUBPATH}"
${GO_CMD} mod download
export CGO_ENABLED=0
${GO_CMD} env -w GO111MODULE=on >/dev/null 2>&1 || true
${GO_CMD} build -v -o "${GO_BINARY_NAME}" ./cmd/server

if [ ! -x "${GO_BINARY_NAME}" ]; then
  err "Build failed: ${REPO_DIR}/${BACKEND_SUBPATH}/${GO_BINARY_NAME} not found or not executable"
  exit 1
fi

# install binary to releases
log "Installing binary to ${INSTALL_DIR} (requires sudo for install and restart)..."
sudo mkdir -p "${RELEASES_DIR}" || { err "sudo mkdir -p ${RELEASES_DIR} failed"; exit 1; }
ts=$(date +%s)
release_path="${RELEASES_DIR}/${ts}"
log "Creating release path: ${release_path}"
sudo mkdir -p "${release_path}" || { err "sudo mkdir -p ${release_path} failed"; exit 1; }

log "Copying binary to release path (preserving mode)"
sudo cp -p "${GO_BINARY_NAME}" "${release_path}/${GO_BINARY_NAME}"
sudo chown root:root "${release_path}/${GO_BINARY_NAME}"
sudo chmod 0755 "${release_path}/${GO_BINARY_NAME}"

log "Symlinking new release to ${INSTALL_DIR}/${GO_BINARY_NAME}"
sudo ln -sfn "${release_path}/${GO_BINARY_NAME}" "${INSTALL_DIR}/${GO_BINARY_NAME}"

log "Pruning old releases (keep ${KEEP_RELEASES})..."
cd "${RELEASES_DIR}"
ls -1t | sed -n "$((KEEP_RELEASES+1)),\$p" | xargs -r sudo rm -rf --

log "Reloading systemd and restarting service ${SYSTEMD_SERVICE}..."
sudo systemctl daemon-reload || true
if ! sudo systemctl restart "${SYSTEMD_SERVICE}"; then
  err "Service restart failed; dumping journal for debugging"
  sudo journalctl -u "${SYSTEMD_SERVICE}" -n 200 --no-pager -o short-iso || true
  exit 1
fi

log "Service restart succeeded; recent logs:"
sudo journalctl -u "${SYSTEMD_SERVICE}" -n 120 --no-pager -o short-iso || true

log "=== tell every time: deploy finished successfully (release=${release_path}) ==="
exit 0
