package ghrelease

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

// current release layout: installer-only Windows, per-arch macOS.
var splitAssets = []github.ReleaseAsset{
	{Name: "qq-farm-windows-amd64-installer.exe"},
	{Name: "qq-farm-darwin-amd64.zip"},
	{Name: "qq-farm-darwin-amd64.dmg"},
	{Name: "qq-farm-darwin-arm64.zip"},
	{Name: "qq-farm-darwin-arm64.dmg"},
	{Name: "SHA256SUMS"},
}

// legacy layout: portable exe + universal zip.
var legacyAssets = []github.ReleaseAsset{
	{Name: "qq-farm-windows-amd64-installer.exe"},
	{Name: "qq-farm-windows-amd64.exe"},
	{Name: "qq-farm-darwin.dmg"},
	{Name: "qq-farm-darwin-universal.zip"},
	{Name: "SHA256SUMS"},
}

func pick(t *testing.T, assets []github.ReleaseAsset, plat, arch string) string {
	t.Helper()
	i := AssetMatcher(updater.CheckRequest{Platform: plat, Arch: arch}, assets)
	if i < 0 {
		return ""
	}
	return assets[i].Name
}

func TestWindowsPicksInstaller(t *testing.T) {
	for _, arch := range []string{"amd64", "x64", "x86_64"} {
		if got := pick(t, splitAssets, "windows", arch); got != "qq-farm-windows-amd64-installer.exe" {
			t.Fatalf("windows %s: got %q", arch, got)
		}
	}
}

func TestDarwinPicksArchZip(t *testing.T) {
	if got := pick(t, splitAssets, "darwin", "arm64"); got != "qq-farm-darwin-arm64.zip" {
		t.Fatalf("darwin arm64: got %q", got)
	}
	if got := pick(t, splitAssets, "darwin", "amd64"); got != "qq-farm-darwin-amd64.zip" {
		t.Fatalf("darwin amd64: got %q", got)
	}
	if got := pick(t, splitAssets, "darwin", "aarch64"); got != "qq-farm-darwin-arm64.zip" {
		t.Fatalf("darwin aarch64: got %q", got)
	}
}

// Never match another architecture's zip just because the right one is absent.
func TestDarwinCrossArchIsRejected(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "qq-farm-darwin-arm64.zip"},
		{Name: "qq-farm-darwin-arm64.dmg"},
		{Name: "SHA256SUMS"},
	}
	if got := pick(t, assets, "darwin", "amd64"); got != "" {
		t.Fatalf("darwin amd64 should not match arm64 zip, got %q", got)
	}
}

func TestLegacyLayoutStillMatches(t *testing.T) {
	// The installer is preferred wherever it exists, even in old releases.
	if got := pick(t, legacyAssets, "windows", "amd64"); got != "qq-farm-windows-amd64-installer.exe" {
		t.Fatalf("legacy windows: got %q", got)
	}
	for _, arch := range []string{"arm64", "amd64"} {
		if got := pick(t, legacyAssets, "darwin", arch); got != "qq-farm-darwin-universal.zip" {
			t.Fatalf("legacy darwin %s: got %q", arch, got)
		}
	}
}
