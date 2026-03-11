# Piper Sidecar

这是一个给 ScoreHub 后端使用的本地 `Piper` sidecar。

它暴露和现有 TTS provider 一致的两个接口：

- `GET /v1/audio/voices`
- `POST /v1/audio/speech`

这样后端无需改代码，只要切换环境变量就能接入 `Piper`。

## 功能

- 基于 `FastAPI` 暴露 `/health`、`/audio/voices`、`/audio/speech`
- 用 `piper` CLI 调用本地 ONNX 模型进行语音合成
- 通过 `PIPER_VOICES_JSON` 定义对外可选声音列表
- 支持按 voice id 或请求里的 `model` 字段路由到指定 Piper 声音
- 运行时检查模型文件是否存在，不存在则自动下载
- 可选在启动时预拉取当前配置里的全部模型
- 输出 `wav`

## 默认示例声音

我已经把示例默认配置成 3 个中文声音：

- `chaowen`
- `huayan`
- `xiao_ya`

对应的示例编排在 `sidecars/piper/docker-compose.example.yml:1`。

## 自动下载说明

默认开启自动下载，示例里同时开启启动预拉取：

- 环境变量：`PIPER_AUTO_DOWNLOAD=true`
- 环境变量：`PIPER_PRELOAD_MODELS=true`
- 启动预拉取：容器启动时会检查并下载 `PIPER_VOICES_JSON` 里配置的全部模型
- 按需下载：如果没开启启动预拉取，首次请求某个 voice 合成时，如果 `/models` 下缺少对应的 `.onnx` 或 `.onnx.json`，sidecar 会自动下载
- 下载目标：`PIPER_MODELS_DIR`，默认 `/models`
- 注意：如果你希望自动下载生效，模型目录不能挂成只读

健康检查不会主动下载模型；它只会在返回里标记当前缺少哪些模型。

## 目录约定

Piper 通常需要一对模型文件：

- `xxx.onnx`
- `xxx.onnx.json`

默认建议把它们放到容器内的 `/models` 目录。

## 官方下载地址

以下地址基于官方仓库 `rhasspy/piper-voices`：

- 仓库主页：`https://huggingface.co/rhasspy/piper-voices`
- 中文目录：`https://huggingface.co/rhasspy/piper-voices/tree/main/zh/zh_CN`

### `chaowen`

- 页面目录：`https://huggingface.co/rhasspy/piper-voices/tree/main/zh/zh_CN/chaowen/medium`
- 下载 `onnx`：`https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/chaowen/medium/zh_CN-chaowen-medium.onnx`
- 下载 `json`：`https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/chaowen/medium/zh_CN-chaowen-medium.onnx.json`
- 模型卡：`https://huggingface.co/rhasspy/piper-voices/blob/main/zh/zh_CN/chaowen/medium/MODEL_CARD`

### `huayan`

- 页面目录：`https://huggingface.co/rhasspy/piper-voices/tree/main/zh/zh_CN/huayan/medium`
- 下载 `onnx`：`https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/huayan/medium/zh_CN-huayan-medium.onnx`
- 下载 `json`：`https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/huayan/medium/zh_CN-huayan-medium.onnx.json`
- 模型卡：`https://huggingface.co/rhasspy/piper-voices/blob/main/zh/zh_CN/huayan/medium/MODEL_CARD`

### `xiao_ya`

- 页面目录：`https://huggingface.co/rhasspy/piper-voices/tree/main/zh/zh_CN/xiao_ya/medium`
- 下载 `onnx`：`https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/xiao_ya/medium/zh_CN-xiao_ya-medium.onnx`
- 下载 `json`：`https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/xiao_ya/medium/zh_CN-xiao_ya-medium.onnx.json`
- 模型卡：`https://huggingface.co/rhasspy/piper-voices/blob/main/zh/zh_CN/xiao_ya/medium/MODEL_CARD`

## 构建

在仓库根目录执行：

```bash
docker build -f sidecars/piper/Dockerfile -t scorehub-piper-tts .
```

## 运行

### 推荐：自动下载模式

首次运行时，即使 `./models` 是空目录，也可以先启动；第一次合成时会自动下载缺失模型。

