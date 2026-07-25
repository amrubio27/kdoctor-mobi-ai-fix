#!/bin/sh
# install.sh — Installer for kdoctor on macOS & Linux
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/install.sh | sh

set -e

echo "Installing kdoctor..."

# 1. Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64|amd64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported architecture $ARCH"
    exit 1
    ;;
esac

case "$OS" in
  darwin)
    BINARY_NAME="kdoctor-darwin-${ARCH}"
    ;;
  linux)
    BINARY_NAME="kdoctor-linux-${ARCH}"
    ;;
  *)
    echo "Error: Unsupported OS $OS"
    exit 1
    ;;
esac

# 2. Determine target directory
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"
TARGET="$INSTALL_DIR/kdoctor"

# 3. Download binary from GitHub Releases
URL="https://github.com/amrubio27/kdoctor-mobi-ai-fix/releases/latest/download/${BINARY_NAME}"
echo "Downloading ${BINARY_NAME} from ${URL}..."

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$URL" -o "$TARGET"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$TARGET" "$URL"
else
  echo "Error: curl or wget is required to download kdoctor."
  exit 1
fi

chmod +x "$TARGET"

echo ""
echo "kdoctor installed successfully to $TARGET"

# 4. Check if INSTALL_DIR is in PATH
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    echo ""
    echo "Notice: $INSTALL_DIR is not in your PATH."
    echo "Add it to your shell configuration file (~/.bashrc, ~/.zshrc):"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    ;;
esac

echo ""
"$TARGET" --version || true
