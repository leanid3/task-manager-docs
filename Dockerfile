# ============================================
# Базовый образ
# ============================================
FROM golang:1.25 AS base-build

# Устанавливаем зависимости
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    librdkafka-dev \
    pkg-config \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Кэширование зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем swag для генерации swagger документации
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12

# ============================================
# Сборка приложения
# ============================================
FROM base-build AS builder

WORKDIR /build

# Копируем весь исходный код для корректной работы swag
COPY . .

# Генерируем swagger документацию
RUN swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

# Собираем бинарник с включенным CGO
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# ============================================
# Dev образ
# ============================================
FROM base-build AS dev
WORKDIR /app

# Копируем только go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Устанавливаем air для hot reload (опционально)
RUN go install github.com/air-verse/air@latest

ENV SERVER_PORT=8080
ENV SERVER_HOST=0.0.0.0
EXPOSE ${SERVER_PORT}

# Исходники монтируются через volume в docker-compose
# Генерация swagger происходит при запуске через команду
CMD ["go", "run", "cmd/api/main.go"]

# ============================================
# Базовый образ runtime
# ============================================
FROM debian:bookworm-slim AS base

# Устанавливаем runtime зависимости
RUN apt-get update && apt-get install -y \
    ca-certificates \
    librdkafka1 \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 1000 appuser


WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /build/app .

# Создаём пользователя без прав root
USER appuser

ENV SERVER_PORT=8080
ENV SERVER_HOST=0.0.0.0
EXPOSE ${SERVER_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${SERVER_PORT}/health/live || exit 1


FROM base AS prod

CMD ["./app"]