```bash
docker run -d \
  --name scorehub-piper-tts \
  -p 18091:18091 \
  -e PIPER_AUTO_DOWNLOAD=true \
  -e PIPER_PRELOAD_MODELS=true \
  -e PIPER_MODELS_DIR=/models \
  -e 'PIPER_VOICES_JSON=[
    {"id":"chaowen","label":"超文","language":"zh-CN","model":"piper-zh-cn-chaowen","modelPath":"zh_CN-chaowen-medium.onnx","description":"中文中性声线","isDefault":true},
    {"id":"huayan","label":"华妍","language":"zh-CN","model":"piper-zh-cn-huayan","modelPath":"zh_CN-huayan-medium.onnx","description":"中文女声"},
    {"id":"xiao_ya","label":"小雅","language":"zh-CN","model":"piper-zh-cn-xiao-ya","modelPath":"zh_CN-xiao_ya-medium.onnx","description":"中文女声（仅限非商用）"}
  ]' \
  -v /your/piper-models:/models \
  --restart unless-stopped \
  scorehub-piper-tts
```

### 预下载模式

如果你不想运行时联网，也可以先把文件下载好再挂载进去。

```bash
mkdir -p sidecars/piper/models

curl -L -o sidecars/piper/models/zh_CN-chaowen-medium.onnx \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/chaowen/medium/zh_CN-chaowen-medium.onnx
curl -L -o sidecars/piper/models/zh_CN-chaowen-medium.onnx.json \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/chaowen/medium/zh_CN-chaowen-medium.onnx.json

curl -L -o sidecars/piper/models/zh_CN-huayan-medium.onnx \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/huayan/medium/zh_CN-huayan-medium.onnx
curl -L -o sidecars/piper/models/zh_CN-huayan-medium.onnx.json \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/huayan/medium/zh_CN-huayan-medium.onnx.json

curl -L -o sidecars/piper/models/zh_CN-xiao_ya-medium.onnx \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/xiao_ya/medium/zh_CN-xiao_ya-medium.onnx
curl -L -o sidecars/piper/models/zh_CN-xiao_ya-medium.onnx.json \
  https://huggingface.co/rhasspy/piper-voices/resolve/main/zh/zh_CN/xiao_ya/medium/zh_CN-xiao_ya-medium.onnx.json
```

## 健康检查

```bash
curl -sS http://127.0.0.1:18091/health
```

示例返回会包含：

- `autoDownload`
- `preloadModels`
- `missingModels`
- `ready`

## Voice 列表接口

```bash
curl -sS http://127.0.0.1:18091/v1/audio/voices
```

## 测试步骤

建议按下面顺序验证：

### 1. 看健康状态

```bash
curl -sS http://127.0.0.1:18091/health
```

确认这几个字段：

- `ready`
- `autoDownload`
- `preloadModels`
- `missingModels`

### 2. 看 voice 列表

```bash
curl -sS http://127.0.0.1:18091/v1/audio/voices
```

先确认返回里确实有你要用的 voice，例如：

- `chaowen`
- `huayan`
- `xiao_ya`

### 3. 直接测试合成

```bash
curl -sS \
  -X POST http://127.0.0.1:18091/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -o /tmp/piper-test.wav \
  -d '{"model":"piper-zh-cn-chaowen","voice":"chaowen","input":"收到8分","response_format":"wav"}'
```

检查文件：

```bash
ls -lh /tmp/piper-test.wav
file /tmp/piper-test.wav
```

macOS 可以直接试听：

```bash
afplay /tmp/piper-test.wav
```

### 4. 报错时先看返回 body

不要一上来就 `-o wav`，先直接看错误体：

```bash
curl -i \
  -X POST http://127.0.0.1:18091/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -d '{"model":"piper-zh-cn-chaowen","voice":"chaowen","input":"收到8分","response_format":"wav"}'
```

如果要更清楚一点，可以这样：

```bash
curl -sS \
  -X POST http://127.0.0.1:18091/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -d '{"model":"piper-zh-cn-chaowen","voice":"chaowen","input":"收到8分","response_format":"wav"}' | jq .
```

## 合成接口

请求：

