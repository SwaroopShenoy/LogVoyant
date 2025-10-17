# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy source
COPY . .

# Generate go.sum and download dependencies
RUN go mod download && go mod tidy

# Build binary (statically linked)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o logvoyant \
    .

# Stage 2: Runtime (ultra-small)
FROM busybox:1.36-musl

# Copy binary
COPY --from=builder /build/logvoyant /usr/local/bin/logvoyant

# Create directories
RUN mkdir -p /data /logs

WORKDIR /data

# Expose port
EXPOSE 3100

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3100/ || exit 1

# Run as root to read host logs (read-only mounts are safe)
# Default command
ENTRYPOINT ["/usr/local/bin/logvoyant"]
CMD ["start", "--port", "3100"]