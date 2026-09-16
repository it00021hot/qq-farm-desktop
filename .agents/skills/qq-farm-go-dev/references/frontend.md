# frontend 前端开发参考（qq-farm-web 子模块）

SoybeanAdmin 2.2.1 改造的管理端：Vue 3.5 + Vite 8 + TypeScript(strict) + Naive UI + UnoCSS + pinia + vue-i18n，pnpm workspace monorepo。业务页面在 `src/views/farm/`。

## 目录结构

```
frontend/
├── build/                  # vite 配置辅助（非 vite.config 本体）
│   ├── config/proxy.ts     # dev 代理（→ 127.0.0.1:9528，ws: true）
│   └── plugins/            # elegant-router、unocss、unplugin（NaiveUI 自动导入）等
├── packages/               # pnpm workspace，@sa/* 内部包
│   ├── axios/              # createFlatRequest 请求封装（code===0 判成功）
│   ├── hooks/ scripts/ uno-preset/ materials/ color/ utils/
├── src/
│   ├── service/
│   │   ├── api/            # auth.ts farm.ts system-manage.ts（fetchXxx 函数）
│   │   └── request/        # request 实例 + 拦截器（Authorization、token 过期码）
│   ├── views/farm/         # 全部业务页面（dashboard/account/friends/activity/analytics/
│   │                       #   game-mall/mystery-shop/game-config/personal/settings）
│   ├── router/elegant/     # elegant-router 自动生成（勿手改）
│   ├── store/modules/      # pinia setup stores（app/auth/farm-account/route/tab/theme）
│   ├── hooks/business/     # farm-ws.ts（实时 WS）、auth.ts
│   ├── hooks/common/       # table.ts(useNaivePaginatedTable) form.ts use-managed-interval.ts
│   ├── typings/api/        # namespace Api.Farm / Api.Auth 等全局类型
│   ├── locales/langs/      # zh-cn.ts en-us.ts（route.* 与 page.* 文案）
│   └── utils/desktop.ts    # isDesktop / 桌面壳判断（VITE_IS_DESKTOP 唯一读取点）
├── bindings/               # wails 生成（已 gitignore）
└── .env / .env.test / .env.prod / .env.desktop
```

## 加一个页面的完整流程

以 `src/views/farm/<new-page>/` 为例：

1. **视图**：`index.vue` 用 `<script setup lang="ts">`，`defineOptions({ name: 'Farm<NewPage>' })`——组件名必须等于路由名（keep-alive 依赖）。私有子组件放 `modules/`（kebab-case 文件名，相对导入）；面板式页面用同级 PascalCase（参考 `personal/BagPanel.vue`）；跨页复用放 `src/views/farm/shared/`。
2. **路由**：elegant-router 插件按 views 目录自动生成（目录名连字符化：`game-config` → 路由 `farm_game-config`），`src/router/elegant/*` 勿手改。也可 `pnpm gen-route` 交互式生成。
3. **文案**：在 `src/locales/langs/zh-cn.ts` 和 `en-us.ts` 的 `route` 对象里各加标题（meta 的 `i18nKey` 自动生成为 `route.farm_<new-page>`）；页面内文案用 `$t`（`page.*`）。
4. **类型**：`src/typings/api/` 下 `namespace Api.Farm` 加 `XxxSearchParams` / `XxxList` 等。
5. **API**：`src/service/api/farm.ts`（index.ts 已 re-export）：
   ```ts
   export function fetchGetFarmAccountList(params?: Api.Farm.AccountSearchParams) {
     return request<Api.Farm.AccountList>({ url: '/farm/account/list', method: 'get', params });
   }
   ```
   命名一律 `fetchXxx`；URL 写后端相对路径，baseURL/代理由 request 层与 env 处理。传参前先过 clean 函数剔除空值（参考 `cleanAccountListParams`）。
6. **页面装配**：列表页用 `useNaivePaginatedTable` + `defaultTransform` + `useTableOperate`（`src/hooks/common/table.ts`）；实时刷新用 `useFarmWs`；跨页"当前管理账号"用 `useFarmAccountStore`。