```json
{
  "model": "piper-zh-cn-chaowen",
  "voice": "chaowen",
  "input": "收到8分",
  "response_format": "wav"
}
```

返回：

- `200 OK`
- `Content-Type: audio/wav`
- body 为 wav 音频字节

## 常见报错

### `400 Bad Request`

当前 sidecar 里会返回 400 的常见原因只有 3 个：

- `{"detail":"invalid voice"}`
- `{"detail":"input is required"}`
- `{"detail":"piper sidecar currently supports wav only"}`

你这次返回头里有：

- `HTTP/1.1 400 Bad Request`
- `content-length: 26`

这和 `{"detail":"invalid voice"}` 的长度是对上的，所以这次**大概率就是 voice 不存在**。

也就是说，你请求里的 `voice` 很可能不是当前容器实际配置的 voice。

先执行：

```bash
curl -sS http://127.0.0.1:18091/v1/audio/voices
```

然后只用返回里的 `id` 去测，例如：

```bash
curl -i \
  -X POST http://127.0.0.1:18091/v1/audio/speech \
  -H 'Content-Type: application/json' \
  -d '{"model":"piper-zh-cn-chaowen","voice":"chaowen","input":"收到8分","response_format":"wav"}'
```

特别注意：

- `xiao_ya` 是下划线，不是 `xiaoya`
- `xiao_ya` 也不是 `xiao-ya`
- `voice` 要和 `/v1/audio/voices` 返回的 `id` 完全一致
- `response_format` 目前只能是 `wav`
- `input` 不能为空

### `502 Bad Gateway`

常见表示：

- 自动下载失败
- 模型文件下载不完整
- `piper` 命令执行失败
- 镜像里缺少 `piper` 运行时依赖

这时优先看：

```bash
docker logs <piper-container-name>
```

如果日志里出现类似：

```text
ModuleNotFoundError: No module named 'pathvalidate'
```

或：

```text
ModuleNotFoundError: No module named 'g2pw'
```

说明当前本地镜像还是旧版本，或镜像里缺少中文 voice 运行时依赖，需要重新构建镜像并重建容器：

```bash
docker compose -f sidecars/piper/docker-compose.example.yml down
docker compose -f sidecars/piper/docker-compose.example.yml up -d --build --force-recreate
```

如果你不是用 `compose`，则执行：

```bash
docker rm -f scorehub-piper-tts
docker build -f sidecars/piper/Dockerfile -t scorehub-piper-tts .
docker run -d --name scorehub-piper-tts -p 18091:18091 -e PIPER_AUTO_DOWNLOAD=true -e PIPER_PRELOAD_MODELS=true -e PIPER_MODELS_DIR=/models -e 'PIPER_VOICES_JSON=[{"id":"chaowen","label":"超文","language":"zh-CN","model":"piper-zh-cn-chaowen","modelPath":"zh_CN-chaowen-medium.onnx","description":"中文中性声线","isDefault":true},{"id":"huayan","label":"华妍","language":"zh-CN","model":"piper-zh-cn-huayan","modelPath":"zh_CN-huayan-medium.onnx","description":"中文女声"},{"id":"xiao_ya","label":"小雅","language":"zh-CN","model":"piper-zh-cn-xiao-ya","modelPath":"zh_CN-xiao_ya-medium.onnx","description":"中文女声（仅限非商用）"}]' -v "$(pwd)/sidecars/piper/models:/models" --restart unless-stopped scorehub-piper-tts
```

另外，日志里那条 `onnxruntime` 的 PCI warning 通常不是这次失败的主因；真正导致失败的是后面的 Python 异常。


`xiao_ya`、`huayan`、`chaowen` 这类中文 voice 在当前 Python 运行方式下会走中文音素流程；如果缺少 `g2pw`，就会在合成阶段返回 `502`。

### `500 Internal Server Error`

常见表示：

- 自动下载关闭了，但模型文件不存在
- `/models` 没写权限
- 容器卷挂载方式不对

## ScoreHub 后端配置

如果 `api` 和 `piper-tts` 在同一个 Docker Compose 网络里：

