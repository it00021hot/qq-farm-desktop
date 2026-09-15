# Release verification checklist

Use after pushing a `v*` tag (or `workflow_dispatch` with a tag).

## CI / Release page

- [ ] Actions workflow **Release** is green (windows + macos (amd64/arm64 matrix) + publish)
- [ ] GitHub Release for the tag includes:
  - `qq-farm-windows-amd64-installer.exe`
  - `qq-farm-darwin-amd64.zip` / `qq-farm-darwin-amd64.dmg`
  - `qq-farm-darwin-arm64.zip` / `qq-farm-darwin-arm64.dmg`
  - `SHA256SUMS`
- [ ] `SHA256SUMS` digests match the five artifacts
- [ ] No portable `qq-farm-windows-amd64.exe` and no `*-universal.zip` (dropped in v0.1.22)
- [ ] Update assets: Windows update = the **installer** (applied via silent install); mac update = **`.zip`**, not `.dmg`

## Windows

- [ ] Run the installer (per-user, no admin prompt)
- [ ] App starts; tray shows **检查更新**
- [ ] Publish a higher version tag → **检查更新** downloads the installer, verifies SHA-256, quits the app and installs silently

## macOS

- [ ] Open the DMG matching the Mac's architecture, drag to Applications (or `~/Applications`)
- [ ] First open may need Privacy & Security allow (ad-hoc signed)
- [ ] Tray / 应用 menu **检查更新** works against a newer Release zip
- [ ] Intel Mac only sees `qq-farm-darwin-amd64.zip` as update; Apple Silicon only `qq-farm-darwin-arm64.zip`

## Upgrade notes

- Installs of v0.1.21 or older (old matcher) no longer find updates on Windows
  (portable exe removed) or may pick the wrong mac arch zip — reinstall from
  the new Release once.

## Local smoke (optional)

```bash
go test ./internal/ghrelease/
VERSION=0.0.0-dev MAC_ARCH=arm64 ./scripts/build-macos-release.sh   # on macOS
VERSION=0.0.0-dev MAC_ARCH=amd64 ./scripts/build-macos-release.sh   # on macOS (Intel build)
VERSION=0.0.0-dev ./scripts/build-windows-installer.sh  # needs NSIS
```
