#!/usr/bin/env bash

set -e -o pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

REPO="barthjs/swim-timer"
APP_NAME="swim-timer"
INSTALL_MODE=""  # "user" or "system"

print_header() {
  echo -e "${GREEN}=== Swim Timer Installer ===${NC}\n"
}

print_info() {
  echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
  echo -e "${GREEN}✓${NC} $1"
}

print_warn() {
  echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
  echo -e "${RED}✗${NC} $1"
}

show_help() {
  cat << 'EOF'
Usage: install.sh [OPTION]

Options:
  --user      Install for current user only (systemd user service)
  --system    Install system-wide (requires sudo, runs as current user)
  --help      Show this help message

If no option is provided, you will be prompted to choose.
EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --user)
      INSTALL_MODE="user"
      shift
      ;;
    --system)
      INSTALL_MODE="system"
      shift
      ;;
    --help)
      show_help
      exit 0
      ;;
    *)
      print_error "Unknown option: $1"
      show_help
      exit 1
      ;;
  esac
done

print_header

ARCH=$(uname -m)
case $ARCH in
  x86_64)
    ARCH="amd64"
    ;;
  aarch64)
    ARCH="arm64"
    ;;
  *)
    print_error "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

BINARY_NAME="${APP_NAME}_linux_${ARCH}.tar.gz"

if [ -z "$INSTALL_MODE" ]; then
  echo "Choose installation mode:"
  echo "  1) User-wide   (installs for $USER only, uses systemd user service)"
  echo "  2) System-wide (installs for all users, requires sudo, runs as $USER)"
  read -p "Select mode (1 or 2): " -n 1 -r
  echo
  case $REPLY in
    1)
      INSTALL_MODE="user"
      ;;
    2)
      INSTALL_MODE="system"
      ;;
    *)
      print_error "Invalid choice"
      exit 1
      ;;
  esac
fi

print_info "Installation mode: $INSTALL_MODE"

if [ "$INSTALL_MODE" = "user" ]; then
  INSTALL_DIR="${HOME}/.local/bin"
  APPS_DIR="${HOME}/.local/share/applications"
  CONFIG_DIR="${HOME}/.config/swim-timer"
  SYSTEMD_DIR="${HOME}/.config/systemd/user"
  SYSTEMD_UNIT="${SYSTEMD_DIR}/${APP_NAME}.service"
else
  INSTALL_DIR="/usr/local/bin"
  APPS_DIR="/usr/share/applications"
  CONFIG_DIR="/var/lib/${APP_NAME}"
  SYSTEMD_DIR="/etc/systemd/system"
  SYSTEMD_UNIT="${SYSTEMD_DIR}/${APP_NAME}.service"
fi

print_info "Downloading latest ${APP_NAME} binary for Linux ${ARCH}..."

RELEASE_JSON=$(curl -s ${GITHUB_TOKEN:+-H "Authorization: token $GITHUB_TOKEN"} "https://api.github.com/repos/${REPO}/releases/latest")
RELEASE_URL=$(echo "$RELEASE_JSON" | grep -B 5 "\"name\": \"$BINARY_NAME\"" | grep '"url":' | grep '/assets/' | head -1 | cut -d '"' -f 4)

if [ -z "$RELEASE_JSON" ]; then
  print_error "Failed to fetch release information from GitHub"
  exit 1
fi

if [ -z "$RELEASE_URL" ]; then
  print_error "Failed to find binary for $BINARY_NAME"
  exit 1
fi

print_info "Downloading from: $RELEASE_URL"

if [ "$INSTALL_MODE" = "user" ]; then
  mkdir -p "$INSTALL_DIR" "$APPS_DIR" "$CONFIG_DIR" "$SYSTEMD_DIR"
else
  sudo mkdir -p "$INSTALL_DIR" "$APPS_DIR" "$CONFIG_DIR" "$SYSTEMD_DIR"
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf $TMP_DIR' EXIT

if ! curl -sL ${GITHUB_TOKEN:+-H "Authorization: token $GITHUB_TOKEN"} -H "Accept: application/octet-stream" "$RELEASE_URL" -o "$TMP_DIR/$BINARY_NAME"; then
  print_error "Failed to download binary"
  exit 1
fi

CHECKSUMS_URL=$(echo "$RELEASE_JSON" | grep -E -B 5 '"name": ".*checksums.*"' | grep '"url":' | grep '/assets/' | head -1 | cut -d '"' -f 4)
if [ -n "$CHECKSUMS_URL" ]; then
  curl -sL ${GITHUB_TOKEN:+-H "Authorization: token $GITHUB_TOKEN"} -H "Accept: application/octet-stream" "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt" 2>/dev/null
