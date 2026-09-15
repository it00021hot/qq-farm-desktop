package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Farm assets are embedded so a single qq-farm.exe is enough (no zip sidecar).
//
//go:embed all:bundled/resource/farm
var bundledFarm embed.FS

// bump when embedded farm layout changes so existing installs re-extract.
// v3: tsdk.wasm 161084 字节（bot 9709bcb，协议 1.14.0.4 配套）。
// v4: seed_images_named 构建期按内容去重（duplicate_images.json 清单），
//
//	提取时按清单从规范副本复制回重复文件，磁盘布局与去重前一致。
const farmBundleVersion = "4"

// duplicateImagesManifest is generated at build time by
// scripts/dedupe_seed_images.py: keys are duplicate image paths relative to
// gameConfig/seed_images_named, values the canonical file to copy from.
const duplicateImagesManifest = "gameConfig/seed_images_named/duplicate_images.json"

// ensureBundledFarmResources extracts embedded farm assets into resourceRoot/resource/farm
// when the directory is missing or incomplete.
func ensureBundledFarmResources(resourceRoot string) error {
	dest := filepath.Join(resourceRoot, "resource", "farm")
	if farmResourcesReady(dest) {
		return nil
	}
	// Incomplete / outdated extract — replace with current embed.
	_ = os.RemoveAll(dest)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("mkdir farm resources: %w", err)
	}

	root := "bundled/resource/farm"
	if err := fs.WalkDir(bundledFarm, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if rel == filepath.FromSlash(duplicateImagesManifest) {
			// Build-time bookkeeping — never lands on disk.
			return nil
		}
		out := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		in, err := bundledFarm.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(f, in)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}); err != nil {
		return err
	}
	if err := restoreDuplicateImages(dest); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dest, ".bundle-version"), []byte(farmBundleVersion+"\n"), 0o644)
}

// restoreDuplicateImages re-materializes seed images that the build deduped
// out of the embedded tree, so on-disk layout and /game-config/* URLs stay
// identical to a non-deduped bundle.
func restoreDuplicateImages(dest string) error {
	data, err := fs.ReadFile(bundledFarm, "bundled/resource/farm/"+duplicateImagesManifest)
	if errors.Is(err, fs.ErrNotExist) {
		return nil // bundle built without dedup
	}
	if err != nil {
		return fmt.Errorf("read duplicate images manifest: %w", err)
	}
	var manifest map[string]string
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse duplicate images manifest: %w", err)
	}
	base := filepath.Join(dest, filepath.FromSlash("gameConfig/seed_images_named"))
	for dup, canonical := range manifest {
		src := filepath.Join(base, filepath.FromSlash(canonical))
		out := filepath.Join(base, filepath.FromSlash(dup))
		if err := copyFile(src, out); err != nil {
			return fmt.Errorf("restore %s: %w", dup, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, in)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func farmResourcesReady(dest string) bool {
	wasm := filepath.Join(dest, "tsdk.wasm")
	if st, err := os.Stat(wasm); err != nil || st.Size() == 0 {
		return false
	}
	// Crop / activity icons must be real files (not a leftover symlink).
	iconSample := filepath.Join(dest, "gameConfig", "seed_images_named", "seed_images", "100001.webp")
	st, err := os.Lstat(iconSample)
	if err != nil || st.Size() == 0 || st.Mode()&os.ModeSymlink != 0 {
		return false
	}
	ver, err := os.ReadFile(filepath.Join(dest, ".bundle-version"))
	if err != nil {
		return false
	}
	return string(ver) == farmBundleVersion+"\n" || string(ver) == farmBundleVersion
}
