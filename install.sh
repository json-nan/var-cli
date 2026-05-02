#!/bin/sh
set -e

OWNER="${OWNER:-json-nan}"
REPO="${REPO:-var-cli}"
BINARY="var-cli"

OS=$(uname -s)
case "$OS" in
    Linux*)     OS=Linux;;
    Darwin*)    OS=Darwin;;
    CYGWIN*|MINGW*|MSYS*) OS=Windows;;
    *)          echo "Unsupported OS: $OS"; exit 1;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)  ARCH=x86_64;;
    arm64|aarch64) ARCH=arm64;;
    i386|i686)     ARCH=i386;;
    *)             echo "Unsupported architecture: $ARCH"; exit 1;;
esac

if [ -n "$INSTALL_DIR" ]; then
    INSTALL_DIR="$INSTALL_DIR"
elif [ -d "$HOME/.local/bin" ] && case ":$PATH:" in *":$HOME/.local/bin:"*) true;; *) false;; esac; then
    INSTALL_DIR="$HOME/.local/bin"
elif [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

API_URL="https://api.github.com/repos/${OWNER}/${REPO}/releases/latest"
echo "⬇️  Fetching latest release..."
LATEST=$(curl -fsSL "$API_URL" | grep '"tag_name":' | head -n 1 | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
    echo "❌ Failed to fetch latest release"
    exit 1
fi

# GitHub release tag has "v" prefix, but filename does not
VERSION=$(echo "$LATEST" | sed 's/^v//')

echo "📦 Latest version: ${LATEST}"

if [ "$OS" = "Windows" ]; then
    FILENAME="${REPO}_${VERSION}_${OS}_${ARCH}.zip"
else
    FILENAME="${REPO}_${VERSION}_${OS}_${ARCH}.tar.gz"
fi

DOWNLOAD_URL="https://github.com/${OWNER}/${REPO}/releases/download/${LATEST}/${FILENAME}"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "⬇️  Downloading ${FILENAME}..."
curl -fsSL --progress-bar -o "${TMP_DIR}/${FILENAME}" "$DOWNLOAD_URL"

echo "📂 Extracting..."
cd "$TMP_DIR"
if [ "$OS" = "Windows" ]; then
    unzip -q "$FILENAME"
else
    tar -xzf "$FILENAME"
fi

if [ ! -f "$BINARY" ]; then
    echo "❌ Binary '${BINARY}' not found in archive"
    exit 1
fi

chmod +x "$BINARY"
mkdir -p "$INSTALL_DIR"

if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY" "${INSTALL_DIR}/${BINARY}"
    echo "✅ ${BINARY} installed to ${INSTALL_DIR}"
else
    echo "🔐 Need write permission for ${INSTALL_DIR}"
    echo "   Run: sudo mv $(pwd)/${BINARY} ${INSTALL_DIR}/${BINARY}"
    exit 1
fi

echo ""
${BINARY} --version

# Check if install dir is in PATH
case ":$PATH:" in
    *":${INSTALL_DIR}:"*)
        echo ""
        echo "🚀 Run '${BINARY}' to get started"
        ;;
    *)
        echo ""
        echo "⚠️  ${INSTALL_DIR} is not in your PATH"
        echo ""
        echo "   Add this line to your shell profile:"
        echo ""
        echo "     export PATH=\"${INSTALL_DIR}:\$PATH\""
        echo ""
        echo "   Then reload:"
        echo ""
        echo "     source ~/.bashrc    # for Bash"
        echo "     source ~/.zshrc     # for Zsh"
        echo ""
        ;;
esac
