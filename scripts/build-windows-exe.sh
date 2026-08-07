#!/usr/bin/env bash
# Build a standalone Windows amd64 qq-farm.exe (resources embedded).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$PATH:$(go env GOPATH)/bin"

# Keep embedded farm assets in sync (dereference seed_images_named symlink)
bash scripts/sync-farm-bundle.sh

# Frontend for embed
(cd frontend && node scripts/build.mjs)

rm -f wails_windows_amd64.syso
wails3 generate syso -arch amd64 \
  -icon build/windows/icon.ico \
  -manifest build/windows/wails.exe.manifest \
  -info build/windows/info.json \
  -out wails_windows_amd64.syso

GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -tags production -trimpath -ldflags="-w -s -H windowsgui" -o bin/qq-farm.exe .
rm -f wails_windows_amd64.syso

ls -lh bin/qq-farm.exe
file bin/qq-farm.exe
echo "OK: bin/qq-farm.exe (standalone — double-click to run)"
