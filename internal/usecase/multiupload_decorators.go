package usecase

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/usecase/multiupload"
	"app/pkg/logger"
	"app/pkg/metrics"
	"context"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

// LoggingDecoratorMultiUpload декоратор для добавления логирования к MultiUploadUC
type LoggingDecoratorMultiUpload struct {
	next   multiupload.MultiUploadUCInterface
	logger logger.Interface
}

// NewLoggingDecoratorMultiUpload создает новый декоратор логирования для MultiUploadUC
func NewLoggingDecoratorMultiUpload(next multiupload.MultiUploadUCInterface, logger logger.Interface) *LoggingDecoratorMultiUpload {
	return &LoggingDecoratorMultiUpload{
		next:   next,
		logger: logger,
	}
}

// CreateMultiUploadTask создает задачу многофайловой загрузки с логированием
func (ld *LoggingDecoratorMultiUpload) CreateMultiUploadTask(ctx context.Context, files []*multipart.FileHeader, requestID string) (uuid.UUID, error) {
	ld.logger.Info("creating multi-upload task", "request_id", requestID, "file_count", len(files))
	taskID, err := ld.next.CreateMultiUploadTask(ctx, files, requestID)
	if err != nil {
		ld.logger.Error("failed to create multi-upload task", "error", err, "request_id", requestID, "file_count", len(files))
	} else {
		ld.logger.Info("multi-upload task created successfully", "task_id", taskID, "request_id", requestID)
	}
	return taskID, err
}

// GetTaskByID возвращает задачу по ID с логированием
func (ld *LoggingDecoratorMultiUpload) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	ld.logger.Debug("getting multi-upload task by ID", "task_id", id)
	task, err := ld.next.GetTaskByID(ctx, id)
	if err != nil {
		ld.logger.Error("failed to get multi-upload task", "error", err, "task_id", id)
	} else {
		ld.logger.Debug("multi-upload task retrieved successfully", "task_id", id, "status", task.GetStatus())
	}
	return task, err
}

// MetricsDecoratorMultiUpload декоратор для добавления метрик к MultiUploadUC
type MetricsDecoratorMultiUpload struct {
	next    multiupload.MultiUploadUCInterface
	metrics metrics.Interface
}

// NewMetricsDecoratorMultiUpload создает новый декоратор метрик для MultiUploadUC
func NewMetricsDecoratorMultiUpload(next multiupload.MultiUploadUCInterface, metrics metrics.Interface) *MetricsDecoratorMultiUpload {
	return &MetricsDecoratorMultiUpload{
		next:    next,
		metrics: metrics,
	}
}

// CreateMultiUploadTask создает задачу многофайловой загрузки с измерением метрик
func (md *MetricsDecoratorMultiUpload) CreateMultiUploadTask(ctx context.Context, files []*multipart.FileHeader, requestID string) (uuid.UUID, error) {
	start := time.Now()
	taskID, err := md.next.CreateMultiUploadTask(ctx, files, requestID)
	duration := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
		if appErr, ok := apperrors.IsAppError(err); ok {
			status = string(appErr.Code)
		}
	}

	md.metrics.ObserveTaskProcessingDuration("multiupload", "create", duration.Seconds())
	md.metrics.IncTasksTotal("multiupload", status)

	return taskID, err
}

// GetTaskByID возвращает задачу по ID с измерением метрик
func (md *MetricsDecoratorMultiUpload) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
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

	md.metrics.ObserveTaskProcessingDuration("multiupload", "get", duration.Seconds())
	md.metrics.IncTasksTotal("multiupload", status)

	return task, err
}
