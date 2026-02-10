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

