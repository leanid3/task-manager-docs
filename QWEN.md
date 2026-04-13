# Task Manager — QWEN.md

## Project Overview

**Task Manager** is a Go-based microservice for task orchestration and management. It serves as an API gateway for external services to submit tasks to workers and receive results. Built using **Clean Architecture** principles with clear separation of concerns.

### Core Responsibilities
1. **API Entry Point** — REST API for external services
2. **Task Dispatch** — Submit tasks to workers via Kafka broker
3. **Result Collection** — Receive and store task results from workers

### External Integrations
- **Kafka** — Consumer & Producer for async task messaging
- **PostgreSQL** — Primary data store (via `pgx/v5`)
- **MinIO/S3** — Object storage for task-related files

### Tech Stack
- **Language:** Go 1.24+
- **HTTP Framework:** Gin
- **Message Broker:** Kafka (confluent-kafka-go)
- **Database:** PostgreSQL (pgx/v5)
- **Object Storage:** MinIO
- **Metrics:** Prometheus
- **API Docs:** Swagger (swaggo)
- **Testing:** testcontainers-go for integration tests
- **Build:** Docker multi-stage, Makefile

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ 1. HANDLERS (Presentation Layer)                            │
│    internal/handlers/                                        │
│    - restapi/  — HTTP handlers (Gin)                        │
│      - v1/     — API v1 handlers                            │
│    - broker/   — Kafka message handlers                     │
│      - workflows/ — typed workflows (llm, etc.)             │
└───────────────────┬─────────────────────────────────────────┘
                    │ depends on
┌───────────────────▼─────────────────────────────────────────┐
│ 2. USE CASES (Business Logic Layer)                         │
│    internal/usecase/                                         │
│    - interface.go     — TaskManager, TaskProcessor, etc.    │
│    - decorator/       — universal Logging/Metrics decorators│
│    - task_llm_*.go    — LLM task usecase                    │
│    - unified_task_uc.go — unified task processing           │
│    - multiupload/     — multi-file upload usecase           │
└───────┬───────────────┴───────────────┬─────────────────────┘
        │ depends on                    │ depends on
┌───────▼───────────────┐   ┌──────────▼──────────────────────┐
│ 3. SERVICE (Abstractions)│  │ 4. REPOSITORY (Data Access)   │
│    internal/service/   │   │    internal/entity/repository/ │
│    - broker.go   — Kafka interface  │    - task_repository.go         │
│    - storage.go  — MinIO interface  │    - Task interface             │
└───────┬───────────────┘   └──────────┬──────────────────────┘
        │ implemented by               │ implemented by
┌───────▼───────────────┐   ┌──────────▼──────────────────────┐
│ 5. INFRASTRUCTURE (Adapters)                                │
│    internal/infrastructure/adapter/                          │
│    - database/postgres/ — PostgreSQL Task repository        │
│    - storage/minio/     — MinIO Storage adapter             │
│    pkg/                                                      │
│    - kafka/     — Kafka producer (implements service.Broker)│
│    - minio/     — MinIO connector                           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ 6. DOMAIN (Models & Value Objects)                          │
│    internal/entity/domain/                                   │
│    - base_task.go   — BaseTask struct, Task interface       │
│    - task_llm.go    — TaskLLM, LLMMetadata                  │
│    - task_unified.go — TaskType, TaskDefinition             │
│    - task_broker.go — BrokerCommand, TaskContractHeaders    │
│    - broker_llm.go  — TaskLLMCommand, TaskLLMStatusEvent    │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Patterns
- **Repository Pattern** — abstract data access (`repository.Task`)
- **Service Abstraction** — external systems behind interfaces (`service.Broker`, `service.Storage`)
- **Use Case Pattern** — isolated business logic with clear interfaces
- **Decorator Pattern** — universal `decorator.Logging` and `decorator.Metrics` for any UseCase
- **Dependency Injection** — all dependencies passed via constructors

---

## Project Structure

