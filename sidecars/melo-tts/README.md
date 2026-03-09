# MeloTTS Sidecar

这是一个给 ScoreHub 后端使用的本地 `MeloTTS` sidecar。  
它对外暴露两组 OpenAI 兼容风格接口：

- `GET /v1/audio/voices`
- `POST /v1/audio/speech`

这样现有后端可以先拉 voice 列表，再按 voice id 合成语音。

## 目录内容

- `app.py`
  - FastAPI 服务，提供 `/health`、`/audio/voices`、`/audio/speech`
- `Dockerfile`
  - 单独构建本地 MeloTTS sidecar
- `requirements.txt`
  - Python 依赖
- `docker-compose.example.yml`
  - 独立部署示例

## 当前实现

- 默认走中文 `ZH`
- 默认输出 `wav`
- `voice` 不再写死在 ScoreHub 后端环境变量里
- sidecar 通过 `MELO_VOICES_JSON` 定义对外可选声音列表

注意：

- MeloTTS 的基础中文模型本质上通常还是单 speaker。
- 这里的多个声音通常是通过不同 `speed / speaker / language` 组合做映射。
- 如果你后面接自定义 speaker 或别的中文模型，只需要改 `MELO_VOICES_JSON`。

## 构建

当前 Dockerfile 已经改成多阶段构建：

- builder 阶段安装 `build-essential / git`
- 运行阶段只保留 Python 运行环境和必要系统库

这样能明显减小最终镜像，相比单阶段更省。

在仓库根目录执行：

```bash
docker build -f sidecars/melo-tts/Dockerfile -t scorehub-melo-tts .
```

## 运行

```bash
docker run -d \
  --name scorehub-melo-tts \
  -p 18090:18090 \
  -e MELO_DEVICE=cpu \
  -e MELO_DEFAULT_LANGUAGE=ZH \
  -e MELO_PRELOAD_LANGUAGES=ZH \
  -e 'MELO_VOICES_JSON=[{"id":"warm","label":"温柔","language":"ZH","speaker":"ZH","speed":0.96,"isDefault":true},{"id":"bright","label":"清亮","language":"ZH","speaker":"ZH","speed":1.06},{"id":"steady","label":"沉稳","language":"ZH","speaker":"ZH","speed":1.0}]' \
  -v scorehub_melo_hf:/models/huggingface \
  --restart unless-stopped \
  scorehub-melo-tts
```

首次启动或首次请求时，MeloTTS 可能会下载模型文件；建议保留 `/models/huggingface` 卷做缓存。
如果你没挂这个卷，模型会落在容器可写层里，容器体积会继续涨；这通常比镜像本身更大。

可以分别看这两个体积：

```bash
docker image ls scorehub-melo-tts
docker ps -s --filter name=scorehub-melo-tts
```

## 健康检查

```bash
curl -sS http://127.0.0.1:18090/health
```

## Voice 列表接口

```bash
curl -sS http://127.0.0.1:18090/v1/audio/voices
```

示例返回：

```json
{
  "defaultVoice": "warm",
  "voices": [
    {
      "id": "warm",
      "label": "温柔",
      "language": "ZH",
      "description": "默认柔和语速",
      "isDefault": true
    },
    {
      "id": "bright",
      "label": "清亮",
      "language": "ZH",
      "description": "更快一点的语速",
      "isDefault": false
    }
  ]
}
```

## 合成接口

请求：

```json
{
  "model": "melo-zh",
  "voice": "warm",
  "input": "收到8分",
  "response_format": "wav"
}
```

返回：

- `200 OK`
- `Content-Type: audio/wav`
- body 为 wav 音频字节

## ScoreHub 后端配置

如果 `api` 和 `melo-tts` 在同一个 Docker Compose 网络里：

```env
SCOREHUB_TTS_API_BASE=http://melo-tts:18090/v1
SCOREHUB_TTS_API_SPEECH_PATH=/audio/speech
SCOREHUB_TTS_API_VOICES_PATH=/audio/voices
SCOREHUB_TTS_API_KEY=local
SCOREHUB_TTS_MODEL=melo-zh
SCOREHUB_TTS_AUDIO_FORMAT=wav
SCOREHUB_TTS_DEFAULT_VOICE=warm
```

如果你希望后端自己定义暴露给前端的声音列表，也可以直接配置：

```env
SCOREHUB_TTS_VOICES_JSON=[{"id":"warm","label":"温柔","providerVoice":"warm","description":"默认柔和语速","isDefault":true},{"id":"bright","label":"清亮","providerVoice":"bright","description":"更快一点的语速"},{"id":"steady","label":"沉稳","providerVoice":"steady","description":"标准语速"}]
```

如果你是在宿主机直接跑后端：

```env
SCOREHUB_TTS_API_BASE=http://127.0.0.1:18090/v1
```

## Sidecar 环境变量

- `PORT`
  - 服务端口，默认 `18090`
- `MELO_DEVICE`
  - `cpu` 或 `cuda`，默认 `cpu`
- `MELO_DEFAULT_LANGUAGE`
  - 默认语言，默认 `ZH`
- `MELO_PRELOAD_LANGUAGES`
  - 启动时预加载语言，逗号分隔；不填时按 `MELO_VOICES_JSON` 中出现的语言预加载
- `MELO_MAX_TEXT_RUNES`
  - 单次最大字符数，默认 `80`
- `MELO_SPEAKER_ZH`
  - 中文默认 speaker，默认 `ZH`
- `MELO_VOICES_JSON`
  - voice 列表 JSON 数组；每项支持：
  - `id`
  - `label`
  - `language`
  - `speaker`
  - `speed`
  - `description`
  - `isDefault`

## 部署方式

推荐直接用本目录下的示例：

```bash
docker compose -f sidecars/melo-tts/docker-compose.example.yml up -d --build
```
