package main

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/MQEnergy/go-skeleton/pkg/appserver"
)

// AppService exposes thin desktop helpers to the WebView.
type AppService struct {
	backend *appserver.Server
	version string
}

func (a *AppService) GetApiBaseURL() string {
	if a.backend == nil {
		return "http://127.0.0.1:9528"
	}
	return a.backend.BaseURL()
}

func (a *AppService) GetAppVersion() string {
	if a.version == "" {
		return "0.1.0"
	}
	return a.version
}

func (a *AppService) GetDataDir() string {
	root, err := appserver.ResolveDataRoot()
	if err != nil {
		return ""
	}
	return root
}

func (a *AppService) OpenDataDir() error {
	dir := a.GetDataDir()
	if dir == "" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
