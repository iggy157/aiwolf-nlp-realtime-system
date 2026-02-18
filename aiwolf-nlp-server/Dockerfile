FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o aiwolf-nlp-server main.go

FROM alpine:3.21

WORKDIR /app
COPY --from=builder /app/aiwolf-nlp-server .
COPY config/ ./config/

RUN mkdir -p ./log

EXPOSE 8080

CMD ["./aiwolf-nlp-server", "-c", "config/realtime_5.yml"]
