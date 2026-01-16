# Task Manager Project Overview

## Project Purpose
The Task Manager is a Go-based microservice application designed to:
1. Serve as an API access point for external services
2. Manage task distribution to workers
3. Collect results from workers

The application integrates with multiple external services including Kafka for messaging, PostgreSQL for data persistence, and MinIO for object storage.

## Architecture & Technologies

### Core Technologies
- **Go** (version 1.24.0): Primary programming language
- **Gin Framework**: HTTP web framework for REST API
- **PostgreSQL**: Relational database for task management
- **Kafka**: Message broker for task distribution and result collection
- **MinIO**: Object storage for file uploads and documents
- **Docker & Docker Compose**: Containerization and orchestration
- **Swagger**: API documentation

### Project Structure
```
task-manager/
├── cmd/                    # Application entry points
│   └── api/                # Main API server
├── internal/               # Private application code
│   ├── entity/             # Data structures and models
│   ├── handlers/           # Request handlers (REST API & Kafka consumers)
│   ├── infrastructure/     # External service adapters
│   └── usecase/            # Business logic
├── pkg/                    # Reusable packages
│   ├── database/           # Database connectors
│   ├── httpserver/         # HTTP server wrapper
│   ├── kafka/              # Kafka producers/consumers
│   ├── logger/             # Logging utilities
│   ├── minio/              # MinIO storage adapter
│   └── response/           # API response helpers
├── config/                 # Configuration management
├── docs/                   # Documentation
├── database/               # Database migrations
├── test/                   # Test files
├── go.mod, go.sum          # Go module definitions
├── Dockerfile              # Multi-stage Docker build
├── docker-compose.yaml     # Service orchestration
└── Makefile                # Build and deployment commands
```

## Building and Running

### Prerequisites
- Go 1.24+
- Docker and Docker Compose
- Make

### Development Setup
1. Copy configuration template:
   ```bash
   cp config.example.yaml config.yaml
   ```

2. Build and run in development mode:
   ```bash
   make build-dev    # Build Docker image
   make run-dev      # Run container
   ```

### Alternative Development Options
- **Local build**: `make build-local && make run-local`
- **Production build**: `make prod-build && make prod-run`

### Key Make Commands
- `make help` - Show all available commands
- `make test-all` - Run all tests (unit + integration)
- `make logs` - View container logs
- `make stop` - Stop containers
- `make swagger` - Generate API documentation
- `make clean-all` - Full cleanup

### Docker Targets
- `dev` - Development build using config.yaml
- `prod` - Production build using environment variables only

## Configuration
The application uses a YAML configuration file (`config.yaml`) with sections for:
- Logger settings
- Server configuration
- Database connection
- Kafka broker settings
- MinIO storage
- Metrics and Swagger

Environment variables can override configuration values in production mode.

## Services Integration
- **Kafka**: Used as both consumer (to receive results from workers) and producer (to send tasks to workers)
- **PostgreSQL**: Stores task metadata and application state
- **MinIO**: Stores uploaded documents and files associated with tasks
- **Swagger UI**: Available at http://localhost:8080/swagger/index.html

## Development Conventions
- Follows Go best practices and idioms
- Uses structured logging with multiple log files for different concerns
- Implements graceful shutdown for proper resource cleanup
- Uses health checks for service monitoring
- Implements proper error handling and validation

## Testing
- Unit tests: `make test-unit`
- Integration tests: `make test-integration`
- All tests: `make test-all`

## Deployment
The application is designed for containerized deployment using Docker and Docker Compose. The docker-compose.yaml defines all required services including PostgreSQL, Kafka, Zookeeper, MinIO, and supporting UI tools.

## Monitoring
- Metrics available at `/metrics` endpoint on port 9090
- Comprehensive logging to multiple files based on concern (app, health, http, kafka, minio, task, s3)