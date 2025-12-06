# Стадия сборки
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Собираем бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# Финальный образ
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Копируем только бинарник
COPY --from=builder /build/app .

# Получаем порт через build argument (по умолчанию 8080)
ARG PORT=9080
EXPOSE ${PORT}

# Создаём пользователя без прав root
RUN adduser -D -u 1000 appuser
USER appuser

ENV PORT=${PORT}
ENV APP_PORT=${PORT}

CMD ["./app"]