```
task-manager/
├── cmd/api/main.go          # Application entry point (DI composition root)
├── config/                   # Configuration loading (env-based)
├── internal/
│   ├── entity/
│   │   ├── domain/          # Domain models (Task, TaskLLM, BaseTask, etc.)
│   │   ├── broker/          # Broker interface (Producer, Consumer)
│   │   ├── repository/      # Data access interfaces (Task)
│   │   └── errors/          # Application error types
│   ├── service/             # Service abstractions (usecase depends on these)
│   │   ├── broker.go        # service.Broker — message broker interface
│   │   └── storage.go       # service.Storage — file storage interface
│   ├── handlers/
│   │   ├── restapi/         # HTTP REST handlers (Gin)
│   │   │   ├── v1/          # API v1 handlers
│   │   │   │   ├── router.go       # Route registration
│   │   │   │   ├── handle.go       # Task LLM handlers
│   │   │   │   └── multiupload/    # Multi-upload handler
│   │   │   └── middleware/  # Gin middleware (logging, metrics, etc.)
│   │   └── broker/          # Kafka message handlers
│   │       ├── kafka.go     # Kafka message dispatcher
│   │       ├── registry.go  # Handler registry
│   │       ├── route.go     # Generic typed route
│   │       └── workflows/   # Typed workflows (llm/, etc.)
│   ├── usecase/             # Business logic
│   │   ├── interface.go     # TaskManager, TaskProcessor, TaskInput, etc.
│   │   ├── decorator/       # Universal Logging & Metrics decorators
│   │   ├── usecase.go       # UseCases container
│   │   ├── task_llm_*.go    # TaskLLMUC + decorators
│   │   ├── unified_task_uc.go  # UnifiedTaskUC
│   │   └── multiupload/     # MultiUploadUC
│   └── infrastructure/
│       └── adapter/
│           ├── database/postgres/  # PostgreSQL Task repository
│           └── storage/minio/      # MinIO Storage adapter
├── pkg/                     # Shared packages
│   ├── kafka/               # Kafka producer (implements service.Broker)
│   ├── minio/               # MinIO connector (implements service.Storage)
│   ├── database/            # PostgreSQL connector
│   ├── logger/              # Structured logging
│   ├── metrics/             # Prometheus metrics
│   ├── httpserver/          # HTTP server wrapper
│   ├── response/            # HTTP response helpers
│   └── ...
├── docker-compose/          # Docker Compose files
├── doc/                     # Documentation
├── docs/                    # Generated Swagger docs
├── test/                    # Test utilities and integration tests
├── Dockerfile               # Multi-stage Docker build
├── Makefile                 # Build/run/test commands
├── go.mod / go.sum          # Go dependencies
└── .env.example             # Environment variables template
```

---

## Building and Running

### Prerequisites
- Go 1.24+
- Docker & Docker Compose
- `make`
- `librdkafka-dev` (for local builds with confluent-kafka-go)

### Quick Start

#### Local Development (native Go)
```bash
cp .env.example .env
make build-local   # Build binary to bin/task-manager
make run-local     # Run with .env
```

#### Docker Dev Mode (hot reload via `go run`)
```bash
cp .env.example .env
make build-dev     # Build dev image
make run-dev       # Run dev container (code mounted as volume)
```

#### Docker Compose — Full Stack (DB + Kafka + MinIO + App)
```bash
cp .env.example .env

# Dev mode (hot reload)
make up-dev

# Prod mode (optimized binary)
make up-prod

# Infrastructure only (no app)
make up-standalone
```

#### Production Build
```bash
cp .env.example .env
# Edit .env with real credentials/hosts
make prod-build
make prod-run
```

> **Important:** In prod mode, configuration is loaded **only** from environment variables (`.env`). No `config.yaml` is used.

### Key Makefile Commands

| Command | Description |
|---------|-------------|
| `make help` | Show all available commands |
| `make up-dev` | Start full stack in dev mode |
| `make up-prod` | Start full stack in prod mode |
| `make down` | Stop all services |
| `make logs SERVICE=xxx` | View service logs |
| `make status` | Show service status |
| `make test-unit` | Run unit tests |
| `make test-integration` | Run integration tests |
| `make test-all` | Run all tests |
| `make fmt` | Format code (`go fmt`) |
| `make vet` | Run `go vet` |
| `make swagger` | Generate Swagger docs |
| `make clean` | Clean binaries and cache |
| `make shell` | Shell into container |

