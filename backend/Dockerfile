# Multi-stage build for Go backend

# Stage 1: Build
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Stage 2: Runtime
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates and wget for HTTPS and health checks
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -S -g 1001 goapp && \
    adduser -S -u 1001 -G goapp goapp

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/data ./data

# Change ownership to non-root user
RUN chown -R goapp:goapp /app

# Switch to non-root user
USER goapp

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/health || exit 1

# Run
CMD ["./server"]
