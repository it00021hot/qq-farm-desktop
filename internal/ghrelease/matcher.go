package ghrelease

import (
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// AssetMatcher picks the auto-update binary (not the first-install package).
func AssetMatcher(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	plat := strings.ToLower(req.Platform)
	arch := strings.ToLower(req.Arch)

	if plat == "darwin" {
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, "darwin-universal") && strings.HasSuffix(name, ".zip") {
				return i
			}
		}
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, "darwin") && strings.HasSuffix(name, ".zip") &&
				!strings.Contains(name, "installer") {
				return i
			}
		}
		return -1
	}

	if plat == "windows" {
		for i, a := range assets {
			name := strings.ToLower(a.Name)
			if strings.Contains(name, "-installer.") || strings.Contains(name, "_installer.") {
				continue
			}
			if !strings.Contains(name, "windows") {
				continue
			}
			if arch != "" && !strings.Contains(name, arch) &&
				!(arch == "amd64" && (strings.Contains(name, "x64") || strings.Contains(name, "x86_64"))) {
				continue
			}
			if strings.HasSuffix(name, ".exe") || strings.HasSuffix(name, ".zip") {
				return i
			}
		}
	}

	return github.DefaultAssetMatcher(req, assets)
}
