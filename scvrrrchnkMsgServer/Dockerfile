# Этап 1: Сборка приложения
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Собираем бинарник. Обрати внимание на имя в -o
RUN CGO_ENABLED=0 GOOS=linux go build -o /msg-server ./cmd/server

# Этап 2: Финальный образ
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /msg-server .
EXPOSE 8081
# Команда для запуска
CMD ["./msg-server"]