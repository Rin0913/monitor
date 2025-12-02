FROM golang:1.23.2 AS builder
WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o monitor ./cmd/server/monitor.go

FROM alpine:3.19
WORKDIR /app

COPY --from=builder /app/monitor .

EXPOSE 8080
CMD ["./monitor"]
