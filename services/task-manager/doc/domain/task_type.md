
# продумываю сущности для использования в проекте. Я уже сделал producer и consumer для kafka, в данный момент мне нужно сделать taskMessageConsumerLLM struct - структура сообщения которое будет отправлено в топик, она должна представлять из себя базовое сообещение с key, header, value - но каждый из этих 3ей опций содержит свою структуру сообщения, именно это мне нужно как то связать, чтобы корректно использовать kafka.Message, как базовый класс. я использую пакет confluentinc, от туда в данный  момент для handler я использую стандартный из пакета type Message struct {

    TopicPartition TopicPartition
    Value          []byte
    Key            []byte
    Timestamp      time.Time
    TimestampType  TimestampType
    Opaque         interface{}
    Headers        []Header
    LeaderEpoch    *int32 // Deprecated: LeaderEpoch or nil if not available. Use m.TopicPartition.LeaderEpoch instead.
    составим универсальный вариант для добовления новых типов задач, например что для состовления структуры TaskMessageConsumerAnylyzes мне бы пришлось минимально изменять структуру

Отличный подход! Создадим **универсальную структуру** для Kafka сообщений с поддержкой полиморфизма по типам задач. Используем `TaskType` как дискриминатор и вложенные структуры.

## Универсальная структура TaskMessage

### 1. Базовые структуры домена

```go
// internal/entity/domain/task_types.go
package domain

import (
    "encoding/json"
    "time"
    
    "github.com/google/uuid"
)

type TaskType string

const (
    TaskTypeParsing    TaskType = "parsing"
    TaskTypeAlgorithms TaskType = "algorithms"
    TaskTypeLLM        TaskType = "llm"
    TaskTypeAnalyze    TaskType = "analyze"
)

// TaskCommand базовая команда для всех типов задач
type TaskCommand struct {
    TaskID       uuid.UUID              `json:"task_id"`
    TaskType     TaskType               `json:"task_type"`
    TraceID      string                 `json:"trace_id"`
    RequestID    string                 `json:"request_id"`
    CreatedAt    time.Time              `json:"created_at"`
    Priority     int                    `json:"priority"`     // 1-10
    TimeoutSec   int                    `json:"timeout_sec"`  // TTL
    RetryCount   int                    `json:"retry_count"`
    MaxRetries   int                    `json:"max_retries"`
    
    // Полиморфные данные по типу задачи
    Payload      json.RawMessage        `json:"payload"`      // JSON с специфичными данными
}

// TaskEvent базовое событие от worker'ов
type TaskEvent struct {
    TaskID       uuid.UUID              `json:"task_id"`
    TaskType     TaskType               `json:"task_type"`
    WorkerID     string                 `json:"worker_id"`
    EventType    TaskEventType          `json:"event_type"`
    TraceID      string                 `json:"trace_id"`
    Timestamp    time.Time              `json:"timestamp"`
    
    // Полиморфные данные
    Payload      json.RawMessage        `json:"payload"`
}

type TaskEventType string

const (
    TaskEventProcessing TaskEventType = "processing"
    TaskEventCompleted  TaskEventType = "completed"
    TaskEventFailed     TaskEventType = "failed"
)
```


### 2. Универсальная Kafka структура

