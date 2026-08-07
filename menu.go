package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// setupMenusAndTray wires the macOS/Windows app menu, system tray, and
// close-to-tray behaviour. Quit is only via menu (or Cmd+Q / tray Quit).
func setupMenusAndTray(app *application.App, window *application.WebviewWindow, appService *AppService) {
	// macOS: keep a minimal system menu bar (About / Quit via AppMenu).
	// Windows/Linux: no in-window menu bar — actions live on the tray menu only.
	if runtime.GOOS == "darwin" {
		setupApplicationMenu(app, appService)
	}
	setupSystemTray(app, window, appService)

	// Closing the window hides to tray instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	if runtime.GOOS == "darwin" {
		app.Event.OnApplicationEvent(events.Mac.ApplicationShouldHandleReopen, func(event *application.ApplicationEvent) {
			window.Show().Focus()
		})
	}
}

func setupApplicationMenu(app *application.App, appService *AppService) {
	menu := app.NewMenu()
	menu.AddRole(application.AppMenu)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.WindowMenu)

	appMenu := menu.AddSubmenu("应用")
	appMenu.Add("打开数据目录").OnClick(func(ctx *application.Context) {
		_ = appService.OpenDataDir()
	})
	appMenu.Add("检查更新").OnClick(func(ctx *application.Context) {
		checkForUpdates(app)
	})

	app.Menu.Set(menu)
}

func setupSystemTray(app *application.App, window *application.WebviewWindow, appService *AppService) {
	tray := app.SystemTray.New()
	tray.SetTooltip("QQ农场智能助手")

	if runtime.GOOS == "darwin" {
		// Menu bar: template icon (silhouette + alpha) so macOS tints for light/dark.
		if len(trayTemplateIcon) > 0 {
			tray.SetTemplateIcon(trayTemplateIcon)
		} else if len(trayIcon) > 0 {
			tray.SetIcon(trayIcon)
		}
	} else if len(trayIcon) > 0 {
		tray.SetIcon(trayIcon)
		tray.SetDarkModeIcon(trayIcon)
	}

	trayMenu := app.NewMenu()
	trayMenu.Add("显示主窗口").OnClick(func(ctx *application.Context) {
		window.Show().Focus()
	})
	trayMenu.Add("打开数据目录").OnClick(func(ctx *application.Context) {
		_ = appService.OpenDataDir()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("检查更新").OnClick(func(ctx *application.Context) {
		checkForUpdates(app)
	})
	trayMenu.Add("关于").OnClick(func(ctx *application.Context) {
		app.Menu.ShowAbout()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)

	tray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show().Focus()
		}
	})
}
