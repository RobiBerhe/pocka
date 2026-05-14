# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Install dependencies for CGO if needed (sqlite etc), but we use postgres so alpine is fine
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pocka-bot ./cmd/bot

# Final stage
FROM alpine:latest  

RUN apk --no-cache add ca-certificates tzdata ttf-dejavu ttf-freefont font-noto-emoji fontconfig

WORKDIR /root/

COPY --from=builder /app/pocka-bot .

# Expose port
EXPOSE 8080

CMD ["./pocka-bot"]
