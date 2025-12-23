package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// @name TaskStatus
type TaskStatus string

const (
	//TODO сделать статус коды
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusProcessing TaskStatus = "PROCESSING"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusFailed     TaskStatus = "FAILED"
)

// type TaskType string

// const (
// 	TaskTypeParsing    TaskType = "PARSING"
// 	TaskTypeAlgorithms TaskType = "ALGORITHMS"
// 	TaskTypeLLM        TaskType = "LLM"
// 	TaskTypeAnalyze    TaskType = "ANALYZE"
// )

// TODO! синхронизировать структуру Task по итогам согласования с LLM сервисом
// @name Task
type Task struct {
	TaskID uuid.UUID `json:"task_id"`
	// TaskType TaskType   `json:"task_type"`
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
