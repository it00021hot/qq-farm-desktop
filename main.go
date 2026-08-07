package main

import (
	"context"
	"embed"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/it00021hot/qq-farm-core/pkg/appserver"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/trayicon.png
var trayIcon []byte

//go:embed build/trayicon-template.png
var trayTemplateIcon []byte

const appVersion = "0.1.0"

func main() {
	dataRoot, err := appserver.ResolveDataRoot()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(dataRoot, 0o755); err != nil {
		log.Fatal(err)
	}
	// Prefer writable data dir for extracted bundled resources (single-exe mode).
	resourceRoot := dataRoot
	if err := ensureBundledFarmResources(resourceRoot); err != nil {
		log.Fatal("extract farm resources: ", err)
	}

	backend, err := appserver.Start(appserver.Options{
		Env:          "prod",
		Host:         appserver.DefaultHost,
		Port:         envOr("QQFARM_API_PORT", appserver.DefaultPort),
		DesktopMode:  true,
		ResourceRoot: resourceRoot,
		DataRoot:     dataRoot,
	})
	if err != nil {
		log.Fatal(err)
	}

	appService := &AppService{backend: backend, version: appVersion}

	app := application.New(application.Options{
		Name:        "QQ农场智能助手",
		Description: "QQ农场智能助手桌面端",
		Icon:        appIcon,
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Keep process alive when the last window is hidden to the tray.
			ApplicationShouldTerminateAfterLastWindowClosed: false,
		},
		OnShutdown: func() {
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			if err := backend.Shutdown(ctx); err != nil {
				log.Printf("backend shutdown: %v", err)
			}
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		// Leave native title empty — brand is already in the Vue header.
		Title:  "",
		Width:  1280,
		Height: 800,
		// Windows: custom chrome. macOS: keep AppKit frame + traffic lights.
		Frameless:                  runtime.GOOS != "darwin",
		DefaultContextMenuDisabled: true,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropNormal,
			// Hidden title + inset traffic lights (aligns with ~56px header/toolbar).
			TitleBar: application.MacTitleBarHiddenInset,
		},
		Windows: application.WindowsWindow{
			// Keep Aero shadow / rounded corners on Windows frameless.
			DisableFramelessWindowDecorations: false,
		},
		BackgroundColour:   application.NewRGB(245, 247, 250),
		URL:                "/",
		UseApplicationMenu: runtime.GOOS == "darwin",
	})

	setupMenusAndTray(app, window, appService)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
