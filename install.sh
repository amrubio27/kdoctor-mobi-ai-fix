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

# 3. Download from GitHub Releases.
#
# KDOCTOR_VERSION pins a tag (e.g. KDOCTOR_VERSION=v0.7.0); the default tracks
# the latest release.
REPO="amrubio27/kdoctor-mobi-ai-fix"
if [ -n "${KDOCTOR_VERSION:-}" ]; then
  RELEASE_PATH="download/${KDOCTOR_VERSION}"
  echo "Version pinned to ${KDOCTOR_VERSION}"
else
  RELEASE_PATH="latest/download"
fi

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -qO "$2" "$1"; }
else
  echo "Error: curl or wget is required to download kdoctor." >&2
  exit 1
fi

URL="https://github.com/${REPO}/releases/${RELEASE_PATH}/${BINARY_NAME}"
echo "Downloading ${BINARY_NAME}..."
if ! fetch "$URL" "$TARGET"; then
  # Name the URL that failed. A 404 here usually means the release is missing
  # this platform's asset, which is a different problem from having no network.
  cat >&2 <<EOF

Error: could not download kdoctor from:
  ${URL}

If that URL 404s, this platform is missing from the release.
You can build from source instead:
  git clone https://github.com/${REPO}.git
  cd kdoctor-mobi-ai-fix
  go build -o kdoctor ./cmd/kdoctor
EOF
  exit 1
fi
chmod +x "$TARGET"

# kdoctor-mcp is best-effort: older releases did not publish it, and the MCP
# server is optional.
MCP_NAME="kdoctor-mcp-${OS}-${ARCH}"
MCP_TARGET="$INSTALL_DIR/kdoctor-mcp"
if fetch "https://github.com/${REPO}/releases/${RELEASE_PATH}/${MCP_NAME}" "$MCP_TARGET" 2>/dev/null; then
  chmod +x "$MCP_TARGET"
  echo "Also installed kdoctor-mcp (MCP server)."
else
  rm -f "$MCP_TARGET"
fi

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

echo ""
echo "Note: the first scan downloads detekt (~50 MB) into ~/.kdoctor/tools"
echo "and needs a JDK 11-21. Run 'kdoctor doctor' to check your setup."
