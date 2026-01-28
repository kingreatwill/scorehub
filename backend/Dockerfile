FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/scorehub ./cmd/api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=builder /out/scorehub /app/scorehub

EXPOSE 8080
ENTRYPOINT ["/app/scorehub"]

# docker build -t scorehub .
# docker tag scorehub kingreatwill/scorehub:v1.0
# docker login -u kingreatwill
# docker push kingreatwill/scorehub:v1.0
# docker run -d --name scorehub --link postgresql   -p 10005:8080   -v /data/dockerv/scorehub/.env:/app/.env:ro   -v /etc/localtime:/etc/localtime:ro   --restart unless-stopped kingreatwill/scorehub:v1.0
