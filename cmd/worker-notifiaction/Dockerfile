# Build stage
FROM golang:1.25-alpine AS builder

RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and configs
COPY . .

# Build the job worker
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker-notifiaction

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary, config files, and firebase credentials
COPY --from=builder /app/worker .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/chat-appliaction-19fd5-firebase-adminsdk-fbsvc-51d924664f.json .

RUN chown -R appuser:appgroup /app

USER appuser

# Default environment variables
ENV APP_ENV=production
ENV TZ=Asia/Jakarta

CMD ["./worker"]
