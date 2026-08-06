# Use the official Go image with version 1.25.10
FROM golang:1.25.10-alpine AS builder

RUN apk update && \
    apk add --no-cache \
    curl \
    "zlib>=1.3.2-r0"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -a -ldflags="-w -s" -o main ./cmd && \
    go clean -cache -modcache

RUN adduser -D -h /home/appuser appuser && \
    mkdir -p logs resources && \
    chown -R appuser:appuser logs/ resources/ && \
    chmod +r resources/* && \
    chmod +w logs && \
    chmod +x main

USER appuser

CMD ["./main"]
