# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o worker-alpaca ./worker-alpaca

# Runtime stage
FROM alpine:latest

# Install runtime dependencies for SQLite
RUN apk add --no-cache ca-certificates sqlite

# Create app directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/worker-alpaca .

# Copy .env file if it exists (optional - can also use environment variables)
# COPY .env .env

# Create directory for database
RUN mkdir -p /app/data

# Set environment variable for database path
ENV SQLITE_DB_PATH=/app/data/alpaca.db

# Run the application
CMD ["./worker-alpaca"]
