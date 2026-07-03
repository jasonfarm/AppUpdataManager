#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "==> Building server backend..."
cd server/backend
GOPROXY=https://proxy.golang.org go mod tidy
GOPROXY=https://proxy.golang.org go build -o ../../dist/server/appUpdateManager-server ./cmd/server

echo "==> Building server frontend..."
cd ../frontend
npm install
npm run build

echo "==> Copying config..."
cd ../../..
mkdir -p dist/server
cp server/backend/config/accounts.txt dist/server/

echo "==> Server build complete: dist/server/"
