# Multi-stage Dockerfile for iTrigger Go server

# Stage 1: Build the Go application
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Configure resilient Go proxies with fallback to ensure fast, reliable downloads during docker compose
ENV GOPROXY=https://goproxy.io,https://proxy.golang.org,direct
ENV GOSUMDB=off

# Copy module definition files first for Docker layer caching
COPY go.mod go.sum ./

# Download dependencies automatically during build
RUN go mod download

# Copy rest of application source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o server ./cmd/server

# Stage 2: Final lightweight image
FROM alpine:latest

# Install ca-certificates, mailcap (for MIME types), and tzdata
RUN apk --no-cache add ca-certificates mailcap tzdata

WORKDIR /app

# Copy built binary from builder stage
COPY --from=builder /app/server .

# Expose port 8080
EXPOSE 8080

# Command to run the executable
CMD ["./server"]
