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
	TaskStatusCancelled  TaskStatus = "CANCELLED"
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

func (t *Task) ToDatabaseCode() string {

	switch t.Status {
	case TaskStatusPending:
		return "PENDING"
	case TaskStatusProcessing:
		return "PROCESSING"
	case TaskStatusCompleted:
		return "COMPLETED"
	case TaskStatusFailed:
		return "FAILED"
	case TaskStatusCancelled:
		return "CANCELLED"
	}
	return ""
}

// ToKafkaCode возвращает числовой код для Kafka (1-4)
func (s TaskStatus) ToKafkaCode() int {
	switch s {
	case TaskStatusPending:
		return 1
	case TaskStatusProcessing:
		return 2
	case TaskStatusCompleted:
		return 3
	case TaskStatusFailed:
		return 4
	case TaskStatusCancelled:
		return 5
	default:
		return 4
	}
}

// FromKafkaCode преобразует код обратно в TaskStatus
func (s TaskStatus) FromKafkaCode(code int) (TaskStatus, bool) {
	switch code {
	case 1:
		return TaskStatusPending, true
	case 2:
		return TaskStatusProcessing, true
	case 3:
		return TaskStatusCompleted, true
	case 4:
		return TaskStatusFailed, true
	case 5:
		return TaskStatusCancelled, true
	default:
		return TaskStatusFailed, false
	}
}

// MultiUploadMetadata структура для хранения метаданных задачи многофайловой загрузки
type MultiUploadMetadata struct {
	FileCount int      `json:"file_count"`
	FileKeys  []string `json:"file_keys"`
}
