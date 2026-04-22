# Stage 1: Build the Vite/React frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /alpaca

# Copy package files first for better layer caching
COPY frontend-vite/package*.json ./
RUN npm ci --frozen-lockfile

COPY frontend-vite ./
RUN npm run build

# Stage 2: Build the Go binary
FROM golang:1.26-alpine AS builder

WORKDIR /alpaca

# Copy Go dependency files first (caching)
COPY vertex/api/go.mod vertex/api/go.sum ./vertex/api/
COPY go.mod go.sum ./
RUN cd vertex/api && go mod download

# Copy Go source code
COPY vertex/api ./vertex/api
COPY vertex/config.go vertex/llm_completion.go vertex/llm_router.go vertex/openTelemetry.go vertex/searchservice.go vertex/vector_search.go ./vertex/

# Build the statically linked binary
RUN cd vertex/api && CGO_ENABLED=0 GOOS=linux go build -o /search .

# Final runtime stage
FROM alpine:latest

# Install CA certificates for TLS
RUN apk add --no-cache ca-certificates

WORKDIR /alpaca

# Copy the compiled Go binary from Stage 2
COPY --from=builder /search /search

# Copy the built frontend (correct source path from Stage 1)
COPY --from=frontend-builder /alpaca/dist ./frontend-vite/dist

# Copy config to the correct location expected by the alpacalication
COPY config.yaml ./config.yaml

EXPOSE 8080

CMD ["/search"]