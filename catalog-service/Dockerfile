# Шаг 1: Сборка (Builder)
FROM golang:1.22-alpine AS builder

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы go.mod и go.sum
COPY go.mod go.sum ./
# Скачиваем зависимости (кешируется слоем докера)
RUN go mod download

# Копируем весь исходный код
COPY . .

# Компилируем бинарник.
# CGO_ENABLED=0 критически важно для запуска в пустом alpine-образе
# -ldflags="-s -w" удаляет отладочную информацию, уменьшая вес файла
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o catalog_app ./cmd/app/main.go

# Шаг 2: Финальный легковесный образ (Runner)
FROM alpine:latest

WORKDIR /root/

# Копируем только готовый бинарник из шага builder
COPY --from=builder /app/catalog_app .

# Указываем порт, который слушает приложение (для документации)
EXPOSE 8081

# Команда запуска
CMD ["./catalog_app"]