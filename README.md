### Task-manager
---
### Цель:
 1) Точка доступа в приложение по api для внешних сервисов
 2) Постановка задач для workerов
 3) Получить результат с workerов
---
### Взаимодействие с другими сервисами:
 1) Kafka - consumer, producer
 2) Postgres - connector, adapter
 3) Minio - connector, adapter
---
Возможен доступ по swagger: http://localhost:8080/swagger/index.html
---
## Быстрый старт

### Запуск контейнера в dev среде:

1. Создать конфигурационный файл:
```bash
cp config.example.yaml config.yaml
```

2. Собрать образ:
```bash
make build-dev
```

3. Запустить контейнер:
```bash
make run-dev
```

### Основные команды

- `make help` - показать все доступные команды
- `make logs` - показать логи контейнера
- `make status` - показать статус контейнера
- `make stop` - остановить контейнер
- `make test-all` - запустить все тесты

### Подробная документация

Подробная документация по Dockerfile, запуску, тестированию и устранению неполадок находится в [doc/DOCKER.md](doc/DOCKER.md)