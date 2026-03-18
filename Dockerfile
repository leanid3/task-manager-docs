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

# Собираем бинарник с включенным CGO (требуется для confluent-kafka-go)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/api

# ============================================
# Dev образ - для разработки с go run
# ============================================
FROM golang:1.25 AS dev

# Устанавливаем зависимости для компиляции confluent-kafka-go (требует CGO)
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    librdkafka-dev \
    pkg-config \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Устанавливаем air для hot reload
RUN go install github.com/air-verse/air@latest

WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Устанавливаем swag для генерации swagger документации
RUN go install github.com/swaggo/swag/cmd/swag@v1.8.12

# Генерируем swagger документацию
RUN swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

ENV SERVER_PORT=8080
EXPOSE ${SERVER_PORT}

# CMD будет переопределён в docker-compose для запуска go run или air
CMD ["go", "run", "cmd/api/main.go"]

# ============================================
# Базовый образ runtime (общий слой для prod)
# ============================================
FROM debian:bookworm-slim AS base

# Устанавливаем runtime зависимости для confluent-kafka-go
RUN apt-get update && apt-get install -y \
    ca-certificates \
    librdkafka1 \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Копируем бинарник из builder
COPY --from=builder /build/app .

# Создаём пользователя без прав root
RUN useradd -m -u 1000 appuser
USER appuser

ENV SERVER_PORT=8080
EXPOSE ${SERVER_PORT}

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${SERVER_PORT}/health/live || exit 1

# ============================================
# Prod образ (использует только переменные окружения из .env)
# ============================================
FROM base AS prod

# В prod режиме конфигурация загружается только из переменных окружения
# Файл config.yaml не копируется - используется .env

CMD ["./app"]
