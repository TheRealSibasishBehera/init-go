# Multi-stage build for Go init system
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o init ./cmd/init

# Production image
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /app/init .
# Copy configuration file
COPY fly_run.json /fly/run.json
# Create test app for demonstration
RUN printf '#!/bin/sh\necho "App started with PID $$"\nsleep 300\n' > /usr/bin/myapp && chmod +x /usr/bin/myapp
CMD ["./init"]