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
