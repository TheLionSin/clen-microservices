FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o notification_app ./cmd/app/main.go


FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/notification_app .

CMD ["./notification_app"]