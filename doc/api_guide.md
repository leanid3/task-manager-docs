# API Guide - Task Manager

## Обзор

Task Manager предоставляет REST API для создания и мониторинга задач обработки документов. API использует стандарт OpenAPI 3.0 и доступен через Swagger UI.

### Быстрый старт

1. **Создание задачи**: POST `/api/v1/tasks/{filename}` с бинарным содержимым файла
2. **Получение статуса**: GET `/api/v1/tasks/{task_id}` для проверки статуса задачи
3. **Обработка результатов**: Поле `task_results` содержит JSON строку, которую необходимо распарсить

**Важно**: 
- В ответах **НЕТ** поля `success: true/false` - используйте HTTP статус-коды
- `task_results` - это JSON строка, а не объект (требует парсинга)
- Каждый ответ содержит `request_id` и `timestamp` для трейсинга

## Базовый URL

```
http://localhost:8080/api/v1
```

## Swagger UI

Интерактивная документация доступна по адресу:
```
http://localhost:8080/swagger/index.html
```

## Формат ответов

### Успешный ответ

Все успешные ответы имеют следующую структуру:

```json
{
  "data": {
    // Данные ответа зависят от endpoint
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**Важно**: В ответах **НЕТ** поля `success: true/false`. Успешность операции определяется HTTP статус-кодом:
- `200 OK` - успешный запрос
- `202 Accepted` - запрос принят к обработке
- `400 Bad Request` - ошибка валидации
- `404 Not Found` - ресурс не найден
- `500 Internal Server Error` - внутренняя ошибка сервера

### Ответ с ошибкой

Все ответы с ошибками имеют следующую структуру:

```json
{
  "error": "Описание ошибки",
  "code": "ERROR_CODE",
  "details": {
    // Дополнительные детали ошибки (опционально)
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

### Поля ответа

- `data` (object, только в успешных ответах) - данные ответа
- `error` (string, только в ответах с ошибками) - описание ошибки
- `code` (string, только в ответах с ошибками) - код ошибки
- `details` (object, опционально) - дополнительные детали
- `request_id` (string, UUID) - уникальный идентификатор запроса для трейсинга
- `timestamp` (string, ISO8601) - время формирования ответа

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
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "PENDING"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**400 Bad Request** (ошибка валидации):
```json
{
  "error": "неверный формат запроса",
  "code": "VALIDATION_FAILED",
  "details": {},
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**500 Internal Server Error** (ошибка сервера):
```json
{
  "error": "не удалось загрузить файл в хранилище.",
  "code": "STORAGE_ERROR",
  "details": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "storage_path": "documents/550e8400-e29b-41d4-a716-446655440000/document.pdf"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
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

**Важно**: 
- Поле `task_results` присутствует только когда `status` = `COMPLETED` и содержит JSON строку (не объект), которую необходимо распарсить
- Поле `error_message` присутствует только когда `status` = `FAILED`

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
  "data": {
    "task_id": "3343fcd8-05a7-43e6-ac5d-c08587def150",
    "status": "COMPLETED",
    "task_results": "{\"Статус\": \"Успешно\", \"ДатаДокумента\": \"20250630\", \"НомерДокумента\": \"УПД123710\", \"ВидДокумента\": \"Универсальный передаточный документ\", \"ИдентификаторВидаДокумента\": \"Первичные\", \"ОтправительДокумента\": \"ООО \\\"Софтехно\\\"\", \"ОтправительКонверта\": \"ООО \\\"Софтехно\\\"\", \"АдресОтправителя\": \"Москва г, вн.тер.г. Муниципальный округ Тверской, Достоевского ул, дом №1/21, строение 1, этаж 1, пом. 1\", \"АдресОтправителяКонверта\": \"Москва г, Достогвского ул, дом №1/21, строение 1\", \"ИННОтправителя\": \"7731655492\", \"ГородОтправителя\": \"Москва\", \"ОрганизацияПолучатель\": \"ООО \\\"Управляющая компания \\\"ТрансТехСервис\\\"\", \"ИННПолучателя\": \"1650131524\", \"КПППолучателя\": \"165001001\", \"VIN\": [], \"Содержание\": \"Предоставлено вознаграждение за право пользования базой данных \\\"Информационно-технологическое сопровождение партнеров\\\".\", \"НомерКонверта\": \"12933811581534\"}"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**Важные примечания**: 
- `task_results` - это **JSON строка** (не объект), содержащая результаты обработки в формате JSON. Для использования необходимо распарсить эту строку с помощью `JSON.parse()` (JavaScript) или `json.loads()` (Python).
- `task_results` присутствует **только** когда `status` = `COMPLETED`, в остальных случаях поле отсутствует или равно `null`
- `error_message` присутствует **только** когда `status` = `FAILED`, в остальных случаях поле отсутствует или равно `null`
- Структура данных внутри `task_results` зависит от типа обрабатываемого документа и определяется воркером LLM

**Пример задачи в статусе FAILED**:
```json
{
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "FAILED",
    "error_message": "Ошибка обработки документа: неверный формат файла"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**400 Bad Request** (неверный формат ID):
```json
{
  "error": "неверный формат ID задачи",
  "code": "INVALID_TASK_ID",
  "details": {},
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**404 Not Found** (задача не найдена):
```json
{
  "error": "не удалось получить задачу из базы данных",
  "code": "DATABASE_ERROR",
  "details": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
}
```

**500 Internal Server Error** (ошибка сервера):
```json
{
  "error": "не удалось получить задачу из базы данных",
  "code": "DATABASE_ERROR",
  "details": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
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
  "data": {
    "status": "live"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
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
  "data": {
    "status": "ready"
  },
  "request_id": "98c33a7b-b779-4902-98de-37c0bc14f0d8",
  "timestamp": "2025-12-29T11:30:33Z"
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

Каждый запрос получает уникальный `request_id` (UUID), который:
- Генерируется автоматически в middleware
- Используется для трейсинга запросов
- Возвращается в каждом ответе (успешном и с ошибкой)
- Логируется во всех операциях
- Может быть использован для поиска логов конкретного запроса

## Timestamp

Каждый ответ содержит поле `timestamp` в формате ISO8601 (RFC3339):
- Формат: `YYYY-MM-DDTHH:MM:SSZ`
- Пример: `2025-12-29T11:30:33Z`
- Время указывается в UTC

## Работа с task_results

Поле `task_results` в ответе GET `/api/v1/tasks/{task_id}` содержит **JSON строку**, а не объект. Это означает, что для использования результатов необходимо:

1. Получить строку из ответа
2. Распарсить её как JSON

### Примеры парсинга

**JavaScript/TypeScript**:
```javascript
const response = await fetch(`http://localhost:8080/api/v1/tasks/${taskId}`);
const data = await response.json();

if (data.data.status === 'COMPLETED' && data.data.task_results) {
  // task_results - это строка, нужно распарсить
  const results = JSON.parse(data.data.task_results);
  console.log(results);
}
```

**Python**:
```python
import json
import requests

response = requests.get(f"http://localhost:8080/api/v1/tasks/{task_id}")
data = response.json()

if data["data"]["status"] == "COMPLETED" and data["data"].get("task_results"):
    # task_results - это строка, нужно распарсить
    results = json.loads(data["data"]["task_results"])
    print(results)
```

**Go**:
```go
import (
    "encoding/json"
    "net/http"
)

type TaskResponse struct {
    Data struct {
        TaskID      string `json:"task_id"`
        Status      string `json:"status"`
        TaskResults string `json:"task_results,omitempty"`
    } `json:"data"`
}

var taskData map[string]interface{}
if resp.Data.Status == "COMPLETED" && resp.Data.TaskResults != "" {
    json.Unmarshal([]byte(resp.Data.TaskResults), &taskData)
}
```

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
  RESPONSE=$(curl -s "http://localhost:8080/api/v1/tasks/$TASK_ID")
  STATUS=$(echo $RESPONSE | jq -r '.data.status')
  echo "Status: $STATUS"
  
  if [ "$STATUS" = "COMPLETED" ]; then
    echo "Task completed!"
    # task_results - это JSON строка, нужно распарсить
    echo $RESPONSE | jq -r '.data.task_results' | jq .
    break
  elif [ "$STATUS" = "FAILED" ]; then
    echo "Task failed!"
    echo $RESPONSE | jq -r '.data.error_message'
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
import json

with open("document.pdf", "rb") as f:
    file_content = f.read()

task_id = create_task("document.pdf", file_content)
print(f"Task created: {task_id}")

result = wait_for_completion(task_id)

# task_results - это JSON строка, нужно распарсить
if result.get("task_results"):
    task_results = json.loads(result["task_results"])
    print(f"Result: {json.dumps(task_results, ensure_ascii=False, indent=2)}")
else:
    print(f"Error: {result.get('error_message')}")
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

