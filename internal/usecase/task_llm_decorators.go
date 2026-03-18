package usecase

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/pkg/logger"
	"app/pkg/metrics"
	"context"
	"io"
	"time"

	"github.com/google/uuid"
)

// LoggingDecoratorTaskLLM декоратор для добавления логирования к TaskLLMUC
type LoggingDecoratorTaskLLM struct {
	next   TaskLLMUCInterface
	logger logger.Interface
}

// NewLoggingDecoratorTaskLLM создает новый декоратор логирования для TaskLLMUC
func NewLoggingDecoratorTaskLLM(next TaskLLMUCInterface, logger logger.Interface) *LoggingDecoratorTaskLLM {
	return &LoggingDecoratorTaskLLM{
		next:   next,
		logger: logger,
	}
}

// CreateTask создает задачу с логированием
func (ld *LoggingDecoratorTaskLLM) CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error) {
	ld.logger.Info("creating LLM task", "request_id", requestID, "filename", filename, "filesize", filesize)
	taskID, err := ld.next.CreateTask(ctx, reader, filename, filesize, requestID)
	if err != nil {
		ld.logger.Error("failed to create LLM task", "error", err, "request_id", requestID, "filename", filename)
	} else {
		ld.logger.Info("LLM task created successfully", "task_id", taskID, "request_id", requestID)
	}
	return taskID, err
}

// GetTaskByID возвращает задачу по ID с логированием
func (ld *LoggingDecoratorTaskLLM) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	ld.logger.Debug("getting LLM task by ID", "task_id", id)
	task, err := ld.next.GetTaskByID(ctx, id)
	if err != nil {
		ld.logger.Error("failed to get LLM task", "error", err, "task_id", id)
	} else {
		ld.logger.Debug("LLM task retrieved successfully", "task_id", id, "status", task.GetStatus())
	}
	return task, err
}

// UpdateTaskStatus обновляет статус задачи с логированием
func (ld *LoggingDecoratorTaskLLM) UpdateTaskStatus(ctx context.Context, evt domain.BrokerCommand[domain.TaskLLMStatusEventPayload]) error {
	ld.logger.Debug("updating LLM task status", "task_id", evt.Key.TaskID, "status_code", evt.Headers.Status)
	err := ld.next.UpdateTaskStatus(ctx, evt)
	if err != nil {
		ld.logger.Error("failed to update LLM task status", "error", err, "task_id", evt.Key.TaskID)
	} else {
		ld.logger.Info("LLM task status updated successfully", "task_id", evt.Key.TaskID, "status_code", evt.Headers.Status)
	}
	return err
}

// MetricsDecoratorTaskLLM декоратор для добавления метрик к TaskLLMUC
type MetricsDecoratorTaskLLM struct {
	next    TaskLLMUCInterface
	metrics metrics.Interface
}

// NewMetricsDecoratorTaskLLM создает новый декоратор метрик для TaskLLMUC
func NewMetricsDecoratorTaskLLM(next TaskLLMUCInterface, metrics metrics.Interface) *MetricsDecoratorTaskLLM {
	return &MetricsDecoratorTaskLLM{
		next:    next,
		metrics: metrics,
	}
}

// CreateTask создает задачу с измерением метрик
func (md *MetricsDecoratorTaskLLM) CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error) {
	start := time.Now()
	taskID, err := md.next.CreateTask(ctx, reader, filename, filesize, requestID)
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
		if appErr, ok := apperrors.IsAppError(err); ok {
			status = string(appErr.Code)
		}
	}

	md.metrics.ObserveTaskProcessingDuration("llm", "create", duration.Seconds())
	md.metrics.IncTasksTotal("llm", status)

	return taskID, err
}

// GetTaskByID возвращает задачу по ID с измерением метрик
func (md *MetricsDecoratorTaskLLM) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	start := time.Now()
	task, err := md.next.GetTaskByID(ctx, id)
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
		if appErr, ok := apperrors.IsAppError(err); ok {
			status = string(appErr.Code)
		}
	}

	md.metrics.ObserveTaskProcessingDuration("llm", "get", duration.Seconds())
	md.metrics.IncTasksTotal("llm", status)

	return task, err
}

// UpdateTaskStatus обновляет статус задачи с измерением метрик
func (md *MetricsDecoratorTaskLLM) UpdateTaskStatus(ctx context.Context, evt domain.BrokerCommand[domain.TaskLLMStatusEventPayload]) error {
	start := time.Now()
	err := md.next.UpdateTaskStatus(ctx, evt)
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
		if appErr, ok := apperrors.IsAppError(err); ok {
			status = string(appErr.Code)
		}
	}

	md.metrics.ObserveTaskProcessingDuration("llm", "update_status", duration.Seconds())
	md.metrics.IncTasksTotal("llm", status)

	return err
}
