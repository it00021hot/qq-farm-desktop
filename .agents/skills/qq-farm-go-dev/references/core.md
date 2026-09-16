# core 后端开发参考（qq-farm-core）

模块 `github.com/it00021hot/qq-farm-core`，Go 1.26。**技术栈：Fiber v3 + GORM + Viper + log/slog + Turso（SQLite 兼容）**。这不是 GoFrame 项目（曾误引入的 `gogf/gf` 依赖已移除），不要使用 GoFrame 写法（无 g.Meta、无 g.Cfg、无 g.DB、无 gerror）。

## 目录结构

```
core/
├── cmd/
│   ├── app/main.go          # HTTP 服务入口（urfave/cli：-e dev|test|prod，-p 端口，默认 9528）
│   └── cli/main.go          # 代码生成 CLI（genModel 等）
├── configs/                 # config.{dev,test,prod}.yaml，init.go 用 embed.FS 内嵌
├── internal/
│   ├── app/
│   │   ├── controller/      # HTTP handler：auth/ system/ backend/ frontend/ farm/<域>/
│   │   ├── service/         # 业务逻辑（与 controller 同构分域）
│   │   ├── dao/             # gorm.io/gen 生成的类型安全 DAO（*.gen.go 勿手改）
│   │   ├── entity/          # gen 辅助
│   │   ├── model/           # 手写 GORM 表模型 cn_*.go（表名前缀 cn_）
│   │   └── pkg/             # validator、内存版 redis 等
│   ├── bootstrap/           # init.go BootService() + boots/{config,logger,turso,migrate,dao,farm,paths}.go
│   ├── command/             # 代码生成命令 + tpls/ 模板
│   ├── farm/                # 农场核心（与 HTTP 层解耦）：protocol/ proto/ game/ runtime/ hub/ logic/ tsdk/ …
│   ├── middleware/          # auth.go(JWT) cache.go logger.go whitelist.go
│   ├── router/              # register.go（Fiber 组装）+ routes/{common,auth,system,farm,frontend}.go
│   ├── types/               # 请求/响应 DTO：admin/ farm/ user/auth/
│   └── vars/variable.go     # 全局单例：vars.Config / vars.DB / vars.MDB / vars.Logger / DesktopMode
├── pkg/
│   ├── appserver/           # 桌面端进程内启动 API（Start/Shutdown/mountWebUI）
│   ├── config/              # viper 封装（带 smap 缓存）
│   ├── database/            # gorm.Open + driver/turso 方言 + migrate/ AutoMigrate
│   ├── logger/              # slog JSON + lumberjack 轮转
│   ├── response/            # 统一响应 {code,requestId,msg,data} + 分页
│   └── jwtauth/ cache/ crontab/ encrypt/ restyHttp/ wecom/ …
├── resource/farm/           # tsdk.wasm + gameConfig JSON（打包时同步进桌面 bundle）
└── Makefile
```

## 分层与一次请求的生命周期

`routes/*.go → controller（仅绑定/校验）→ service（业务）→ vars.DB（原生 gorm）或 dao → model`。
DTO 全部集中在 `internal/types/<域>/`，controller/service 用包级单例（`var Auth = &AuthController{}`）。

以 `POST /auth/login` 为例：

1. 路由 `internal/router/routes/common.go`：`router.Post("/auth/login", auth.Auth.Login).Name("登录")`（public 组，无 auth 中间件）
2. DTO `internal/types/user/auth/auth.go`：`LoginReq` 带 `form:"userName" json:"userName" validate:"required"`（go-playground 校验 tag）
3. Controller `internal/app/controller/auth/auth.go`：`c.Validate(ctx, &req)` → `auth2.Auth.Login(req)` → `response.SuccessJSON` / `response.BadRequestException`；swag 注释生成文档
4. Service `internal/app/service/auth/auth.go`：`vars.DB.Where("account = ?", ...).First(&admin)` → 密码校验 → `pkg/jwtauth` 签发 token，refresh jti 存内存缓存
5. Model `internal/app/model/cn_sys_admin.go`：`TableName() = "cn_sys_admin"`

**加一个接口的完整流程**（顺序固定）：

1. **Model**：`internal/app/model/cn_<name>.go` 新建表模型，并加入 `pkg/database/migrate` 的 `Models()` 列表（每次启动 AutoMigrate）
2. **Types**：`internal/types/<域>/` 定义 Req/Resp，`json` tag 用 camelCase，校验用 `validate:"..."` tag
3. **Service**：`internal/app/service/<域>/`，内嵌 `service.Service` 基类，包级单例导出
4. **Controller**：`internal/app/controller/<域>/`，内嵌 `controller.Controller`（有 `Validate()`），handler 只做绑定校验 + 调 service + 组响应；补 swag 注释
5. **Route**：`internal/router/routes/<组>.go` 加一行，`.Name("中文名")`；新组要在 `internal/router/register.go` 注册
6. **文档**：`make docs`（swag 生成到 ./docs）

路由约定：只用 GET/POST；命名 `list|add|modify|delete` 前缀；中间件分 public（Logger+WhiteIp）与 protected（另加 JWT+Cache）。

## 响应与错误处理