```env
SCOREHUB_TTS_API_BASE=http://piper-tts:18091/v1
SCOREHUB_TTS_API_SPEECH_PATH=/audio/speech
SCOREHUB_TTS_API_VOICES_PATH=/audio/voices
SCOREHUB_TTS_API_KEY=local
SCOREHUB_TTS_MODEL=piper-zh-cn-chaowen
SCOREHUB_TTS_AUDIO_FORMAT=wav
SCOREHUB_TTS_DEFAULT_VOICE=chaowen
```

如果你希望后端固定展示自定义 voice 列表，也可以直接配置：

```env
SCOREHUB_TTS_VOICES_JSON=[{"id":"chaowen","label":"超文","providerVoice":"chaowen","description":"中文中性声线","isDefault":true},{"id":"huayan","label":"华妍","providerVoice":"huayan","description":"中文女声"},{"id":"xiao_ya","label":"小雅","providerVoice":"xiao_ya","description":"中文女声（仅限非商用）"}]
```

如果你是在宿主机直接跑后端：

```env
SCOREHUB_TTS_API_BASE=http://127.0.0.1:18091/v1
```

## Sidecar 环境变量

- `PORT`
  - 服务端口，默认 `18091`
- `PIPER_BIN`
  - `piper` 可执行文件路径，默认 `piper`
- `PIPER_MODELS_DIR`
  - 模型目录，默认 `/models`
- `PIPER_AUTO_DOWNLOAD`
  - 缺文件时是否自动下载，默认 `true`
- `PIPER_PRELOAD_MODELS`
  - 启动时是否预拉取当前配置里的全部模型，默认 `false`
- `PIPER_DOWNLOAD_TIMEOUT_SECS`
  - 单个模型文件下载超时秒数，默认 `120`
- `PIPER_MODEL_PATH`
  - 单 voice 模式使用的模型路径，默认 `model.onnx`
- `PIPER_MODEL_DOWNLOAD_URL`
  - 单 voice 模式显式指定模型下载地址；不填时会按官方路径规则推断
- `PIPER_MODEL_CONFIG_URL`
  - 单 voice 模式显式指定配置下载地址；不填时会按官方路径规则推断
- `PIPER_DEFAULT_VOICE_ID`
  - 单 voice 模式对外 voice id，默认 `default`
- `PIPER_DEFAULT_VOICE_LABEL`
  - 单 voice 模式展示名称，默认 `默认`
- `PIPER_DEFAULT_LANGUAGE`
  - 默认语言，默认 `zh-CN`
- `PIPER_DEFAULT_MODEL`
  - 默认请求模型标识，默认 `piper-zh-cn`
- `PIPER_DEFAULT_SPEAKER`
  - 多 speaker 模型默认 speaker id
- `PIPER_LENGTH_SCALE`
  - 语速缩放，默认 `1.0`
- `PIPER_NOISE_SCALE`
  - 噪声参数，默认 `0.667`
- `PIPER_NOISE_W`
  - 音素宽度噪声参数，默认 `0.8`
- `PIPER_SAMPLE_RATE`
  - 用于说明当前 voice 采样率，默认 `22050`
- `PIPER_TIMEOUT_SECS`
  - 单次合成超时秒数，默认 `30`
- `PIPER_MAX_TEXT_RUNES`
  - 单次最大字符数，默认 `120`
- `PIPER_VOICES_JSON`
  - voice 列表 JSON 数组；每项支持：
  - `id`
  - `label`
  - `language`
  - `model` 或 `modelAlias`
  - `modelPath`
  - `downloadURL`
  - `configURL`
  - `speaker`
  - `lengthScale`
  - `noiseScale`
  - `noiseW`
  - `sampleRate`
  - `description`
  - `isDefault`

## 注意事项

- `xiao_ya` 的官方模型卡注明：需要 `Piper` Python 版 `1.4+`
- `xiao_ya` 的官方模型卡注明：数据集许可为非商用，商用前请先确认授权
- 如果你把 `/models` 挂成只读卷，自动下载会失败
- 自动下载依赖容器在运行时能访问 Hugging Face

## 部署方式

推荐直接用本目录下的示例：

```bash
docker compose -f sidecars/piper/docker-compose.example.yml up -d --build
```
