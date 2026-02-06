# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the API binary with sqlite_fts5 tag for full-text search support
RUN CGO_ENABLED=1 GOOS=linux go build -tags sqlite_fts5 -a -installsuffix cgo -o wacli-api cmd/wacli-api/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/wacli-api .

# Copy the web directory for frontend
COPY --from=builder /app/web ./web

# Create data directory
RUN mkdir -p /root/.wacli

# Expose port
EXPOSE 8080

# Run the API server
CMD ["./wacli-api"]
