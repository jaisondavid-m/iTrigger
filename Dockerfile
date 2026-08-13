# Multi-stage Dockerfile for iTrigger Go server

# Stage 1: Build the Go application
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Enable GOPROXY with fallback
ENV GOPROXY=https://proxy.golang.org,direct

# Copy all source code (including vendor directory if created)
COPY . .

# Download dependencies if vendor directory is not present
RUN if [ ! -d "vendor" ]; then go mod download; fi

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
