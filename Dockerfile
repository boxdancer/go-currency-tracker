# Stage 1: build
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Копируем go.mod/go.sum, чтобы кэшировать зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем бинарь
RUN go build -o currency-tracker ./cmd/app

# Stage 2: минимальный образ
FROM alpine:3.18

WORKDIR /app
COPY --from=builder /app/currency-tracker .

EXPOSE 8080

CMD ["./currency-tracker"]
