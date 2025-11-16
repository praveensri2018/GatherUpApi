    #!/usr/bin/env bash
# Place: deploy/deploy.sh
# Usage: ./deploy/deploy.sh [environment]
# Example: ./deploy/deploy.sh production
#
# This script:
#  - updates repo to origin/main
#  - builds the Go backend (backend/go)
#  - places binary under /opt/gatherup/releases/<ts>/gatherup-server
#  - symlinks /opt/gatherup/gatherup-server -> latest release
#  - restarts systemd service gatherup-server (via sudo)
#
# Run as the 'deploy' user. Script will use sudo for system-wide operations.

set -euo pipefail
IFS=$'\n\t'

### === CONFIG - change to suit your server ===
ENV="${1:-production}"

REPO_DIR="/srv/apps/gatherup"            # where the repo is checked out
BACKEND_SUBPATH="backend/go"             # backend module path (relative to repo)
GO_BINARY_NAME="gatherup-server"         # compiled binary base name
INSTALL_DIR="/opt/gatherup"              # where binary is installed
RELEASES_DIR="${INSTALL_DIR}/releases"   # releases directory
SYSTEMD_SERVICE="gatherup-server"        # systemd service name to restart
KEEP_RELEASES=5                          # how many old releases to keep
GO_CMD="${GO_CMD:-go}"                   # override if go binary is in a custom path

# logging helper
log(){ printf "[%s] %s\n" "$(date '+%F %T')" "$*"; }
err(){ printf "[%s] ERROR: %s\n" "$(date '+%F %T')" "$*" >&2; }

# sanity checks
if [ -z "${REPO_DIR}" ] || [ -z "${BACKEND_SUBPATH}" ]; then
  err "REPO_DIR or BACKEND_SUBPATH not set"
  exit 2
fi

# ensure we have go
if ! command -v "${GO_CMD}" >/dev/null 2>&1; then
  err "go not found on PATH (set GO_CMD to the go binary if custom). Aborting."
  exit 2
fi

# switch to deploy user recommended; warn if run as root
if [ "$(id -u)" -eq 0 ]; then
  log "Running as root; recommended to run as 'deploy' user. Script will still proceed."
fi

log "Starting deploy (env=${ENV})"

# 1) fetch latest code (safe update)
if [ ! -d "${REPO_DIR}" ]; then
  log "Repo not found at ${REPO_DIR} — cloning placeholder (update URL below)"
  # Replace the URL below with your repository SSH URL if you want auto-clone
  git clone git@github.com:YOUR-ORG/YOUR-REPO.git "${REPO_DIR}"
fi

cd "${REPO_DIR}"

log "Fetching latest from origin..."
# ensure we are using main branch; change branch name if your default is different
git fetch --all --prune
git checkout main || git checkout -b main origin/main || true
git reset --hard origin/main

# 2) build backend
log "Building Go backend..."
cd "${REPO_DIR}/${BACKEND_SUBPATH}"

# download modules
${GO_CMD} mod download

# build (static-ish)
export CGO_ENABLED=0
${GO_CMD} env -w GO111MODULE=on >/dev/null 2>&1 || true
${GO_CMD} build -v -o "${GO_BINARY_NAME}" ./cmd/server

# sanity: ensure build produced binary
if [ ! -x "${GO_BINARY_NAME}" ]; then
  err "Build failed: ${REPO_DIR}/${BACKEND_SUBPATH}/${GO_BINARY_NAME} not found or not executable"
  exit 1
fi

# 3) install binary into releases folder
log "Installing binary to ${INSTALL_DIR} (requires sudo for install and restart)..."
sudo mkdir -p "${RELEASES_DIR}"
ts=$(date +%s)
release_path="${RELEASES_DIR}/${ts}"
sudo mkdir -p "${release_path}"
# copy binary (preserve executable bit)
sudo cp -p "${GO_BINARY_NAME}" "${release_path}/${GO_BINARY_NAME}"
sudo chown root:root "${release_path}/${GO_BINARY_NAME}"
sudo chmod 0755 "${release_path}/${GO_BINARY_NAME}"

# atomic symlink update
sudo ln -sfn "${release_path}/${GO_BINARY_NAME}" "${INSTALL_DIR}/${GO_BINARY_NAME}"

# 4) cleanup old releases
log "Pruning old releases (keep ${KEEP_RELEASES})..."
cd "${RELEASES_DIR}"
# list releases sorted by time (newest first), keep first KEEP_RELEASES, remove remainder
ls -1t | sed -n "$((KEEP_RELEASES+1)),\$p" | xargs -r sudo rm -rf --

# 5) restart service
log "Reloading systemd and restarting service ${SYSTEMD_SERVICE}..."
# reload systemd (safe if unit file unchanged)
sudo systemctl daemon-reload || true
sudo systemctl restart "${SYSTEMD_SERVICE}"

# show recent status lines for quick debug
log "Service status (last 120 chars of output):"
sudo systemctl status "${SYSTEMD_SERVICE}" --no-pager -l | sed -n '1,200p' || true

log "Deploy finished successfully (release=${release_path})."
exit 0
