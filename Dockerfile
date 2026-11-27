# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/informer ./cmd/informer

FROM alpine:3.22
WORKDIR /app
# apk upgrade: pick up security fixes published since the base image was built.
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/informer /app/informer

# Non-root user (repo convention: never run containers as root).
RUN addgroup -g 1001 -S appuser 2>/dev/null; adduser -u 1001 -S -G root appuser 2>/dev/null \
    || adduser -D -u 1001 appuser
RUN chown -R appuser /app
USER appuser

ENTRYPOINT ["/app/informer"]
