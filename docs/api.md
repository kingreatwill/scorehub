# API (v1)

Base URL: `http://localhost:8080/api/v1`

## Auth

### POST /auth/dev_login

仅开发环境（`SCOREHUB_DEV_AUTH=true`）。

Request:

```json
{"openid":"dev-user-1","nickname":"张三","avatarUrl":"https://..."}
```

Response:

```json
{"token":"sh1....","user":{"id":1,"openid":"dev-user-1","nickname":"张三","avatarUrl":"https://..."}} 
```

### POST /auth/wechat_login

微信小程序登录：前端通过 `uni.login()` 获取 `code` 交给后端换取 `openid`，返回 token。
头像/昵称建议通过 `uni.getUserProfile()` 获取后再调用 `PATCH /me` 保存（便于用户后续修改昵称）。

Request:

```json
{"code":"<wx_code>"}
```

## Me

所有接口默认需要 `Authorization: Bearer <token>`。

### GET /me

获取当前用户信息。

### PATCH /me

更新当前用户头像/昵称（前端可先通过 `uni.getUserProfile()` 获取后再提交）。

```json
{"nickname":"张三","avatarUrl":"https://..."}
```

## Location

所有接口默认需要 `Authorization: Bearer <token>`。

### GET /location/reverse_geocode?lat=..&lng=..

根据经纬度反查地址（`locationText`）。需要配置 `SCOREHUB_TENCENT_MAP_KEY`（优先使用腾讯地图 `geocoder/v1` 的 `formatted_addresses.recommend`），否则会回退为 `lat,lng`。

## Scorebooks

所有接口默认需要 `Authorization: Bearer <token>`。

### POST /scorebooks

创建新的得分簿；`name` 为空时默认使用「当前时间 + 位置」生成。

```json
{"name":"","locationText":"上海·徐汇"}
```

### GET /scorebooks

我的得分簿列表。

### GET /scorebooks/:id

得分簿详情（包含成员列表 + 每人累计得分）。

### PATCH /scorebooks/:id

修改名称（仅掌柜/创建者）。

```json
{"name":"周末牌局"}
```

### POST /scorebooks/:id/end

结束（仅掌柜/创建者）。

Response（会返回冠亚季军：按分数降序取前 3 名，且分数必须 > 0；可能为空）：

```json
{"scorebook":{...},"winners":{"champion":{"memberId":"...","nickname":"...","avatarUrl":"...","score":10},"runnerUp":null,"third":null}}
```

### POST /scorebooks/:id/join

加入成员（仅进行中的得分簿可加入；已结束不可加入）。

```json
{"nickname":"李四","avatarUrl":"https://..."}
```

### GET /scorebooks/:id/invite_qrcode

获取该得分簿的小程序码（PNG）。仅成员可获取，且得分簿必须是进行中。

### PATCH /scorebooks/:id/members/me

修改自己在该得分簿内的头像/昵称。

```json
{"nickname":"我自己","avatarUrl":"https://..."}
```

## Records

### POST /scorebooks/:id/records

对某个成员记分（`toMemberId` 为对方 memberId，`delta` 为本次增加的分数，必须 > 0）。

```json
{"toMemberId":"<uuid>","delta":10,"note":"炸胡"}
```

### GET /scorebooks/:id/records?limit=50&offset=0

记录列表（倒序）。

## Invites

### GET /invites/:code

通过邀请码获取得分簿信息。

### POST /invites/:code/join

通过邀请码加入得分簿。

## WebSocket

`ws://localhost:8080/ws/scorebooks/:id?token=<token>`

服务端会广播：

- `record.created`
- `member.joined`
- `member.updated`
- `scorebook.updated`
- `scorebook.ended`
