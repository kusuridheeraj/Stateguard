#!/usr/bin/env sh
set -eu

SOURCE_ROOT="${SOURCE_ROOT:-$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)}"
INSTALL_ROOT="${INSTALL_ROOT:-/opt/stateguard}"
CONFIG_ROOT="${CONFIG_ROOT:-/etc/stateguard}"
ARTIFACT_ROOT="${ARTIFACT_ROOT:-/var/lib/stateguard/artifacts}"
BIN_ROOT="$INSTALL_ROOT/bin"
DIST_ROOT="$SOURCE_ROOT/dist/linux"
DAEMON_SRC="$DIST_ROOT/stateguardd"
CLI_SRC="$DIST_ROOT/stateguard"
API_SRC="$DIST_ROOT/stateguard-dashboard-api"
DAEMON_DST="$BIN_ROOT/stateguardd"
CLI_DST="$BIN_ROOT/stateguard"
API_DST="$BIN_ROOT/stateguard-dashboard-api"
WRAPPER_DST="$BIN_ROOT/stateguard-compose"
CONFIG_PATH="$CONFIG_ROOT/safedata.yaml"

require_binary() {
  if [ ! -f "$1" ]; then
    echo "missing built binary: $1" >&2
    echo "build release binaries into dist/linux before running the installer" >&2
    exit 1
  fi
}

require_binary "$DAEMON_SRC"
require_binary "$CLI_SRC"
require_binary "$API_SRC"

echo "Preparing Stateguard directories..."
mkdir -p "$INSTALL_ROOT" "$BIN_ROOT" "$CONFIG_ROOT" "$ARTIFACT_ROOT"

echo "Copying binaries..."
install -m 0755 "$DAEMON_SRC" "$DAEMON_DST"
install -m 0755 "$CLI_SRC" "$CLI_DST"
install -m 0755 "$API_SRC" "$API_DST"

if [ ! -f "$CONFIG_PATH" ]; then
  cat >"$CONFIG_PATH" <<EOF
version: "1"
project:
  name: stateguard
  environment: production
policy:
  mode: fail-closed
  validation:
    strategy: hybrid
    integrity_check: true
    allow_degraded: false
    restore_test:
      enabled: true
      cadence: periodic
  retention:
    window: 72h
    max_disk_usage_percent: 20
storage:
  local:
    path: $ARTIFACT_ROOT
runtime:
  compose:
    project_boundary: labels+compose_project
  kubernetes:
    mode: beta
daemon:
  host: 127.0.0.1
  port: 7010
api:
  host: 127.0.0.1
  port: 7011
EOF
fi

cat >"$WRAPPER_DST" <<EOF
#!/usr/bin/env sh
exec "$CLI_DST" intercept compose "\$@"
EOF
chmod 0755 "$WRAPPER_DST"

cat >/etc/systemd/system/stateguard-daemon.service <<EOF
[Unit]
Description=Stateguard Daemon
After=network-online.target docker.service
Wants=network-online.target

[Service]
Type=simple
Environment=STATEGUARD_CONFIG=$CONFIG_PATH
ExecStart=$DAEMON_DST
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

cat >/etc/systemd/system/stateguard-dashboard.service <<EOF
[Unit]
Description=Stateguard Dashboard API
After=network-online.target stateguard-daemon.service
Wants=network-online.target

[Service]
Type=simple
Environment=STATEGUARD_CONFIG=$CONFIG_PATH
ExecStart=$API_DST
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable --now stateguard-daemon.service
systemctl enable --now stateguard-dashboard.service

echo "Stateguard installed."
echo "CLI: $CLI_DST"
echo "Compose wrapper: $WRAPPER_DST"
