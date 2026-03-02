package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskProcessor интерфейс для обработки задач
type TaskProcessor interface {
	// Process обрабатывает задачу
	Process(ctx context.Context, task Task) error
	// GetType возвращает тип задачи
	GetType() TaskType
	// Validate проверяет задачу на корректность
	Validate(task Task) error
	// GetTimeout возвращает таймаут для задачи
	GetTimeout() time.Duration
}


// TaskEvent универсальное событие для всех типов задач
type TaskEvent struct {
	Key     TaskContractKey      `json:"key"`
	Headers TaskContractHeaders  `json:"headers"`
	Value   interface{}          `json:"value"`
}

// TaskType тип задачи
type TaskType string

const (
	TaskTypeLLM       TaskType = "llm"
	TaskTypeParsing   TaskType = "parsing"
	TaskTypeAlgorithms TaskType = "algorithms"
	TaskTypeAnalyze   TaskType = "analyze"
)

// TaskDefinition определяет спецификацию задачи
type TaskDefinition struct {
	Type         TaskType           `json:"type"`
	Name         string             `json:"name"`
	Topic        string             `json:"topic"`
	Timeout      time.Duration      `json:"timeout"`
	MaxRetries   int                `json:"max_retries"`
	StoragePath  string             `json:"storage_path"`
	Enabled      bool               `json:"enabled"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// TaskResult результат выполнения задачи
type TaskResult struct {
	TaskID       uuid.UUID         `json:"task_id"`
	Status       TaskStatus        `json:"status"`
	Result       json.RawMessage   `json:"result,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// TaskInput входные данные для задачи
type TaskInput struct {
	TaskID    uuid.UUID                    `json:"task_id"`
	Filename  string                       `json:"filename"`
	Filesize  int64                        `json:"filesize"`
	Reader    interface{}                  `json:"-"` // io.Reader не может быть сериализован
	RequestID string                       `json:"request_id"`
	Metadata  map[string]interface{}       `json:"metadata,omitempty"`
}

// TaskProcessorFactory фабрика для создания обработчиков задач
type TaskProcessorFactory interface {
	Create(taskType TaskType) (TaskProcessor, error)
	Register(taskType TaskType, processor TaskProcessor) error
}

// TaskManager интерфейс для управления задачами
type TaskManager interface {
	// CreateTask создает новую задачу
	CreateTask(ctx context.Context, input TaskInput) (uuid.UUID, error)
	// GetTaskByID возвращает задачу по ID
	GetTaskByID(ctx context.Context, id uuid.UUID) (Task, error)
	// UpdateTaskStatus обновляет статус задачи
	UpdateTaskStatus(ctx context.Context, event TaskEvent) error
	// ProcessTask обрабатывает задачу
	ProcessTask(ctx context.Context, taskID uuid.UUID) error
}