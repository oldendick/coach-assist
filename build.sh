#!/bin/bash

# Build script for Coach Assist
# This script cross-compiles the application for the following platforms:
# 1. macOS Intel (amd64)
# 2. macOS Apple Silicon (arm64)
# 3. Windows (amd64)
# 4. Linux (amd64)

set -e

# Get version from Git
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")

APP_NAME="coachassist"
SRC_PATH="./cmd/coachassist"
OUT_DIR="dist"

# Create output directory
mkdir -p "$OUT_DIR"

echo "--- Starting Coach Assist ${VERSION} Build ---"

# 1. macOS Intel
echo "[1/4] Building for macOS (Intel amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.Version=${VERSION}" -o "$OUT_DIR/${APP_NAME}-darwin-amd64" "$SRC_PATH"

# 2. macOS Apple Silicon
echo "[2/4] Building for macOS (Apple Silicon arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Version=${VERSION}" -o "$OUT_DIR/${APP_NAME}-darwin-arm64" "$SRC_PATH"

# 3. Windows 64-bit
echo "[3/4] Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-X main.Version=${VERSION}" -o "$OUT_DIR/${APP_NAME}-windows-amd64.exe" "$SRC_PATH"

# 4. Linux 64-bit
echo "[4/4] Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-X main.Version=${VERSION}" -o "$OUT_DIR/${APP_NAME}-linux-amd64" "$SRC_PATH"

echo "--- Build Complete! ---"

# --- Packaging Phase ---
echo "--- Starting Packaging Phase ---"
for PLATFORM in "darwin-amd64" "darwin-arm64" "windows-amd64" "linux-amd64"; do
    OS=$(echo $PLATFORM | cut -d'-' -f1)
    ARCH=$(echo $PLATFORM | cut -d'-' -f2)
    
    echo "Packaging for ${PLATFORM}..."
    PKG_NAME="coach-assist-${VERSION}-${PLATFORM}"
    # The user wants a top-level folder in the archive called coach-assist-vX.Y.Z
    BASE_DIR="coach-assist-${VERSION}"
    STAGING_ROOT="${OUT_DIR}/${PKG_NAME}"
    TEMP_DIR="${STAGING_ROOT}/${BASE_DIR}"
    
    rm -rf "${STAGING_ROOT}"
    mkdir -p "${TEMP_DIR}/bin"
    
    # Copy app binary
    if [[ "$OS" == "windows" ]]; then
        if [ -f "${OUT_DIR}/${APP_NAME}-${PLATFORM}.exe" ]; then
            cp "${OUT_DIR}/${APP_NAME}-${PLATFORM}.exe" "${TEMP_DIR}/${APP_NAME}.exe"
        fi
        # Copy relevant gws binary to generic name
        if [ -f "bin/gws-windows-amd64.exe" ]; then
            cp "bin/gws-windows-amd64.exe" "${TEMP_DIR}/bin/gws.exe"
        fi
    else
        if [ -f "${OUT_DIR}/${APP_NAME}-${PLATFORM}" ]; then
            cp "${OUT_DIR}/${APP_NAME}-${PLATFORM}" "${TEMP_DIR}/${APP_NAME}"
            chmod +x "${TEMP_DIR}/${APP_NAME}"
        fi
        # Copy relevant gws binary to generic name
        if [ -f "bin/gws-${OS}-${ARCH}" ]; then
            cp "bin/gws-${OS}-${ARCH}" "${TEMP_DIR}/bin/gws"
            chmod +x "${TEMP_DIR}/bin/gws"
        fi
    fi
    
    # Copy configuration and docs
    [ -f "config.json" ] && cp config.json "${TEMP_DIR}/"
    [ -f "config.example.json" ] && cp config.example.json "${TEMP_DIR}/"
    [ -f "README.md" ] && cp README.md "${TEMP_DIR}/"
    
    # Create Archive
    if [[ "$OS" == "linux" ]]; then
        (cd "${STAGING_ROOT}" && tar -cJf "../${PKG_NAME}.tar.xz" "${BASE_DIR}")
    else
        (cd "${STAGING_ROOT}" && zip -r "../${PKG_NAME}.zip" "${BASE_DIR}" > /dev/null)
    fi
    
    # Cleanup staging root
    rm -rf "${STAGING_ROOT}"
done

echo "--- Packaging Complete! ---"
echo "Release archives are located in the '$OUT_DIR/' directory:"
ls -lh "$OUT_DIR/"*.zip "$OUT_DIR/"*.tar.xz 2>/dev/null
