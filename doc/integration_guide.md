# Integration Guide - Руководство по интеграции

## Обзор

Это руководство описывает, как интегрировать внешние сервисы с Task Manager. Task Manager работает как API Gateway и координатор задач, взаимодействуя с воркерами через Kafka.

## Архитектура интеграции

```
┌─────────────────┐
│  Client App     │
│  (External)     │
└────────┬────────┘
         │ HTTP REST API
         ↓
┌─────────────────────────────────────┐
│      Task Manager                  │
│  - API Gateway                     │
│  - Task Coordinator                │
└────────┬───────────────────────────┘
         │ Kafka
         ↓
┌─────────────────┐
│  LLM Worker     │
│  (External)     │
└─────────────────┘
```

## Интеграция как клиент (HTTP API)

### Шаг 1: Подключение к API

Task Manager предоставляет REST API на порту 8080 (по умолчанию).

**Базовый URL**:
```
http://localhost:8080/api/v1
```

### Шаг 2: Создание задачи

Отправьте файл для обработки:

```bash
curl -X POST "http://localhost:8080/api/v1/tasks/document.pdf" \
  -H "Content-Length: 1024" \
  --data-binary "@document.pdf"
```

**Ответ**:
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

### Шаг 3: Мониторинг статуса

Периодически опрашивайте статус задачи:

```bash
curl "http://localhost:8080/api/v1/tasks/550e8400-e29b-41d4-a716-446655440000"
```

**Ответы**:
- `PENDING` - задача создана, ожидает обработки
- `PROCESSING` - задача обрабатывается
- `COMPLETED` - задача завершена, результат доступен
- `FAILED` - задача завершилась с ошибкой

### Шаг 4: Получение результата

Когда статус `COMPLETED`, получите результат:

```json
{
  "success": true,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "COMPLETED",
    "task_results": {
      "extracted_data": "...",
      "tokens_used": 1250
    }
  }
}
```

### Пример полной интеграции (Python)

```python
import requests
import time
import uuid
from typing import Optional, Dict, Any

class TaskManagerClient:
    def __init__(self, base_url: str = "http://localhost:8080/api/v1"):
        self.base_url = base_url
    
    def create_task(self, filename: str, file_content: bytes) -> uuid.UUID:
        """Создает задачу обработки файла"""
        url = f"{self.base_url}/tasks/{filename}"
        headers = {"Content-Length": str(len(file_content))}
        
        response = requests.post(url, data=file_content, headers=headers)
        response.raise_for_status()
        
        data = response.json()
        return uuid.UUID(data["data"]["task_id"])
    
    def get_task(self, task_id: uuid.UUID) -> Dict[str, Any]:
        """Получает информацию о задаче"""
        url = f"{self.base_url}/tasks/{task_id}"
        response = requests.get(url)
        response.raise_for_status()
        return response.json()["data"]
    
    def wait_for_completion(
        self, 
        task_id: uuid.UUID, 
        timeout: int = 300,
        poll_interval: int = 2
    ) -> Dict[str, Any]:
        """Ожидает завершения задачи"""
        start_time = time.time()
        
        while time.time() - start_time < timeout:
            task = self.get_task(task_id)
            status = task["status"]
            
            if status == "COMPLETED":
                return task
            elif status == "FAILED":
                error_msg = task.get("error_message", "Unknown error")
                raise Exception(f"Task failed: {error_msg}")
            
            time.sleep(poll_interval)
        
        raise TimeoutError("Task did not complete in time")
    
    def process_file(self, filename: str, file_content: bytes) -> Dict[str, Any]:
        """Полный цикл обработки файла"""
        # Создать задачу
        task_id = self.create_task(filename, file_content)
        print(f"Task created: {task_id}")
        
        # Ожидать завершения
        result = self.wait_for_completion(task_id)
        print(f"Task completed: {result['task_results']}")
        
        return result

# Использование
client = TaskManagerClient()

with open("document.pdf", "rb") as f:
    file_content = f.read()

result = client.process_file("document.pdf", file_content)
print(result)
```

## Интеграция как Worker (Kafka)

### Шаг 1: Подключение к Kafka

Подключитесь к Kafka брокеру:

```python
from confluent_kafka import Consumer, Producer

consumer = Consumer({
    'bootstrap.servers': 'kafka:29092',
    'group.id': 'llm-worker-group',
    'auto.offset.reset': 'earliest'
})

producer = Producer({
    'bootstrap.servers': 'kafka:29092'
})
```

### Шаг 2: Подписка на топик

Подпишитесь на топик `tasks_llm`:

```python
consumer.subscribe(['tasks_llm'])
```

