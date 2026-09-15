package ghrelease

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// AssetMatcher picks the auto-update asset for the running platform/arch.
//
// Release layout since the per-arch split (v0.1.22+):
//   - qq-farm-windows-<arch>-installer.exe   Windows install + auto-update
//   - qq-farm-darwin-<arch>.zip              macOS auto-update
//   - qq-farm-darwin-<arch>.dmg              macOS first install
//
// darwin-universal.zip stays accepted as a fallback so installs built before
// the split keep updating; the portable qq-farm-windows-<arch>.exe is kept as
// a legacy fallback for pre-split releases.
func AssetMatcher(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	plat := strings.ToLower(req.Platform)
	arch := normalizeArch(strings.ToLower(req.Arch))

	if plat == "darwin" {
		if arch != "" {
			for i, a := range assets {
				name := strings.ToLower(a.Name)
				if strings.HasSuffix(name, ".zip") &&
					strings.Contains(name, "darwin") && archIn(name, arch) {
					return i
				}
			}
		}
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, "darwin-universal") && strings.HasSuffix(name, ".zip") {
				return i
			}
		}
		return -1
	}

	if plat == "windows" && arch != "" {
		// Installer-only releases: the installer itself is the update vehicle.
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if isInstaller(name) && strings.Contains(name, "windows") && archIn(name, arch) {
				return i
			}
		}
		// Legacy releases shipped a portable exe / zip for updates.
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if isInstaller(name) || !strings.Contains(name, "windows") || !archIn(name, arch) {
				continue
			}
			if strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip") {
				return i
			}
		}
	}

	return github.DefaultAssetMatcher(req, assets)
}

func normalizeArch(arch string) string {
	switch arch {
	case "x86_64", "x64":
		return "amd64"
	case "aarch64":
		return "arm64"
	default:
		return arch
	}
}

func archIn(name, arch string) bool {
	if strings.Contains(name, arch) {
		return true
	}
	if arch == "amd64" && (strings.Contains(name, "x86_64") || strings.Contains(name, "x64")) {
		return true
	}
	if arch == "arm64" && strings.Contains(name, "aarch64") {
		return true
	}
	return false
}

func isInstaller(name string) bool {
	return strings.Contains(name, "-installer.") ||
		strings.Contains(name, "_installer.") ||
		name == "installer.exe"
}