fi
if [ -f "$TMP_DIR/checksums.txt" ]; then
  if command -v sha256sum &> /dev/null; then
    if grep "$BINARY_NAME" "$TMP_DIR/checksums.txt" | sed 's/^\.\/*//' | (cd "$TMP_DIR" && sha256sum -c - >/dev/null 2>&1); then
      print_success "Binary integrity verified"
    else
      print_warn "Binary integrity check failed"
    fi
  fi
else
  print_warn "Checksums file not available"
fi

cd "$TMP_DIR"
if ! tar -xzf "$BINARY_NAME" 2>/dev/null; then
  print_error "Failed to extract binary from archive"
  exit 1
fi

BINARY_PATH="$TMP_DIR/$APP_NAME"

if [ ! -f "$BINARY_PATH" ]; then
  print_error "Failed to extract binary from archive"
  exit 1
fi

chmod +x "$BINARY_PATH"

if [ "$INSTALL_MODE" = "user" ]; then
  mv "$BINARY_PATH" "$INSTALL_DIR/$APP_NAME"
  print_success "Binary installed to $INSTALL_DIR/$APP_NAME"
else
  sudo mv "$BINARY_PATH" "$INSTALL_DIR/$APP_NAME"
  sudo chown root:root "$INSTALL_DIR/$APP_NAME"
  print_success "Binary installed to $INSTALL_DIR/$APP_NAME"
fi

if [ "$INSTALL_MODE" = "user" ] && [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  print_warn "$INSTALL_DIR is not in your PATH. Add it with:"
  echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

DESKTOP_FILE="$APPS_DIR/$APP_NAME.desktop"

APP_VERSION=$(echo "$RELEASE_JSON" | grep '"tag_name"' | head -1 | cut -d '"' -f 4)
if [ -z "$APP_VERSION" ]; then
  APP_VERSION="dev"
fi

read -r -d '' DESKTOP_CONTENT << EOF || true
[Desktop Entry]
Name=Swim Timer
Comment=HTML overlays with timing data for swimming competitions
Type=Application
Exec=$INSTALL_DIR/$APP_NAME
Terminal=false
Icon=chronometer
Categories=Utility;
X-AppVersion=$APP_VERSION
EOF

if [ "$INSTALL_MODE" = "user" ]; then
  echo "$DESKTOP_CONTENT" > "$DESKTOP_FILE"
else
  echo "$DESKTOP_CONTENT" | sudo tee "$DESKTOP_FILE" > /dev/null
fi
print_success "Desktop shortcut created at $DESKTOP_FILE"

echo ""
read -p "Do you want to install as a systemd service? (Y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] || [ -z "$REPLY" ]; then
  print_info "Installing as systemd $([ "$INSTALL_MODE" = "user" ] && echo "user" || echo "system") service..."

  WANTED_BY=$([ "$INSTALL_MODE" = "user" ] && echo "default.target" || echo "multi-user.target")

  read -r -d '' SERVICE_CONTENT << EOF || true
[Unit]
Description=Swim Timer
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$CONFIG_DIR
ExecStart=$INSTALL_DIR/$APP_NAME
Restart=on-failure
RestartSec=5

[Install]
WantedBy=$WANTED_BY
EOF

  if [ "$INSTALL_MODE" = "user" ]; then
    echo "$SERVICE_CONTENT" > "$SYSTEMD_UNIT"
  else
    echo "$SERVICE_CONTENT" | sudo tee "$SYSTEMD_UNIT" > /dev/null
  fi

  print_success "Service file created at $SYSTEMD_UNIT"

  if [ "$INSTALL_MODE" = "user" ]; then
    systemctl --user daemon-reload
  else
    sudo systemctl daemon-reload
  fi

  echo ""
  read -p "Enable and start the service now? (Y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]] || [ -z "$REPLY" ]; then
    if [ "$INSTALL_MODE" = "user" ]; then
      systemctl --user enable "$APP_NAME"
      systemctl --user start "$APP_NAME"
      print_success "Service enabled and started"
      echo "Check status with: systemctl --user status $APP_NAME"

      print_info "User service will auto-start at next login"
    else
      sudo systemctl enable "$APP_NAME"
      sudo systemctl start "$APP_NAME"
      print_success "Service enabled and started"
      echo "Check status with: sudo systemctl status $APP_NAME"
    fi
  fi
else
  print_info "Skipped service installation"
  echo ""
  echo "You can run $APP_NAME directly by:"
  echo "  1. Starting it from your applications launcher"
  echo "  2. Running: $APP_NAME"
fi

echo ""
print_success "Installation complete!"

if [ "$INSTALL_MODE" = "user" ]; then
  if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
    echo "Start the app with: $APP_NAME"
  else
    echo "Start the app with: $INSTALL_DIR/$APP_NAME"
  fi
else
  echo "Start the app with: $APP_NAME"
fi
