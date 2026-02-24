# Multi-stage Dockerfile for Tecla

# Stage 1: Build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache make git

# Copy source and build
COPY . .
RUN go mod download
RUN make build

# Stage 2: Final Image
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache git curl ca-certificates

# Copy binary from builder
COPY --from=builder /app/tecla /usr/local/bin/tecla

# Create config directory
RUN mkdir -p /root/.config/tecla

# Run tecla by default
ENTRYPOINT ["tecla"]
CMD ["--help"]
