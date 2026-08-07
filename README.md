# QQ农场智能助手 · 桌面端 (Wails v3)

基于 [Wails v3](https://v3.wails.io/) 的跨平台桌面壳：嵌入 [qq-farm-web](https://github.com/it00021hot/qq-farm-web) 前端，进程内启动 [qq-farm-core](https://github.com/it00021hot/qq-farm-core) Fiber API（`127.0.0.1:9528`），保留农场 WASM / WebSocket 能力。

> Wails v3 目前为 beta；CLI 请锁定 `wails3@v3.0.0-beta.4` 或与本仓库 `go.mod` 一致。

Go 模块：`github.com/it00021hot/qq-farm-desktop`  
依赖：`github.com/it00021hot/qq-farm-core`（默认从 GitHub tag 拉取，例如 `v0.1.0`）

## 架构

```
WebView (embedded Vue dist)
    │  HTTP + WS  →  http://127.0.0.1:9528
    ▼
Wails process
  ├─ frontend/dist (embed)
  └─ qq-farm-core/pkg/appserver  →  Fiber + farm runtime
```

- 无头 / Web 开发模式：`qq-farm-core` 的 `make run` + `qq-farm-web` 的 `pnpm dev`
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

同级目录建议同时克隆：

```bash
# .../qq-farm/
#   qq-farm-core/
#   qq-farm-web/
#   qq-farm-desktop/   ← 本仓库
```

## 开发

推荐双进程热更（与 Web 模式一致）：

```bash
# 终端 1
cd ../qq-farm-core && make run

# 终端 2
cd ../qq-farm-web && pnpm dev
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

## 打包

### Windows x64

```bash
cd qq-farm-desktop
chmod +x scripts/build-windows-exe.sh
./scripts/build-windows-exe.sh
# 产物：bin/qq-farm.exe（单文件，农场资源已内嵌）
```

| 文件 | 说明 |
|------|------|
| `bin/qq-farm.exe` | **推荐** Windows x64 单文件可执行程序 |
| `bin/qq-farm-windows-amd64.zip` | 可选便携包（含 WebView2 安装引导） |

目标机需 Windows 10+ 与 WebView2。打包脚本会先 `sync-farm-bundle`（展开 `seed_images_named`）并构建 `qq-farm-web` 的 `build:desktop`。

### macOS

```bash
cd qq-farm-desktop
bash scripts/sync-farm-bundle.sh
(cd frontend && node scripts/build.mjs)
CGO_ENABLED=1 go build -tags production -trimpath -ldflags="-w -s" -o bin/qq-farm .
# 组装 .app 见 build/darwin/Taskfile.yml，或沿用现有 bin/qq-farm.app 结构
```

## 窗口与托盘

- 关闭窗口 → 隐藏到系统托盘（不退出）
- 托盘：显示主窗口 / 打开数据目录 / 关于 / 退出
- macOS：隐藏标题栏 + 原生圆角与红绿灯；侧栏底部为品牌与账号切换
- Windows：无边框；最小化 / 最大化 / 关闭并入顶栏右侧

## 与纯 HTTP 服务的关系

| 入口 | 用途 |
|------|------|
| [`qq-farm-core/cmd/app`](../qq-farm-core/cmd/app) | 纯 HTTP 服务（浏览器 / 远程） |
| [`qq-farm-desktop/`](.) | 桌面窗口 + 同进程 API |

业务逻辑在 `qq-farm-core`；桌面通过公开包 [`pkg/appserver`](../qq-farm-core/pkg/appserver) 启动（避免跨 module 引用 `internal/`）。本地联调可临时 `replace`，发布依赖 GitHub tag。

## 前端 desktop mode

- [`qq-farm-web/.env.desktop`](../qq-farm-web/.env.desktop)：`VITE_IS_DESKTOP=Y`，`VITE_SERVICE_BASE_URL=http://127.0.0.1:9528`，hash 路由，关闭代理
- `pnpm build:desktop`
- WebSocket 从 `VITE_SERVICE_BASE_URL` 推导 host（见 `src/hooks/business/farm-ws.ts`）
