# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gameday-sim .

# Run stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/gameday-sim .

# Copy default config
COPY --from=builder /app/config.yaml .

# Run the binary
ENTRYPOINT ["./gameday-sim"]

# Allow config file to be overridden
CMD ["-config", "config.yaml"]
