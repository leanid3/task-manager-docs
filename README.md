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

> 💡 Все команды можно скопировать и вставить напрямую.  
> Для работы требуется: `go`, `docker`, `docker-compose`, `make`.

### 1. Нативный запуск (Go)

Используется для разработки и отладки. Запускается как локальный бинарник, читает `config.yaml`.

```bash
# 1. Создать конфигурацию (если ещё нет)
cp config.example.yaml config.yaml

# 2. Собрать бинарник
make build-local

# 3. Запустить приложение
make run-local
```

✅ Работает на порту `8080`, метрики — `9090`.  
⚠️ Убедитесь, что `config.yaml` существует и содержит корректные параметры подключения к DB/Kafka/MinIO.

---

### 2. Docker (dev и prod)

#### 🔹 Dev-режим (рекомендуется для разработки)
Использует `config.yaml` из проекта + `.env` (если есть). Подходит для быстрой отладки.

```bash
# 1. Создать конфигурацию
cp config.example.yaml config.yaml

# 2. Собрать образ (dev target)
make build-dev

# 3. Запустить контейнер
make run-dev
```

#### 🔹 Prod-режим (для production)
**Не использует `config.yaml`!** Всё берётся из `.env`. Файл `config.yaml` игнорируется.

```bash
# 1. Создать .env (можно скопировать из .env.example)
cp .env.example .env

# 2. Отредактировать .env (обязательно: DB, Kafka, MinIO)
#    Пример: POSTGRES_USER=myuser, KAFKA_BOOTSTRAP=kafka:9092 и т.д.

# 3. Собрать prod-образ
make prod-build

# 4. Запустить prod-контейнер
make prod-run
```

📌 **Важно**:  
- В `prod` режиме нельзя использовать `config.yaml` — только `.env`.  
- Для переопределения портов/сетей используйте переменные Makefile:  
  ```bash
  make run-dev PORT_C=3000 NET_C=my-net
  ```

---

### 3. Docker Compose (полный стек)

Запускает всё: PostgreSQL, Kafka, MinIO, миграции и сам `task-manager` в одном команде.

```bash
# 1. Создать конфигурацию (для dev-режима внутри контейнера)
cp config.example.yaml config.yaml

# 2. Запустить весь стек (автоматически собирает образ и запускает зависимости)
docker-compose up --build
```

✅ Автоматически:
- создаёт БД и миграции,
- инициализирует MinIO-бакет,
- запускает `task-manager` в режиме `dev` (использует `config.yaml` из проекта),
- открывает порты: `8080` (API), `9090` (метрики), `9000` (MinIO), `9100` (Kafka UI), `9099` (Adminer).

💡 Для остановки: `docker-compose down`

---

## ⚙️ Конфигурация

Сервис поддерживает два формата конфигурации:

| Режим | Источник | Файл | Переменные окружения |
|-------|----------|------|----------------------|
| **dev** | YAML + ENV override | `config.yaml` | `.env` (необязательно) |
| **prod** | Только ENV | — | `.env` (**обязательно**) |

### Формат логов

В секции `logger` в `config.yaml` задаются:
```yaml
logger:
  level: debug          # debug, info, warn, error
  format: json          # text, json, color
  mode: stdout          # stdout или files
  max_size: 10          # MB на файл
  max_files: 5          # кол-во ротаций
  clear_on_start: true  # очищать при старте
```

- `mode: stdout` — логи выводятся в терминал (подходит для Docker и разработки).
- `mode: files` — логи пишутся в файлы (например, `./logs/app.log`).  
  Пути можно указать явно:
  ```yaml
  app_path: "/app/logs/app.log"
  health_path: "/app/logs/health.log"
  # и т.д.
  ```

### Переопределение через ENV

Любой параметр из `config.yaml` можно переопределить через переменные окружения в `.env` или при запуске:

```bash
# Пример: изменить порт сервера
SERVER_PORT=8081 make run-dev
```

Правило именования:  
`SECTION_NAME_PARAM_NAME` → `LOGGER_LEVEL`, `SERVER_PORT`, `DATABASE_HOST`, `MINIO_ENDPOINT` и т.д.

> Полный список переменных можно получить из `config.example.yaml` и кода (пакет `config`).

---

## 📚 Дополнительно

- [Подробная документация по Docker](doc/DOCKER.md)  
- `make help` — показать все доступные команды  
- `make logs` — посмотреть логи контейнера  
- `make test-all` — запустить все тесты (unit + integration)

---