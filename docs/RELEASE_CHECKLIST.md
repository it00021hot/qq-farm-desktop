# Release verification checklist

Use after pushing a `v*` tag (or `workflow_dispatch` with a tag).

## CI / Release page

- [ ] Actions workflow **Release** is green (windows + macos + publish)
- [ ] GitHub Release for the tag includes:
  - `qq-farm-windows-amd64-installer.exe`
  - `qq-farm-windows-amd64.exe`
  - `qq-farm-darwin.dmg`
  - `qq-farm-darwin-universal.zip`
  - `SHA256SUMS`
- [ ] `SHA256SUMS` digests match the four binaries
- [ ] Update asset names: Windows exe has **no** `installer` in the name; mac update is **`.zip`**, not `.dmg`

## Windows

- [ ] Run the installer (per-user, no admin prompt)
- [ ] App starts; tray shows **检查更新**
- [ ] Publish a higher version tag → **检查更新** downloads exe, replaces, relaunches

## macOS

- [ ] Open DMG, drag to Applications (or `~/Applications`)
- [ ] First open may need Privacy & Security allow (ad-hoc signed)
- [ ] Tray / 应用 menu **检查更新** works against a newer Release zip

## Local smoke (optional)

```bash
go test ./internal/ghrelease/
VERSION=0.0.0-dev ./scripts/build-macos-release.sh   # on macOS
VERSION=0.0.0-dev ./scripts/build-windows-installer.sh  # needs NSIS
```
