# Multi-stage Dockerfile for iTrigger Go server

# Stage 1: Build the Go application
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy all application files (including vendor directory)
COPY . .

# Build static binary using vendor mode (fast & zero network dependency)
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -ldflags="-w -s" -o server ./cmd/server

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
