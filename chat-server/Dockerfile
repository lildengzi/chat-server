FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o chat-server .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/chat-server ./chat-server

EXPOSE 8080

CMD ["./chat-server"]
