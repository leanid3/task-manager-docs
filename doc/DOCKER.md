# Dockerfile - Подробная документация

## Назначение Dockerfile

Dockerfile описывает процесс сборки и запуска сервиса **task-manager** - микросервиса для постановки и управления задачами. Dockerfile использует многостадийную сборку (multi-stage build) для оптимизации размера финального образа и разделения процессов сборки и выполнения.

## Структура Dockerfile

Dockerfile состоит из следующих стадий:

### 1. builder (golang:1.25)

Стадия сборки приложения:
- Устанавливает зависимости для компиляции (gcc, g++, librdkafka-dev)
- Загружает Go-зависимости (go mod download)
- Устанавливает swag для генерации Swagger документации
- Генерирует Swagger документацию из аннотаций в коде
- Компилирует бинарник приложения с включенным CGO (требуется для confluent-kafka-go)

### 2. base (debian:bookworm-slim)

Базовый runtime образ:
- Минимальный runtime образ на основе Debian
- Устанавливает runtime зависимости (librdkafka1, ca-certificates)
- Копирует скомпилированный бинарник из стадии builder
- Создает непривилегированного пользователя appuser для безопасности
- Настраивает порт сервера (по умолчанию 8030)

### 3. dev (наследует от base)

Режим разработки:
- Копирует config.yaml из проекта в контейнер
- Используется для локальной разработки и тестирования
- Конфигурация загружается из файла config.yaml

### 4. prod (наследует от base)

Production режим:
- Не копирует config.yaml (конфигурация загружается только из переменных окружения)
- Используется для production развертывания
- Конфигурация загружается из переменных окружения (.env файл)

## Запуск в локальной среде

### Предварительные требования

1. Установленный Docker и Docker Compose
2. Go 1.25+ (для локальной разработки, опционально)
3. Доступ к зависимым сервисам (Kafka, Postgres, Minio)

### Шаги для запуска

#### 1. Создать конфигурационный файл

В корне проекта создать config.yaml на основе примера:

```bash
cp config.example.yaml config.yaml
```

#### 2. Настроить config.yaml

При необходимости настроить параметры:
- Указать адреса Kafka, Postgres, Minio
- Настроить порты и другие параметры
- Настроить логирование и метрики

#### 3. Собрать Docker образ в dev среде

Через Makefile:
```bash
make build-dev
```

Или напрямую через Docker:
```bash
docker build --target dev --build-arg CONFIG_FILE=config.yaml -t task-manager:dev .
```

#### 4. Запустить контейнер

Через Makefile:
```bash
make run-dev
```

Или напрямую через Docker:
```bash
docker run -d \
  --name task-manager-app \
  -p 8080:8080 \
  -p 9090:9090 \
  -v ./logs:/app/logs \
  --network bridge \
  task-manager:dev
```

#### 5. Проверить статус контейнера

```bash
make status
# или
docker ps | grep task-manager-app
```

#### 6. Просмотреть логи

```bash
make logs
# или
docker logs -f task-manager-app
```

### Параметры запуска (через Makefile)

Можно переопределить параметры через переменные окружения:

```bash
make run-dev PORT_C=3000 MPORT_C=9091 NAME_C=my-task-manager
```

Доступные переменные:
- `NAME_C` - имя контейнера (по умолчанию: task-manager-app)
- `PORT_C` - основной порт HTTP API (по умолчанию: 8080)
- `MPORT_C` - порт метрик (по умолчанию: 9090)
- `NET_C` - Docker сеть (по умолчанию: bridge)
- `LOGS_C` - путь к директории логов (по умолчанию: ./logs)

Для просмотра всех доступных переменных:
```bash
make help-vars
```

### Production режим

Для сборки и запуска в production режиме:

```bash
# Сборка
make prod-build

# Запуск
make prod-run
```

В production режиме конфигурация загружается только из переменных окружения (.env файл). Файл config.yaml не используется.

## Тестирование

### Локальное тестирование (без Docker)

#### Unit тесты

```bash
make test-unit
# или
go test ./... -short -v
```

#### Integration тесты

```bash
make test-integration
# или
go test ./... -tags=integration -v -parallel=1
```

#### Все тесты

```bash
make test-all
```

### Тестирование в Docker контейнере

#### 1. Запустить контейнер

См. раздел "Запуск в локальной среде"

#### 2. Проверить здоровье сервиса

```bash
curl http://localhost:8080/health
```

#### 3. Проверить метрики

```bash
curl http://localhost:9090/metrics
```

#### 4. Проверить Swagger документацию

