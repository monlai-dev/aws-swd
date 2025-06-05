# ---------- Builder stage ----------
FROM golang:1.24 AS builder

WORKDIR /app

# Only copy go.mod and go.sum first
COPY go.mod go.sum ./

# Download dependencies (this will be cached unless deps change)
RUN go mod download

# Now copy the rest of the app
COPY . .

# Build binary
RUN go build -o main .

# ---------- Runtime stage ----------
FROM alpine:latest

WORKDIR /app

# Copy the built binary from builder
COPY --from=builder /app/main .

# Set entrypoint
CMD ["./main"]
