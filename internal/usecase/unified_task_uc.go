package usecase

import (
	"app/internal/entity/domain"
	"app/internal/entity/repository"
	"app/pkg/logger"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UnifiedTaskUC универсальный usecase для обработки различных типов задач
type UnifiedTaskUC struct {
	taskRepo      repository.Task
	storageRepo   repository.Storage
	taskProcessorFactory domain.TaskProcessorFactory
	logger        logger.Interface
}

// NewUnifiedTaskUC создает новый универсальный usecase для задач
func NewUnifiedTaskUC(
	taskRepo repository.Task,
	storageRepo repository.Storage,
	taskProcessorFactory domain.TaskProcessorFactory,
	logger logger.Interface,
) *UnifiedTaskUC {
	return &UnifiedTaskUC{
		taskRepo:      taskRepo,
		storageRepo:   storageRepo,
		taskProcessorFactory: taskProcessorFactory,
		logger:        logger,
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
	task := &domain.Task{
		TaskID:    taskID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now(),
		TraceID:   &traceID,
		RequestID: input.RequestID,
		Metadata:  input.Metadata,
	}

	// Сохраняем задачу в базе данных
	if err := uc.taskRepo.Create(ctx, task); err != nil {
		uc.logger.Error("failed to create task in database",
			"error", err,
			"task_id", taskID,
			"request_id", input.RequestID,
		)
		return uuid.Nil, err
	}

	uc.logger.Info("task created successfully",
		"task_id", taskID,
		"status", domain.TaskStatusPending,
		"request_id", input.RequestID,
	)

	return taskID, nil
}

// GetTaskByID возвращает задачу по ID
func (uc *UnifiedTaskUC) GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	uc.logger.Debug("getting task by ID", "task_id", id)

	task, err := uc.taskRepo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Error("failed to get task from database",
			"error", err,
			"task_id", id,
		)
		return nil, err
	}

	uc.logger.Debug("task retrieved successfully", "task_id", id, "status", task.Status)
	return task, nil
}

// UpdateTaskStatus обновляет статус задачи
func (uc *UnifiedTaskUC) UpdateTaskStatus(ctx context.Context, event domain.TaskEvent) error {
	taskID := event.Key.TaskID
	
	uc.logger.Debug("updating task status",
		"task_id", taskID,
		"status_code", event.Headers.Status,
	)

	// Преобразуем статус из числового кода в enum
	taskStatus, ok := domain.TaskStatus("").FromKafkaCode(event.Headers.Status)
	if !ok {
		uc.logger.Error("unknown task status code",
			"task_id", taskID,
			"status_code", event.Headers.Status,
		)
		return nil // или возвращаем ошибку в зависимости от требований
	}

	// Получаем текущую задачу
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		uc.logger.Error("failed to get task for status update",
			"error", err,
			"task_id", taskID,
		)
		return err
	}

	// Обновляем статус в зависимости от типа
	switch taskStatus {
	case domain.TaskStatusProcessing:
		if task.Status == domain.TaskStatusProcessing {
			uc.logger.Warn("task already processing", "task_id", taskID)
			return nil
		}
		return uc.taskRepo.UpdateWithStatus(ctx, taskID, event.Headers.WorkerID, domain.TaskStatusProcessing)
	case domain.TaskStatusCompleted:
		if task.Status == domain.TaskStatusCompleted {
			uc.logger.Warn("task already completed", "task_id", taskID)
			return nil
		}
		// Извлекаем результат из события
		result, ok := event.Value.(map[string]interface{})
		if !ok {
			uc.logger.Error("invalid result format in event", "task_id", taskID)
			return nil
		}
		jsonResult, err := json.Marshal(result)
		if err != nil {
			uc.logger.Error("failed to marshal result", "error", err, "task_id", taskID)
			return err
		}
		return uc.taskRepo.UpdateWithResult(ctx, taskID, event.Headers.WorkerID, domain.TaskStatusCompleted, jsonResult)
	case domain.TaskStatusFailed:
		if task.Status == domain.TaskStatusFailed {
			uc.logger.Warn("task already failed", "task_id", taskID)
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
		uc.logger.Debug("task status is pending, no update needed", "task_id", taskID)
		return nil
	case domain.TaskStatusCancelled:
		uc.logger.Debug("task status is cancelled, no update needed", "task_id", taskID)
		return nil
	default:
		uc.logger.Error("unknown task status", "task_id", taskID, "status", taskStatus)
		return nil
	}
}

// ProcessTask обрабатывает задачу с использованием соответствующего процессора
func (uc *UnifiedTaskUC) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	// Получаем задачу из базы данных
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		uc.logger.Error("failed to get task for processing", "error", err, "task_id", taskID)
		return err
	}

	// Определяем тип задачи из метаданных
	taskTypeStr, exists := task.Metadata["task_type"].(string)
	if !exists {
		uc.logger.Error("task type not specified in metadata", "task_id", taskID)
		return nil
	}

	taskType := domain.TaskType(taskTypeStr)

	// Получаем соответствующий процессор задачи
	processor, err := uc.taskProcessorFactory.Create(taskType)
	if err != nil {
		uc.logger.Error("failed to get processor for task type", "error", err, "task_id", taskID, "task_type", taskType)
		return err
	}

	// Обновляем статус на PROCESSING
	if err := uc.taskRepo.UpdateWithStatus(ctx, taskID, "", domain.TaskStatusProcessing); err != nil {
		uc.logger.Error("failed to update task status to processing", "error", err, "task_id", taskID)
		return err
	}

	// Обрабатываем задачу
	if err := processor.Process(ctx, task); err != nil {
		uc.logger.Error("failed to process task", "error", err, "task_id", taskID)
		// Обновляем статус на FAILED
		if updateErr := uc.taskRepo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, err.Error()); updateErr != nil {
			uc.logger.Error("failed to update task status to failed", "error", updateErr, "task_id", taskID)
		}
		return err
	}

	uc.logger.Info("task processed successfully", "task_id", taskID)
	return nil
}