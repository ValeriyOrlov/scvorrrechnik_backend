# Этап 1: Сборка приложения
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Собираем бинарник. Обрати внимание на имя в -o
RUN CGO_ENABLED=0 GOOS=linux go build -o /auth-server ./cmd/server

# Этап 2: Финальный образ
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /auth-server .
EXPOSE 8080
# Команда для запуска
CMD ["./auth-server"]