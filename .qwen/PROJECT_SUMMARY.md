# Project Summary

## Overall Goal
Refactor the Task Manager microservice to adopt a unified, interface-based architecture for task handling, replacing concrete `*domain.Task` structs with a `domain.Task` interface and `domain.BaseTask` implementation to ensure predictability, extensibility, and simplicity when adding new task types (e.g., LLM, Parsing, Multifield).

## Key Knowledge
- **Core Technology Stack**: Go 1.24+, Gin framework, PostgreSQL, Kafka, MinIO, Docker.
- **Architecture Pattern**: Domain-Driven Design with clear separation: `internal/entity/domain` (models), `internal/usecase` (business logic), `internal/handlers` (API/broker), `internal/infrastructure/adapter` (DB/storage).
- **Critical Refactoring Decision**: `domain.Task` is now an interface; all concrete task types (e.g., `TaskLLM`) embed `domain.BaseTask`, which implements the interface. Direct field access (e.g., `task.Status`) is replaced by method calls (e.g., `task.GetStatus()`).
- **Generic Broker Commands**: Introduced `domain.BrokerCommand[T]` for type-safe Kafka event handling, replacing raw `any` or concrete event structs like `domain.TaskLLMStatusEvent`.
- **Build & Test Commands**:
  - Build: `go build ./...`
  - Test: `go test ./...`
  - Local run: `make build-local && make run-local`
  - Dev run: `make build-dev && make run-dev`
- **User Preference**: Output language is Russian, but technical artifacts (code, paths, logs) remain in English.

## Recent Actions
- **Successfully Implemented Core Architecture**:
  - Created foundational files: `task_interface.go`, `base_task.go`, `broker_command.go`.
  - Updated domain layer: `task.go` (constants only), `task_llm.go` (embeds `BaseTask`), `task_unified.go` (uses `Task` interface).
  - Updated repository layer: `task_repository.go` and `task_repository_with_metrics.go` now use `domain.Task` instead of `*domain.Task`.
  - Updated use cases: `task_llm_usecase.go`, `multiupload_usecase.go`, `unified_task_uc.go` create tasks via `&domain.BaseTask{...}` and use interface methods.
  - Updated all decorators (`logging_decorator.go`, `metrics_decorator.go`, `task_llm_decorators.go`, `multiupload_decorators.go`) to use `domain.Task` and `GetStatus()`.
  - Updated REST API handlers (`handle.go`) to use interface methods (`GetID()`, `GetStatus()`, etc.).
  - Updated broker workflow validators and routes (`validator_llm.go`, `route_llm.go`) to use `BrokerCommand[...]`.
  - Fixed generic type inference in `broker/route.go` by changing `ValidatorFunc[T]` from `func(*kafka.Message) (*T, error)` to `func(*kafka.Message) (T, error)`.
  - Updated test mocks (`test/mocks/task_llm_usecase.go`, `task_llm_test.go`) to align with new types.
- **Compilation Status**: The project now builds successfully (`go build ./...` exits with code 0). All core application code is refactored and type-correct.
- **Remaining Issues**: Integration and unit tests fail due to outdated mocks (e.g., `MockRepository` still uses `*domain.Task`, `MockProducer` uses `interface{}` instead of `[]byte`) and direct struct literals (`domain.Task{...}`) or field access (`task.Status`) in test code.

## Current Plan
1. [DONE] Define `domain.Task` interface and `domain.BaseTask` implementation.
2. [DONE] Refactor domain entities (`task_llm.go`, `task_unified.go`) to embed `BaseTask`.
3. [DONE] Update repository layer (`task_repository.go`, `task_repository_with_metrics.go`) to use `domain.Task`.
4. [DONE] Update use cases (`task_llm_usecase.go`, `multiupload_usecase.go`, `unified_task_uc.go`) to create `BaseTask` and use interface methods.
5. [DONE] Update all decorators (logging, metrics, multiupload, LLM) to use `domain.Task`.
6. [DONE] Update REST API handlers (`handle.go`) and broker workflows (`validator_llm.go`, `route_llm.go`).
7. [DONE] Fix generic type definitions in `broker/route.go`.
8. [IN PROGRESS] Update test mocks and integration tests to match new architecture.
   - [TODO] Update `MockRepository` methods (`Create`, `GetByID`, `ListByStatus`) to use `domain.Task` instead of `*domain.Task`.
   - [TODO] Update `MockProducer.Send` signature to accept `[]byte` instead of `interface{}`.
   - [TODO] Update `MockTaskLLMUCInterface` and `MockMultiUploadUCInterface` to use `(domain.Task, error)` for `GetTaskByID`.
   - [TODO] Replace all `domain.Task{...}` literals in tests with `&domain.BaseTask{...}`.
   - [TODO] Replace all direct field accesses (e.g., `task.Status`, `task.RequestID`) in tests with method calls (e.g., `task.GetStatus()`, `task.GetRequestID()`).
   - [TODO] Update all test cases that reference `domain.TaskLLMStatusEvent` to use `domain.BrokerCommand[domain.TaskLLMStatusEventPayload]`.

---

## Summary Metadata
**Update time**: 2026-03-02T11:36:26.580Z 
