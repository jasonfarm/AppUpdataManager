#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "==> Building server frontend..."
cd server/frontend
npm install
npm run build

echo "==> Building server backend..."
cd ../backend

# 交叉编译配置：Linux AMD64
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0  # 禁用 CGO，避免依赖 macOS 的 C 库

GOPROXY=https://proxy.golang.org go mod tidy
GOPROXY=https://proxy.golang.org go build -o ../../dist/server/appUpdateManager-server ./cmd/server

echo "==> Copying config and static assets..."
cd ../..
mkdir -p dist/server
rm -rf dist/server/static
if [ ! -d "server/backend/static" ]; then
  echo "ERROR: server/backend/static not found. Frontend build may have failed."
  exit 1
fi
cp -R server/backend/static dist/server/
if [ ! -f "dist/server/static/index.html" ]; then
  echo "ERROR: dist/server/static/index.html not found. Frontend build did not produce index.html."
  exit 1
fi
cp server/backend/config/accounts.txt dist/server/

echo "==> Server build complete: dist/server/"
