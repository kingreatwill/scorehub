# 得分簿 (ScoreHub)

微信小程序「得分簿」：用于多人实时记分。
得分簿在「记录中」状态下若连续 7 天没有新的记分记录，会自动结束。

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

如需通过 Docker 构建后端镜像并内置 H5 前端，直接在仓库根目录构建 `backend/Dockerfile`（会在构建阶段执行 `npm run build:h5-backend`）：

```bash
docker build -f backend/Dockerfile -t scorehub .
```

2) 执行初始化 SQL（按顺序执行所有 migration）

```bash
for f in backend/sql/migrations/*.sql; do
  psql "postgres://postgres:scorehub@localhost:5432/scorehub?sslmode=disable" -f "$f"
done
```

如果你之前已经启动过 `docker compose`（本地 `pgdata` 卷已存在），新增的 migration 不会自动应用。
此时你有两种方式：

- 方式 A（推荐开发环境）：删除卷重建数据库

```bash
docker compose down -v
docker compose up -d postgres
```

- 方式 B：手动执行新增 migration（例如本次新增的 `0004_auth.sql`）

```bash
psql "postgres://postgres:scorehub@localhost:5432/scorehub?sslmode=disable" -f backend/sql/migrations/0004_auth.sql
```

3) 启动后端

```bash
cd backend
cp .env.example .env
go run ./cmd/api
```

后端会自动加载 `.env`（支持在仓库根目录或 `backend/` 目录启动）；也可通过环境变量 `SCOREHUB_ENV_FILE` 指定自定义路径。
如需将定位经纬度自动反查为位置名称（例如「上海·徐汇」），请在 `.env` 中配置 `SCOREHUB_TENCENT_MAP_KEY`（腾讯位置服务 key）、`SCOREHUB_AMAP_KEY`（高德开放平台 key）或 `SCOREHUB_BAIDU_MAP_AK`（百度地图开放平台 AK）。若同时配置，后端会根据各家 QPS 限制选择可用服务（每次请求只调用一家，避免依次调用导致额度浪费）。

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

如需构建 H5 并交给后端统一托管（支持前端路由刷新）：

```bash
cd frontend/miniapp
npm run build:h5-backend
```

该命令会把 H5 产物同步到 `backend/assets/h5/`，后端启动后可直接访问：

- `http://localhost:8080/`
- 前端路由刷新（如 `http://localhost:8080/pages/home/index`）会回退到 H5 `index.html`

## 文档

- `docs/api.md` 后端接口说明
- 微信小程序开发文档: https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/userProfile.html
- 微信小程序开发平台: https://mp.weixin.qq.com/wxamp/devprofile/get_profile?token=44927146&lang=zh_CN
- 腾讯地图(并发5次/秒, 6000次/日): https://lbs.qq.com/service/webService/webServiceGuide/address/Gcoder
- 高德地图(并发3次/秒, 150000次/月): https://lbs.amap.com/api/webservice/summary
- 百度地图(并发3次/秒, 300次/日): https://baidumap.apifox.cn/api-32790722


## 说明
MP-WEIXIN 不是项目里某个文件“手动定义”的变量，而是 uni-app 的条件编译平台标识：当构建/运行目标是微信小程序时，编译器自动认为 MP-WEIXIN 为真。
它由构建命令的 -p mp-weixin 决定

## 其他
### 框架
### UI组件
https://github.com/dcloudio/uni-app

#### tdesign
https://github.com/Tencent/tdesign-miniprogram
https://github.com/Tencent/tdesign-miniprogram/tree/develop/packages/tdesign-uniapp
https://tdesign.tencent.com/uniapp/getting-started

#### weui
原生视觉体验
https://github.com/Tencent/weui-wxss