```go
// internal/entity/kafka/task_message.go
package kafka

import (
    "encoding/json"
    
    "github.com/confluentinc/confluent-kafka-go/kafka"
    "app/internal/entity/domain"
)

// TaskMessage универсальное сообщение Kafka
type TaskMessage struct {
    // Kafka поля
    Topic     string                 `json:"-"`
    Key       []byte                 `json:"-"`
    Value     []byte                 `json:"-"`
    Headers   []kafka.Header         `json:"-"`
    
    // Разобранные данные
    Command   *domain.TaskCommand    `json:"-"`
    Event     *domain.TaskEvent      `json:"-"`
    
    // Метаданные
    MessageID string                 `json:"-"`
    Offset    int64                  `json:"-"`
}

// NewTaskMessageProducer создаёт сообщение для producer
func NewTaskMessageProducer(cmd *domain.TaskCommand, topic string) (*TaskMessage, error) {
    value, err := json.Marshal(cmd)
    if err != nil {
        return nil, err
    }
    
    key := []byte(cmd.TaskID.String())
    
    headers := []kafka.Header{
        {"message_type", []byte("task_command")},
        {"task_type", []byte(string(cmd.TaskType))},
        {"trace_id", []byte(cmd.TraceID)},
    }
    
    return &TaskMessage{
        Topic:   topic,
        Key:     key,
        Value:   value,
        Headers: headers,
        Command: cmd,
    }, nil
}

// NewTaskMessageEvent создаёт событие от worker'а
func NewTaskMessageEvent(event *domain.TaskEvent, topic string) (*TaskMessage, error) {
    value, err := json.Marshal(event)
    if err != nil {
        return nil, err
    }
    
    key := []byte(event.TaskID.String())
    
    headers := []kafka.Header{
        {"message_type", []byte("task_event")},
        {"event_type", []byte(string(event.EventType))},
        {"task_type", []byte(string(event.TaskType))},
        {"worker_id", []byte(event.WorkerID)},
    }
    
    return &TaskMessage{
        Topic:   topic,
        Key:     key,
        Value:   value,
        Headers: headers,
        Event:   event,
    }, nil
}

// ToKafkaMessage конвертирует в confluent-kafka Message
func (tm *TaskMessage) ToKafkaMessage(partition int32) *kafka.Message {
    msg := &kafka.Message{
        TopicPartition: kafka.TopicPartition{
            Topic:     &tm.Topic,
            Partition: partition,
        },
        Key:     tm.Key,
        Value:   tm.Value,
        Headers: tm.Headers,
    }
    return msg
}

// ParseFromKafkaMessage парсит из confluent-kafka Message
func ParseFromKafkaMessage(msg *kafka.Message) (*TaskMessage, error) {
    tm := &TaskMessage{
        Key:     msg.Key,
        Value:   msg.Value,
        Headers: msg.Headers,
        Offset:  msg.TopicPartition.Offset,
    }
    
    // Парсим message_type из headers
    var messageType string
    for _, header := range msg.Headers {
        if string(header.Key) == "message_type" {
            messageType = string(header.Value)
            break
        }
    }
    
    switch messageType {
    case "task_command":
        var cmd domain.TaskCommand
        if err := json.Unmarshal(msg.Value, &cmd); err != nil {
            return nil, err
        }
        tm.Command = &cmd
        
    case "task_event":
        var event domain.TaskEvent
        if err := json.Unmarshal(msg.Value, &event); err != nil {
            return nil, err
        }
        tm.Event = &event
        
    default:
        return nil, fmt.Errorf("unknown message_type: %s", messageType)
    }
    
    return tm, nil
}
```


### 3. Специфичные структуры для Payload

```go
// internal/entity/domain/task_payloads.go
package domain

// ParsingPayload специфичные данные для parsing
type ParsingPayload struct {
    StoragePath     string `json:"storage_path"`     // s3://bucket/path
    StorageSize     int64  `json:"storage_size"`
    OriginalFilename string `json:"original_filename"`
    ContentType     string `json:"content_type"`
}

// ParsingResult результат parsing
type ParsingResult struct {
    Text            string   `json:"text"`
    Pages           int      `json:"pages"`
    Confidence      float64  `json:"confidence"`
    ProcessingTimeMS int     `json:"processing_time_ms"`
}

// AlgorithmsPayload для algorithms worker'а
type AlgorithmsPayload struct {
    InputText      string `json:"input_text"`
    Algorithm      string `json:"algorithm"` // "ner", "classification", etc.
    Language       string `json:"language"`
}

// AlgorithmsResult результат
type AlgorithmsResult struct {
    Entities       []string `json:"entities"`
    Classification string  `json:"classification"`
    Score          float64  `json:"score"`
}

// LLMAnalyzePayload для LLM analyze
type LLMAnalyzePayload struct {
    InputText      string  `json:"input_text"`
    PromptTemplate string  `json:"prompt_template"`
    Model          string  `json:"model"`
    MaxTokens      int     `json:"max_tokens"`
}

// LLMAnalyzeResult результат
type LLMAnalyzeResult struct {
    Summary        string   `json:"summary"`
    KeyPoints      []string `json:"key_points"`
    TokensUsed     int      `json:"tokens_used"`
}
```


### 4. Универсальные Producer/Consumer

