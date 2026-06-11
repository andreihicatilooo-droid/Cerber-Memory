# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO (required by go-sqlite3)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o /cerber ./cmd/cerber

# Stage 2: Minimal runtime image
FROM alpine:latest

# Install sqlite and ca-certificates
RUN apk add --no-cache sqlite ca-certificates

WORKDIR /app

# Copy the compiled binary from builder stage
COPY --from=builder /cerber /app/cerber

# Create data directory for SQLite database and uploads
RUN mkdir -p /app/data

# Expose default port
EXPOSE 8080

# Run the server flag by default
ENTRYPOINT ["/app/cerber", "--server"]
