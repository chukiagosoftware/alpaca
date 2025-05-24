FROM golang:1.23-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o topology-app main.go

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/topology-app .
COPY .env .
COPY static/ ./static/
COPY templates/ ./templates/
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
EXPOSE 8080
CMD ["./topology-app"]