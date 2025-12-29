# API Guide - Task Manager

## Обзор

Task Manager предоставляет REST API для создания и мониторинга задач обработки документов. API использует стандарт OpenAPI 3.0 и доступен через Swagger UI.

## Базовый URL

```
http://localhost:8080/api/v1
```

## Swagger UI

Интерактивная документация доступна по адресу:
```
http://localhost:8080/swagger/index.html
```

## Endpoints

### 1. Создание задачи

**POST** `/api/v1/tasks/{filename}`

Создает новую задачу обработки файла.

#### Параметры

**Path Parameters**:
- `filename` (string, required) - Имя файла с расширением. Пример: `document.pdf`

**Headers**:
- `Content-Length` (int, required) - Размер файла в байтах
- `Content-Type` (string, optional) - Тип содержимого (по умолчанию `application/octet-stream`)

**Body**:
- Бинарное содержимое файла (application/octet-stream)

#### Пример запроса

```bash
curl -X POST "http://localhost:8080/api/v1/tasks/document.pdf" \
  -H "Content-Length: 1024" \
  --data-binary "@document.pdf"
```

#### Ответы

**202 Accepted** (успех):
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "PENDING"
  },
  "request_id": "req-123456"
}
```

**400 Bad Request** (ошибка валидации):
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "filename cannot be empty",
    "details": {}
  },
  "request_id": "req-123456"
}
```

**500 Internal Server Error** (ошибка сервера):
```json
{
  "success": false,
  "error": {
    "code": "STORAGE_ERROR",
    "message": "не удалось загрузить файл в хранилище.",
    "details": {
      "task_id": "550e8400-e29b-41d4-a716-446655440000",
      "storage_path": "documents/550e8400-e29b-41d4-a716-446655440000/document.pdf"
    }
  },
  "request_id": "req-123456"
}
```

#### Жизненный цикл после создания

1. Файл загружается в MinIO/S3
2. Задача создается в PostgreSQL со статусом `PENDING`
3. Команда публикуется в Kafka топик `tasks_llm`
4. Если любая операция завершается ошибкой, выполняется компенсация (удаление файла)

### 2. Получение задачи

**GET** `/api/v1/tasks/{task_id}`

Получает информацию о задаче по её ID.

#### Параметры

**Path Parameters**:
- `task_id` (UUID, required) - Идентификатор задачи

#### Пример запроса

```bash
curl -X GET "http://localhost:8080/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000"
```

#### Ответы

**200 OK** (задача найдена):
```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "COMPLETED",
    "task_results": {
      "extracted_data": "...",
      "tokens_used": 1250
    },
    "error_message": null
  },
  "request_id": "req-123456"
}
```

**400 Bad Request** (неверный формат ID):
```json
{
  "success": false,
  "error": {
    "code": "INVALID_TASK_ID",
    "message": "неверный формат ID задачи",
    "details": {}
  },
  "request_id": "req-123456"
}
```

**404 Not Found** (задача не найдена):
```json
{
  "success": false,
  "error": {
    "code": "DATABASE_ERROR",
    "message": "не удалось получить задачу из базы данных",
    "details": {
      "task_id": "550e8400-e29b-41d4-a716-446655440000"
    }
  },
  "request_id": "req-123456"
}
```

**500 Internal Server Error** (ошибка сервера):
```json
{
  "success": false,
  "error": {
    "code": "DATABASE_ERROR",
    "message": "не удалось получить задачу из базы данных",
    "details": {
      "task_id": "550e8400-e29b-41d4-a716-446655440000"
    }
  },
  "request_id": "req-123456"
}
```

### 3. Health Check

**GET** `/health/live`

Проверка жизнеспособности сервиса (liveness probe).

#### Пример запроса

```bash
curl -X GET "http://localhost:8080/health/live"
```

#### Ответ

**200 OK**:
```json
{
  "success": true,
  "data": {
    "status": "live"
  },
  "request_id": "req-123456"
}
```

### 4. Readiness Check

**GET** `/health/ready`

Проверка готовности сервиса (readiness probe).

#### Пример запроса

```bash
curl -X GET "http://localhost:8080/health/ready"
```

#### Ответ

**200 OK**:
```json
{
  "success": true,
  "data": {
    "status": "ready"
  },
  "request_id": "req-123456"
}
```

## Статусы задач

### TaskStatus

| Статус      | Описание                                    |
|-------------|---------------------------------------------|
| `PENDING`   | Задача создана, ожидает обработки          |
| `PROCESSING`| Задача обрабатывается воркером             |
| `COMPLETED` | Задача успешно завершена                    |
| `FAILED`    | Задача завершилась с ошибкой                |
| `CANCELLED` | Задача отменена                             |

### Переходы статусов

```
PENDING → PROCESSING → COMPLETED
                    ↓
                  FAILED
```

## Коды ошибок

### Общие коды

