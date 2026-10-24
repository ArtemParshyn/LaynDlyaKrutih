FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ledger main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/ledger .

EXPOSE 8081

CMD ["./ledger"]
