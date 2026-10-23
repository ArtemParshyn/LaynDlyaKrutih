FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gateway main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/gateway .
COPY migrations ./migrations

EXPOSE 8080

CMD ["./gateway"]