| Код                    | HTTP Status | Описание                          |
|------------------------|-------------|-----------------------------------|
| `VALIDATION_FAILED`    | 400         | Ошибка валидации входных данных   |
| `INVALID_REQUEST`      | 400         | Неверный формат запроса           |
| `INVALID_TASK_ID`      | 400         | Неверный формат ID задачи         |
| `DATABASE_ERROR`       | 500         | Ошибка работы с базой данных      |
| `STORAGE_ERROR`        | 500         | Ошибка работы с хранилищем        |
| `KAFKA_ERROR`          | 500         | Ошибка работы с Kafka             |
| `INTERNAL_ERROR`       | 500         | Внутренняя ошибка сервера         |

### Специфичные коды

| Код                        | Описание                                    |
|----------------------------|---------------------------------------------|
| `TASK_ALREADY_PROCESSING`  | Задача уже обрабатывается                   |
| `TASK_ALREADY_COMPLETED`   | Задача уже завершена                        |
| `TASK_ALREADY_FAILED`      | Задача уже завершилась с ошибкой           |
| `INVALID_MESSAGE_FORMAT`   | Неверный формат сообщения из Kafka          |

## Request ID

Каждый запрос получает уникальный `request_id`, который:
- Генерируется автоматически в middleware
- Используется для трейсинга запросов
- Возвращается в ответе
- Логируется во всех операциях

## Примеры использования

### Создание задачи и ожидание результата

```bash
#!/bin/bash

# 1. Создать задачу
RESPONSE=$(curl -s -X POST "http://localhost:8080/api/v1/tasks/document.pdf" \
  -H "Content-Length: $(stat -f%z document.pdf)" \
  --data-binary "@document.pdf")

TASK_ID=$(echo $RESPONSE | jq -r '.data.task_id')
echo "Task created: $TASK_ID"

# 2. Ожидание завершения (polling)
while true; do
  STATUS=$(curl -s "http://localhost:8080/api/v1/tasks/$TASK_ID" | jq -r '.data.status')
  echo "Status: $STATUS"
  
  if [ "$STATUS" = "COMPLETED" ]; then
    echo "Task completed!"
    curl -s "http://localhost:8080/api/v1/tasks/$TASK_ID" | jq '.data.task_results'
    break
  elif [ "$STATUS" = "FAILED" ]; then
    echo "Task failed!"
    curl -s "http://localhost:8080/api/v1/tasks/$TASK_ID" | jq '.data.error_message'
    break
  fi
  
  sleep 2
done
```

### Использование с Python

```python
import requests
import time
import uuid

def create_task(filename: str, file_content: bytes) -> uuid.UUID:
    """Создает задачу обработки файла"""
    url = f"http://localhost:8080/api/v1/tasks/{filename}"
    headers = {"Content-Length": str(len(file_content))}
    
    response = requests.post(url, data=file_content, headers=headers)
    response.raise_for_status()
    
    data = response.json()
    return uuid.UUID(data["data"]["task_id"])

def get_task(task_id: uuid.UUID) -> dict:
    """Получает информацию о задаче"""
    url = f"http://localhost:8080/api/v1/tasks/{task_id}"
    response = requests.get(url)
    response.raise_for_status()
    return response.json()["data"]

def wait_for_completion(task_id: uuid.UUID, timeout: int = 300) -> dict:
    """Ожидает завершения задачи"""
    start_time = time.time()
    
    while time.time() - start_time < timeout:
        task = get_task(task_id)
        status = task["status"]
        
        if status == "COMPLETED":
            return task
        elif status == "FAILED":
            raise Exception(f"Task failed: {task.get('error_message')}")
        
        time.sleep(2)
    
    raise TimeoutError("Task did not complete in time")

# Использование
with open("document.pdf", "rb") as f:
    file_content = f.read()

task_id = create_task("document.pdf", file_content)
print(f"Task created: {task_id}")

result = wait_for_completion(task_id)
print(f"Result: {result['task_results']}")
```

## Rate Limiting

В текущей версии rate limiting не реализован. Рекомендуется:
- Ограничивать количество запросов на клиентской стороне
- Использовать очередь для batch обработки
- В production добавить rate limiting middleware

## Версионирование API

Текущая версия: **v1**

API использует версионирование через путь: `/api/v1/`

При изменении API создается новая версия: `/api/v2/`

## Безопасность

### Текущее состояние

- Нет аутентификации
- Нет авторизации
- Нет rate limiting
- Нет валидации размера файла (ограничено только конфигурацией)

### Рекомендации для production

- [ ] JWT токены
- [ ] API ключи
- [ ] Rate limiting
- [ ] Валидация размера файла
- [ ] Валидация типа файла
- [ ] SSL/TLS
- [ ] CORS настройки

## Ограничения

### Размер файла

Ограничения на размер файла не установлены на уровне API, но могут быть ограничены:
- Конфигурацией HTTP сервера
- Настройками MinIO
- Доступной памятью

### Типы файлов

В текущей версии принимаются любые типы файлов. Рекомендуется добавить валидацию на уровне бизнес-логики.

### Таймауты

- HTTP запрос: настраивается через `server.timeout` в конфигурации
- Обработка задачи: зависит от воркера (LLM Worker)