- 成功：`response.SuccessJSON(ctx, data)`；失败：`response.BadRequestException(ctx, code, msg)` ——**HTTP 状态码仍是 200**，业务码放 body 的 `code`（soybean 前端按 `code===0` 判成功）。
- 错误用标准库：`fmt.Errorf("...: %w", err)` 包装，`errors.Is(err, gorm.ErrRecordNotFound)` 判空。**不用 gerror**。
- service 层返回 error，由 controller 统一翻译成响应；不要在 service 里直接写 HTTP 响应。

## 日志

`log/slog` JSON + lumberjack 轮转（`pkg/logger`），启动时初始化为 `vars.Logger`。用法：`slog.Info/Warn/Error` 或 `vars.Logger`。`server.mode=production` 时只写文件（runtime/logs），否则 stdout+文件双写。HTTP 请求日志由 `middleware.LoggerMiddleware`（slog-fiber）负责。**不要用 g.Log/glog**。

## 配置

- 文件 `configs/config.{dev,test,prod}.yaml`，经 embed 内嵌；env 由启动参数 `-e` 决定（非环境变量），默认 prod（桌面 appserver 场景）。
- 启动时 `boots.InitConfig()` 读入全局 `vars.Config`（viper 封装）：`vars.Config.GetString/GetInt/GetBool/GetDuration`。
- 主要配置块：`server`（mode/prefork/whiteList）、`jwt`（secret/expire）、`log`、`database.turso`（path/autoMigrate/maxOpenConn）、`farm`（gatewayUrl wss、wasmPath、gameConfigDir、clientVersion、推送渠道）、`wecom`。
- 桌面路径环境变量只有两个：`QQFARM_RESOURCE_ROOT`、`QQFARM_DATA_ROOT`（见 `pkg/appserver`）。

## 数据库

- 驱动：GORM + 自研 Turso 方言（`pkg/database/driver/turso`），SQLite 兼容，WAL 模式；库文件 `runtime/data/qq-farm.db`，表名 `cn_` 前缀、`SingularTable: true`。
- 迁移：**无 SQL 文件**，`boots.InitMigrate()` 每次启动 `AutoMigrate` 全部模型 + 播种管理员 `admin/admin888`。改表结构 = 改 model + 重启。
- DAO：`gorm.io/gen` 生成 `internal/app/dao/*.gen.go`（`go run ./cmd/cli genModel`，模板在 `internal/command/tpls/`）。生成物勿手改；service 首选直接用 `vars.DB`，dao 是可选的类型安全路径。

## 农场自动化域（internal/farm）

与 HTTP 层解耦，是本仓库的核心业务：

- `protocol/`：游戏 WSS 网关客户端（gorilla/websocket + protobuf），Login 由 Session 发起、Heartbeat 每 25s；加解密走 `tsdk/`（wazero 跑 WASM TSDK）。协议语义与 qq-farm-bot / qq-farm-rust 逐字节对齐——改协议必须对照这两者的注释与测试。
- `proto/`：*.proto + 生成代码；改了跑 `make proto`（scripts/gen-farm-proto.sh）。
- `game/`：游戏 RPC 封装（lands/friend/mall/dog/daily…）。
- `runtime/`：`AccountManager`（StartAccount/StopAccount/ApplyConfig/StopAll）+ 每账号 `Session`（世代号防串、巡田、好友互动、重连）。
- `hub/`：进程内 pub/sub；浏览器侧 `GET /farm/ws?token=<jwt>` 订阅（WS 不能带 Authorization 头，token 走 query——见 `middleware/auth.go` 提取链）。
- 前端契约字段例外：`logic.AutomationConfig/AccountConfig` 用 snake_case 对齐 bot/rust，其余 DTO 一律 camelCase。

## 桌面集成（pkg/appserver）

桌面壳通过 `appserver.Start(Options{Env, Host, Port, ResourceRoot, DataRoot, DesktopMode, WebFS})` 进程内启动本服务：跑 `bootstrap.BootService()` → `router.Register` → 可选挂载 Vue SPA（WebFS）→ 监听 `127.0.0.1:9528`。`Shutdown` 会 `farmruntime.Default.StopAll()` 停掉所有农场会话。改启动顺序/初始化逻辑要同时验证两种入口：`make run` 与桌面 `wails3 task dev`。

## Makefile 与工具链

| 命令 | 作用 |
|------|------|
| `make run` | 增量构建并以 `-e=dev -p=9528` 运行（macOS 会 ad-hoc 签名） |
| `make build` / `make windows\|linux\|darwin` | 构建/交叉编译 app+cli |
| `make lint` | gofumpt 格式化（提交前必跑；无 .golangci） |
| `make proto` | 重新生成 farm protobuf |
| `make docs` | swag 生成 Swagger 文档 |
| `make clean` / `make help` | 清理 / 帮助 |

- 热重载：`air`（`.air.toml`；Windows 输出名必须带 .exe）。
- 测试：直接 `go test ./...`（52 个测试文件集中在 `internal/farm/`，stdlib testing + 表驱动，中文注释说明对齐依据）。改 protocol/runtime 必须跑全量测试。
- 注释风格：中文为主，导出符号一行 `//` 注释；涉及协议对齐的改动在注释里写明参照（bot/rust 哪个文件）。
