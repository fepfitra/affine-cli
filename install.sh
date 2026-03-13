#!/bin/bash

set -e

REPO="tomohiro/affine-cli"
BINARY_NAME="affine"
INSTALL_DIR="$HOME/.local/bin"

cleanup() {
    rm -f "$TMPFILE"
}
trap cleanup EXIT

echo '{"status":"checking","message":"Checking for latest version..."}'

if command -v jq &> /dev/null; then
    LATEST_TAG=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | jq -r '.tag_name // empty')
else
    LATEST_TAG=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p')
fi

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Could not fetch latest version. Check your network connection."
    exit 1
fi

echo "Latest version: $LATEST_TAG"

if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    CURRENT_VERSION=$("$INSTALL_DIR/$BINARY_NAME" --version 2>/dev/null | awk '{print $2}')
    if [ "$CURRENT_VERSION" = "$LATEST_TAG" ]; then
        echo "Already at the latest version ($LATEST_TAG)"
        exit 0
    fi
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
    linux*)
        case "$ARCH" in
            x86_64) ASSET_NAME="affine-linux-amd64" ;;
            aarch64) ASSET_NAME="affine-linux-arm64" ;;
            *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
        esac
        ;;
    darwin*)
        case "$ARCH" in
            x86_64) ASSET_NAME="affine-darwin-amd64" ;;
            arm64) ASSET_NAME="affine-darwin-arm64" ;;
            *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
        esac
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$ASSET_NAME"

echo "Downloading from: $DOWNLOAD_URL"

mkdir -p "$INSTALL_DIR"

echo "Downloading..."
TMPFILE=$(mktemp "$INSTALL_DIR/$BINARY_NAME.XXXXXX")
curl -fL "$DOWNLOAD_URL" -o "$TMPFILE"
mv "$TMPFILE" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

if ! file "$INSTALL_DIR/$BINARY_NAME" | grep -qE "(ELF|Mach-O)"; then
    echo "Error: Downloaded file is not a valid binary"
    rm -f "$INSTALL_DIR/$BINARY_NAME"
    exit 1
fi

echo "{\"status\":\"success\",\"version\":\"$LATEST_TAG\",\"path\":\"$INSTALL_DIR/$BINARY_NAME\"}"
