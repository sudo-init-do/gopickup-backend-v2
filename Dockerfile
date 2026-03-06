# Build Stage
FROM golang:1.23.3-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/migrate ./cmd/migrate

# Runtime Stage
FROM alpine:3.19

WORKDIR /app

# Install necessary runtime dependencies (ca-certificates for HTTPS, tzdata for timezones)
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user
RUN adduser -D -g '' appuser

# Copy the binaries from the builder stage
COPY --from=builder /app/server .
COPY --from=builder /app/migrate .
COPY scripts/entrypoint.sh .
RUN chmod +x entrypoint.sh

# Use the non-root user
USER appuser

# Expose the application port
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./entrypoint.sh"]
