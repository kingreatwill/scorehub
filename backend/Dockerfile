FROM node:20-alpine AS h5_builder

WORKDIR /workspace

COPY frontend/miniapp/package.json frontend/miniapp/package-lock.json ./frontend/miniapp/
RUN npm -C frontend/miniapp ci --no-fund --no-audit

COPY frontend/miniapp ./frontend/miniapp
COPY backend ./backend
COPY favicon.ico ./favicon.ico
RUN npm -C frontend/miniapp run build:h5-backend

FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend ./
COPY --from=h5_builder /workspace/backend/assets/h5 ./assets/h5
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/scorehub ./cmd/api

FROM alpine:3
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=builder /out/scorehub /app/scorehub

EXPOSE 8080
ENTRYPOINT ["/app/scorehub"]

# run from repository root:
# docker login -u kingreatwill

# 1
# docker buildx build -f backend/Dockerfile -t kingreatwill/scorehub:v2.3.8 --push .

# 2
# docker build -f backend/Dockerfile -t scorehub .
# docker tag scorehub kingreatwill/scorehub:v2.3.8
# docker push kingreatwill/scorehub:v2.3.8

# 3
# docker build -f backend/Dockerfile -t kingreatwill/scorehub:v2.3.8 . && docker push kingreatwill/scorehub:v2.3.8

# docker run -d --name scorehub --link postgresql   -p 10005:8080   -v /data/dockerv/scorehub/.env:/app/.env:ro   -v /etc/localtime:/etc/localtime:ro   --restart unless-stopped kingreatwill/scorehub:v2.3.8
