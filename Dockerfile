FROM golang:1.23-alpine AS builder
WORKDIR /app

# Install git (needed for go modules, if not already cached)
RUN apk add --no-cache git

# Copy go.mod and go.sum to download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code.
COPY . .

# Build the Go binary with CGO disabled for a static binary.
RUN CGO_ENABLED=0 go build -o telegram_app .

# Final stage
FROM alpine:latest
WORKDIR /app

# Copy the binary and .env file (if needed) from the builder stage.
COPY --from=builder /app/telegram_app .
COPY .env .

EXPOSE 10000

# Run the application.
CMD ["./telegram_app"]
