# QQ农场智能助手 · 桌面端 (Wails v3)

> 维护状态：本仓库已恢复维护（2026-09-11 起随 Go 主力版本一同更新桌面端）。

基于 [Wails v3](https://v3.wails.io/) 的跨平台桌面壳：嵌入 [qq-farm-web](https://github.com/it00021hot/qq-farm-web) 前端，进程内启动 [qq-farm-core](https://github.com/it00021hot/qq-farm-core) Fiber API（`127.0.0.1:9528`），保留农场 WASM / WebSocket 能力。

> Wails v3 目前为 beta；CLI 请锁定 `wails3@v3.0.0-beta.4` 或与本仓库 `go.mod` 一致。

Go 模块：`github.com/it00021hot/qq-farm-desktop`  
依赖：`github.com/it00021hot/qq-farm-core` 与 `qq-farm-web` 均以 **git submodule** 形式随本仓库分发（`core/`、`frontend/`），Go 侧通过 `replace => ./core` 本地链接。

## 架构

```
WebView (embedded Vue dist)          Browser (本机)
    │                                      │
    │  Wails AssetServer                   │  SPA + API
    │                                      ▼
    │                               http://127.0.0.1:9528/
    │  HTTP + WS  ─────────────────────────┘
    ▼
Wails process
  ├─ frontend/dist (embed → WebView + Fiber)
  └─ qq-farm-core/pkg/appserver  →  Fiber API + 可选 Web UI
```

- 桌面运行后，本机浏览器可打开 **http://127.0.0.1:9528/** 使用完整管理页面（与窗口共用同一套 API / JWT；托盘或 macOS「应用」菜单可点「在浏览器中打开」）
- 无头 / 纯 Web 开发：`qq-farm-core` 的 `make run` + `qq-farm-web` 的 `pnpm dev`（core 单独启动不托管 SPA）
- 桌面数据目录（sqlite / logs / tsdk）
  - macOS：`~/Library/Application Support/QQFarm`
  - Windows：`%LOCALAPPDATA%\QQFarm`
- 农场资源：打包时 `go:embed` 进二进制，首次运行解压到数据目录下的 `resource/farm`

## 前置

```bash
# Go 1.25+、（macOS）Xcode CLT + CGO、pnpm、Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.4
wails3 doctor
```

克隆本仓库（连同子模块）：

```bash
git clone --recurse-submodules git@github.com:it00021hot/qq-farm-desktop.git
# 已克隆过的补拉子模块：
git submodule update --init --recursive
```

仓库结构：

```
qq-farm-desktop/
├── core/       ← submodule: qq-farm-core（Go 后端）
├── frontend/   ← submodule: qq-farm-web（Vue3 前端，构建产物 dist 被 go:embed）
└── ...         ← Wails 桌面壳
```

## 开发

推荐双进程热更（与 Web 模式一致）：

```bash
# 终端 1
cd core && make run

# 终端 2
cd frontend && pnpm dev
```

桌面一体调试：

```bash
cd qq-farm-desktop
wails3 task darwin:run   # macOS：构建并启动 .dev.app
# 或直接：
go build -tags production -o bin/qq-farm . && open bin/qq-farm.app
```

环境变量：

| 变量 | 说明 |
|------|------|
| `QQFARM_API_PORT` | Fiber 端口，默认 `9528` |
| `QQFARM_RESOURCE_ROOT` | 覆盖资源根目录 |
| `QQFARM_DATA_ROOT` | 覆盖可写数据目录 |

## 安装与自动更新

正式分发使用**安装包**（非便携双击 exe）。打 Git tag 后由 GitHub Actions 自动构建并上传到 [Releases](https://github.com/it00021hot/qq-farm-desktop/releases)。

| 平台 | 首次安装 | 自动更新资产 |
|------|----------|--------------|
| Windows x64 | `qq-farm-windows-amd64-installer.exe`（用户级，无需管理员） | 同左（更新器下载安装器静默安装） |
| macOS（Apple Silicon） | `qq-farm-darwin-arm64.dmg`（拖到 Applications） | `qq-farm-darwin-arm64.zip`（整包 `.app`） |
| macOS（Intel） | `qq-farm-darwin-amd64.dmg`（拖到 Applications） | `qq-farm-darwin-amd64.zip`（整包 `.app`） |

客户端内置 Wails Updater：启动约 5 秒后静默检查 GitHub Releases；托盘 / macOS「应用」菜单有「检查更新」。校验依赖同 Release 中的 `SHA256SUMS`。Windows 上更新走「下载安装器 → 校验 → 退出应用 → 静默安装」流程；macOS 仍为二进制原地替换。

### 发版

发布流程完全由 GitHub Actions 流水线触发，**本地不需要手动构建前端或同步 dist**。`frontend/dist` 不入库（子模块自身 `.gitignore`）；打 tag 后流水线会：

1. checkout 本仓库并 `submodules: recursive` 拉取 `core/` 与 `frontend/`
2. 在流水线内 `pnpm install --frozen-lockfile` + `pnpm run build:desktop` 构建前端 dist 并嵌入（见 [`scripts/build-macos-release.sh`](scripts/build-macos-release.sh) / [`scripts/build-windows-installer.sh`](scripts/build-windows-installer.sh)）
3. 并行构建 Windows 安装包 + macOS 双架构（amd64/arm64 矩阵），汇总校验和后创建同名 Release

```bash
git tag v0.2.0
git push origin v0.2.0
```

Actions 会并行构建 Windows + macOS，汇总校验和后创建同名 Release。也可用 Actions 里的 `workflow_dispatch` 手动重跑。

发版后验收见 [`docs/RELEASE_CHECKLIST.md`](docs/RELEASE_CHECKLIST.md)。

### 本地打包（调试）

仅限调试，正式分发一律走上面的流水线：

```bash
# Windows（需 makensis；可在 macOS/Linux 交叉编译 exe，NSIS 建议在 Windows 上打）
VERSION=0.2.0 ./scripts/build-windows-installer.sh

# macOS（须在 macOS 上；按架构分别构建）
MAC_ARCH=arm64 VERSION=0.2.0 ./scripts/build-macos-release.sh  # Apple Silicon
MAC_ARCH=amd64 VERSION=0.2.0 ./scripts/build-macos-release.sh  # Intel
```

本地调试用裸 exe（不分发）：

```bash
VERSION=0.2.0 ./scripts/build-windows-exe.sh
```

目标机需 Windows 10+ 与 WebView2；macOS 12+。无 Apple Developer ID 时，macOS 首次打开可能需在「隐私与安全性」中允许。

## 窗口与托盘

- 关闭窗口 → 隐藏到系统托盘（不退出）
- 托盘：显示主窗口 / **在浏览器中打开** / 打开数据目录 / 检查更新 / 关于 / 退出
- macOS：隐藏标题栏 + 原生圆角与红绿灯；侧栏底部为品牌与账号切换；「应用」菜单含「在浏览器中打开」
- Windows：无边框；最小化 / 最大化 / 关闭并入顶栏右侧（仅 WebView；浏览器打开同一页面时不显示窗控）

## 与纯 HTTP 服务的关系

| 入口 | 用途 |
|------|------|
| [`core/cmd/app`](core/cmd/app) | 纯 HTTP API（需另挂 `qq-farm-web` dist 或 `pnpm dev`） |
| [`qq-farm-desktop/`](.) | 桌面窗口 + 同进程 API **并托管 Web 页面**（本机 `http://127.0.0.1:9528/`） |

业务逻辑在 `core`（qq-farm-core）；桌面通过公开包 [`pkg/appserver`](core/pkg/appserver) 启动（避免跨 module 引用 `internal/`）。本地开发由 `replace => ./core` 指向子模块，发布时升 core 版本号后更新子模块指针即可。

## 前端 desktop mode

- [`frontend/.env.desktop`](frontend/.env.desktop)：`VITE_IS_DESKTOP=Y`，`VITE_SERVICE_BASE_URL=http://127.0.0.1:9528`，hash 路由，关闭代理
- 前端产物由发版流水线构建并嵌入，**本地无需 `pnpm build:desktop` 或手动拷贝 dist**
- 本地前端联调直接 `cd frontend && pnpm dev`（桌面窗口内用同一套 API）
- WebSocket 从 `VITE_SERVICE_BASE_URL` 推导 host（见 `src/hooks/business/farm-ws.ts`）