### Шаг 3: Чтение команд

Читайте команды из Kafka:

```python
import json
import uuid

while True:
    msg = consumer.poll(timeout=1.0)
    
    if msg is None:
        continue
    
    if msg.error():
        print(f"Consumer error: {msg.error()}")
        continue
    
    # Парсинг сообщения
    task_id = msg.key().decode('utf-8')
    headers = {h[0]: h[1].decode('utf-8') for h in msg.headers()}
    value = json.loads(msg.value().decode('utf-8'))
    
    status = int(headers.get('status', '1'))
    
    # Обрабатываем только команды (status=1, PENDING)
    if status == 1:
        process_task(task_id, value, headers)
```

### Шаг 4: Обработка задачи

Обработайте задачу и отправьте статусы:

```python
def process_task(task_id: str, command: dict, headers: dict):
    """Обрабатывает задачу"""
    storage_path = command['storage_path']
    trace_id = headers.get('trace_id')
    
    try:
        # 1. Отправить статус PROCESSING
        publish_status(task_id, status=2, trace_id=trace_id)
        
        # 2. Загрузить файл из MinIO
        file_content = load_from_minio(storage_path)
        
        # 3. Обработать файл (LLM, анализ и т.д.)
        result = process_with_llm(file_content)
        
        # 4. Отправить статус COMPLETED с результатом
        publish_status(
            task_id, 
            status=3, 
            result=result,
            trace_id=trace_id
        )
        
    except Exception as e:
        # 5. Отправить статус FAILED с ошибкой
        publish_status(
            task_id,
            status=4,
            error=str(e),
            trace_id=trace_id
        )
```

### Шаг 5: Публикация событий

Публикуйте события статуса обратно в Kafka:

```python
def publish_status(
    task_id: str, 
    status: int, 
    result: dict = None, 
    error: str = None,
    trace_id: str = None
):
    """Публикует событие статуса задачи"""
    headers = {
        'status': str(status),
        'worker_id': 'llm-worker-1'
    }
    
    if trace_id:
        headers['trace_id'] = trace_id
    
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
```

### Пример полной интеграции Worker (Python)

```python
from confluent_kafka import Consumer, Producer
import json
import uuid
from minio import Minio

class LLMWorker:
    def __init__(self, kafka_servers: str, minio_config: dict):
        self.consumer = Consumer({
            'bootstrap.servers': kafka_servers,
            'group.id': 'llm-worker-group',
            'auto.offset.reset': 'earliest'
        })
        
        self.producer = Producer({
            'bootstrap.servers': kafka_servers
        })
        
        self.minio_client = Minio(
            minio_config['endpoint'],
            access_key=minio_config['access_key'],
            secret_key=minio_config['secret_key'],
            secure=minio_config.get('use_ssl', False)
        )
        
        self.bucket = minio_config['bucket']
    
    def start(self):
        """Запускает worker"""
        self.consumer.subscribe(['tasks_llm'])
        print("Worker started, waiting for tasks...")
        
        try:
            while True:
                msg = self.consumer.poll(timeout=1.0)
                
                if msg is None:
                    continue
                
                if msg.error():
                    print(f"Consumer error: {msg.error()}")
                    continue
                
                self.handle_message(msg)
                
        except KeyboardInterrupt:
            print("Stopping worker...")
        finally:
            self.consumer.close()
    
    def handle_message(self, msg):
        """Обрабатывает сообщение из Kafka"""
        task_id = msg.key().decode('utf-8')
        headers = {h[0]: h[1].decode('utf-8') for h in msg.headers()}
        value = json.loads(msg.value().decode('utf-8'))
        
        status = int(headers.get('status', '1'))
        trace_id = headers.get('trace_id')
        
        # Обрабатываем только команды (PENDING)
        if status == 1:
            self.process_task(task_id, value, trace_id)
    
    def process_task(self, task_id: str, command: dict, trace_id: str = None):
        """Обрабатывает задачу"""
        storage_path = command['storage_path']
        
        try:
            # Отправить PROCESSING
            self.publish_status(task_id, 2, trace_id=trace_id)
            
            # Загрузить файл
            file_content = self.load_file(storage_path)
            
            # Обработать
            result = self.process_with_llm(file_content)
            
            # Отправить COMPLETED
            self.publish_status(task_id, 3, result=result, trace_id=trace_id)
            
        except Exception as e:
            # Отправить FAILED
            self.publish_status(task_id, 4, error=str(e), trace_id=trace_id)
    
    def load_file(self, storage_path: str) -> bytes:
        """Загружает файл из MinIO"""
        # Извлекаем путь относительно bucket
        object_name = storage_path.replace(f"{self.bucket}/", "", 1)
        
        response = self.minio_client.get_object(self.bucket, object_name)
        content = response.read()
        response.close()
        
        return content
    
    def process_with_llm(self, file_content: bytes) -> dict:
        """Обрабатывает файл с помощью LLM"""
        # Здесь ваша логика обработки
        # Например, отправка в GigaChat API
        
        return {
            "extracted_data": "Результат обработки",
            "tokens_used": 1250,
            "processing_time_ms": 3450.5
        }
    
    def publish_status(
        self, 
        task_id: str, 
        status: int, 
        result: dict = None, 
        error: str = None,
        trace_id: str = None
    ):
        """Публикует событие статуса"""
        headers = {
            'status': str(status),
            'worker_id': 'llm-worker-1'
        }
        
        if trace_id:
            headers['trace_id'] = trace_id
        
        value = {
            'result': result,
            'error_message': error
        }
        
        self.producer.produce(
            topic='tasks_llm',
            key=task_id,
            value=json.dumps(value),
            headers=[(k, v.encode()) for k, v in headers.items()]
        )
        
        self.producer.flush()

# Запуск worker
worker = LLMWorker(
    kafka_servers='kafka:29092',
    minio_config={
        'endpoint': 'minio:9000',
        'access_key': 'minioadmin',
        'secret_key': 'minioadmin',
        'bucket': 'documents',
        'use_ssl': False
    }
)

worker.start()
```

