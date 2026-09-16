---
name: qq-farm-go-dev
description: QQ农场智能助手 qq-farm-go 仓库（Wails v3 桌面壳 + core/ Go 后端 + frontend/ Vue3 管理端）的开发规范、架构与流程。凡在该仓库内新增或修改 Go/Vue 代码、加接口、加页面、构建打包、更新子模块、升级依赖、发版，都应使用本 skill。注意：core 实际是 Fiber v3 + GORM 技术栈，不是 GoFrame——不要套用 goframe-v2 skill 的模式。
---

# qq-farm-go 开发指南

本仓库是**单仓库三组件**：仓库根 = Wails v3 桌面壳（Go），`core/` 与 `frontend/` 是 git 子模块（后端 API、管理端前端）。子模块随仓库分发，不要引用仓库外的兄弟目录。

## 架构总览

```
桌面模式（本仓库产物）
┌─────────────────────────────────────────────┐
│ Wails 进程 (qq-farm.exe)                     │
│  ├─ WebView：frontend/dist (go:embed 内嵌)   │
│  ├─ 进程内启动 core pkg/appserver            │
│  │   → Fiber API http://127.0.0.1:9528      │
│  ├─ farm WASM/资源 bundle (bundle_farm.go)   │
│  └─ 自动更新 (updater*.go → GitHub Releases) │
└─────────────────────────────────────────────┘

Web 模式（开发/部署管理端）
  core `make run` (Fiber :9528)  ←proxy──  frontend `pnpm dev` (:9529)
```

- 前端与 API 的契约：HTTP 200 + JSON 包络 `{code, requestId, msg, data}`，`code=0` 为成功（见 `pkg/response`，前端 `@sa/axios` 对应解析）。
- 实时事件：前端 `useFarmWs` 连 `GET /farm/ws?token=...`，订阅 core 进程内 `internal/farm/hub`。
- 桌面数据目录：Windows `%LOCALAPPDATA%\QQFarm`，macOS `~/Library/Application Support/QQFarm`。

## 仓库布局

```
qq-farm-go/                  ← 仓库根 = 桌面壳
├── core/                    ← 子模块 github.com/it00021hot/qq-farm-core（Go 后端）
├── frontend/                ← 子模块 github.com/it00021hot/qq-farm-web（Vue3 前端）
├── main.go                  # //go:embed all:frontend/dist
├── appservice.go            # 桌面服务注册
├── bundle_farm.go           # //go:embed all:bundled/resource/farm
├── menu.go / updater*.go    # 托盘菜单 / 平台更新器
├── internal/ghrelease/      # Release 资产匹配 + SHA256 校验
├── go.mod                   # replace github.com/it00021hot/qq-farm-core => ./core
├── build/                   # wails3 任务/打包资源 (Taskfile.yml, config.yml, windows/, darwin/)
├── scripts/                 # 打包与资源同步脚本（见 references/desktop.md）
├── bundled/resource/farm/   # 同步自 core/resource/farm（脚本生成，勿手改）
└── docs/RELEASE_CHECKLIST.md
```

## 技术栈速览

| 组件 | 技术 | 入口 |
|------|------|------|
| 桌面壳（本仓库） | Go 1.26 + Wails v3 **beta.22**（锁定） | `main.go` |
| core 后端 | **Fiber v3 + GORM + Viper + slog** + Turso(SQLite) | `core/cmd/app`，`core/pkg/appserver`（桌面用） |
| frontend 前端 | SoybeanAdmin 2.2.1（Vue 3.5 + Vite 8 + TS strict + Naive UI + UnoCSS + pinia） | `frontend/src`，构建 `pnpm build:desktop` |

## 常用命令

```bash
# Web 联调（两个终端）
cd core && make run            # Fiber API :9528（-e dev）
cd frontend && pnpm dev        # Vite :9529，代理到 9528

# 桌面一体调试（内嵌前端 :9245 热更）
wails3 task dev

# 桌面构建
wails3 task build                                   # 标准链路
VERSION=x.y.z ./scripts/build-windows-exe.sh        # 裸 exe 调试
VERSION=x.y.z ./scripts/build-windows-installer.sh  # 正式安装包（需 NSIS）

# 测试 / 检查
cd core && go test ./... && make lint     # gofumpt
cd frontend && pnpm typecheck && pnpm lint && pnpm fmt
cd . && go test ./internal/ghrelease/
```

## 硬性规则

1. **core 不是 GoFrame**。技术栈是 Fiber v3 + GORM + Viper + slog；分层为 `routes → controller → service → vars.DB/dao → model`。写 core 代码前先读 `references/core.md`，不要套用任何 GoFrame 的 api/g.Meta/dao 生成模式。
2. **wails3 版本三处一致**：`go.mod`、本机 CLI、CI 的 `WAILS_VERSION`（当前 `v3.0.0-beta.22`）。升级时三处必须同步，且跑一遍 `wails3 task build` 验证。
3. **生成物勿手改**：`core/internal/app/dao/*.gen.go`（gorm gen）、`frontend/src/router/elegant/*`（elegant-router 自动生成）、`frontend/bindings/`（wails 代码生成，已在 frontend .gitignore）、`bundled/resource/farm/`（sync 脚本生成）、`build/windows/info.json` 与 `build/darwin/Info.plist` 的版本字段（构建脚本按 VERSION 改写）。
4. **子模块变更流程**：在 `core/` 或 `frontend/` 里改 → 子模块内 commit + push → 回仓库根 `git add core frontend` 记录新指针 → 提交推送。克隆新机器用 `git clone --recurse-submodules`。
5. **提交信息**：conventional commits（`feat:`/`fix:`/`chore:`/`docs:`/`build:`），中文描述。frontend 有 pre-commit（`typecheck && lint && fmt && git diff --exit-code`）与 commit-msg 校验（`sa git-commit-verify`）；仅文档/配置类小改可 `git -c core.hooksPath=/dev/null commit` 临时跳过，代码改动必须过 hook。
6. **DTO 命名**：前后端契约字段一律 camelCase（`json` tag）；例外：`core/internal/farm/logic` 的 `AutomationConfig/AccountConfig` 用 snake_case 对齐 bot/rust 协议。
7. **YAML 陷阱**：改 `build/**/*.yml` 时，纯标量值里含 `: `（如 `(default: per-user)`）必须加引号，否则新版 task 解析失败。
8. 仓库内路径引用一律用 `./core`、`./frontend` 相对路径（go.mod replace、脚本、CI 都基于此布局）；不要引入 `../qq-farm-*` 这类仓库外引用。

## 按任务查参考（先读对应文件再动手）

| 任务 | 读 |
|------|-----|
| 加/改后端接口、加表、写 service、配置、日志、农场自动化 | `references/core.md` |
| 加/改页面、加 API 调用、路由/菜单、WS、桌面模式分支 | `references/frontend.md` |
| 桌面壳代码、构建打包、子模块指针、发版、更新器 | `references/desktop.md` |

core 内另有一份更细的后端规范（`.cursor/skills/go-skeleton-dev/SKILL.md`），与本参考一致，可交叉查阅。
