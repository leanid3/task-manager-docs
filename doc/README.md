# Task Manager - Документация

Task Manager - это API Gateway сервис для управления задачами обработки документов через LLM и другие воркеры. Сервис обеспечивает единую точку входа для внешних клиентов, управление жизненным циклом задач и интеграцию с системой обработки через Kafka.

## Быстрый старт

### 1. Подготовка окружения

```bash
# Скопируйте конфигурационный файл
cp config.example.yaml config.yaml

# Отредактируйте config.yaml с вашими настройками
```

### 2. Запуск через Docker

```bash
# Сборка образа
make build-dev

# Запуск контейнера
make run-dev
```

### 3. Проверка работоспособности

```bash
# Health check
curl http://localhost:8080/health/live

# Swagger документация
# Откройте в браузере: http://localhost:8080/swagger/index.html
```

## Основные возможности

### ✅ Реализовано

1. **REST API**
   - Создание задач обработки файлов (POST `/api/v1/tasks/{filename}`)
   - Получение статуса задачи (GET `/api/v1/tasks/{task_id}`)
   - Health check endpoints (`/health/live`, `/health/ready`)

2. **Kafka Integration**
   - Producer: публикация задач в топик `tasks_llm`
   - Consumer: обработка событий изменения статуса задач
   - Поддержка идемпотентности и гарантий доставки

3. **Storage Integration**
   - Загрузка файлов в MinIO/S3
   - Управление путями хранения
   - Автоматическая очистка при ошибках

4. **Database Integration**
   - Хранение задач в PostgreSQL
   - Отслеживание статусов и результатов
   - Поддержка трейсинга (trace_id, request_id)

5. **Observability**
   - Структурированное логирование
   - Разделение логов по компонентам
   - Поддержка trace_id для распределенного трейсинга

6. **Graceful Shutdown**
   - Корректное завершение HTTP сервера
   - Остановка Kafka consumer
   - Закрытие соединений с БД

## Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    External Clients                          │
└───────────────────────┬─────────────────────────────────────┘
                        │ HTTP REST API
                        ↓
┌─────────────────────────────────────────────────────────────┐
│                    Task Manager Service                      │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ REST API     │  │ Kafka        │  │ Use Cases    │      │
│  │ Handlers     │  │ Handlers     │  │ (Business    │      │
│  │              │  │              │  │  Logic)      │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                 │              │
│         └──────────────────┴─────────────────┘              │
│                            │                                 │
│                   ┌────────▼────────┐                        │
│                   │   Repositories  │                        │
│                   │  - Task         │                        │
│                   │  - Storage      │                        │
│                   └────────┬────────┘                        │
└────────────────────────────┼─────────────────────────────────┘
                             │
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ↓                    ↓                    ↓
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  PostgreSQL  │    │    Kafka     │    │   MinIO/S3   │
│  (Database)  │    │   (Broker)   │    │  (Storage)   │
└──────────────┘    └──────────────┘    └──────────────┘
                             │
                             ↓
                    ┌──────────────┐
                    │ LLM Worker   │
                    │ (External)   │
                    └──────────────┘
```

## Поток обработки задачи

1. **Создание задачи** (HTTP POST)
   - Клиент отправляет файл через REST API
   - Сервис создает задачу в БД со статусом `PENDING`
   - Файл загружается в MinIO/S3
   - Задача публикуется в Kafka топик `tasks_llm`

2. **Обработка задачи** (Kafka Consumer)
   - LLM Worker получает задачу из Kafka
   - Отправляет событие `PROCESSING` обратно в Kafka
   - Task Manager обновляет статус задачи в БД

3. **Завершение задачи** (Kafka Consumer)
   - LLM Worker отправляет событие `COMPLETED` или `FAILED`
   - Task Manager обновляет результат/ошибку в БД
   - Клиент может получить результат через GET запрос

## Взаимодействие с сервисами

### Kafka
- **Producer**: Публикует команды на обработку задач
- **Consumer**: Подписывается на события изменения статуса
- **Топики**: `tasks_llm` (основной топик для LLM задач)

### PostgreSQL
- Хранение задач и их статусов
- Метаданные и результаты обработки
- Поддержка трейсинга (trace_id, request_id)

### MinIO/S3
- Хранение загруженных файлов
- Структура: `{bucket}/{task_id}/{filename}`
- Автоматическая очистка при ошибках

## Документация

- [[architecture_summary|Архитектура сервиса]] - детальное описание архитектуры
- [[api_guide|API Guide]] - описание REST API endpoints
- [[events_guide|Events Guide]] - форматы сообщений и события Kafka
- [[integration_guide|Integration Guide]] - руководство по интеграции
- [[lifecycle_guide|Lifecycle Guide]] - жизненный цикл задач
- [[broker/kafka|Kafka Integration]] - детали работы с Kafka
- [[DOCKER|Docker Guide]] - развертывание через Docker

## Конфигурация

Основные настройки находятся в `config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  host: "postgres-task-manager"
  port: 5432
  database: "task_manager"

broker:
  bootstrap_service: "kafka:29092"
  topics:
    - tasks_llm

minio:
  endpoint: "minio:9000"
  bucket: "documents"
```

Подробнее: см. `config.example.yaml`

## Мониторинг

### Health Checks

```bash
# Liveness probe
curl http://localhost:8080/health/live

# Readiness probe
curl http://localhost:8080/health/ready
```

### Логирование

Логи разделены по компонентам:
- `app.log` - основные логи приложения
- `http.log` - HTTP запросы
- `kafka.log` - операции с Kafka
- `task.log` - обработка задач
- `minio.log` - операции с хранилищем

## Развертывание

### Docker

```bash
# Dev режим
make build-dev
make run-dev

# Prod режим
docker build -t task-manager:prod --target prod .
```

### Переменные окружения

В production можно использовать переменные окружения вместо `config.yaml`:

```bash
export SERVER_PORT=8080
export DATABASE_HOST=postgres
export KAFKA_BOOTSTRAP_SERVERS=kafka:29092
```

## Troubleshooting

### Сервис не запускается

```bash
# Проверьте логи
make logs

# Проверьте конфигурацию
cat config.yaml

# Проверьте доступность зависимостей
docker ps | grep -E "(postgres|kafka|minio)"
```

### Задачи не обрабатываются

```bash
# Проверьте Kafka consumer
docker logs task-manager | grep kafka

# Проверьте топик
kafka-topics --bootstrap-server localhost:9092 --list

# Проверьте сообщения в топике
kafka-console-consumer --bootstrap-server localhost:9092 --topic tasks_llm
```

## Production Readiness

### Критичные доделки

- [ ] Метрики Prometheus (`/metrics`)
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Rate limiting
- [ ] Authentication/Authorization
- [ ] SSL/TLS для Kafka и MinIO

### Опциональные улучшения

- [ ] Кэширование задач (Redis)
- [ ] Webhook уведомления
- [ ] Batch обработка задач
- [ ] Retry механизм для failed задач

## Контакты / Поддержка

При возникновении проблем:
1. Проверьте логи: `make logs`
2. Проверьте документацию в папке `doc/`
3. Проверьте Swagger: http://localhost:8080/swagger/index.html

