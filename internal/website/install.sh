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
    armv7l) ARCH="armv7" ;;
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
    URL="https://github.com/micro/go-micro/releases/latest/download/micro_${OS}_${ARCH}.tar.gz"
else
    URL="https://github.com/micro/go-micro/releases/download/${VERSION}/micro_${OS}_${ARCH}.tar.gz"
fi

# Create temp directory for extraction
TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/micro.tar.gz"

# Download
if command -v curl &> /dev/null; then
    curl -fsSL "$URL" -o "$TMP_FILE"
elif command -v wget &> /dev/null; then
    wget -q "$URL" -O "$TMP_FILE"
else
    echo "Error: curl or wget required"
    rm -rf "$TMP_DIR"
    exit 1
fi

# Extract and install
tar -xzf "$TMP_FILE" -C "$TMP_DIR"

# Handle different archive structures (binary at root or in subdirectory)
if [ -f "${TMP_DIR}/micro" ]; then
    BINARY_PATH="${TMP_DIR}/micro"
elif [ -f "${TMP_DIR}/micro_${OS}_${ARCH}/micro" ]; then
    BINARY_PATH="${TMP_DIR}/micro_${OS}_${ARCH}/micro"
else
    # Try to find any executable named 'micro' in the extracted content
    BINARY_PATH=$(find "$TMP_DIR" -name "micro" -type f -executable | head -n1)
fi

if [ -z "$BINARY_PATH" ] || [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Could not find micro binary in archive"
    echo "Archive contents:"
    tar -tzf "$TMP_FILE"
    rm -rf "$TMP_DIR"
    exit 1
fi

chmod +x "$BINARY_PATH"
mv "$BINARY_PATH" "$INSTALL_DIR/micro"

# Cleanup
rm -rf "$TMP_DIR"

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
