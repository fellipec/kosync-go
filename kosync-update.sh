#!/usr/bin/env bash
set -euo pipefail

SERVICE_NAME="kosync"
INSTALL_PATH="/usr/local/bin/kosync-go"
TMP_DIR="$(mktemp -d)"
API_URL="https://api.github.com/repos/fellipec/kosync-go/releases/latest"

echo "Fetching latest release info..."
LATEST_URL=$(curl -fsSL "$API_URL" | grep browser_download_url | grep linux-amd64 | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
    echo "Could not find the latest binary download URL."
    exit 1
fi

echo "Downloading latest binary..."
curl -fsSL "$LATEST_URL" -o "$TMP_DIR/kosync-go"

echo "Stopping service..."
sudo systemctl stop "$SERVICE_NAME"

echo "Setting executable permissions..."
chmod +x "$TMP_DIR/kosync-go"

echo "Replacing binary..."
sudo mv "$TMP_DIR/kosync-go" "$INSTALL_PATH"

echo "Starting service..."
sudo systemctl start "$SERVICE_NAME"

echo "Cleaning up..."
rm -rf "$TMP_DIR"

echo "Update complete!"
