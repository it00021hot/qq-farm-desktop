#!/usr/bin/env python3
"""Deduplicate seed images by content after sync-farm-bundle.sh.

Removes byte-identical duplicates under
bundled/resource/farm/gameConfig/seed_images_named and writes a manifest
(duplicate_images.json) mapping each removed path to the kept canonical file.
bundle_farm.go re-materializes the duplicates at extraction time, so the
embedded tree shrinks (~6 MB) while the on-disk layout stays unchanged.

The manifest is deterministic (sorted keys, canonical = lexicographically
first path per hash group) to keep release builds reproducible.
"""

import hashlib
import json
import os
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
ICONS_DIR = os.path.join(
    ROOT, "bundled", "resource", "farm", "gameConfig", "seed_images_named"
)
MANIFEST_NAME = "duplicate_images.json"


def main() -> int:
    if not os.path.isdir(ICONS_DIR):
        print(f"seed_images_named missing: {ICONS_DIR}", file=sys.stderr)
        return 1

    groups: dict[str, list[str]] = {}
    for dirpath, _dirnames, filenames in os.walk(ICONS_DIR):
        for name in filenames:
            if name == MANIFEST_NAME:
                continue
            path = os.path.join(dirpath, name)
            with open(path, "rb") as f:
                digest = hashlib.md5(f.read()).hexdigest()
            rel = os.path.relpath(path, ICONS_DIR).replace(os.sep, "/")
            groups.setdefault(digest, []).append(rel)

    manifest: dict[str, str] = {}
    removed = 0
    kept_bytes = 0
    for rels in groups.values():
        rels.sort()
        canonical = rels[0]
        for dup in rels[1:]:
            manifest[dup] = canonical
            os.remove(os.path.join(ICONS_DIR, *dup.split("/")))
            removed += 1
        kept_bytes += os.path.getsize(os.path.join(ICONS_DIR, *canonical.split("/")))

    manifest_path = os.path.join(ICONS_DIR, MANIFEST_NAME)
    with open(manifest_path, "w", encoding="utf-8", newline="\n") as f:
        json.dump(manifest, f, sort_keys=True, separators=(",", ":"))

    print(
        f"dedupe: removed {removed} duplicate images, "
        f"manifest entries {len(manifest)} -> {manifest_path}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
