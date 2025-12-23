package domain

import (
	"encoding/json"
)

// тело сообщения,которое будет отправлено в broker по задаче LLM
type TaskLLMCommand struct {
	Key     TaskContractKey
	Headers TaskContractHeaders
	Value   TaskLLMCommandPayload
}

type TaskLLMCommandPayload struct {
	StoragePath string      `json:"storage_path"`
	StorageSize int64       `json:"storage_size,omitempty"`
	Metadata    LLMMetadata `json:"metadata,omitempty"`
}

// NewTaskLLMCommand создает команду для задачи LLM
func NewTaskLLMCommand(task *TaskLLM) *TaskLLMCommand {
	return &TaskLLMCommand{
		Key: TaskContractKey{
			TaskID: task.TaskID,
		},
		Headers: TaskContractHeaders{
			Status:   task.Status.ToKafkaCode(),
			TraceID:  task.TraceID,
			WorkerID: task.WorkerID,
		},
		Value: TaskLLMCommandPayload{
			StoragePath: task.StoragePath,
			StorageSize: task.StorageSize,
			Metadata:    task.Metadata,
		},
	}
}

// Тело события,которое будет получена от broker LLM сервиса
type TaskLLMStatusEvent struct {
	Key     TaskContractKey
	Headers TaskContractHeaders
	Value   TaskLLMStatusEventPayload
}

type TaskLLMStatusEventPayload struct {
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// NewTaskLLMStatusEvent создает событие статуса задачи LLM
func NewTaskLLMStatusEvent(task *TaskLLM) *TaskLLMStatusEvent {
	return &TaskLLMStatusEvent{
		Key: TaskContractKey{
			TaskID: task.TaskID,
		},
		Headers: TaskContractHeaders{
			Status:   task.Status.ToKafkaCode(),
			TraceID:  task.TraceID,
			WorkerID: task.WorkerID,
		},
		Value: TaskLLMStatusEventPayload{
			Result:       task.Result,
			ErrorMessage: task.ErrorMessage,
		},
	}
}
