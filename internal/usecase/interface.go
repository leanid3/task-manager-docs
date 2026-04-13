package usecase

import (
	"app/internal/entity/domain"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TaskProcessor интерфейс для обработки задач конкретного типа
type TaskProcessor interface {
	// Process обрабатывает задачу
	Process(ctx context.Context, task domain.Task) error
	// GetType возвращает тип задачи
	GetType() domain.TaskType
	// Validate проверяет задачу на корректность
	Validate(task domain.Task) error
	// GetTimeout возвращает таймаут для задачи
	GetTimeout() time.Duration
}

// TaskProcessorFactory фабрика для создания обработчиков задач
type TaskProcessorFactory interface {
	Create(taskType domain.TaskType) (TaskProcessor, error)
	Register(taskType domain.TaskType, processor TaskProcessor) error
}

// TaskInput входные данные для создания задачи
type TaskInput struct {
	TaskID    uuid.UUID              `json:"task_id"`
	Filename  string                 `json:"filename"`
	Filesize  int64                  `json:"filesize"`
	Reader    interface{}            `json:"-"` // io.Reader не может быть сериализован
	RequestID string                 `json:"request_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TaskEvent универсальное событие для обновления статуса задачи
type TaskEvent struct {
	Key     domain.TaskContractKey     `json:"key"`
	Headers domain.TaskContractHeaders `json:"headers"`
	Value   interface{}                `json:"value"`
}

// TaskResult результат выполнения задачи
type TaskResult struct {
	TaskID       uuid.UUID        `json:"task_id"`
	Status       domain.TaskStatus `json:"status"`
	Result       json.RawMessage  `json:"result,omitempty"`
	ErrorMessage string           `json:"error_message,omitempty"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
}

// TaskManager универсальный интерфейс управления задачами.
// Реализуется конкретными usecase (TaskLLMUC, UnifiedTaskUC, MultiUploadUC и т.д.)
type TaskManager interface {
	// CreateTask создает новую задачу
	CreateTask(ctx context.Context, input TaskInput) (uuid.UUID, error)
	// GetTaskByID возвращает задачу по ID
	GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error)
	// UpdateTaskStatus обновляет статус задачи
	UpdateTaskStatus(ctx context.Context, event TaskEvent) error
	// ProcessTask обрабатывает задачу
	ProcessTask(ctx context.Context, taskID uuid.UUID) error
}