```go
// internal/adapter/kafka/task_producer.go
type TaskProducer struct {
    producer *kafka.Producer
    logger   logger.Interface
}

func NewTaskProducer(producer *kafka.Producer, logger logger.Interface) *TaskProducer {
    return &TaskProducer{producer: producer, logger: logger}
}

// PublishTaskCommand отправляет команду worker'у
func (p *TaskProducer) PublishTaskCommand(ctx context.Context, cmd *domain.TaskCommand) error {
    // Создаём специфичный payload
    var payload json.RawMessage
    switch cmd.TaskType {
    case domain.TaskTypeParsing:
        parsingPayload := &domain.ParsingPayload{
            // заполняем из cmd или из БД
        }
        data, _ := json.Marshal(parsingPayload)
        payload = data
        
    case domain.TaskTypeAnalyze:
        analyzePayload := &domain.LLMAnalyzePayload{
            // ...
        }
        data, _ := json.Marshal(analyzePayload)
        payload = data
    }
    
    cmd.Payload = payload
    
    // Создаём сообщение Kafka
    msg, err := NewTaskMessageProducer(cmd, "tasks.parsing") // или по типу
    if err != nil {
        return err
    }
    
    // Отправляем
    return p.producer.Produce(msg.ToKafkaMessage(kafka.PartitionAny), ctx)
}

// PublishTaskEvent публикует событие от worker'а
func (p *TaskProducer) PublishTaskEvent(ctx context.Context, event *domain.TaskEvent) error {
    // Аналогично: payload в зависимости от TaskType
    msg, err := NewTaskMessageEvent(event, "events.tasks")
    if err != nil {
        return err
    }
    return p.producer.Produce(msg.ToKafkaMessage(kafka.PartitionAny), ctx)
}
```


### 5. Consumer для LLM Analyze

```go
// internal/worker/llm_analyze/consumer.go
type TaskMessageConsumerLLMAnalyze struct {
    consumer kafka.Consumer
    uc       usecase.LLMAnalyzeUseCase
    logger   logger.Logger
}

func (c *TaskMessageConsumerLLMAnalyze) Consume(ctx context.Context, topic string, groupID string) error {
    return c.consumer.SubscribeTopics([]string{topic}, nil)
    
    for {
        msg, err := c.consumer.ReadMessage(-1)
        if err != nil {
            c.logger.Error("consumer error", "error", err)
            continue
        }
        
        // Парсим универсальное сообщение
        taskMsg, err := ParseFromKafkaMessage(msg)
        if err != nil {
            c.logger.Error("failed to parse message", "error", err)
            continue
        }
        
        // Проверяем тип
        if taskMsg.Command == nil || taskMsg.Command.TaskType != domain.TaskTypeAnalyze {
            c.logger.Warn("skipping non-analyze task", "task_type", taskMsg.Command.TaskType)
            continue
        }
        
        // Парсим специфичный payload
        var payload domain.LLMAnalyzePayload
        if err := json.Unmarshal(taskMsg.Command.Payload, &payload); err != nil {
            c.logger.Error("failed to parse analyze payload", "error", err)
            continue
        }
        
        // Обрабатываем
        c.uc.ProcessTask(ctx, taskMsg.Command.TaskID, payload)
    }
}
```


## Пример создания Analyze задачи

```go
// API-gateway создаёт задачу analyze
func createAnalyzeTask() {
    cmd := &domain.TaskCommand{
        TaskID:   uuid.New(),
        TaskType: domain.TaskTypeAnalyze,
        TraceID:  "trace-123",
        Payload: json.RawMessage(`{
            "input_text": "текст для анализа",
            "prompt_template": "проанализируй документ",
            "model": "gpt-4",
            "max_tokens": 1000
        }`),
    }
    
    producer.PublishTaskCommand(ctx, cmd)  // автоматически попадёт в правильный топик
}
```


## Преимущества архитектуры

1. **Добавление нового типа задачи** = просто:

```go
// 1. Добавляем TaskTypeAnalyze
// 2. Создаём AnalyzePayload + AnalyzeResult структуры
// 3. В producer добавляем case TaskTypeAnalyze
// 4. Создаём consumer для нового типа
```

2. **Автоматическая маршрутизация**: `TaskType` → правильный топик/worker
3. **Type safety**: compile-time проверка payload'ов
4. **Расширяемость**: новые поля в payload не ломают существующие consumer'ы
5. **Headers**: метаданные доступны без парсинга JSON

Готово! Универсальная структура для всех типов задач 🚀

