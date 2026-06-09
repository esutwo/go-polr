# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Cache module downloads in their own layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary (no CGO; sqlite is only used in tests)
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/go-polr ./cmd/server

# Final stage
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata curl \
    && addgroup -S -g 10001 app \
    && adduser  -S -u 10001 -G app app

WORKDIR /app

COPY --from=builder --chown=app:app /out/go-polr ./go-polr
COPY --from=builder --chown=app:app /app/web      ./web

USER app:app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/go-polr"]
