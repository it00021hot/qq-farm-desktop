package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/it00021hot/qq-farm-desktop/internal/ghrelease"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const githubReleaseRepo = "it00021hot/qq-farm-desktop"

// newUpdaterHTTPClient builds the HTTP client used for the GitHub API and
// asset downloads. The github provider's default client has an overall
// 30s Timeout that spans the whole request including the body stream, which
// truncates larger downloads. We keep per-phase timeouts (dial, TLS, first
// response byte) but no overall deadline — the caller's context governs the
// total duration, so a slow-but-progressing download is never cut off.
func newUpdaterHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   4,
			ForceAttemptHTTP2:     true,
		},
	}
}

// setupUpdater wires GitHub Releases auto-update. Startup only prompts when
// an update is available; the menu action reports "already up to date".
func setupUpdater(app *application.App) {
	gh, err := github.New(github.Config{
		Repository:    githubReleaseRepo,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  ghrelease.AssetMatcher,
		HTTPClient:    newUpdaterHTTPClient(),
	})
	if err != nil {
		log.Printf("updater: github provider: %v", err)
		return
	}

	// Open at the full update-flow size. Wails defaults to a compact 348×161
	// card and grows later via SetSize; on macOS that resize is easy to miss,
	// which clips the Restart & Apply buttons until the user enlarges manually.
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: appVersion,
		Providers:      []updater.Provider{gh},
		Window: &updater.BuiltinWindow{
			Options: updater.WindowOptions{
				Title:  "软件更新",
				Width:  560,
				Height: 620,
			},
		},
	}); err != nil {
		log.Printf("updater: init: %v", err)
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		runUpdateCheck(app, false)
	}()
}

func checkForUpdates(app *application.App) {
	go runUpdateCheck(app, true)
}

// runUpdateCheck drives one update check. macOS keeps the wails builtin
// update window (binary swap); Windows uses the installer-based flow in
// updater_windows.go because a swap cannot apply an NSIS installer.
func runUpdateCheck(app *application.App, manual bool) {
	if runtime.GOOS == "windows" {
		windowsUpdateFlow(app, manual)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if !manual {
		rel, err := app.Updater.Check(ctx)
		if err != nil {
			log.Printf("updater: background check: %v", err)
			return
		}
		if rel == nil {
			return
		}
	}
	if err := app.Updater.CheckAndInstall(ctx); err != nil {
		log.Printf("updater: check: %v", err)
	}
}
