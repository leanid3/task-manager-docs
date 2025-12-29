# Events Guide - Форматы сообщений и события

## Обзор

Task Manager использует Apache Kafka для асинхронной коммуникации с воркерами. Система работает по паттерну Command-Event:
- **Command** (Producer → Kafka): Команды на обработку задач
- **Event** (Kafka → Consumer): События об изменении статуса задач

## Топики Kafka

### tasks_llm

Основной топик для LLM задач:
- **Producer**: Task Manager публикует команды
- **Consumer**: Task Manager подписывается на события статуса
- **Worker**: LLM Worker читает команды и публикует события

## Форматы сообщений

### TaskLLMCommand (Producer → Kafka)

Команда для создания/обработки задачи LLM.

#### Структура Kafka Message

- **Topic**: `tasks_llm`
- **Key**: `{task_id}` (UUID в виде строки)
- **Headers**:
  - `status`: `"1"` (PENDING) - числовой код статуса
  - `trace_id`: `"{uuid}"` (опционально) - UUID для трейсинга
  - `worker_id`: `""` (опционально) - ID воркера
  - `version`: `""` (опционально) - версия контракта

- **Value**: JSON с данными задачи

#### Формат Value (JSON)

```json
{
  "storage_path": "documents/550e8400-e29b-41d4-a716-446655440000/document.pdf",
  "storage_size": 1024,
  "metadata": {
    "filename": "document.pdf",
    "filesize": 1024,
    "content_type": "application/octet-stream",
    "storage_path": "documents/550e8400-e29b-41d4-a716-446655440000/document.pdf"
  }
}
```

#### Поля

| Поле         | Тип   | Обязательно | Описание                                    |
|--------------|-------|-------------|---------------------------------------------|
| `storage_path` | string | Да          | Путь к файлу в хранилище (MinIO/S3)         |
| `storage_size`  | int64  | Нет         | Размер файла в байтах                       |
| `metadata`      | object | Нет         | Метаданные файла                            |

#### Metadata

| Поле           | Тип   | Описание                          |
|----------------|-------|-----------------------------------|
| `filename`     | string | Имя файла                         |
| `filesize`     | int64  | Размер файла в байтах             |
| `content_type` | string | MIME тип файла                    |
| `storage_path` | string | Полный путь в хранилище           |

#### Пример полного сообщения

```json
{
  "topic": "tasks_llm",
  "key": "550e8400-e29b-41d4-a716-446655440000",
  "headers": {
    "status": "1",
    "trace_id": "660e8400-e29b-41d4-a716-446655440001"
  },
  "value": {
    "storage_path": "documents/550e8400-e29b-41d4-a716-446655440000/document.pdf",
    "storage_size": 1024,
    "metadata": {
      "filename": "document.pdf",
      "filesize": 1024,
      "content_type": "application/octet-stream",
      "storage_path": "documents/550e8400-e29b-41d4-a716-446655440000/document.pdf"
    }
  }
}
```

### TaskLLMStatusEvent (Kafka → Consumer)

Событие об изменении статуса задачи LLM.

#### Структура Kafka Message

- **Topic**: `tasks_llm`
- **Key**: `{task_id}` (UUID в виде строки)
- **Headers**:
  - `status`: `"2"` (PROCESSING), `"3"` (COMPLETED), `"4"` (FAILED) - числовой код статуса
  - `worker_id`: `"{worker_id}"` (опционально) - ID воркера, обрабатывающего задачу
  - `trace_id`: `"{uuid}"` (опционально) - UUID для трейсинга
  - `version`: `""` (опционально) - версия контракта

- **Value**: JSON с результатом или ошибкой

#### Формат Value (JSON)

**Для статуса COMPLETED (status=3)**:
```json
{
  "result": {
    "extracted_data": "...",
    "tokens_used": 1250,
    "processing_time_ms": 3450.5
  },
  "error_message": null
}
```

