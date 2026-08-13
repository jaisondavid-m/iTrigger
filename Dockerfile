# Multi-stage Dockerfile for iTrigger Go server

# Stage 1: Build the Go application
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Set reliable proxy fallback & disable checksum DB verification inside container to avoid TLS timeouts
ENV GOPROXY=https://goproxy.io,direct
ENV GOSUMDB=off

# Copy Go module definitions for caching
COPY go.mod go.sum ./

# Download dependencies inside builder container
RUN go mod download

# Copy source code
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
