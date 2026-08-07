#!/usr/bin/env bash
# Sync qq-farm-core farm resources into desktop/bundled, dereferencing symlinks
# so go:embed gets real seed_images_named PNGs (not a broken absolute symlink).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$ROOT/../qq-farm-core/resource/farm"
DEST="$ROOT/bundled/resource/farm"

if [[ ! -d "$SRC" ]]; then
  echo "missing farm resources: $SRC" >&2
  exit 1
fi

mkdir -p "$(dirname "$DEST")"
rm -rf "$DEST"

# -R recurse, -L follow symlinks (copy target files/dirs)
# Prefer ditto on macOS for reliable symlink dereference into a fresh tree.
if command -v ditto >/dev/null 2>&1; then
  ditto "$SRC" "$DEST"
  # ditto preserves symlinks; replace seed_images_named if still a link
  if [[ -L "$DEST/gameConfig/seed_images_named" ]]; then
    target="$(readlink "$DEST/gameConfig/seed_images_named")"
    rm -f "$DEST/gameConfig/seed_images_named"
    if [[ -d "$target" ]]; then
      ditto "$target" "$DEST/gameConfig/seed_images_named"
    else
      echo "seed_images_named symlink target missing: $target" >&2
      exit 1
    fi
  fi
else
  cp -RL "$SRC" "$DEST"
fi

# Fail fast if icons were not materialized
icon_dir="$DEST/gameConfig/seed_images_named"
if [[ -L "$icon_dir" ]] || [[ ! -d "$icon_dir" ]]; then
  echo "seed_images_named must be a real directory after sync (got: $(ls -ld "$icon_dir" 2>/dev/null || echo missing))" >&2
  exit 1
fi
count="$(find "$icon_dir" -type f -name '*.png' | wc -l | tr -d ' ')"
if [[ "$count" -lt 1 ]]; then
  echo "no PNG icons under $icon_dir" >&2
  exit 1
fi

echo "synced farm resources → $DEST ($count seed icons)"
