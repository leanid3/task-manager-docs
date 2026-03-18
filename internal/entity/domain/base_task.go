// internal/entity/domain/base_task.go

package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BaseTask — базовая структура для всех типов задач.
// Она содержит общие поля и будет embed-иться в специфичные задачи (например, TaskLLM).
// Реализует интерфейс Task.
type BaseTask struct {
	TaskID       uuid.UUID              `json:"task_id"`
	Status       TaskStatus             `json:"status"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	WorkerID     string                 `json:"worker_id,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	TraceID      *uuid.UUID             `json:"trace_id,omitempty"`
	Result       json.RawMessage        `json:"result,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

func (t *BaseTask) GetID() uuid.UUID                    { return t.TaskID }
func (t *BaseTask) GetStatus() TaskStatus               { return t.Status }
func (t *BaseTask) GetMetadata() map[string]interface{} { return t.Metadata }
func (t *BaseTask) GetWorkerID() string                 { return t.WorkerID }
func (t *BaseTask) GetRequestID() string                { return t.RequestID }
func (t *BaseTask) GetTraceID() *uuid.UUID              { return t.TraceID }
func (t *BaseTask) GetCreatedAt() time.Time             { return t.CreatedAt }
func (t *BaseTask) GetStartedAt() *time.Time            { return t.StartedAt }
func (t *BaseTask) GetCompletedAt() *time.Time          { return t.CompletedAt }
func (t *BaseTask) GetErrorMessage() string             { return t.ErrorMessage }
func (t *BaseTask) GetResult() json.RawMessage          { return t.Result }
func (t *BaseTask) GetPayload() interface{}             { return nil } // Базовая задача не имеет специфичного тела

func (t *BaseTask) SetStatus(status TaskStatus)      { t.Status = status }
func (t *BaseTask) SetWorkerID(workerID string)      { t.WorkerID = workerID }
func (t *BaseTask) SetErrorMessage(errMsg string)    { t.ErrorMessage = errMsg }
func (t *BaseTask) SetResult(result json.RawMessage) { t.Result = result }
func (t *BaseTask) SetStartedAt(newTime time.Time)   { t.StartedAt = &newTime }
func (t *BaseTask) SetCompletedAt(newTime time.Time) { t.CompletedAt = &newTime }
