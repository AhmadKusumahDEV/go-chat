# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache ca-certificates git
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and configs
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/api

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary and config files
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

# Default environment variables
ENV APP_ENV=production
ENV SERVER_PORT=8080


CMD ["./main"]