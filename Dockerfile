# Stage 1: Build
FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

# Stage 2: Runtime
FROM alpine
WORKDIR /app
COPY --from=builder /app/main .
CMD ["./main"]
