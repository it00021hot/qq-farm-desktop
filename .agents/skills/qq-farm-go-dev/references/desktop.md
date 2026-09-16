# desktop 桌面壳 / 构建 / 发版参考

仓库根本身就是 Wails v3 桌面壳（模块 `github.com/it00021hot/qq-farm-desktop`，Wails **v3.0.0-beta.22** 锁定）。

## 根目录职责

```
main.go               # 应用入口；//go:embed all:frontend/dist → fs.Sub 取 assets
appservice.go         # 桌面服务注册（Wails service）
menu.go               # 托盘 / macOS 应用菜单（在浏览器中打开、检查更新…）
bundle_farm.go        # //go:embed all:bundled/resource/farm（首次运行解压到数据目录）
updater.go            # 更新器公共逻辑（检查 Release、SHA256 校验、触发平台安装）
updater_windows.go    # Windows：下载安装器 → 校验 → 退出应用 → 静默安装（NSIS per-user）
updater_notwindows.go # macOS：整包 .zip 原地替换
internal/ghrelease/   # Release 资产名匹配 + SHA256SUMS 校验（有测试，go test ./internal/ghrelease/）
build/                # wails3 任务与打包资源
  Taskfile.yml        # frontend 任务（pnpm-only）、bindings/icons 生成
  config.yml          # wails3 dev 配置 + info.version 版本号（手动维护）
  windows/ darwin/    # 图标、manifest、Info.plist、nsis/
scripts/
  sync-farm-bundle.sh          # core/resource/farm → bundled/resource/farm（解符号链接、去重图标）
  build-windows-exe.sh         # 裸 exe（调试用，不分发）
  build-windows-installer.sh   # 正式 Windows NSIS 安装包（需 makensis）
  build-macos-release.sh       # macOS 单架构 .app + zip(更新用) + dmg(首装)，MAC_ARCH=amd64|arm64
docs/RELEASE_CHECKLIST.md      # 发版验收清单
```

## 启动链与内嵌

`main.go`：`//go:embed all:frontend/dist` → `fs.Sub(assets, "frontend/dist")` → 注册 Wails 服务 → `appserver.Start`（core 的 `pkg/appserver`，WebFS 传 SPA）→ 进程内 Fiber API 监听 `127.0.0.1:9528`。

两个 embed 目录的产出链：

- `frontend/dist`：`cd frontend && pnpm run build:desktop`（产物就在子模块内，无需拷贝）
- `bundled/resource/farm`：`bash scripts/sync-farm-bundle.sh`（从 `core/resource/farm` 同步，必须先跑；`seed_images_named` 必须是真实目录）

`go build` 直接编译要求 `frontend/dist` 已存在（embed 目录缺失会编译失败）——先构建前端或跑 `wails3 task build`。

## wails3 版本一致性（升级必读）

三处必须同步为同一版本（当前 `v3.0.0-beta.22`）：

1. 本仓库 `go.mod` 的 `github.com/wailsapp/wails/v3`
2. 本机 CLI：`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.22`
3. `.github/workflows/release.yml` 的 `WAILS_VERSION`

升级后必跑 `wails3 task build` 验证（新版 task 对 YAML 更严格：`build/**/*.yml` 中纯标量含 `: ` 要加引号）。

## Taskfile 常用任务

| 命令 | 说明 |
|------|------|
| `wails3 task dev` | 一体调试：内嵌构建 + frontend vite desktop 模式（:9245 strictPort）+ 启动应用 |
| `wails3 task build` | 标准构建链：`go mod tidy` → frontend（pnpm install + bindings + `build:desktop`）→ 图标/syso → `go build -tags production` → `bin/qq-farm.exe` |
| `wails3 task package` | 构建并打安装包（Windows NSIS / macOS .app） |
| `wails3 task run` | 运行已构建产物 |

frontend 相关任务已收敛为 pnpm-only（web 是 pnpm workspace，`workspace:*` 依赖 npm 装不了）。

## 本地验证顺序（改动后的标准自检）

```bash
bash scripts/sync-farm-bundle.sh          # core 资源有改动时必跑
cd frontend && pnpm install && pnpm run build:desktop
cd .. && go build ./...
wails3 task build                         # 完整链路
go test ./internal/ghrelease/             # 改过更新器时
```

打包级验证：`VERSION=0.0.0-dev ./scripts/build-windows-exe.sh`（免 NSIS）；正式安装包在 Windows 上 `VERSION=x.y.z ./scripts/build-windows-installer.sh`。

## 子模块操作

```bash
# 新机器克隆
git clone --recurse-submodules git@github.com:it00021hot/qq-farm-desktop.git
git submodule update --init --recursive   # 已克隆补拉

# 升级/修改子模块（在子模块内提交推送后）
git add core frontend        # 记录新指针（这是普通提交内容，不是引用）
git commit -m "chore: 升级 core 至 vX.Y.Z（……）"
```

- core 的 Go 依赖通过 `go.mod` 的 `replace github.com/it00021hot/qq-farm-core => ./core` 生效；改 core 代码后无需动 go.mod，提交子模块指针即可。
- CI 用 `.gitmodules` 里的 SSH 地址拉子模块，workflow 在 checkout 前用 `git config --global url."https://github.com/".insteadOf "git@github.com:"` 重写为 HTTPS 走 token——新增子模块时确认这步仍覆盖到。
- frontend 的 `dist/`、`bindings/` 在子模块自身 .gitignore 中，构建产物不会弄脏指针状态；提交指针前 `git submodule status` 确认无 `+`/脏标记。

## 发版流程

1. 确认 `build/config.yml` 的 `info.version` 已更新到本次版本（手动维护；`build/windows/info.json`、`build/darwin/Info.plist` 的版本字段由打包脚本按 `VERSION` 自动改写，不用手改）。
2. 全部子模块指针就位、`wails3 task build` 本地通过。
3. 打 tag 触发 CI：
   ```bash
   git tag v0.2.0 && git push origin v0.2.0
   ```
4. GitHub Actions `Release` workflow（`.github/workflows/release.yml`）：单次 checkout + `submodules: recursive`，Windows 与 macOS(amd64/arm64 矩阵) 并行构建，`publish` job 汇总生成 `SHA256SUMS` 并创建 GitHub Release。
5. 按 `docs/RELEASE_CHECKLIST.md` 验收：产物五件（Windows installer、mac 双架构 zip+dmg）+ SHA256SUMS；**没有**便携 exe 与 universal 包。

更新器行为（改 updater*.go / internal/ghrelease 时注意）：启动约 5 秒静默检查 Releases；Windows 更新资产 = 安装器（静默安装），macOS = 对应架构的 `.zip`（非 dmg）；校验依赖 Release 内 `SHA256SUMS`。Windows 静默安装完成后由 NSIS 的 `.onInstSuccess`（仅 `/S` 生效）自动重启新版应用——该逻辑在安装器里，所以旧版本应用更新时也能享受；改 `build/windows/nsis/project.nsi` 后须用 makensis 本地编译验证再发版。
