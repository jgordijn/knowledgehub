#!/bin/bash
set -euo pipefail

INSTALL_DIR="/opt/knowledgehub"
BINARY="$INSTALL_DIR/knowledgehub"
REPO="jgordijn/knowledgehub"
VERSION_FILE="$INSTALL_DIR/.current-version"

# Get current version
CURRENT="none"
if [ -f "$VERSION_FILE" ]; then
    CURRENT=$(cat "$VERSION_FILE")
fi

# Get latest release tag from GitHub
LATEST=$(curl -sfL "https://api.github.com/repos/$REPO/releases/latest" | grep -o '"tag_name": *"[^"]*"' | cut -d'"' -f4)

if [ -z "$LATEST" ]; then
    echo "Failed to check latest version"
    exit 1
fi

if [ "$CURRENT" = "$LATEST" ]; then
    exit 0
fi

echo "Updating $CURRENT -> $LATEST"

# Download new binary
TMP=$(mktemp)
curl -sfL "https://github.com/$REPO/releases/download/$LATEST/knowledgehub" -o "$TMP"
chmod +x "$TMP"

# Verify it runs
if ! "$TMP" --version > /dev/null 2>&1; then
    echo "Downloaded binary is invalid"
    rm -f "$TMP"
    exit 1
fi

# Install and restart
mv "$TMP" "$BINARY"
echo "$LATEST" > "$VERSION_FILE"
systemctl restart knowledgehub
echo "Updated to $LATEST"
