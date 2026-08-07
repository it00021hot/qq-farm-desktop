#!/usr/bin/env bash
# Build Windows amd64 portable zip from an already-built bin/qq-farm.exe
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f bin/qq-farm.exe ]]; then
  echo "bin/qq-farm.exe missing. Run: wails3 package GOOS=windows ARCH=amd64" >&2
  exit 1
fi

STAGE="bin/qq-farm-windows-amd64"
rm -rf "$STAGE" "bin/qq-farm-windows-amd64.zip"
mkdir -p "$STAGE/resource"
cp bin/qq-farm.exe "$STAGE/"
mkdir -p bin/resource
if [[ ! -d bin/resource/farm ]]; then
  cp -R ../qq-farm-core/resource/farm bin/resource/farm
fi
cp -R bin/resource/farm "$STAGE/resource/farm"

if [[ -f build/windows/nsis/MicrosoftEdgeWebview2Setup.exe ]]; then
  cp build/windows/nsis/MicrosoftEdgeWebview2Setup.exe "$STAGE/"
fi

cat > "$STAGE/README.txt" <<'EOF'
QQ Farm Assistant (Windows x64)
================================

Requirements
- Windows 10 / 11 x64
- Microsoft Edge WebView2 Runtime
  (usually preinstalled; if the app fails to open, run MicrosoftEdgeWebview2Setup.exe)

Run
1. Unzip this folder anywhere (keep resource/ next to qq-farm.exe)
2. Double-click qq-farm.exe
   or run install-user.bat for per-user install + shortcuts

Data directory
- %LOCALAPPDATA%\QQFarm

Default login
- admin / admin888
EOF

cat > "$STAGE/install-user.bat" <<'EOF'
@echo off
setlocal
set DEST=%LOCALAPPDATA%\Programs\QQ Farm Assistant
echo Installing to "%DEST%" ...
mkdir "%DEST%" 2>nul
xcopy /E /I /Y "%~dp0*" "%DEST%\" >nul
powershell -NoProfile -Command ^
  "$s=(New-Object -ComObject WScript.Shell); $sc=$s.CreateShortcut([Environment]::GetFolderPath('Desktop')+'\QQ Farm Assistant.lnk'); $sc.TargetPath='%DEST%\qq-farm.exe'; $sc.WorkingDirectory='%DEST%'; $sc.Save()"
powershell -NoProfile -Command ^
  "$dir=[Environment]::GetFolderPath('StartMenu')+'\Programs'; New-Item -ItemType Directory -Force -Path $dir | Out-Null; $s=(New-Object -ComObject WScript.Shell); $sc=$s.CreateShortcut($dir+'\QQ Farm Assistant.lnk'); $sc.TargetPath='%DEST%\qq-farm.exe'; $sc.WorkingDirectory='%DEST%'; $sc.Save()"
echo Done. Shortcut created on Desktop.
pause
EOF

(cd bin && zip -r -q qq-farm-windows-amd64.zip qq-farm-windows-amd64)
ls -lh bin/qq-farm-windows-amd64.zip
echo "OK: bin/qq-farm-windows-amd64.zip"
