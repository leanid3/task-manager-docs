package domain

import (
	"strconv"

	"github.com/google/uuid"
)

// TaskContractHeaders заголовки для задачи
type TaskContractHeaders struct {
	TraceID  *uuid.UUID `json:"trace_id"`
	WorkerID string     `json:"worker_id,omitempty"`
	Status   int        `json:"status"`
	Version  string     `json:"version,omitempty"`
}

func (t *TaskContractHeaders) ToMap() map[string]string {
	headers := make(map[string]string)

	if t.TraceID != nil {
		headers["trace_id"] = t.TraceID.String()
	}
	if t.WorkerID != "" {
		headers["worker_id"] = t.WorkerID
	}
	if t.Status != 0 {
		headers["status"] = strconv.Itoa(t.Status)
	}
	if t.Version != "" {
		headers["version"] = t.Version
	}
	return headers
}

// TaskContractKey ключ для задачи
type TaskContractKey struct {
	TaskID uuid.UUID `json:"task_id"`
}

// TaskCommand универсальная команда для всех типов задач - только для DI
type TaskCommand struct {
	Key     TaskContractKey
	Headers TaskContractHeaders
	Value   any
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
	default:
		return TaskStatusFailed, true
	}
}
