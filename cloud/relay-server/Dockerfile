# Multi-stage build for ultra-lightweight Go Relay Server
FROM golang:1.24-alpine AS builder

WORKDIR /build
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o relay-server ./cmd/relay-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/relay-server /app/relay-server

EXPOSE 9000

ENV PORT=9000
ENV RELAY_SECRET=""

ENTRYPOINT ["/app/relay-server"]
