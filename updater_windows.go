//go:build windows

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// Windows updates are installer-based: releases ship only the NSIS
// installer, and wails' binary-swap helper cannot apply an installer. So we
// check via the wails provider, download + SHA-256 the installer ourselves,
// then hand off to a silent install from a detached script after this
// process exits (the installer cannot overwrite a running exe).

const (
	mbOK            = 0x00000000
	mbYesNo         = 0x00000004
	mbIconError     = 0x00000010
	mbIconQuestion  = 0x00000020
	mbIconInfo      = 0x00000040
	mbTopmost       = 0x00040000
	mbSetForeground = 0x00010000
	idYes           = 6
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = user32.NewProc("MessageBoxW")
)

func messageBox(title, text string, flags uint32) {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(flags),
	)
}

func askYesNo(text, title string) bool {
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	ret, _, _ := procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(m)),
		uintptr(unsafe.Pointer(t)),
		uintptr(mbYesNo|mbIconQuestion|mbTopmost|mbSetForeground),
	)
	return ret == idYes
}

// windowsUpdateFlow checks GitHub and, with the user's consent, downloads the
// new installer and runs it silently after quitting.
func windowsUpdateFlow(app *application.App, manual bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	rel, err := app.Updater.Check(ctx)
	if err != nil {
		log.Printf("updater: check: %v", err)
		if manual {
			messageBox("软件更新", fmt.Sprintf("检查更新失败：\n%v", err), mbOK|mbIconError|mbTopmost)
		}
		return
	}
	if rel == nil {
		if manual {
			messageBox("软件更新", "当前已是最新版本 v"+appVersion, mbOK|mbIconInfo|mbTopmost)
		}
		return
	}

	if !askYesNo(fmt.Sprintf("发现新版本 v%s，现在下载并安装吗？", rel.Version), "软件更新") {
		return
	}

	installer, err := downloadVerifiedInstaller(ctx, rel)
	if err != nil {
		log.Printf("updater: download: %v", err)
		messageBox("软件更新", fmt.Sprintf("下载更新失败：\n%v", err), mbOK|mbIconError|mbTopmost)
		return
	}

	if !askYesNo("更新包已下载并通过校验。\n\n点击“是”后应用将退出并自动完成安装，成功后会自动重新打开。", "软件更新") {
		_ = os.Remove(installer)
		return
	}

	if err := scheduleSilentInstall(installer); err != nil {
		log.Printf("updater: schedule install: %v", err)
		messageBox("软件更新", fmt.Sprintf("启动安装程序失败：\n%v", err), mbOK|mbIconError|mbTopmost)
		return
	}
	app.Quit()
}

// downloadVerifiedInstaller streams the release asset to %TEMP% and verifies
// its SHA-256 against the digest the provider parsed from SHA256SUMS.
func downloadVerifiedInstaller(ctx context.Context, rel *updater.Release) (string, error) {
	rawURL, _ := rel.Metadata["github.asset.url"].(string)
	if rawURL == "" {
		return "", fmt.Errorf("release metadata has no asset url")
	}
	if rel.Verification == nil ||
		!strings.EqualFold(rel.Verification.DigestAlgo, "sha256") ||
		len(rel.Verification.Digest) != sha256.Size {
		return "", fmt.Errorf("release carries no sha256 digest; refusing to install")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := newUpdaterHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: http %d", rel.Artifact.Filename, resp.StatusCode)
	}

	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("qq-farm-update-%s.exe", rel.Version))
	out, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(io.MultiWriter(out, hash), resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", closeErr
	}
	if !bytes.Equal(hash.Sum(nil), rel.Verification.Digest) {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("checksum mismatch for %s", rel.Artifact.Filename)
	}
	return tmpPath, nil
}

// updateResultMarker is written by the NSIS installer (.onInstSuccess) right
// before it relaunches the app, holding the freshly installed version. The
// relaunched instance shows a one-shot confirmation and deletes the file — so
// even if the auto-relaunch itself fails, the next manual start still reports
// the completed update.
const updateResultMarker = "qq-farm-update-result.txt"

// notifyUpdateResult reports the completed silent update, if any. Windows
// only; other platforms get the wails updater's own restart window.
func notifyUpdateResult() {
	path := filepath.Join(os.TempDir(), updateResultMarker)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = os.Remove(path)
	version := strings.TrimSpace(string(data))
	if version == "" {
		return
	}
	messageBox("软件更新", fmt.Sprintf("已成功更新到 v%s。", version), mbOK|mbIconInfo|mbTopmost)
}

// scheduleSilentInstall writes a detached batch script that waits for this
// process to exit, then runs the installer with NSIS silent flags; a non-zero
// installer exit pops a failure notice instead of leaving the user guessing.
// The batch deletes itself afterwards.
func scheduleSilentInstall(installer string) error {
	bat := filepath.Join(os.TempDir(), "qq-farm-update-install.cmd")
	failVBS := filepath.Join(os.TempDir(), "qq-farm-update-fail.vbs")
	// The batch must stay pure ASCII (cmd parses it in the system codepage);
	// the Chinese failure notice lives in a separate UTF-16 .vbs instead.
	script := "@echo off\r\n" +
		"timeout /t 8 /nobreak >nul\r\n" +
		fmt.Sprintf("%q /S\r\n", installer) +
		fmt.Sprintf("if errorlevel 1 wscript.exe %q\r\n", failVBS) +
		"del \"%~f0\"\r\n"
	if err := writeVBScript(failVBS, "MsgBox \"自动更新安装失败，应用已退出。请到 GitHub Releases 页面手动下载安装包重新安装。\", vbExclamation, \"QQ Farm 更新\"\r\n"); err != nil {
		return err
	}
	if err := os.WriteFile(bat, []byte(script), 0o644); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", bat)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

// writeVBScript writes content as UTF-16LE with BOM so wscript renders the
// Chinese text correctly regardless of the system codepage.
func writeVBScript(path, content string) error {
	runes := utf16.Encode([]rune(content))
	buf := make([]byte, 2+2*len(runes))
	binary.LittleEndian.PutUint16(buf[0:2], 0xFEFF)
	for i, r := range runes {
		binary.LittleEndian.PutUint16(buf[2+2*i:], r)
	}
	return os.WriteFile(path, buf, 0o644)
}
