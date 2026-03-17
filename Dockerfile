# Build stage
FROM golang:1.25-alpine AS builder

# Set working directory
WORKDIR /alpaca

# Copy only necessary Vertex search files and config (no go.mod yet)
COPY vertex/search/ ./vertex/search/
COPY vertex/openTelemetry.go ./vertex/
COPY vertex/searchservice.go ./vertex/
COPY config.yaml ./

# Initialize a new minimal go.mod based on copied sources (replace module name as needed)
RUN go mod init github.com/chukiagosoftware/alpaca && go mod tidy

# Build the search application only
RUN go build -o search vertex/search/main.go vertex/search/http_handlers.go

FROM alpine:latest
# Install CA certificates for TLS verification
RUN apk add --no-cache ca-certificates

# Copy the built binary and necessary files
COPY --from=builder /alpaca/search /search
COPY --from=builder /alpaca/vertex/search/index.html /vertex/search/
COPY --from=builder /alpaca/vertex/search/static/ /vertex/search/static/
COPY --from=builder /alpaca/config.yaml /

# Expose port
EXPOSE 8080

# Run the application
CMD ["/search"]