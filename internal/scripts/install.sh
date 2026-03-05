#!/bin/bash
# Install script for micro CLI
# Usage: curl -fsSL https://go-micro.dev/install.sh | sh

set -e

VERSION="${MICRO_VERSION:-latest}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture
case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l) ARCH="arm" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Normalize OS
case $OS in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Determine install directory
if [ "$EUID" -eq 0 ] || [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

echo "Installing micro ${VERSION} for ${OS}/${ARCH}..."

# Download URL
if [ "$VERSION" = "latest" ]; then
    URL="https://github.com/micro/go-micro/releases/latest/download/micro-${OS}-${ARCH}"
else
    URL="https://github.com/micro/go-micro/releases/download/${VERSION}/micro-${OS}-${ARCH}"
fi

# Download
TMP_FILE=$(mktemp)
if command -v curl &> /dev/null; then
    curl -fsSL "$URL" -o "$TMP_FILE"
elif command -v wget &> /dev/null; then
    wget -q "$URL" -O "$TMP_FILE"
else
    echo "Error: curl or wget required"
    exit 1
fi

# Install
chmod +x "$TMP_FILE"
mv "$TMP_FILE" "$INSTALL_DIR/micro"

echo ""
echo "✓ Installed micro to $INSTALL_DIR/micro"
echo ""

# Verify
if command -v micro &> /dev/null; then
    micro --version
else
    echo "Note: Add $INSTALL_DIR to your PATH:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo ""
echo "Get started:"
echo "  micro new myservice    # Create a new service"
echo "  micro run              # Run locally"
echo "  micro deploy           # Deploy to server"
echo ""
