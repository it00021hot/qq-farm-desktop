package main

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/it00021hot/qq-farm-core/pkg/appserver"
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
	return openURI(dir)
}

func (a *AppService) OpenInBrowser() error {
	return openURI(a.GetApiBaseURL() + "/")
}

func openURI(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}
