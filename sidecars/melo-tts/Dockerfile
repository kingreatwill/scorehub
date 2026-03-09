FROM python:3.9-slim AS builder

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    git \
    && rm -rf /var/lib/apt/lists/*

RUN python -m venv /opt/venv
ENV PATH=/opt/venv/bin:$PATH

COPY sidecars/melo-tts/requirements.txt /tmp/requirements.txt
RUN python -m pip install --upgrade pip setuptools wheel && \
    python -m pip install --no-compile -r /tmp/requirements.txt && \
    python -m unidic download && \
    find /opt/venv -type d -name '__pycache__' -prune -exec rm -rf {} + && \
    find /opt/venv -type f \( -name '*.pyc' -o -name '*.pyo' \) -delete

FROM python:3.9-slim

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_NO_CACHE_DIR=1 \
    PATH=/opt/venv/bin:$PATH \
    PORT=18090 \
    MELO_DEVICE=cpu \
    MELO_DEFAULT_LANGUAGE=ZH \
    MELO_PRELOAD_LANGUAGES=ZH \
    HF_HOME=/models/huggingface

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    espeak-ng \
    libgomp1 \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/* && \
    useradd --create-home --uid 10001 app

COPY --from=builder /opt/venv /opt/venv
COPY sidecars/melo-tts/app.py /app/app.py

RUN mkdir -p /models/huggingface && chown -R app:app /app /models/huggingface

USER app

EXPOSE 18090

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "18090"]

# docker login -u kingreatwill

# docker buildx build -f sidecars/melo-tts/Dockerfile -t kingreatwill/scorehub-melo-tts:v0.0.1 --push .

# docker run -d \
#   --name scorehub-melo-tts \
#   -p 18090:18090 \
#   -e MELO_DEVICE=cpu \
#   -e MELO_DEFAULT_LANGUAGE=ZH \
#   -e MELO_PRELOAD_LANGUAGES=ZH \
#   -e 'MELO_VOICES_JSON=[{"id":"warm","label":"温柔","language":"ZH","speaker":"ZH","speed":0.96,"isDefault":true},{"id":"bright","label":"清亮","language":"ZH","speaker":"ZH","speed":1.06},{"id":"steady","label":"沉稳","language":"ZH","speaker":"ZH","speed":1.0}]' \
#   -v /data/dockerv/scorehub/melo_tts_hf:/models/huggingface \
#   --restart unless-stopped \
#   kingreatwill/scorehub-melo-tts:v0.0.2

# curl -sS \
#   -X POST http://127.0.0.1:18090/v1/audio/speech \
#   -H 'Content-Type: application/json' \
#   -o /tmp/melo-test.wav \
#   -d '{"model":"melo-zh","voice":"warm","input":"收到8分","response_format":"wav"}'