Открыть в браузере: http://localhost:8080/swagger/index.html

#### 5. Проверить логи на наличие ошибок

```bash
make logs
# или
docker logs task-manager-app | grep -i error
```

### Тестирование API endpoints

После запуска контейнера можно тестировать API через Swagger UI или напрямую:

```bash
# Пример запроса (зависит от реализованных endpoints)
curl -X GET http://localhost:8080/api/v1/tasks
```

## Полезные команды

### Основные команды

- `make help` - показать все доступные команды
- `make help-vars` - показать доступные переменные
- `make stop` - остановить контейнер
- `make restart` - перезапустить контейнер
- `make logs` - показать логи контейнера
- `make status` - показать статус контейнера

### Управление контейнерами и образами

- `make rm-container` - удалить контейнер
- `make rm-image` - удалить образ
- `make rm-all` - удалить контейнер и образы

### Короткие алиасы

- `h` / `hv` - help / help-vars
- `bd` / `rd` - build-dev / run-dev
- `pb` / `pr` - prod-build / prod-run
- `tu` / `ti` / `ta` - test-unit / test-integration / test-all
- `l` / `s` / `st` / `rs` - logs / status / stop / restart
- `rmi` / `rmc` / `ra` - rm-image / rm-container / rm-all

## Устранение неполадок

### Контейнер не запускается

1. **Проверьте логи**:
   ```bash
   make logs
   ```

2. **Убедитесь, что порты не заняты**:
   ```bash
   netstat -tulpn | grep 8080
   # или
   lsof -i :8080
   ```

3. **Проверьте наличие config.yaml**:
   ```bash
   ls -la config.yaml
   ```

4. **Проверьте права доступа к директории логов**:
   ```bash
   ls -la logs/
   chmod 755 logs/
   ```

### Ошибки подключения к Kafka/Postgres/Minio

1. **Проверьте настройки в config.yaml**:
   - Убедитесь, что адреса сервисов указаны правильно
   - Проверьте учетные данные (username, password)

2. **Убедитесь, что зависимые сервисы запущены и доступны**:
   ```bash
   # Проверка Kafka
   docker ps | grep kafka
   
   # Проверка Postgres
   docker ps | grep postgres
   
   # Проверка Minio
   docker ps | grep minio
   ```

3. **Проверьте сетевые настройки Docker**:
   - Убедитесь, что контейнеры находятся в одной сети
   - Проверьте настройки сети в docker-compose или Makefile

4. **Проверьте доступность сервисов из контейнера**:
   ```bash
   docker exec -it task-manager-app ping kafka
   docker exec -it task-manager-app ping postgres
   ```

### Проблемы со сборкой

1. **Очистите кэш Docker**:
   ```bash
   make build-dev NOCACHE_C=true
   ```

2. **Проверьте наличие всех зависимостей в go.mod**:
   ```bash
   go mod verify
   go mod tidy
   ```

3. **Проверьте версию Go**:
   - Убедитесь, что используется Go 1.25 или выше
   - Проверьте Dockerfile на соответствие версии

4. **Проверьте наличие всех системных зависимостей**:
   - librdkafka-dev должен быть доступен в образе builder
   - Проверьте логи сборки на наличие ошибок компиляции

### Проблемы с правами доступа

1. **Проблемы с логами**:
   ```bash
   # Создать директорию логов с правильными правами
   mkdir -p logs
   chmod 755 logs
   ```

2. **Проблемы с config.yaml**:
   ```bash
   # Проверить права доступа
   ls -la config.yaml
   chmod 644 config.yaml
   ```

### Проблемы с производительностью

1. **Использование кэша при сборке**:
   - Docker автоматически использует кэш слоев
   - Для принудительной пересборки используйте `NOCACHE_C=true`

2. **Оптимизация размера образа**:
   - Используется многостадийная сборка для минимизации размера
   - Финальный образ основан на debian:bookworm-slim

## Дополнительная информация

### Переменные окружения

В production режиме конфигурация загружается из переменных окружения. Создайте файл `.env` в корне проекта с необходимыми переменными.

### Сетевая конфигурация

По умолчанию контейнер использует сеть `bridge`. Для работы с другими сервисами убедитесь, что они находятся в той же сети или используйте docker-compose.

### Volumes

Контейнер монтирует директорию `./logs` в `/app/logs` для сохранения логов на хосте.

### Порты

- **8080** - HTTP API сервер
- **9090** - Метрики (Prometheus)

### Swagger документация

После запуска контейнера Swagger документация доступна по адресу:
http://localhost:8080/swagger/index.html

