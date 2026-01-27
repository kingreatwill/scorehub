# 得分簿 (ScoreHub)

微信小程序「得分簿」：用于多人实时记分。

## 技术栈

- 前端：uni-app (Vue3) → 编译到微信小程序
- 后端：Golang + Hertz
- 数据库：PostgreSQL

## 目录结构

- `backend/` 后端服务（Hertz）
- `backend/sql/migrations/` PostgreSQL 初始化 SQL
- `frontend/miniapp/` uni-app (Vue3) 小程序前端

## 本地启动（开发）

1) 准备数据库（示例：Docker）

```bash
docker run --name scorehub-pg -e POSTGRES_PASSWORD=scorehub -e POSTGRES_DB=scorehub -p 5432:5432 -d postgres:16
```

或使用 `docker compose`（会自动初始化表结构）：

```bash
docker compose up -d postgres
```

2) 执行初始化 SQL

```bash
psql "postgres://postgres:scorehub@localhost:5432/scorehub?sslmode=disable" -f backend/sql/migrations/0001_init.sql
```

3) 启动后端

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

后端会自动加载 `.env`（支持在仓库根目录或 `backend/` 目录启动）；也可通过环境变量 `SCOREHUB_ENV_FILE` 指定自定义路径。
如需将定位经纬度自动反查为位置名称（例如「上海·徐汇」），请在 `.env` 中配置 `SCOREHUB_TENCENT_MAP_KEY`（腾讯位置服务 key）。

4) 启动前端

前端基于 uni-app（Vue3），推荐开发模式（会输出到 `dist/dev/mp-weixin`）：

```bash
cd frontend/miniapp
npm install
npm run dev:mp-weixin
```

然后在微信开发者工具中导入：

- 推荐：导入 `frontend/miniapp/`（已提供 `project.config.json`，指向 `dist/dev/mp-weixin/`）
- 或直接导入 `frontend/miniapp/dist/dev/mp-weixin/`

发布/打包可用：

```bash
cd frontend/miniapp
npm run build:mp-weixin
```

## 文档

- `docs/api.md` 后端接口说明
- 微信小程序开发文档: https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/userProfile.html
- 微信小程序开发平台: https://mp.weixin.qq.com/wxamp/devprofile/get_profile?token=44927146&lang=zh_CN
