# Build stage
FROM golang:1.25.10-alpine AS builder

RUN apk add --no-cache git ca-certificates "zlib>=1.3.2-r0"

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /out/main ./cmd

# Runtime stage — keep image small so ACR push is fast
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /home/appuser appuser && \
    mkdir -p /app/logs /app/resources && \
    chown -R appuser:appuser /app

WORKDIR /app

COPY --from=builder /out/main /app/main
COPY --from=builder /src/resources/ /app/resources/

USER appuser

CMD ["/app/main"]
