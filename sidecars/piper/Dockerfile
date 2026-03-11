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
    python -m pip install --no-compile piper-tts && \
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

# docker buildx build -f sidecars/piper/Dockerfile -t kingreatwill/scorehub-piper-tts:v0.0.1 --push .

# docker run -d \
#   --name scorehub-piper-tts \
#   -p 18091:18091 \
#   -e PIPER_MODEL_PATH=/models/zh_CN-huayan-medium.onnx \
#   -e PIPER_DEFAULT_VOICE_ID=huayan \
#   -e PIPER_DEFAULT_VOICE_LABEL=华妍 \
#   -e PIPER_DEFAULT_LANGUAGE=zh-CN \
#   -e PIPER_DEFAULT_MODEL=piper-zh-cn \
#   -v /data/dockerv/scorehub/piper:/models:ro \
#   --restart unless-stopped \
#   kingreatwill/scorehub-piper-tts:v0.0.1
