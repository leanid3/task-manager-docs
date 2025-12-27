### Task-manager
---
### Цель:
 1) Точка доступа в приложние по api для внешних сервисов
 2) Постановка задач для workerов
 3) Получить результат с workerов
---
 ###Взаимодействие с другими сервисами:
 1) Kakfa - consumer, producer
 2) Postgres - connector, adapter
 3) Minio - connector, adapter
---
Возможен доступ по swagger: ["http://localhost:8080/swagger/index.html"]
---
Запуск контейнера в dev среде:
1) создать совой config в корне проекта:
```bash
    cp config.example.yaml config.yaml
```
2) Собрать образ в dev среде(параметры запуска можно изменить в Makefile):
```bash
    make build-dev
```
3) Запустить
```bash
    make run-dev
```
Запустить:
```bash
    
```