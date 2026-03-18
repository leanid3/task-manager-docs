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

Сервис предоставляет метрики в формате Prometheus по адресу: `http://localhost:9090/metrics` (порт и путь настраиваются в конфигурации).

Поддерживаются следующие метрики:
- HTTP-запросы (время выполнения, количество)
- Обработка задач (время выполнения, статусы)
- Сообщения Kafka (потребленные, произведенные)
- Операции с базой данных (время выполнения запросов)
- Операции с MinIO (время выполнения, объем переданных данных)

---

## 🚀 Быстрый старт

### 1. Нативный запуск (Go)

```bash
# 1. Создать конфигурацию (если ещё нет)
cp .env.example .env

# 2. Собрать бинарник
make build-local

# 3. Запустить приложение
make run-local
```

---

### 2. Docker (dev и prod)

#### 🔹 Dev-режим (рекомендуется для разработки)
Исходники монтируются в контейнер, приложение запускается через `go run`. Позволяет вносить изменения в код без пересборки образа.

```bash
# 1. Создать конфигурацию
cp .env.example .env

# 2. Собрать образ (dev target)
make build-dev

# 3. Запустить контейнер с поддержкой горячей перезагрузки
make run-dev
```

#### 🔹 Prod-режим (для production)

```bash
# 1. Создать .env (можно скопировать из .env.example)
cp .env.example .env


# 2. Собрать prod-образ
make prod-build

# 3. Запустить prod-контейнер
make prod-run
```
---

### 3. Docker Compose (полный стек)

#### 🔹 Dev-режим 

Исходники монтируются в контейнер, приложение запускается через `go run`.

```bash
# 1. Создать .env (можно скопировать из .env.example)
cp .env.example .env

# 2. Запустить весь стек в dev режиме
make up-dev

# Или напрямую:
docker compose -f docker-compose.yaml -f docker-compose.dev.yaml --profile dev up --build
```
#### 🔹 Prod-режим
```bash
# 1. Создать .env (можно скопировать из .env.example)
cp .env.example .env

# 2. Запустить весь стек в prod режиме
make up-prod

# Или напрямую:
docker compose -f docker-compose.yaml -f docker-compose.prod.yaml --profile prod up --build
```
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

- `mode: stdout` — логи выводятся в терминал.
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
---

## 📚 Дополнительно

- [Подробная документация по Docker](doc/DOCKER.md)  
- `make help` — показать все доступные команды  
- `make logs` — посмотреть логи контейнера  
- `make test-all` — запустить все тесты (unit + integration)

---
