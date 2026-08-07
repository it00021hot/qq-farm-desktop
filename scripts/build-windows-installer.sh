#!/usr/bin/env bash
# Build Windows amd64 update binary + NSIS per-user installer.
# Requires: Go, pnpm, wails3, makensis; sibling ../qq-farm-web and ../qq-farm-core.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$PATH:$(go env GOPATH)/bin"

VERSION="${VERSION:-0.1.0}"
VERSION="${VERSION#v}"
ARCH="${ARCH:-amd64}"
INSTALL_SCOPE="${INSTALL_SCOPE:-user}"

bash scripts/sync-farm-bundle.sh
(cd ../qq-farm-web && pnpm install --frozen-lockfile)
(cd frontend && node scripts/build.mjs)

# Keep VERSIONINFO in sync for the .syso
PYTHON=python3
command -v python3 >/dev/null 2>&1 || PYTHON=python
"$PYTHON" - <<PY
import json
from pathlib import Path
p = Path("build/windows/info.json")
data = json.loads(p.read_text())
data["fixed"]["file_version"] = "${VERSION}"
data["info"]["0000"]["ProductVersion"] = "${VERSION}"
p.write_text(json.dumps(data, indent="\t") + "\n")
PY

rm -f "wails_windows_${ARCH}.syso"
wails3 generate syso -arch "$ARCH" \
  -icon build/windows/icon.ico \
  -manifest build/windows/wails.exe.manifest \
  -info build/windows/info.json \
  -out "wails_windows_${ARCH}.syso"

LDFLAGS="-w -s -H windowsgui -X main.appVersion=${VERSION}"
GOOS=windows GOARCH="$ARCH" CGO_ENABLED=0 \
  go build -tags production -trimpath -ldflags="$LDFLAGS" -o bin/qq-farm.exe .
rm -f "wails_windows_${ARCH}.syso"

cp -f bin/qq-farm.exe "bin/qq-farm-windows-${ARCH}.exe"

wails3 generate webview2bootstrapper -dir "$ROOT/build/windows/nsis"

NSIS_SCOPE_FLAGS=(-DWAILS_INSTALL_SCOPE=user -DREQUEST_EXECUTION_LEVEL=user)
if [[ "$INSTALL_SCOPE" == "machine" ]]; then
  NSIS_SCOPE_FLAGS=(-DWAILS_INSTALL_SCOPE=machine)
fi

ARG_FLAG=AMD64
if [[ "$ARCH" == "arm64" ]]; then
  ARG_FLAG=ARM64
fi

NSIS_BINARY="$ROOT/bin/qq-farm.exe"
# makensis on Windows needs a native path (not Git Bash /c/...)
if command -v cygpath >/dev/null 2>&1; then
  NSIS_BINARY="$(cygpath -w "$NSIS_BINARY")"
elif [[ "$(uname -s)" == MINGW* || "$(uname -s)" == MSYS* || "$(uname -s)" == CYGWIN* ]]; then
  NSIS_BINARY="$(echo "$NSIS_BINARY" | sed -e 's|^/\([a-zA-Z]\)/|\1:/|' -e 's|/|\\|g')"
fi

(
  cd build/windows/nsis
  # Prefer makensis on PATH; fall back to common Windows install dirs
  MAKENSIS=makensis
  if ! command -v makensis >/dev/null 2>&1; then
    for candidate in \
      "/c/Program Files (x86)/NSIS/makensis.exe" \
      "/c/Program Files/NSIS/makensis.exe" \
      "/mnt/c/Program Files (x86)/NSIS/makensis.exe"; do
      if [[ -x "$candidate" ]]; then
        MAKENSIS="$candidate"
        break
      fi
    done
  fi
  "$MAKENSIS" \
    -DINFO_PRODUCTVERSION="$VERSION" \
    "${NSIS_SCOPE_FLAGS[@]}" \
    -DARG_WAILS_${ARG_FLAG}_BINARY="$NSIS_BINARY" \
    project.nsi
)

echo "OK: bin/qq-farm-windows-${ARCH}.exe (auto-update asset)"
echo "OK: bin/qq-farm-windows-${ARCH}-installer.exe (first install)"
ls -lh "bin/qq-farm-windows-${ARCH}.exe" "bin/qq-farm-windows-${ARCH}-installer.exe"
