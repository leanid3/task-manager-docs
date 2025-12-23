# Стадия сборки
FROM golang:1.25 AS builder

# Устанавливаем зависимости для компиляции confluent-kafka-go (требует CGO)
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    librdkafka-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем swag для генерации swagger документации (версия должна совпадать с библиотекой в go.mod)
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12

# Копируем исходники
COPY . .

# Генерируем swagger документацию
RUN swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# Собираем бинарник с включенным CGO (требуется для confluent-kafka-go)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# Финальный образ
FROM debian:bookworm-slim

# Устанавливаем runtime зависимости для confluent-kafka-go
RUN apt-get update && apt-get install -y \
    ca-certificates \
    librdkafka1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Копируем только бинарник
COPY --from=builder /build/app .

# Получаем порт через build argument (по умолчанию 8080)
ARG PORT=9080
EXPOSE ${PORT}

# Создаём пользователя без прав root
RUN useradd -m -u 1000 appuser
USER appuser

ENV PORT=${PORT}
ENV APP_PORT=${PORT}

CMD ["./app"]
