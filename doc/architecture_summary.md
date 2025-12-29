# Архитектура Task Manager

## Обзор

Task Manager построен на основе Clean Architecture с разделением на слои:
- **Handlers** - обработка HTTP и Kafka сообщений
- **Use Cases** - бизнес-логика
- **Repositories** - абстракция доступа к данным
- **Infrastructure** - адаптеры для внешних сервисов

## Архитектурные слои

```
┌─────────────────────────────────────────────────────────────┐
│ 1. HANDLERS (Presentation Layer)                          │
│    internal/handlers/                                       │
│    - restapi/ - HTTP handlers (Gin)                        │
│    - broker/ - Kafka message handlers                      │
└───────────────────┬─────────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────────┐
│ 2. USE CASES (Business Logic Layer)                        │
│    internal/usecase/                                        │
│    - TaskLLMUC - бизнес-логика обработки LLM задач         │
│    - UseCases - контейнер use cases                        │
└───────────────────┬─────────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────────┐
│ 3. REPOSITORIES (Data Access Layer)                        │
│    internal/entity/repository/                             │
│    - Task - интерфейс работы с задачами                    │
│    - Storage - интерфейс работы с хранилищем               │
└───────────────────┬─────────────────────────────────────────┘
                    │
┌───────────────────▼─────────────────────────────────────────┐
│ 4. INFRASTRUCTURE (External Adapters)                      │
│    internal/infrastructure/adapter/                         │
│    - database/postgres/ - PostgreSQL адаптер               │
│    - storage/minio/ - MinIO/S3 адаптер                    │
│    pkg/                                                    │
│    - kafka/ - Kafka producer/consumer                      │
│    - minio/ - MinIO connector                              │
│    - database/ - PostgreSQL connector                      │
└─────────────────────────────────────────────────────────────┘
```

## Компоненты

### 1. HTTP API Layer

**Расположение**: `internal/handlers/restapi/`

- **Router** (`router.go`): Настройка маршрутов и middleware
- **V1 Handlers** (`v1/`): Обработчики API v1
  - `createTask` - создание задачи
  - `getTaskByID` - получение задачи

**Middleware**:
- RequestID - генерация уникального ID для каждого запроса
- Logger - логирование HTTP запросов
- Recovery - обработка паник

### 2. Kafka Handlers

**Расположение**: `internal/handlers/broker/`

- **KafkaMessageHandler**: Диспетчер сообщений по топикам
  - `HandleTaskStatusLLM` - обработка событий статуса LLM задач
  - Расширяем для новых типов задач

### 3. Use Cases

**Расположение**: `internal/usecase/`

#### TaskLLMUC

Основной use case для работы с LLM задачами:

```go
type TaskLLMUC struct {
    taskRepo    repository.Task
    producer    broker.Producer
    storageRepo repository.Storage
    topic       string
    l           logger.Interface
}
```

**Методы**:
- `CreateTask` - создание новой задачи
  1. Валидация входных данных
  2. Генерация task_id и trace_id
  3. Загрузка файла в MinIO
  4. Создание записи в БД
  5. Публикация команды в Kafka
  6. Компенсация при ошибках (удаление файла)

- `GetTaskByID` - получение задачи по ID
- `UpdateTaskStatus` - обновление статуса задачи из Kafka события

### 4. Repositories

**Расположение**: `internal/entity/repository/`

#### Task Repository

Интерфейс для работы с задачами:

```go
type Task interface {
    Create(ctx context.Context, task *domain.Task) error
    GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error)
    UpdateWithStatus(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus) error
    UpdateWithResult(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, result json.RawMessage) error
    UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error
}
```

**Реализация**: `internal/infrastructure/adapter/database/postgres/task_repository.go`

#### Storage Repository

Интерфейс для работы с хранилищем:

```go
type Storage interface {
    UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.UploadOptions) (*minio.ObjectInfo, error)
    DeleteObject(ctx context.Context, objectName string) error
    GenerateStoragePath(taskID uuid.UUID, filename string) string
}
```

**Реализация**: `internal/infrastructure/adapter/storage/minio/minio_repository.go`

### 5. Domain Models

**Расположение**: `internal/entity/domain/`

#### Task

Базовая модель задачи:

```go
type Task struct {
    TaskID       uuid.UUID
    Status       TaskStatus  // PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED
    Metadata     map[string]interface{}
    WorkerID     string
    RequestID    string
    TraceID      *uuid.UUID
    Result       json.RawMessage
    CreatedAt    time.Time
    StartedAt    *time.Time
    CompletedAt  *time.Time
    ErrorMessage string
}
```

#### TaskLLM

Расширенная модель для LLM задач:

```go
type TaskLLM struct {
    Task
    StoragePath string
    StorageSize int64
    Metadata    LLMMetadata
}
```

#### TaskLLMCommand / TaskLLMStatusEvent

Модели для Kafka сообщений (см. [[events_guide|Events Guide]])

### 6. Infrastructure

#### Kafka

**Producer** (`pkg/kafka/producer.go`):
- Публикация команд в топики
- Поддержка идемпотентности
- Retry механизм

**Consumer** (`pkg/kafka/consumer.go`):
- Подписка на топики
- Автоматический коммит офсетов
- Graceful shutdown

#### PostgreSQL

