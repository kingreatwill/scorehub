# ScoreHub 项目理解记录

## 概览
ScoreHub 是一个基于微信小程序的多人实时记分与轻量记账/存款/生日管理应用。前端使用 uni-app (Vue3)，后端使用 Go + Hertz，数据库为 PostgreSQL。

## 技术栈
- 前端：uni-app (Vue3) → 构建为微信小程序，UI 组件包含 tdesign。
- 后端：Golang + Hertz，WebSocket 用于实时推送。
- 数据库：PostgreSQL，SQL 迁移在 `backend/sql/migrations/`。

## 目录结构
- 后端代码：`backend/`
- 前端小程序：`frontend/miniapp/`
- 文档：`docs/api.md`

## 后端概览
入口与路由：
- 入口：`backend/cmd/api/main.go`
- 路由：`/api/v1/*` + `/ws/scorebooks/:id` + `/static/*`
- 静态资源读取：`backend/cmd/api/main.go` 的 `staticAssetsHandler`

核心模块：
- 认证：`backend/internal/auth/`、`backend/internal/http/handlers/auth.go`
- 配置：`backend/internal/config/config.go`  
  主要环境变量：
  - `SCOREHUB_ADDR`
  - `SCOREHUB_DB_DSN`
  - `SCOREHUB_TOKEN_SECRET`
  - `SCOREHUB_DEV_AUTH`
  - `SCOREHUB_WECHAT_APPID` / `SCOREHUB_WECHAT_SECRET`
  - `SCOREHUB_TENCENT_MAP_KEY` / `SCOREHUB_AMAP_KEY` / `SCOREHUB_BAIDU_MAP_AK`
- 业务处理：`backend/internal/http/handlers/`  
  包含 `scorebook`、`ledger`、`birthday`、`deposit`、`location`、`me` 等。
- 数据访问：`backend/internal/store/`  
  统一处理 DB 操作，使用 pgx。
- 实时推送：`backend/internal/realtime/hub.go`  
  基于 WebSocket 维护房间广播。
- 自动结束：`backend/cmd/api/auto_end.go`  
  7 天无记录自动结束得分簿。

## 数据模型（迁移）
基础表：
- `users`

得分簿：
- `scorebooks` (book_type: `scorebook` / `ledger`)
- `scorebook_members`
- `score_records`

生日：
- `birthday_contacts`

存款：
- `deposit_accounts`
- `deposit_records`

迁移文件：
- `backend/sql/migrations/0001_init.sql`
- `backend/sql/migrations/0002_birthday.sql`
- `backend/sql/migrations/0003_deposit.sql`

## 主要功能模块
### 得分簿（Scorebook）
- 创建/加入/修改/结束、成员管理、记分记录。
- 记录通过 WebSocket 广播：`record.created`、`member.joined`、`member.updated`、`scorebook.updated`、`scorebook.ended`。
- 7 天无记录自动结束。

### 记账簿（Ledger）
- 复用 `scorebooks` 表，`book_type = ledger`。
- 记录存储在 `score_records`，`delta` 正负表示收入/支出。
- 通过 `GET /ledgers/:id` 返回成员与记录。

### 生日薄（Birthday）
- `birthday_contacts` 表，支持公历/农历。
- 前端用 `frontend/miniapp/src/utils/lunar-calendar.mjs` 做农历到公历转换并回写。

### 存款（Deposit）
- 账户：`deposit_accounts`
- 记录：`deposit_records`  
  支持状态、标签、附件、统计与筛选。

## 前端概览
入口与配置：
- 入口：`frontend/miniapp/src/main.ts`
- App 级样式与主题变量：`frontend/miniapp/src/App.vue`
- 页面配置：`frontend/miniapp/src/pages.json`

页面与模块：
- 得分簿：`frontend/miniapp/src/pages/scorebook/*`
- 记账：`frontend/miniapp/src/pages/ledger/*`
- 生日：`frontend/miniapp/src/pages/birthday/*`
- 存款：`frontend/miniapp/src/pages/deposit/*`
- 个人页：`frontend/miniapp/src/pages/my/*`

API 封装：
- `frontend/miniapp/src/utils/api.ts`

银行 logo：
- 元数据：`frontend/miniapp/src/utils/banks.ts`
- 资源路径：`/static/img/pay/*.svg`

## 分页与加载（已统一）
- 列表与记录默认分页大小为 20。
- 列表滚动到底部自动加载下一页。
- 文案统一为：
  - “滑动加载下一页”
  - “已全部加载完毕”
- 相关页面：
  - `frontend/miniapp/src/pages/scorebook/list.vue`
  - `frontend/miniapp/src/pages/scorebook/detail.vue`
  - `frontend/miniapp/src/pages/ledger/list.vue`
  - `frontend/miniapp/src/pages/ledger/detail.vue`
  - `frontend/miniapp/src/pages/birthday/list.vue`
  - `frontend/miniapp/src/pages/deposit/list.vue`

## 开发与运行
后端：
1. 启动 PostgreSQL
2. 执行 SQL 迁移
3. `go run ./cmd/api`

前端：
1. `npm install`
2. `npm run dev:mp-weixin`
3. 用微信开发者工具导入 `frontend/miniapp/`

## 备注
- `MP-WEIXIN` 为 uni-app 条件编译标识，构建目标为微信小程序时自动为真。