**Для статуса FAILED (status=4)**:
```json
{
  "result": null,
  "error_message": "Circuit breaker OPEN - LLM service unavailable"
}
```

**Для статуса PROCESSING (status=2)**:
```json
{
  "result": null,
  "error_message": null
}
```

#### Поля

| Поле           | Тип         | Обязательно | Описание                                    |
|----------------|-------------|-------------|---------------------------------------------|
| `result`       | object/null | Нет         | Результат обработки (для COMPLETED)         |
| `error_message`| string/null | Нет         | Сообщение об ошибке (для FAILED)            |

#### Пример полного сообщения (COMPLETED)

```json
{
  "topic": "tasks_llm",
  "key": "550e8400-e29b-41d4-a716-446655440000",
  "headers": {
    "status": "3",
    "worker_id": "llm-worker-1",
    "trace_id": "660e8400-e29b-41d4-a716-446655440001"
  },
  "value": {
    "result": {
      "extracted_data": "Извлеченные данные из документа",
      "tokens_used": 1250,
      "processing_time_ms": 3450.5
    },
    "error_message": null
  }
}
```

#### Пример полного сообщения (FAILED)

```json
{
  "topic": "tasks_llm",
  "key": "550e8400-e29b-41d4-a716-446655440000",
  "headers": {
    "status": "4",
    "worker_id": "llm-worker-1",
    "trace_id": "660e8400-e29b-41d4-a716-446655440001"
  },
  "value": {
    "result": null,
    "error_message": "Failed to process file: connection timeout"
  }
}
```

## Коды статусов

### Маппинг статусов

| Статус        | Код Kafka | Описание                                    |
|---------------|-----------|---------------------------------------------|
| `PENDING`     | 1         | Задача создана, ожидает обработки          |
| `PROCESSING`  | 2         | Задача обрабатывается воркером             |
| `COMPLETED`   | 3         | Задача успешно завершена                    |
| `FAILED`      | 4         | Задача завершилась с ошибкой                |
| `CANCELLED`   | 5         | Задача отменена                             |

### Преобразование статусов

В коде используются методы:
- `TaskStatus.ToKafkaCode()` - преобразование статуса в код Kafka
- `TaskStatus.FromKafkaCode()` - преобразование кода Kafka в статус

## События и их обработка

### 1. Создание задачи (PENDING)

**Отправитель**: Task Manager (Producer)
**Получатель**: LLM Worker (Consumer)

**Поток**:
1. HTTP запрос на создание задачи
2. Файл загружается в MinIO
3. Задача создается в PostgreSQL со статусом `PENDING`
4. Команда публикуется в Kafka с `status=1`

**Обработка в Worker**:
- Worker получает команду
- Начинает обработку файла
- Отправляет событие `PROCESSING` обратно в Kafka

### 2. Начало обработки (PROCESSING)

**Отправитель**: LLM Worker (Producer)
**Получатель**: Task Manager (Consumer)

**Поток**:
1. Worker отправляет событие с `status=2`
2. Task Manager получает событие
3. Обновляет статус задачи в PostgreSQL на `PROCESSING`
4. Устанавливает `started_at` (если применимо)

**Обработка в Task Manager**:
```go
case domain.TaskStatusProcessing:
    // Проверка, что задача не уже обрабатывается
    if task.Status == domain.TaskStatusProcessing {
        return error("task already processing")
    }
    return uc.taskRepo.UpdateWithStatus(ctx, taskID, taskStatus)
```

### 3. Завершение обработки (COMPLETED)

**Отправитель**: LLM Worker (Producer)
**Получатель**: Task Manager (Consumer)

**Поток**:
1. Worker завершает обработку успешно
2. Отправляет событие с `status=3` и результатом
3. Task Manager получает событие
4. Обновляет статус задачи в PostgreSQL на `COMPLETED`
5. Сохраняет результат в поле `result`

