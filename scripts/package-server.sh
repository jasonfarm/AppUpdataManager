#!/bin/bash
set -e

cd "$(dirname "$0")/.."

./scripts/build-server.sh

cd dist/server
tar czvf ../appUpdateManager-server.tar.gz .

cd ../..
echo "==> Server package: dist/appUpdateManager-server.tar.gz"
