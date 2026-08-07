package ghrelease

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestAssetMatcher(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "qq-farm-windows-amd64-installer.exe"},
		{Name: "qq-farm-windows-amd64.exe"},
		{Name: "qq-farm-darwin.dmg"},
		{Name: "qq-farm-darwin-universal.zip"},
		{Name: "SHA256SUMS"},
	}

	win := AssetMatcher(updater.CheckRequest{Platform: "windows", Arch: "amd64"}, assets)
	if win < 0 || assets[win].Name != "qq-farm-windows-amd64.exe" {
		t.Fatalf("windows: got index %d", win)
	}

	mac := AssetMatcher(updater.CheckRequest{Platform: "darwin", Arch: "arm64"}, assets)
	if mac < 0 || assets[mac].Name != "qq-farm-darwin-universal.zip" {
		t.Fatalf("darwin: got index %d", mac)
	}

	macAmd := AssetMatcher(updater.CheckRequest{Platform: "darwin", Arch: "amd64"}, assets)
	if macAmd < 0 || assets[macAmd].Name != "qq-farm-darwin-universal.zip" {
		t.Fatalf("darwin amd64: got index %d", macAmd)
	}
}
