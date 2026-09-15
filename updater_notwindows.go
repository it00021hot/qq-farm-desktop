//go:build !windows

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// windowsUpdateFlow is implemented only on Windows (updater_windows.go):
// other platforms use the wails updater's binary-swap flow and never call it.
// The stub exists so the non-windows builds compile.
func windowsUpdateFlow(app *application.App, manual bool) {
	_ = app
	_ = manual
}
