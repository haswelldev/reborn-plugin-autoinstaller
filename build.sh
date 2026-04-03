#!/usr/bin/env bash
# Build script — Reborn Plugin Autoinstaller
# Cross-compiles from macOS to Windows x64.
#
# One-time setup:
#   brew install go mingw-w64
#   go install github.com/akavel/rsrc@latest
#
# For the NSIS installer, use:
#   - Linux:   sudo apt install nsis && makensis installer.nsi
#   - Windows: build-installer.bat
#   - CI:      .github/workflows/build.yml (automatic on tag push)

set -e
export PATH="/opt/homebrew/bin:$PATH"

DIST="dist"
APP_EXE="$DIST/reborn-plugin-autoinstaller.exe"
RSRC_BIN="$(go env GOPATH)/bin/rsrc"

echo "================================================================"
echo " Reborn Plugin Autoinstaller — Windows Build"
echo "================================================================"

mkdir -p "$DIST"

# ── Step 1: Embed manifest + icon into rsrc.syso ──────────────────────
echo ""
echo "[1/2] Embedding manifest and icon..."
"$RSRC_BIN" \
    -manifest app.manifest \
    -ico resources/icon.ico \
    -o rsrc.syso
echo "      rsrc.syso created"

# ── Step 2: Cross-compile Go binary for Windows x64 ───────────────────
echo ""
echo "[2/2] Compiling for Windows x64..."
CC=x86_64-w64-mingw32-gcc \
GOOS=windows \
GOARCH=amd64 \
CGO_ENABLED=1 \
go build \
    -ldflags="-H windowsgui -s -w" \
    -o "$APP_EXE" \
    .
echo "      $APP_EXE — $(du -sh "$APP_EXE" | cut -f1)"

echo ""
echo "================================================================"
echo " Done! -> $APP_EXE"
echo ""
echo " To build the NSIS installer:"
echo "   Linux:   sudo apt install nsis && makensis installer.nsi"
echo "   Windows: build-installer.bat"
echo "   CI:      push a tag to GitHub"
echo "================================================================"
