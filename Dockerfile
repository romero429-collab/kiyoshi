FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY apps/cli/go.mod apps/cli/go.sum ./
RUN go mod download

COPY apps/cli/*.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o kiyoshi-cli .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/kiyoshi-cli .

EXPOSE 8080

CMD ["./kiyoshi-cli", "-mode=http", "-addr=:8080"]
