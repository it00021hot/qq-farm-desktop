package main

import (
	"embed"
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
const farmBundleVersion = "2"

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

	return os.WriteFile(filepath.Join(dest, ".bundle-version"), []byte(farmBundleVersion+"\n"), 0o644)
}

func farmResourcesReady(dest string) bool {
	wasm := filepath.Join(dest, "tsdk.wasm")
	if st, err := os.Stat(wasm); err != nil || st.Size() == 0 {
		return false
	}
	// Crop / activity icons must be real files (not a leftover symlink).
	iconSample := filepath.Join(dest, "gameConfig", "seed_images_named", "100001.png")
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
