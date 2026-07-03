#!/bin/bash
set -e

cd "$(dirname "$0")/.."

if ! command -v x86_64-w64-mingw32-gcc &> /dev/null; then
  echo "Error: x86_64-w64-mingw32-gcc not found."
  echo "Please install mingw-w64 first:"
  echo "  brew install mingw-w64"
  exit 1
fi

if ! command -v x86_64-w64-mingw32-windres &> /dev/null; then
  echo "Error: x86_64-w64-mingw32-windres not found."
  echo "Please install mingw-w64 first:"
  echo "  brew install mingw-w64"
  exit 1
fi

if [ ! -f "client/assets/icon.ico" ]; then
  echo "Error: client/assets/icon.ico not found."
  exit 1
fi

if [ ! -f "client/assets/icon.rc" ]; then
  echo "Error: client/assets/icon.rc not found."
  echo "Please run ./scripts/apply-icon.sh first, or create it manually."
  exit 1
fi

echo "==> Compiling Windows icon resource..."
cd client
x86_64-w64-mingw32-windres assets/icon.rc -o icon.syso

echo "==> Building Windows client..."
GOPROXY=https://proxy.golang.org go mod tidy
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
  CC=x86_64-w64-mingw32-gcc \
  go build -ldflags -H=windowsgui -o ../dist/client/appUpdateManager-client.exe .

# Clean up temporary syso
rm -f icon.syso

cd ..
echo "==> Client build complete: dist/client/appUpdateManager-client.exe"
