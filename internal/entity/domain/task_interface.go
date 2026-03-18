// internal/entity/domain/task_interface.go

package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Task — это основной интерфейс для всех типов задач.
// Он определяет контракт для доступа к данным задачи, что позволяет избежать type assertion
// и делает код более гибким и расширяемым.
type Task interface {
	// Общие методы для получения данных
	GetID() uuid.UUID
	GetStatus() TaskStatus
	GetMetadata() map[string]interface{}
	GetWorkerID() string
	GetRequestID() string
	GetTraceID() *uuid.UUID
	GetCreatedAt() time.Time
	GetStartedAt() *time.Time
	GetCompletedAt() *time.Time
	GetErrorMessage() string
	GetResult() json.RawMessage

	// Метод для получения специфичного тела задачи.
	// Возвращает nil для базовых задач, или конкретную структуру для специфичных (например, *TaskLLMPayload).
	GetPayload() interface{}

	// Методы для изменения состояния задачи
	SetStatus(status TaskStatus)
	SetWorkerID(workerID string)
	SetErrorMessage(errMsg string)
	SetResult(result json.RawMessage)
	SetStartedAt(t time.Time)
	SetCompletedAt(t time.Time)
}

// BaseTask — базовая структура, которая реализует интерфейс Task.
// Все специфичные задачи (TaskLLM, TaskParsing и т.д.) будут embed-ить эту структуру.
// Определение BaseTask находится в base_task.go.
