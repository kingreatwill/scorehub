FROM python:3.9-slim

ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PORT=18090 \
    MELO_DEVICE=cpu \
    MELO_DEFAULT_LANGUAGE=ZH \
    MELO_PRELOAD_LANGUAGES=ZH \
    HF_HOME=/models/huggingface

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    espeak-ng \
    git \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*

COPY sidecars/melo-tts/requirements.txt /tmp/requirements.txt
RUN python -m pip install --upgrade pip && \
    python -m pip install -r /tmp/requirements.txt && \
    python -m unidic download

COPY sidecars/melo-tts/app.py /app/app.py

RUN mkdir -p /models/huggingface

EXPOSE 18090

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "18090"]


# docker login -u kingreatwill

# docker buildx build -f sidecars/melo-tts/Dockerfile -t kingreatwill/scorehub-melo-tts:v0.0.1 --push .
