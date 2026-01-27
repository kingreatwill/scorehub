FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/scorehub ./cmd/api

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
WORKDIR /app
COPY --from=builder /out/scorehub /app/scorehub

EXPOSE 8080
ENTRYPOINT ["/app/scorehub"]
