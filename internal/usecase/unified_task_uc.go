package usecase

import (
	"app/internal/entity/domain"
	"app/internal/entity/repository"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UnifiedTaskUC универсальный usecase для обработки различных типов задач
type UnifiedTaskUC struct {
	taskRepo             repository.Task
	storageRepo          repository.Storage
	taskProcessorFactory domain.TaskProcessorFactory
}

// NewUnifiedTaskUC создает новый универсальный usecase для задач
func NewUnifiedTaskUC(
	taskRepo repository.Task,
	storageRepo repository.Storage,
	taskProcessorFactory domain.TaskProcessorFactory,
) *UnifiedTaskUC {
	return &UnifiedTaskUC{
		taskRepo:             taskRepo,
		storageRepo:          storageRepo,
		taskProcessorFactory: taskProcessorFactory,
	}
}

// CreateTask создает задачу любого типа
func (uc *UnifiedTaskUC) CreateTask(ctx context.Context, input domain.TaskInput) (uuid.UUID, error) {
	taskID := input.TaskID
	if taskID == uuid.Nil {
		taskID = uuid.New()
	}

	traceID := uuid.New()

	// Создаем задачу в базе данных
	task := &domain.BaseTask{
		TaskID:    taskID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now(),
		TraceID:   &traceID,
		RequestID: input.RequestID,
		Metadata:  input.Metadata,
	}

	// Сохраняем задачу в базе данных
	if err := uc.taskRepo.Create(ctx, task); err != nil {
		return uuid.Nil, err
	}

	return taskID, nil
}

// GetTaskByID возвращает задачу по ID
func (uc *UnifiedTaskUC) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	task, err := uc.taskRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// UpdateTaskStatus обновляет статус задачи
func (uc *UnifiedTaskUC) UpdateTaskStatus(ctx context.Context, event domain.TaskEvent) error {
	taskID := event.Key.TaskID

	// Преобразуем статус из числового кода в enum
	taskStatus, ok := domain.TaskStatus("").FromKafkaCode(event.Headers.Status)
	if !ok {
		return nil // или возвращаем ошибку в зависимости от требований
	}

	// Получаем текущую задачу
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Обновляем статус в зависимости от типа
	switch taskStatus {
	case domain.TaskStatusProcessing:
		if task.GetStatus() == domain.TaskStatusProcessing {
			return nil // уже в процессе
		}
		return uc.taskRepo.UpdateWithStatus(ctx, taskID, event.Headers.WorkerID, domain.TaskStatusProcessing)
	case domain.TaskStatusCompleted:
		if task.GetStatus() == domain.TaskStatusCompleted {
			return nil
		}
		// Извлекаем результат из события
		result, ok := event.Value.(map[string]interface{})
		if !ok {
			return nil
		}
		jsonResult, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return uc.taskRepo.UpdateWithResult(ctx, taskID, event.Headers.WorkerID, domain.TaskStatusCompleted, jsonResult)
	case domain.TaskStatusFailed:
		if task.GetStatus() == domain.TaskStatusFailed {
			return nil
		}
		// Извлекаем сообщение об ошибке из события
		errorMsg := ""
		if result, ok := event.Value.(map[string]interface{}); ok {
			if errMsg, exists := result["error_message"]; exists {
				errorMsg = errMsg.(string)
			}
		}
		return uc.taskRepo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, errorMsg)
	case domain.TaskStatusPending:
		return nil
	case domain.TaskStatusCancelled:
		return nil
	default:
		return nil
	}
}

// ProcessTask обрабатывает задачу с использованием соответствующего процессора
func (uc *UnifiedTaskUC) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	// Получаем задачу из базы данных
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	// Определяем тип задачи из метаданных
	taskTypeStr, exists := task.GetMetadata()["task_type"].(string)
	if !exists {
		return nil
	}

	taskType := domain.TaskType(taskTypeStr)

	// Получаем соответствующий процессор задачи
	processor, err := uc.taskProcessorFactory.Create(taskType)
	if err != nil {
		return err
	}

	// Обновляем статус на PROCESSING
	if err := uc.taskRepo.UpdateWithStatus(ctx, taskID, "", domain.TaskStatusProcessing); err != nil {
		return err
	}

	// Обрабатываем задачу
	if err := processor.Process(ctx, task); err != nil {
		// Обновляем статус на FAILED
		if updateErr := uc.taskRepo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, err.Error()); updateErr != nil {
			return updateErr
		}
		return err
	}

	return nil
}
