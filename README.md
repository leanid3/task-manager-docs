### Task-manager
---
### Цель:
 1) Точка доступа в приложение по API для внешних сервисов
 2) Постановка задач для workerов
 3) Получение результатов с workerов

---
### Взаимодействие с другими сервисами:
 1) Kafka — consumer и producer
 2) Postgres — connector и adapter
 3) MinIO — connector и adapter

---
Возможен доступ по Swagger: http://localhost:8080/swagger/index.html

---
## Метрики

Сервис предоставляет метрики в формате Prometheus по адресу: `http://localhost:9091/metrics` (порт и путь настраиваются в конфигурации).

Поддерживаются следующие метрики:
- HTTP-запросы (время выполнения, количество)
- Обработка задач (время выполнения, статусы)
- Сообщения Kafka (потребленные, произведенные)
- Операции с базой данных (время выполнения запросов)
- Операции с MinIO (время выполнения, объем переданных данных)

---

## 🚀 Быстрый старт

> 💡 Все команды можно скопировать и вставить напрямую.
> Для работы требуется: `go`, `docker`, `docker-compose`, `make`.

### 1. Нативный запуск (Go)

Используется для разработки и отладки. Запускается как локальный бинарник, читает переменные окружения.

```bash
# 1. Создать .env (если ещё нет)
cp .env.example .env

# 2. Собрать бинарник
make build-local

# 3. Запустить приложение
make run-local
```

✅ Работает на порту `8080`, метрики — `9090`.

---

### 2. Docker (dev и prod)

#### 🔹 Dev-режим (рекомендуется для разработки)
Исходники монтируются в контейнер, приложение запускается через `go run`. Позволяет вносить изменения в код без пересборки образа.

```bash
# 1. Создать .env
cp .env.example .env

# 2. Собрать образ (dev target)
make build-dev

# 3. Запустить контейнер
make run-dev
```

#### 🔹 Prod-режим (для production)
**Не использует `config.yaml`!** Всё берётся из `.env`.

```bash
# 1. Создать .env
cp .env.example .env

# 2. Отредактировать .env (обязательно: DB, Kafka, MinIO)

# 3. Собрать prod-образ
make prod-build

# 4. Запустить prod-контейнер
make prod-run
```

📌 **Важно**:
- В `prod` режиме конфигурация загружается **только** из переменных окружения (`.env`).

---

### 3. Docker Compose (полный стек)

Запускает всё: PostgreSQL, Kafka, MinIO, миграции и сам `task-manager` в одной команде.

#### 🔹 Dev-режим (с горячей перезагрузкой кода)

Исходники монтируются в контейнер, приложение запускается через `go run`.

```bash
# 1. Создать .env (можно скопировать из .env.example)
cp .env.example .env

# 2. Запустить весь стек в dev режиме
make up-dev

# Или напрямую:
docker compose -f docker-compose/base.yaml -f docker-compose/dev.yaml --profile dev up --build
```

✅ Автоматически:
- создаёт БД и миграции,
- инициализирует MinIO-бакет,
- запускает `task-manager` в режиме `dev` (исходники монтируются, используется `go run cmd/api/main.go`),
- открывает порты: `8080` (API), `9091` (метрики), `9000` (MinIO), `2181` (Zookeeper), `9092` (Kafka).

#### 🔹 Prod-режим (оптимизированный образ)

**Не использует `config.yaml`!** Всё берётся из `.env`.

```bash
# 1. Создать .env (можно скопировать из .env.example)
cp .env.example .env

# 2. Запустить весь стек в prod режиме
make up-prod

# Или напрямую:
docker compose -f docker-compose/base.yaml -f docker-compose/prod.yaml --profile prod up --build
```

📌 **Важно**:
- В `prod` режиме конфигурация загружается **только** из переменных окружения (`.env`).
- Используется multi-stage build с оптимизированным образом.
- Логирование в JSON формате для лучшей интеграции с системами мониторинга.

#### 🔹 Остановка и управление

```bash
# Остановить все сервисы
make down

# Просмотр логов
make logs SERVICE=task-manager

# Статус сервисов
make status

# Пересборка образа
make build SERVICE=task-manager
```

#### 🔹 Структура файлов

```
docker-compose/
├── base.yaml       # Базовая инфраструктура (БД, Kafka, MinIO)
├── dev.yaml        # Dev профиль (go run, volume с кодом)
├── prod.yaml       # Prod профиль (бинарник)
└── standalone.yaml # Только инфраструктура
```

---

## ⚙️ Конфигурация

Сервис использует **только переменные окружения** для конфигурации (загружаются из `.env`).

### Формат логов

Конфигурация логирования задаётся через переменные окружения:
```bash
LOGGER_LEVEL=debug          # debug, info, warn, error
LOGGER_FORMAT=text          # text, json (prod использует json)
LOGGER_MODE=stdout          # stdout или files
```

- `mode: stdout` — логи выводятся в терминал (подходит для Docker и разработки).
- `mode: files` — логи пишутся в файлы (требует указания путей в конфиге).

### Переменные окружения

Все параметры задаются через переменные окружения в `.env`:

```bash
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database
DATABASE_HOST=task-manager-database
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_DATABASE=task_manager

# Broker (Kafka)
BROKER_BOOTSTRAP_SERVICE=kafka:29092
BROKER_TOPICS=tasks_llm

# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=documents
```

> Полный список переменных можно получить из `.env.example` и кода (пакет `config`).

---

## 📚 Дополнительно

- [Подробная документация по Docker](doc/DOCKER.md)
- `make help` — показать все доступные команды
- `make logs` — посмотреть логи контейнера
- `make test-all` — запустить все тесты (unit + integration)

---
