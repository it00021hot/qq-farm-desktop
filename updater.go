package main

import (
	"context"
	"log"
	"time"

	"github.com/it00021hot/qq-farm-desktop/internal/ghrelease"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const githubReleaseRepo = "it00021hot/qq-farm-desktop"

// setupUpdater wires GitHub Releases auto-update. Manual checks use the
// builtin window; startup only opens it when an update is available.
func setupUpdater(app *application.App) {
	gh, err := github.New(github.Config{
		Repository:    githubReleaseRepo,
		ChecksumAsset: "SHA256SUMS",
		AssetMatcher:  ghrelease.AssetMatcher,
	})
	if err != nil {
		log.Printf("updater: github provider: %v", err)
		return
	}

	if err := app.Updater.Init(updater.Config{
		CurrentVersion: appVersion,
		Providers:      []updater.Provider{gh},
	}); err != nil {
		log.Printf("updater: init: %v", err)
		return
	}

	go func() {
		time.Sleep(5 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		rel, err := app.Updater.Check(ctx)
		if err != nil {
			log.Printf("updater: background check: %v", err)
			return
		}
		if rel == nil {
			return
		}
		if err := app.Updater.CheckAndInstall(ctx); err != nil {
			log.Printf("updater: install: %v", err)
		}
	}()
}

func checkForUpdates(app *application.App) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := app.Updater.CheckAndInstall(ctx); err != nil {
			log.Printf("updater: check: %v", err)
		}
	}()
}