**Connector** (`pkg/database/connector/sql/postgres/`):
- Connection pooling
- Health checks
- Graceful shutdown

**Repository** (`internal/infrastructure/adapter/database/postgres/`):
- CRUD операции для задач
- Транзакции (TODO)

#### MinIO/S3

**Connector** (`pkg/minio/connector.go`):
- Подключение к MinIO/S3
- Health checks

**Adapter** (`internal/infrastructure/adapter/storage/minio/`):
- Загрузка файлов (streaming)
- Удаление файлов
- Генерация путей хранения

## Поток данных

### Создание задачи

```
HTTP Request
    ↓
REST Handler (createTask)
    ↓
TaskLLMUC.CreateTask
    ↓
    ├─→ Storage.UploadStream (MinIO)
    ├─→ TaskRepository.Create (PostgreSQL)
    └─→ Producer.Send (Kafka)
```

### Обновление статуса

```
Kafka Message
    ↓
KafkaMessageHandler.HandleTaskStatusLLM
    ↓
TaskLLMUC.UpdateTaskStatus
    ↓
TaskRepository.UpdateWithStatus/Result/Error (PostgreSQL)
```

## Паттерны проектирования

### 1. Repository Pattern

Абстракция доступа к данным через интерфейсы:
- Легкая замена реализации (PostgreSQL → MySQL)
- Тестируемость (моки)
- Изоляция бизнес-логики от инфраструктуры

### 2. Use Case Pattern

Изоляция бизнес-логики:
- Независимость от HTTP/Kafka
- Переиспользование
- Тестируемость

### 3. Dependency Injection

Зависимости передаются через конструкторы:
- Легкое тестирование
- Гибкая конфигурация
- Явные зависимости

### 4. Strategy Pattern

Разные реализации интерфейсов:
- Storage: MinIO, S3, локальное хранилище
- Database: PostgreSQL, MySQL (будущее)

## Жизненный цикл приложения

### Инициализация (`cmd/api/main.go`)

1. Загрузка конфигурации
2. Инициализация логгеров
3. Подключение к PostgreSQL
4. Подключение к MinIO
5. Создание Kafka Producer
6. Создание репозиториев
7. Создание Use Cases
8. Создание Kafka Consumer
9. Запуск HTTP сервера
10. Запуск Kafka Consumer (goroutine)

### Graceful Shutdown

1. Получение SIGTERM/SIGINT
2. Отмена контекста (остановка Consumer)
3. Остановка HTTP сервера
4. Остановка Kafka Consumer
5. Закрытие соединений с БД

## Масштабируемость

### Горизонтальное масштабирование

- **HTTP сервер**: Можно запускать несколько инстансов за load balancer
- **Kafka Consumer**: Consumer Group обеспечивает распределение нагрузки
- **Stateless**: Сервис не хранит состояние в памяти

### Вертикальное масштабирование

- Connection pooling для PostgreSQL
- Настройка batch size для Kafka Producer
- Настройка timeout'ов

## Безопасность

### Текущее состояние

- Нет аутентификации/авторизации
- Нет SSL/TLS для Kafka (PLAINTEXT)
- Нет SSL для MinIO (UseSSL: false)

### Рекомендации для production

- [ ] JWT токены для API
- [ ] SASL_PLAINTEXT или SASL_SSL для Kafka
- [ ] SSL/TLS для MinIO
- [ ] Secrets management (Vault, K8s Secrets)
- [ ] Rate limiting
- [ ] Input validation и sanitization

## Observability

### Логирование

Структурированное логирование с разделением по компонентам:
- `app.log` - основные логи
- `http.log` - HTTP запросы
- `kafka.log` - Kafka операции
- `task.log` - обработка задач
- `minio.log` - операции с хранилищем

### Трейсинг

- `request_id` - уникальный ID для каждого HTTP запроса
- `trace_id` - распределенный трейсинг (UUID)
- `task_id` - идентификатор задачи

### Метрики

- TODO: Prometheus metrics endpoint
- TODO: Метрики обработки задач
- TODO: Метрики Kafka lag

## Расширяемость

### Добавление нового типа задач

1. Создать новый Use Case (например, `TaskAnalysisUC`)
2. Добавить обработчик в `KafkaMessageHandler`
3. Добавить новый топик в конфигурацию
4. Расширить domain модели при необходимости

### Добавление нового хранилища

1. Реализовать интерфейс `repository.Storage`
2. Создать connector в `pkg/`
3. Создать adapter в `internal/infrastructure/adapter/storage/`
4. Обновить инициализацию в `main.go`

## Тестирование

### Unit тесты

- Use Cases с моками репозиториев
- Handlers с моками Use Cases
- Domain модели

### Integration тесты

- Тесты с реальной БД (testcontainers)
- Тесты с реальным Kafka (testcontainers)
- Тесты с реальным MinIO (testcontainers)

## Преимущества архитектуры

### ✅ Гибкость

- Легкая замена реализаций
- Добавление новых типов задач
- Расширение функциональности

### ✅ Тестируемость

- Изолированные компоненты
- Моки для внешних зависимостей
- Unit и integration тесты

### ✅ Поддерживаемость

- Четкое разделение ответственности
- Понятная структура
- Документированный код

### ✅ Production-ready

- Graceful shutdown
- Health checks
- Error handling
- Логирование

