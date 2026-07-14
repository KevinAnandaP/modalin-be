# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary statically (CGO_ENABLED=0) and strip debug symbols (-ldflags="-s -w")
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/main cmd/api/main.go

# Stage 2: Run the binary in a minimal image
FROM alpine:latest

# Install basic packages (ca-certificates for HTTPS, tzdata for timezone management)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config folder (excluding files listed in .dockerignore)
COPY configs/ ./configs/

# Expose port
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