## Интеграция с MinIO

### Загрузка файла

Task Manager автоматически загружает файлы в MinIO при создании задачи. Worker должен читать файлы из того же хранилища.

**Структура путей**:
```
{bucket}/{task_id}/{filename}
```

Пример: `documents/550e8400-e29b-41d4-a716-446655440000/document.pdf`

### Чтение файла из MinIO

```python
from minio import Minio

minio_client = Minio(
    'minio:9000',
    access_key='minioadmin',
    secret_key='minioadmin',
    secure=False
)

# Получить файл
object_name = f"{task_id}/{filename}"
response = minio_client.get_object('documents', object_name)
content = response.read()
response.close()
```

## Интеграция с PostgreSQL

Task Manager хранит задачи в PostgreSQL. Для мониторинга можно подключиться к той же БД:

```python
import psycopg2

conn = psycopg2.connect(
    host='postgres-task-manager',
    port=5432,
    database='task_manager',
    user='postgres',
    password='postgres'
)

cursor = conn.cursor()
cursor.execute("SELECT * FROM tasks WHERE task_id = %s", (task_id,))
task = cursor.fetchone()
```

## Обработка ошибок

### Retry механизм

При ошибках обработки рекомендуется:
1. Логировать ошибку
2. Отправлять статус `FAILED` в Kafka
3. Не коммитить offset в Kafka (для повторной обработки)

### Circuit Breaker

Для защиты от каскадных ошибок рекомендуется использовать Circuit Breaker:
- При превышении порога ошибок - временно прекращать обработку
- Автоматическое восстановление через некоторое время

## Мониторинг интеграции

### Метрики для отслеживания

- Количество созданных задач
- Количество завершенных задач
- Количество failed задач
- Время обработки задач
- Kafka lag (для worker)

### Логирование

Используйте `trace_id` для трейсинга запросов:
- Передавайте `trace_id` через все компоненты
- Логируйте `trace_id` во всех операциях
- Используйте для поиска проблем

## Best Practices

### 1. Идемпотентность

- Проверяйте статус задачи перед обработкой
- Не обрабатывайте задачу дважды
- Используйте уникальные идентификаторы

### 2. Обработка ошибок

- Всегда отправляйте статус `FAILED` при ошибках
- Включайте детальное сообщение об ошибке
- Логируйте все ошибки

### 3. Производительность

- Используйте connection pooling
- Батчируйте операции где возможно
- Настройте правильные timeout'ы

### 4. Безопасность

- Используйте SSL/TLS для Kafka в production
- Используйте аутентификацию для MinIO
- Не логируйте чувствительные данные

## Troubleshooting

### Задачи не обрабатываются

1. Проверьте подключение к Kafka
2. Проверьте, что worker подписан на правильный топик
3. Проверьте логи worker'а
4. Проверьте доступность MinIO

### Задачи застревают в PROCESSING

1. Проверьте, что worker отправляет статусы
2. Проверьте Kafka consumer в Task Manager
3. Проверьте логи Task Manager

### Ошибки при загрузке файлов

1. Проверьте доступность MinIO
2. Проверьте права доступа
3. Проверьте структуру путей

