package usecase

import (
	"app/internal/entity/domain"
	"app/pkg/logger"
	"context"
	"time"

	"github.com/google/uuid"
)

// LoggingDecorator декоратор для добавления логирования к usecase
type LoggingDecorator struct {
	next   domain.TaskManager
	logger logger.Interface
}

// NewLoggingDecorator создает новый декоратор логирования
func NewLoggingDecorator(next domain.TaskManager, logger logger.Interface) *LoggingDecorator {
	return &LoggingDecorator{
		next:   next,
		logger: logger,
	}
}

// CreateTask создает задачу с логированием
func (ld *LoggingDecorator) CreateTask(ctx context.Context, input domain.TaskInput) (uuid.UUID, error) {
	ld.logger.Info("creating task", "request_id", input.RequestID)
	taskID, err := ld.next.CreateTask(ctx, input)
	if err != nil {
		ld.logger.Error("failed to create task", "error", err, "request_id", input.RequestID)
	} else {
		ld.logger.Info("task created successfully", "task_id", taskID, "request_id", input.RequestID)
	}
	return taskID, err
}

// GetTaskByID возвращает задачу по ID с логированием
func (ld *LoggingDecorator) GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	ld.logger.Debug("getting task by ID", "task_id", id)
	task, err := ld.next.GetTaskByID(ctx, id)
	if err != nil {
		ld.logger.Error("failed to get task", "error", err, "task_id", id)
	} else {
		ld.logger.Debug("task retrieved successfully", "task_id", id, "status", task.Status)
	}
	return task, err
}

// UpdateTaskStatus обновляет статус задачи с логированием
func (ld *LoggingDecorator) UpdateTaskStatus(ctx context.Context, event domain.TaskEvent) error {
	ld.logger.Debug("updating task status", "task_id", event.Key.TaskID, "status_code", event.Headers.Status)
	err := ld.next.UpdateTaskStatus(ctx, event)
	if err != nil {
		ld.logger.Error("failed to update task status", "error", err, "task_id", event.Key.TaskID)
	} else {
		ld.logger.Info("task status updated successfully", "task_id", event.Key.TaskID, "status_code", event.Headers.Status)
	}
	return err
}

// ProcessTask обрабатывает задачу с логированием
func (ld *LoggingDecorator) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	ld.logger.Info("processing task", "task_id", taskID)
	start := time.Now()
	err := ld.next.ProcessTask(ctx, taskID)
	duration := time.Since(start)

	if err != nil {
		ld.logger.Error("failed to process task", "error", err, "task_id", taskID, "duration", duration)
	} else {
		ld.logger.Info("task processed successfully", "task_id", taskID, "duration", duration)
	}
	return err
}