**Обработка в Task Manager**:
```go
case domain.TaskStatusCompleted:
    // Проверка, что задача не уже завершена
    if task.Status == domain.TaskStatusCompleted {
        return error("task already completed")
    }
    return uc.taskRepo.UpdateWithResult(ctx, taskID, taskStatus, evt.Value.Result)
```

### 4. Ошибка обработки (FAILED)

**Отправитель**: LLM Worker (Producer)
**Получатель**: Task Manager (Consumer)

**Поток**:
1. Worker получает ошибку при обработке
2. Отправляет событие с `status=4` и сообщением об ошибке
3. Task Manager получает событие
4. Обновляет статус задачи в PostgreSQL на `FAILED`
5. Сохраняет сообщение об ошибке в поле `error_message`

**Обработка в Task Manager**:
```go
case domain.TaskStatusFailed:
    // Проверка, что задача не уже завершилась с ошибкой
    if task.Status == domain.TaskStatusFailed {
        return error("task already failed")
    }
    return uc.taskRepo.UpdateWithError(ctx, taskID, taskStatus, evt.Value.ErrorMessage)
```

## Идемпотентность

### Защита от дубликатов

Task Manager проверяет текущий статус задачи перед обновлением:

- **PROCESSING**: Проверяет, что задача не уже обрабатывается
- **COMPLETED**: Проверяет, что задача не уже завершена
- **FAILED**: Проверяет, что задача не уже завершилась с ошибкой

Это обеспечивает идемпотентность обработки событий даже при повторной доставке сообщений.

## Гарантии доставки

### Producer

- **acks**: `-1` (all) - ожидание подтверждения от всех реплик
- **enable.idempotence**: `true` - идемпотентность сообщений
- **retries**: `3` - количество повторных попыток

### Consumer

- **enable.auto.commit**: `true` - автоматический коммит офсетов
- **auto.commit.interval.ms**: `5000` - интервал коммита (5 секунд)

### Обработка ошибок

Если обработка события завершается ошибкой:
- Сообщение не коммитится
- Kafka доставит сообщение повторно
- Обеспечивается "at-least-once" доставка

## Примеры использования

### Публикация команды (Go)

```go
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
err := producer.Send(ctx, "tasks_llm", taskID.String(), headers, cmd.Value)
```

### Обработка события (Go)

```go
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

### Публикация события из Worker (Python пример)

```python
from confluent_kafka import Producer
import json
import uuid

def publish_status_event(task_id: str, status: int, result: dict = None, error: str = None):
    producer = Producer({'bootstrap.servers': 'kafka:29092'})
    
    headers = {
        'status': str(status),
        'worker_id': 'llm-worker-1',
        'trace_id': str(uuid.uuid4())
    }
    
    value = {
        'result': result,
        'error_message': error
    }
    
    producer.produce(
        topic='tasks_llm',
        key=task_id,
        value=json.dumps(value),
        headers=[(k, v.encode()) for k, v in headers.items()]
    )
    
    producer.flush()

# Пример использования
publish_status_event(
    task_id='550e8400-e29b-41d4-a716-446655440000',
    status=3,  # COMPLETED
    result={'extracted_data': '...', 'tokens_used': 1250}
)
```

## Расширение форматов

### Добавление нового поля

1. Обновить структуру в `internal/entity/domain/broker_llm.go`
2. Обновить сериализацию/десериализацию
3. Обновить обработчики в Worker и Task Manager
4. Обновить документацию

### Добавление нового статуса

1. Добавить константу в `domain.TaskStatus`
2. Обновить `ToKafkaCode()` и `FromKafkaCode()`
3. Добавить обработку в `UpdateTaskStatus()`
4. Обновить Worker для поддержки нового статуса

## Версионирование контрактов

Для обратной совместимости можно использовать поле `version` в headers:

```json
{
  "headers": {
    "status": "3",
    "version": "1.0"
  }
}
```

При изменении формата сообщений:
1. Увеличить версию контракта
2. Поддержать старые версии в обработчиках
3. Постепенно мигрировать на новую версию

