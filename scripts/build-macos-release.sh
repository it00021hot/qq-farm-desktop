#!/usr/bin/env bash
# Build a single-architecture macOS .app, zip (auto-update) and DMG (first
# install). Run once per architecture: MAC_ARCH=amd64 (Intel) or MAC_ARCH=arm64
# (Apple Silicon); CI runs the two architectures as separate jobs.
# Must run on macOS with CGO. Requires the core/ and frontend/ (qq-farm-web)
# submodules (git clone --recurse-submodules).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$PATH:$(go env GOPATH)/bin"

VERSION="${VERSION:-0.1.0}"
VERSION="${VERSION#v}"
APP_NAME="${APP_NAME:-qq-farm}"
BIN_DIR="${BIN_DIR:-bin}"
MAC_ARCH="${MAC_ARCH:?set MAC_ARCH to amd64 or arm64}"

case "$MAC_ARCH" in
  amd64|arm64) ;;
  *) echo "MAC_ARCH must be amd64 or arm64 (got: $MAC_ARCH)" >&2; exit 1 ;;
esac

python3 - <<PY
from pathlib import Path
import re
p = Path("build/darwin/Info.plist")
text = p.read_text()
def set_key(xml: str, key: str, value: str) -> str:
    return re.sub(
        rf"(<key>{re.escape(key)}</key>\s*<string>)[^<]*(</string>)",
        rf"\g<1>{value}\g<2>",
        xml,
        count=1,
    )
text = set_key(text, "CFBundleVersion", "${VERSION}")
text = set_key(text, "CFBundleShortVersionString", "${VERSION}")
p.write_text(text)
print(f"Info.plist version → ${VERSION}")
PY

mkdir -p "${BIN_DIR}"

# Stale universal artifacts from pre-split releases must never linger.
rm -f "${BIN_DIR}/qq-farm-darwin-universal.zip" "${BIN_DIR}/qq-farm-darwin.dmg"

bash scripts/sync-farm-bundle.sh
(cd frontend && pnpm install --frozen-lockfile && pnpm run build:desktop)

export CGO_ENABLED=1
export MACOSX_DEPLOYMENT_TARGET=12.0
LDFLAGS="-w -s -X main.appVersion=${VERSION}"

BIN="${BIN_DIR}/${APP_NAME}-${MAC_ARCH}"
GOOS=darwin GOARCH="$MAC_ARCH" \
  CGO_CFLAGS="-mmacosx-version-min=12.0" \
  CGO_LDFLAGS="-mmacosx-version-min=12.0" \
  go build -tags production -trimpath -ldflags="$LDFLAGS" -o "$BIN" .

# Assemble .app (mirrors build/darwin create:app:bundle). The bundle keeps the
# plain app name so the auto-updater's swap target is stable across arches.
APP="${BIN_DIR}/${APP_NAME}.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp build/darwin/icons.icns "$APP/Contents/Resources/"
rm -f "$APP/Contents/Resources/Assets.car"
mkdir -p "$APP/Contents/Resources/resource"
rm -rf "$APP/Contents/Resources/resource/farm"
cp -R bundled/resource/farm "$APP/Contents/Resources/resource/farm"
cp "$BIN" "$APP/Contents/MacOS/"
cp build/darwin/Info.plist "$APP/Contents/"
codesign --force --deep --sign - "$APP"
rm -f "$BIN"

# Zip for auto-update (single top-level .app entry)
ZIP_OUT="${BIN_DIR}/qq-farm-darwin-${MAC_ARCH}.zip"
rm -f "$ZIP_OUT"
(
  cd "$BIN_DIR"
  ditto -c -k --keepParent "${APP_NAME}.app" "qq-farm-darwin-${MAC_ARCH}.zip"
)

# DMG for first-time install
DMG_OUT="${BIN_DIR}/qq-farm-darwin-${MAC_ARCH}.dmg"
rm -f "${BIN_DIR}/${APP_NAME}.dmg"
if command -v wails3 >/dev/null 2>&1; then
  wails3 tool package --format dmg --name "$APP_NAME" --out "$BIN_DIR" \
    --background build/darwin/dmg-background.png \
    --volume-icon build/darwin/icons.icns \
    --file-icon build/darwin/dmg-file-icon.icns \
    --window-width 540 --window-height 380 || true
fi

DMG_SRC="${BIN_DIR}/${APP_NAME}.dmg"
if [[ -f "$DMG_SRC" ]]; then
  mv -f "$DMG_SRC" "$DMG_OUT"
elif [[ ! -f "$DMG_OUT" ]]; then
  STAGE="${BIN_DIR}/dmg-stage"
  rm -rf "$STAGE"
  mkdir -p "$STAGE"
  cp -R "$APP" "$STAGE/"
  ln -sf /Applications "$STAGE/Applications"
  hdiutil create -volname "QQ Farm Assistant" -srcfolder "$STAGE" -ov -format UDZO "$DMG_OUT"
  rm -rf "$STAGE"
fi

echo "OK: ${APP} (${MAC_ARCH})"
echo "OK: ${ZIP_OUT} (auto-update asset)"
echo "OK: ${DMG_OUT} (first install)"
ls -lh "$ZIP_OUT" "$DMG_OUT"
du -sh "$APP"
