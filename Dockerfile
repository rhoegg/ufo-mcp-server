# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with version info
RUN make build

# Final stage - minimal image
FROM scratch

# Copy SSL certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/build/ufo-mcp /ufo-mcp

# Create data directory
VOLUME ["/data"]

# Expose HTTP port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/ufo-mcp"]

# Default to HTTP transport
CMD ["--transport", "http", "--port", "8080", "--effects-file", "/data/effects.json"]