---

## Configuration

All configuration is done via **environment variables** (loaded from `.env`).

### Essential Variables

```bash
# Server
SERVER_PORT=8080
SERVER_HOST=0.0.0.0

# Database
DATABASE_HOST=task-manager-database
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_DATABASE=task_manager

# Kafka
BROKER_BOOTSTRAP_SERVICE=kafka:29092
BROKER_TOPICS=tasks_llm

# MinIO
MINIO_ENDPOINT=minio:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=documents

# Metrics
METRICS_PORT=9091
METRICS_PATH=/metrics

# Logging
LOGGER_LEVEL=debug
LOGGER_FORMAT=text        # text or json (prod uses json)
LOGGER_MODE=stdout        # stdout or files
```

See `.env.example` for the full list.

---

## API & Swagger

Swagger UI is available at: `http://localhost:8080/swagger/index.html`

Regenerate Swagger docs:
```bash
make swagger
```

---

## Metrics

Prometheus metrics are exposed at: `http://localhost:9091/metrics`

Available metrics:
- HTTP requests (duration, count)
- Task processing (duration, statuses)
- Kafka messages (consumed, produced)
- Database operations (query time)
- MinIO operations (duration, bytes transferred)

---

## Testing

### Unit Tests
```bash
make test-unit
# or: go test ./... -short -v
```

### Integration Tests (uses testcontainers)
```bash
make test-integration
# or: go test ./... -tags=integration -v -parallel=1
```

### All Tests
```bash
make test-all
```

---

## Development Conventions

### Code Style
- Standard Go formatting (`go fmt`)
- Run `go vet` before commits
- Use `make check` for pre-commit validation

### Adding a New Task Type
See `doc/how-to-add-new-task-type.md` for a step-by-step guide. In brief:
1. Add `TaskType` constant in `internal/entity/domain/task_unified.go`
2. Create domain struct embedding `BaseTask`
3. Implement usecase in `internal/usecase/`
4. Create HTTP handler in `internal/handlers/restapi/v1/`
5. Register route in router
6. (Optional) Add Kafka consumer/processor
7. Write tests

### Architecture Guidelines
- Use **Repository Pattern** for data access
- Use **Use Case Pattern** for business logic
- Inject all dependencies via constructors
- Wrap usecases with decorators for metrics/logging
- Handle errors with compensation (rollback created resources)

---

## Docker Compose Services

| Service | Description |
|---------|-------------|
| `base.yaml` | PostgreSQL, Kafka (Zookeeper + Broker), MinIO |
| `dev.yaml` | Dev profile — `go run`, code volume mount |
| `prod.yaml` | Prod profile — compiled binary, env-only config |
| `standalone.yaml` | Infrastructure only (no task-manager app) |

### Ports (default)
| Service | Port |
|---------|------|
| Task Manager API | 8080 |
| Task Manager Metrics | 9091 |
| MinIO API | 9000 |
| MinIO Console | 9001 |
| Kafka | 9092 |
| Zookeeper | 2181 |
| Kafka UI | 9100 |

---

## Graceful Shutdown

The application handles `SIGINT`/`SIGTERM`:
1. Cancel context (stops Kafka consumer)
2. Shutdown HTTP server
3. Shutdown metrics server
4. Stop Kafka consumer
5. Close database connection

---

## Documentation Files

| File | Description |
|------|-------------|
| `doc/architecture_summary.md` | Full architecture overview |
| `doc/how-to-add-new-task-type.md` | Guide for adding new task types |
| `doc/DOCKER.md` | Docker build/run documentation |
| `doc/TESTING.md` | Testing guide |
| `doc/integration_guide.md` | Integration guide |
| `doc/lifecycle_guide.md` | Application lifecycle |
| `doc/api_guide.md` | API usage guide |
