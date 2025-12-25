## Запуск контейнеров

запускаем task-manager отдельно
```bash
#!/bin/bash

# Команда для запуска контейнера task-manager
docker run -d \
  --name task-manager-app \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 9090:9090 \
  --env-file env/task-manager.env \
  -v /var/log/task-manager:/app/logs \
  --network document-flow_document-flow-network \
  192.168.78.61:3000/ravil-developer/task-manager:v1
```