## 请求与后端契约

- 实例：`src/service/request/index.ts`，`createFlatRequest`（来自 `@sa/axios`）。成功判定 `code === VITE_SERVICE_SUCCESS_CODE`（0）；401 类过期码、登出码走 env 配置。
- dev 模式（`.env.test`，`VITE_HTTP_PROXY=Y`）baseURL 是 `/proxy-default`，由 vite 代理到 `http://127.0.0.1:9528`（`build/config/proxy.ts`，支持 WS 升级）。
- 后端响应是 HTTP 200 + `{code, requestId, msg, data}`——前端不需要处理非 200 业务错误；新增后端接口时保持这个约定。
- Authorization 头由 `onRequest` 注入（`getToken()`，localStorage `SOY_` 前缀键）。

## 路由 / 菜单 / 权限

- `VITE_AUTH_ROUTE_MODE=static`（各 env 均为 static），菜单由静态路由生成。
- 常量路由列表、meta 默认值见 `build/plugins/router.ts` 的 `onRouteMetaGen`；自定义手工路由在 `src/router/routes/builtin.ts`。
- 路由 history：web 用 history，桌面用 hash（env 驱动，勿在代码里写死）。

## 桌面模式（Wails）

- 分支判断**只**用 `src/utils/desktop.ts`：`isDesktop`（构建期 env `VITE_IS_DESKTOP === 'Y'`）与 `isDesktopShell`（运行时探测 `/wails/runtime.js`）。不要在别处直接读 `import.meta.env.VITE_IS_DESKTOP`。
- `.env.desktop`：`VITE_SERVICE_BASE_URL=http://127.0.0.1:9528`、hash 路由、关代理、关更新检测。桌面 WebView 与 API 非同源，所以 WS 必须显式指向 `VITE_SERVICE_BASE_URL`。
- WS：`src/hooks/business/farm-ws.ts` 的 `useFarmWs`，连 `/farm/ws?token=<getToken()>`（token 走 query）。内置指数退避重连（2s 起，30s 封顶）；消息包络 `{ type, payload, accountId }`，用 `onMessage(type, payload, raw)` 订阅。已在 account/dashboard/friends 页使用。

## 状态管理

- pinia setup store：`defineStore(SetupStoreId.X, () => {...})`，id 从 `src/enum` 取。
- token：`localStg.set('token'|'refreshToken'|'lastLoginUserId')`；登录态 `isLogin = Boolean(token)`。
- `farm-account` store 提供全局"当前管理账号"（`currentAccountId`/`accountOptions`），农场各页共享。

## 规范与检查

- TypeScript strict；全局类型命名空间 `Api.*`、`App.*`（`src/typings/`）。
- 文件/目录 kebab-case；组件 PascalCase 且 NaiveUI 模板强制大写（eslint 规则，自动导入无需 import）。
- UnoCSS：presetWind3(class 暗黑) + `@sa/uno-preset`，快捷方式 `card-wrapper`；工具类直接写在模板。
- i18n：所有用户可见文案过 `$t`，中英两份 langs 必须同步加 key。
- 脚本：

| 脚本 | 作用 |
|------|------|
| `pnpm dev` / `pnpm dev:prod` | dev server（:9529），test/prod 模式 + 代理 |
| `pnpm build:desktop` | 桌面构建（hash、:9528、产物 dist/ 被桌面内嵌） |
| `pnpm typecheck` | vue-tsc 全量类型检查 |
| `pnpm lint` / `pnpm fmt` | oxlint+eslint --fix / oxfmt |
| `pnpm commit` | conventional commit 交互式生成 |

- **pre-commit**：`pnpm typecheck && pnpm lint && pnpm fmt && git diff --exit-code`——提交前本地先跑这三个，避免 hook 自动修复后 diff 非空导致提交失败。commit-msg 由 `sa git-commit-verify` 校验 conventional 格式。
