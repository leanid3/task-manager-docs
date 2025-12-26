package domain

import (
	"encoding/json"
)

// Сообщение для брокера
type TaskLLMCommand struct {
	Key     TaskContractKey
	Headers TaskContractHeaders
	Value   TaskLLMCommandPayload
}

// тело сообщения,которое будет отправлено в broker по задаче LLM
type TaskLLMCommandPayload struct {
	StoragePath string      `json:"storage_path"`
	StorageSize int64       `json:"storage_size,omitempty"` //необязательно
	Metadata    LLMMetadata `json:"metadata,omitempty"`     //необязательно
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

// Сообщение ожидаемое от брокера - consumer
type TaskLLMStatusEvent struct {
	Key     TaskContractKey
	Headers TaskContractHeaders
	Value   TaskLLMStatusEventPayload
}

// Тело события,которое будет получена от broker LLM сервиса
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
