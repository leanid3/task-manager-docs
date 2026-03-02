package domain

import (
	"encoding/json"
)

// тело сообщения,которое будет отправлено в broker по задаче LLM
type TaskLLMCommandPayload struct {
	StoragePath string      `json:"storage_path"`
	StorageSize int64       `json:"storage_size,omitempty"` //необязательно
	Metadata    LLMMetadata `json:"metadata,omitempty"`     //необязательно
}

// NewTaskLLMCommand создает команду для задачи LLM
func NewTaskLLMCommand(task *TaskLLM) BrokerCommand[TaskLLMCommandPayload] {
	return BrokerCommand[TaskLLMCommandPayload]{
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
type TaskLLMStatusEventPayload struct {
	Result       json.RawMessage `json:"result,omitempty"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// TaskLLMStatusEvent — псевдоним для BrokerCommand с payload TaskLLMStatusEventPayload
type TaskLLMStatusEvent = BrokerCommand[TaskLLMStatusEventPayload]

// NewTaskLLMStatusEvent создает событие статуса задачи LLM
func NewTaskLLMStatusEvent(task *TaskLLM) BrokerCommand[TaskLLMStatusEventPayload] {
	return BrokerCommand[TaskLLMStatusEventPayload]{
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
