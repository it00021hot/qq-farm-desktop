#!/usr/bin/env bash
# Build macOS universal .app, DMG (first install), and zip (auto-update).
# Must run on macOS with CGO. Requires sibling ../qq-farm-web and ../qq-farm-core.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$PATH:$(go env GOPATH)/bin"

VERSION="${VERSION:-0.1.0}"
VERSION="${VERSION#v}"
APP_NAME="${APP_NAME:-qq-farm}"
BIN_DIR="${BIN_DIR:-bin}"

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

bash scripts/sync-farm-bundle.sh
(cd ../qq-farm-web && pnpm install --frozen-lockfile)
(cd frontend && node scripts/build.mjs)

export CGO_ENABLED=1
export MACOSX_DEPLOYMENT_TARGET=12.0
LDFLAGS="-w -s -X main.appVersion=${VERSION}"

build_arch() {
  local arch="$1"
  local out="${BIN_DIR}/${APP_NAME}-${arch}"
  GOOS=darwin GOARCH="$arch" \
    CGO_CFLAGS="-mmacosx-version-min=12.0" \
    CGO_LDFLAGS="-mmacosx-version-min=12.0" \
    go build -tags production -trimpath -ldflags="$LDFLAGS" -o "$out" .
}

build_arch amd64
build_arch arm64
lipo -create -output "${BIN_DIR}/${APP_NAME}" \
  "${BIN_DIR}/${APP_NAME}-amd64" "${BIN_DIR}/${APP_NAME}-arm64"
rm -f "${BIN_DIR}/${APP_NAME}-amd64" "${BIN_DIR}/${APP_NAME}-arm64"

# Assemble .app (mirrors build/darwin create:app:bundle)
APP="${BIN_DIR}/${APP_NAME}.app"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp build/darwin/icons.icns "$APP/Contents/Resources/"
rm -f "$APP/Contents/Resources/Assets.car"
mkdir -p "$APP/Contents/Resources/resource"
rm -rf "$APP/Contents/Resources/resource/farm"
cp -R bundled/resource/farm "$APP/Contents/Resources/resource/farm"
cp "${BIN_DIR}/${APP_NAME}" "$APP/Contents/MacOS/"
cp build/darwin/Info.plist "$APP/Contents/"
codesign --force --deep --sign - "$APP"

# Zip for auto-update (single top-level .app entry)
ZIP_OUT="${BIN_DIR}/qq-farm-darwin-universal.zip"
rm -f "$ZIP_OUT"
(
  cd "$BIN_DIR"
  ditto -c -k --keepParent "${APP_NAME}.app" "qq-farm-darwin-universal.zip"
)

# DMG for first-time install
DMG_OUT="${BIN_DIR}/qq-farm-darwin.dmg"
rm -f "$DMG_OUT" "${BIN_DIR}/${APP_NAME}.dmg"
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

echo "OK: ${APP}"
echo "OK: ${ZIP_OUT} (auto-update asset)"
echo "OK: ${DMG_OUT} (first install)"
ls -lh "$ZIP_OUT" "$DMG_OUT"
du -sh "$APP"
