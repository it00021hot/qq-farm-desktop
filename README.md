# QQ农场智能助手 · 桌面端 (Wails v3)

基于 [Wails v3](https://v3.wails.io/) 的 macOS 桌面壳：嵌入 Vue 前端，进程内启动现有 Fiber API（`127.0.0.1:9528`），保留农场 WASM / WebSocket 能力。

> Wails v3 目前为 beta；CLI 请锁定 `wails3@v3.0.0-beta.4` 或与本仓库 `go.mod` 一致。

## 架构

```
WebView (embedded Vue dist)
    │  HTTP + WS  →  http://127.0.0.1:9528
    ▼
Wails process
  ├─ frontend/dist (embed)
  └─ go-skeleton/pkg/appserver  →  Fiber + farm runtime
```

- 无头 / Web 开发模式不变：`go-framework` 的 `make run` + `vue-framework` 的 `pnpm dev`
- 桌面数据目录（sqlite / logs / tsdk）：`~/Library/Application Support/QQFarm`
- 只读资源（wasm / gameConfig）：`.app/Contents/Resources/resource/farm`

## 前置

```bash
# Go 1.25+、Xcode CLT、CGO、pnpm、Wails CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.4
wails3 doctor
```

## 开发

推荐继续用双进程热更（与原来一致）：

```bash
# 终端 1
cd go-framework && make run

# 终端 2
cd vue-framework && pnpm dev
```

桌面一体调试：

```bash
cd desktop
wails3 build DEV=true   # 或 wails3 package 后打开 .app
# 或：
wails3 task darwin:run  # 构建并启动 .dev.app（会拷贝 farm 资源）
```

环境变量：

| 变量 | 说明 |
|------|------|
| `QQFARM_API_PORT` | Fiber 端口，默认 `9528` |
| `QQFARM_RESOURCE_ROOT` | 覆盖资源根目录 |
| `QQFARM_DATA_ROOT` | 覆盖可写数据目录 |

## 打包 (Windows x64)

```bash
cd desktop
chmod +x scripts/build-windows-exe.sh
./scripts/build-windows-exe.sh
# 产物：bin/qq-farm.exe（单文件，农场资源已内嵌，双击即可）
```

| 文件 | 说明 |
|------|------|
| `bin/qq-farm.exe` | **推荐** Windows x64 单文件可执行程序 |
| `bin/qq-farm-windows-amd64.zip` | 可选便携包（含 WebView2 安装引导） |

说明：本机 Homebrew `makensis` 无法稳定生成 NSIS 安装向导，因此主交付物为单个 `.exe`（资源 `go:embed` 进二进制，首次运行解压到 `%LOCALAPPDATA%\QQFarm`）。目标机需 Windows 10+ 与 WebView2。

打包前会执行 `scripts/sync-farm-bundle.sh`，把 `seed_images_named`（作物/活动图标）从符号链接展开为真实文件再嵌入；升级后若图标仍缺失，删除数据目录下的 `resource/farm` 即可触发重新解压。

托盘与菜单：

- 关闭窗口 → 隐藏到系统托盘（不退出）
- 托盘菜单：显示主窗口 / 打开数据目录 / 关于 / 退出
- macOS 菜单栏：标准 App 菜单（含退出 Cmd+Q）；点击 Dock 图标可重新显示窗口

## 与 cmd/app 的关系

| 入口 | 用途 |
|------|------|
| [`go-framework/cmd/app`](../go-framework/cmd/app) | 纯 HTTP 服务（浏览器 / 远程） |
| [`desktop/`](.) | 桌面窗口 + 同进程 API |

业务逻辑共用 `go-skeleton`；桌面通过公开包 [`pkg/appserver`](../go-framework/pkg/appserver) 启动（避免跨 module 引用 `internal/`）。

## 前端 desktop mode

- [`vue-framework/.env.desktop`](../vue-framework/.env.desktop)：`VITE_SERVICE_BASE_URL=http://127.0.0.1:9528`，`hash` 路由，关闭代理
- `pnpm build:desktop`
- WebSocket 从 `VITE_SERVICE_BASE_URL` 推导 host（见 `src/hooks/business/farm-ws.ts`）
