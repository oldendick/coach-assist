#!/bin/bash

# Coach Assist One-Liner Installer
# Usage: curl -sSL https://raw.githubusercontent.com/oldendick/coach-assist/main/install.sh | sh

set -e

REPO="oldendick/coach-assist"
APP_NAME="coachassist"

echo "--- Coach Assist Installer ---"
echo "Installing into: $PWD"
echo ""

# Detect Platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) GOARCH="amd64" ;;
    arm64|aarch64) GOARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin) PLATFORM="darwin-$GOARCH"; EXT="zip" ;;
    linux) PLATFORM="linux-$GOARCH"; EXT="tar.xz" ;;
    *) echo "Unsupported OS: $OS. For Windows, please download the .zip manually from GitHub Releases."; exit 1 ;;
esac

echo "[1/4] Detected Platform: $PLATFORM"

# Fetch Latest Release info from GitHub API
echo "[2/4] Fetching latest release info from GitHub..."
RELEASE_DATA=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
VERSION=$(echo "$RELEASE_DATA" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
    echo "Error: Could not determine latest version from GitHub API."
    exit 1
fi

# Find the download URL for the specific platform
ASSET_URL=$(echo "$RELEASE_DATA" | grep "browser_download_url" | grep "$PLATFORM.$EXT" | head -n 1 | sed -E 's/.*"(https[^"]+)".*/\1/')

if [ -z "$ASSET_URL" ]; then
    echo "Error: Could not find download asset for $PLATFORM.$EXT in release $VERSION."
    exit 1
fi

echo "[3/4] Downloading $VERSION ($PLATFORM)..."
curl -L -o "coach-assist-$VERSION.$EXT" "$ASSET_URL"

# Extract
echo "[4/4] Extracting $VERSION..."
if [ "$EXT" = "zip" ]; then
    unzip -q "coach-assist-$VERSION.$EXT"
else
    tar -xJf "coach-assist-$VERSION.$EXT"
fi

# Cleanup
rm "coach-assist-$VERSION.$EXT"

# The package structure contains a folder named coach-assist-$VERSION
INSTALL_DIR="coach-assist-$VERSION"

echo ""
echo "--- Installation Successful! ---"
echo "Coach Assist $VERSION has been installed to: $PWD/$INSTALL_DIR"
echo ""
echo "Next Steps:"
echo "1. Configure the app:"
echo "   - cd $INSTALL_DIR && cp config.example.json config.json"
echo "   - Edit 'config.json' with your coach details."
echo "2. Run the application:"
echo "   - ./$APP_NAME"
echo "   - (The app will automatically guide you through the Google Workspace client setup on your first run)"
echo ""
