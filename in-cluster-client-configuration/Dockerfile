FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o in-cluster

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/in-cluster .

CMD ["./in-cluster"]
