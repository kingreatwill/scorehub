FROM python:3.11-slim AS builder

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

RUN python -m venv /opt/venv
ENV PATH=/opt/venv/bin:$PATH

COPY sidecars/piper/requirements.txt /tmp/requirements.txt
RUN python -m pip install --upgrade pip setuptools wheel && \
    python -m pip install --no-compile -r /tmp/requirements.txt && \
    python -m pip install --no-compile piper-tts pathvalidate g2pw requests && \
    find /opt/venv -type d -name '__pycache__' -prune -exec rm -rf {} + && \
    find /opt/venv -type f \( -name '*.pyc' -o -name '*.pyo' \) -delete

FROM python:3.11-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    PATH=/opt/venv/bin:$PATH \
    PORT=18091 \
    PIPER_MODELS_DIR=/models \
    PIPER_MODEL_PATH=model.onnx \
    PIPER_DEFAULT_VOICE_ID=default \
    PIPER_DEFAULT_VOICE_LABEL=默认 \
    PIPER_DEFAULT_LANGUAGE=zh-CN \
    PIPER_DEFAULT_MODEL=piper-zh-cn

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    libstdc++6 \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/* && \
    useradd --create-home --uid 10001 app

COPY --from=builder /opt/venv /opt/venv
COPY sidecars/piper/app.py /app/app.py

RUN mkdir -p /models && chown -R app:app /app /models

USER app

EXPOSE 18091

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "18091"]

# docker login -u kingreatwill

# docker buildx build -f sidecars/piper/Dockerfile -t kingreatwill/scorehub-piper-tts:v0.0.1 --push .

# chown -R 10001:10001 /data/dockerv/scorehub/piper-tts/models
# chmod -R 755 /data/dockerv/scorehub/piper-tts/models

# docker run -d --name scorehub-tts -p 18091:18091 -e PIPER_AUTO_DOWNLOAD=true -e PIPER_PRELOAD_MODELS=true -e PIPER_MODELS_DIR=/models -e 'PIPER_VOICES_JSON=[{"id":"chaowen","label":"超文","language":"zh-CN","model":"piper-zh-cn-chaowen","modelPath":"zh_CN-chaowen-medium.onnx","description":"中文中性声线","isDefault":true},{"id":"huayan","label":"华妍","language":"zh-CN","model":"piper-zh-cn-huayan","modelPath":"zh_CN-huayan-medium.onnx","description":"中文女声"},{"id":"xiao_ya","label":"小雅","language":"zh-CN","model":"piper-zh-cn-xiao-ya","modelPath":"zh_CN-xiao_ya-medium.onnx","description":"中文女声（仅限非商用）"}]' -v "/data/dockerv/scorehub/piper-tts/models:/models" --restart unless-stopped kingreatwill/scorehub-piper-tts:v0.0.2
