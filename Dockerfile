# ============================================
# Стадия сборки (общая для dev и prod)
# ============================================
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

# Устанавливаем swag для генерации swagger документации
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12

# Копируем исходники
COPY . .

# Генерируем swagger документацию
RUN swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

#TODO найти альтернативу для confluent-kafka-go, чтобы убрать CGO
# Собираем бинарник с включенным CGO (требуется для confluent-kafka-go)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# ============================================
# Базовый образ runtime (общий слой)
# ============================================
FROM debian:bookworm-slim AS base

# Устанавливаем runtime зависимости для confluent-kafka-go
RUN apt-get update && apt-get install -y \
    ca-certificates \
    librdkafka1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /build/app .

# Создаём пользователя без прав root
RUN useradd -m -u 1000 appuser
USER appuser

# Получаем порт через build argument (по умолчанию 8080)
ARG PORT=8080
EXPOSE ${PORT}

ENV PORT=${PORT}
ENV APP_PORT=${PORT}

# ============================================
# Dev образ (использует config.yaml из проекта)
# ============================================
FROM base AS dev

# В dev режиме используем config.yaml из директории проекта
ARG CONFIG_FILE=config.yaml
COPY --chown=appuser:appuser ${CONFIG_FILE} /app/config.yaml

CMD ["./app"]

# ============================================
# Prod образ (использует только переменные окружения из .env)
# ============================================
FROM base AS prod

# В prod режиме конфигурация загружается только из переменных окружения
# Файл config.yaml не копируется - используется .env

CMD ["./app"]

# ============================================
# Финальный образ (по умолчанию dev)
# ============================================
FROM dev

# Использование:
# 
# Dev режим (по умолчанию, использует config.yaml из проекта):
#   docker build -t task-manager:dev --target dev .
#   docker-compose build task-manager  # использует target: dev из docker-compose.yaml
#
# Prod режим (использует только переменные окружения из .env):
#   docker build -t task-manager:prod --target prod .
#   или в docker-compose.yaml изменить target: dev на target: prod
