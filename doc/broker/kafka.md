# Документация по работе с Kafka

## Содержание

1. [Обзор](#обзор)
2. [Архитектура](#архитектура)
3. [Producer (Публикация сообщений)](#producer-публикация-сообщений)
4. [Consumer (Обработка сообщений)](#consumer-обработка-сообщений)
5. [Формат сообщений](#формат-сообщений)
6. [Конфигурация](#конфигурация)
7. [Обработка ошибок](#обработка-ошибок)
8. [Примеры использования](#примеры-использования)

## Обзор

Приложение `task-manager` использует Apache Kafka для асинхронной обработки задач. Система работает по паттерну Producer-Consumer:

- **Producer**: Публикует задачи в топик `tasks_llm` для обработки внешними сервисами (например, LLM Worker)
- **Consumer**: Подписывается на топик `tasks_llm` и обрабатывает события об изменении статуса задач

## Архитектура

### Поток данных

```
External Request → API Gateway → Producer → tasks_llm (status=pending)
                                           ↓
LLM Worker → API Gateway → Producer → tasks_llm (status=processing/completed/failed)
                                           ↓
API Gateway (Consumer) → HandleTaskStatusLLM() → UpdateTaskStatus()
```

### Компоненты

1. **Producer** (`pkg/kafka/producer.go`): Отправляет команды в Kafka
2. **Consumer** (`pkg/kafka/consumer.go`): Читает события из Kafka
3. **Handler** (`internal/handlers/broker/kafka.go`): Маршрутизирует сообщения по топикам
4. **UseCase** (`internal/usecase/task_llm_usecase.go`): Бизнес-логика обработки задач

## Producer (Публикация сообщений)

### Инициализация

Producer создается при старте приложения в `cmd/api/main.go`:

```go
kafkaProducer, err := kafka.NewProducer(kafka.ProducerConfig{
    BootstrapServers:          cfg.Broker.BootstrapService,
    ClientID:                  cfg.Broker.ClientID,
    ProducerAcks:              cfg.Broker.ProducerAcks,
    ProducerEnableIdempotence: cfg.Broker.ProducerIdempotence,
    ProducerCompressionType:   cfg.Broker.ProducerCompression,
    ProducerRetries:           cfg.Broker.ProducerRetries,
}, logMgr.Get("kafka"))
```

### Публикация сообщения

При создании новой задачи (`TaskLLMUC.CreateTask`):

1. Создается задача в базе данных
2. Загружается файл в хранилище (MinIO)
3. Формируется команда `TaskLLMCommand`
4. Команда публикуется в топик `tasks_llm`

```go
cmd := domain.NewTaskLLMCommand(taskLLM)
headers := cmd.Headers.ToMap()
err := uc.producer.Send(ctx, uc.topic, taskID.String(), headers, cmd.Value)
```

### Структура публикуемого сообщения

- **Topic**: `tasks_llm`
- **Key**: UUID задачи (string)
- **Headers**:
  - `status`: числовой код статуса (1-5)
  - `trace_id`: UUID для трейсинга (опционально)
  - `worker_id`: ID воркера (опционально)
  - `version`: версия контракта (опционально)
- **Value**: JSON с данными задачи (см. [Формат сообщений](#формат-сообщений))

### Настройки Producer

- **acks**: `-1` (all) - ожидание подтверждения от всех реплик
- **enable.idempotence**: `true` - идемпотентность сообщений
- **compression.type**: `snappy` - сжатие сообщений
- **retries**: `3` - количество повторных попыток

## Consumer (Обработка сообщений)

### Инициализация

Consumer создается при старте приложения и запускается в отдельной горутине:

```go
kafkaHandler := brokerhandlers.NewKafkaMessageHandler(*usecases, logMgr.Get("kafka"))

cons, err := kafka.NewConsumerWithHandler(ctx, kafkaHandler, cfg.Broker.Topics, kafka.ConsumerConfig{
    BootstrapServers:     cfg.Broker.BootstrapService,
    ClientID:             cfg.Broker.ClientID,
    GroupID:              cfg.Broker.ConsumerGroupID,
    EnableAutoCommit:     cfg.Broker.ConsumerEnableAutoCommit,
    AutoCommitIntervalMs: cfg.Broker.ConsumerAutoCommitIntervalMs,
    SessionTimeoutMs:     cfg.Broker.ConsumerSessionTimeoutMs,
    HeartbeatIntervalMs:  cfg.Broker.ConsumerHeartbeatIntervalMs,
}, logMgr.Get("kafka"))

// Запуск в фоне
go func() {
    if err := cons.Start(ctx); err != nil {
        l.Error("kafka consumer failed", "error", err)
    }
}()
```

### Обработка сообщений

1. **Маршрутизация** (`KafkaMessageHandler.Handle`):
   - Определяет топик сообщения
   - Вызывает соответствующий обработчик

2. **Обработка событий статуса** (`HandleTaskStatusLLM`):
   - Парсит `task_id` из ключа сообщения
   - Извлекает заголовки (status, worker_id, trace_id)
   - Десериализует тело сообщения
   - Вызывает `UpdateTaskStatus` для обновления задачи в БД

3. **Обновление статуса** (`TaskLLMUC.UpdateTaskStatus`):
   - Преобразует код статуса из Kafka в `TaskStatus`
   - Обновляет задачу в базе данных в зависимости от статуса:
     - `PROCESSING`: обновляет статус
     - `COMPLETED`: обновляет статус и результат
     - `FAILED`: обновляет статус и сообщение об ошибке

### Настройки Consumer

- **group.id**: `task_llm_group` - группа потребителей
- **enable.auto.commit**: `true` - автоматический коммит офсетов
- **auto.commit.interval.ms**: `5000` - интервал коммита (5 секунд)
- **session.timeout.ms**: `30000` - таймаут сессии (30 секунд)
- **heartbeat.interval.ms**: `10000` - интервал heartbeat (10 секунд)

### Механизм коммита

Consumer использует батч-коммит:
- Коммит каждые 10 сообщений ИЛИ
- Коммит каждые 5 секунд (что наступит раньше)

Это обеспечивает баланс между производительностью и надежностью.

### Обработка недоступности брокеров

Consumer автоматически обрабатывает ситуации, когда Kafka брокеры недоступны:

- **Экспоненциальный backoff**: задержка увеличивается с каждой попыткой (максимум 30 секунд)
- **Логирование**: предупреждения логируются не чаще раза в минуту
- **Восстановление**: при восстановлении соединения логируется информация о простое

## Формат сообщений

### TaskLLMCommand (Producer → Kafka)

Команда для создания/обработки задачи:

```json
{
  "storage_path": "tasks/{task_id}/filename.ext",
  "storage_size": 1024,
  "metadata": {
    "filename": "document.pdf",
    "filesize": 1024,
    "content_type": "application/octet-stream",
    "storage_path": "tasks/{task_id}/filename.ext"
  }
}
```

**Kafka Message:**
- **Key**: `{task_id}` (UUID в виде строки)
- **Headers**:
  - `status`: `"1"` (PENDING)
  - `trace_id`: `"{uuid}"` (опционально)
  - `worker_id`: `""` (опционально)
- **Value**: JSON выше

### TaskLLMStatusEvent (Kafka → Consumer)

Событие об изменении статуса задачи:

```json
{
  "result": {...},           // опционально, для статуса COMPLETED
  "error_message": "..."     // опционально, для статуса FAILED
}
```

**Kafka Message:**
- **Key**: `{task_id}` (UUID в виде строки)
- **Headers**:
  - `status`: `"2"` (PROCESSING), `"3"` (COMPLETED), `"4"` (FAILED)
  - `worker_id`: `"{worker_id}"` (опционально)
  - `trace_id`: `"{uuid}"` (опционально)
- **Value**: JSON выше

### Коды статусов

| Статус | Код Kafka | Описание |
|--------|-----------|----------|
| PENDING | 1 | Задача создана, ожидает обработки |
| PROCESSING | 2 | Задача обрабатывается |
| COMPLETED | 3 | Задача успешно завершена |
| FAILED | 4 | Задача завершилась с ошибкой |
| CANCELLED | 5 | Задача отменена |

## Конфигурация

Все настройки Kafka находятся в секции `broker` файла `config.yaml`:

```yaml
broker:
  bootstrap_service: "kafka:29092"
  client_id: "task_manager_1_client"
  topics:
    - tasks_llm
  
  # Producer
  producer_acks: "-1"
  producer_idempotence: true
  producer_compression: "snappy"
  producer_retries: 3
  producer_batch_size: 16384
  producer_linger_ms: 10
  producer_max_flight_requests: 5
  producer_timeout_ms: 30000
  
  # Consumer
  consumer_group_id: "task_llm_group"
  consumer_auto_commit: true
  consumer_commit_interval_ms: 5000
  consumer_session_timeout_ms: 30000
  consumer_heartbeat_interval_ms: 10000
  consumer_max_poll_records: 500
  
  # Security
  security_protocol: "PLAINTEXT"  # В продакшене: SASL_PLAINTEXT
  sasl_mechanism: ""
  sasl_username: ""
  sasl_password: ""
  
  # SSL
  ssl_certificate_verification: false
```

### Рекомендации для продакшена

1. **Безопасность**:
   - Использовать `SASL_PLAINTEXT` или `SASL_SSL`
   - Настроить аутентификацию (username/password)
   - Включить SSL/TLS

2. **Надежность**:
   - Увеличить `producer_retries` до 5-10
   - Настроить `producer_timeout_ms` в зависимости от сетевых условий
   - Рассмотреть отключение `consumer_auto_commit` для ручного управления офсетами

3. **Производительность**:
   - Настроить `producer_batch_size` и `producer_linger_ms` в зависимости от нагрузки
   - Увеличить `consumer_max_poll_records` при высокой нагрузке

## Обработка ошибок

### Producer

- **Ошибка сериализации**: возвращается ошибка, задача не публикуется
- **Ошибка доставки**: задача помечается как `FAILED` в БД с сообщением об ошибке
- **Таймаут контекста**: сообщение может быть потеряно, но задача уже создана в БД

### Consumer

- **Ошибка парсинга**: сообщение логируется, но не коммитится (будет обработано повторно)
- **Ошибка валидации**: аналогично ошибке парсинга
- **Ошибка обновления БД**: сообщение логируется, но не коммитится

### Восстановление после сбоев

1. **Недоступность Kafka**: Consumer автоматически переподключается с экспоненциальным backoff
2. **Потеря сообщений**: При восстановлении Consumer продолжит чтение с последнего закоммиченного офсета
3. **Дублирование сообщений**: Идемпотентность обеспечивается проверкой текущего статуса задачи перед обновлением

## Примеры использования

### Создание задачи (Producer)

```go
// В usecase/task_llm_usecase.go
taskLLM := &domain.TaskLLM{
    Task: domain.Task{
        TaskID:    taskID,
        Status:    domain.TaskStatusPending,
        CreatedAt: time.Now(),
        TraceID:   &traceID,
    },
    StoragePath: storagePath,
    StorageSize: filesize,
    Metadata:    llmMetadata,
}

cmd := domain.NewTaskLLMCommand(taskLLM)
headers := cmd.Headers.ToMap()
err := uc.producer.Send(ctx, uc.topic, taskID.String(), headers, cmd.Value)
```

### Обработка события статуса (Consumer)

```go
// В handlers/broker/task_llm.go
func (h *KafkaMessageHandler) HandleTaskStatusLLM(ctx context.Context, msg *kafka.Message) error {
    // Парсинг task_id из ключа
    taskID, err := uuid.Parse(string(msg.Key))
    
    // Парсинг заголовков
    status, workerID, traceID, err := parseTaskStatusHeaders(msg.Headers)
    
    // Парсинг тела сообщения
    var kafkaEvent domain.TaskLLMStatusEvent
    json.Unmarshal(msg.Value, &kafkaEvent.Value)
    
    // Обновление статуса
    kafkaEvent.Key.TaskID = taskID
    kafkaEvent.Headers.Status = status.ToKafkaCode()
    kafkaEvent.Headers.WorkerID = workerID
    kafkaEvent.Headers.TraceID = traceID
    
    return h.uc.TaskLLMUC.UpdateTaskStatus(ctx, kafkaEvent)
}
```

### Обновление статуса задачи

```go
// В usecase/task_llm_usecase.go
func (uc *TaskLLMUC) UpdateTaskStatus(ctx context.Context, evt domain.TaskLLMStatusEvent) error {
    taskStatus, ok := domain.TaskStatus("").FromKafkaCode(evt.Headers.Status)
    
    switch taskStatus {
    case domain.TaskStatusProcessing:
        return uc.taskRepo.UpdateWithStatus(ctx, evt.Key.TaskID, taskStatus)
    case domain.TaskStatusCompleted:
        return uc.taskRepo.UpdateWithResult(ctx, evt.Key.TaskID, taskStatus, evt.Value.Result)
    case domain.TaskStatusFailed:
        return uc.taskRepo.UpdateWithError(ctx, evt.Key.TaskID, taskStatus, evt.Value.ErrorMessage)
    }
    return nil
}
```

## Мониторинг и логирование

Все операции с Kafka логируются:

- **Producer**: логирует отправку сообщений, ошибки доставки
- **Consumer**: логирует получение сообщений, ошибки обработки, недоступность брокеров
- **Handler**: логирует маршрутизацию сообщений и обработку событий

Логи пишутся в файл `/app/logs/kafka.log` (настраивается в `config.yaml`).

## Расширение функциональности

### Добавление нового топика

1. Добавить топик в `config.yaml`:
   ```yaml
   broker:
     topics:
       - tasks_llm
       - tasks_analysis  # новый топик
   ```

2. Добавить обработчик в `KafkaMessageHandler.Handle`:
   ```go
   switch topic := *msg.TopicPartition.Topic; topic {
   case "tasks_llm":
       return h.HandleTaskStatusLLM(ctx, msg)
   case "tasks_analysis":
       return h.HandleTaskAnalysis(ctx, msg)  // новый обработчик
   }
   ```

3. Создать обработчик в `internal/handlers/broker/`

### Добавление нового статуса

1. Добавить константу в `domain.TaskStatus`
2. Обновить методы `ToKafkaCode()` и `FromKafkaCode()`
3. Добавить обработку в `UpdateTaskStatus()`


