package usecase

import (
	"app/internal/entity/domain"
	"app/pkg/metrics"
	"context"
	"time"

	"github.com/google/uuid"
)

// MetricsDecorator декоратор для добавления метрик к usecase
type MetricsDecorator struct {
	next    TaskManager
	metrics metrics.Interface
}

// NewMetricsDecorator создает новый декоратор метрик
func NewMetricsDecorator(next TaskManager, metrics metrics.Interface) *MetricsDecorator {
	return &MetricsDecorator{
		next:    next,
		metrics: metrics,
	}
}

// CreateTask создает задачу с измерением метрик
func (md *MetricsDecorator) CreateTask(ctx context.Context, input TaskInput) (uuid.UUID, error) {
	start := time.Now()
	taskID, err := md.next.CreateTask(ctx, input)
	duration := time.Since(start)

	md.metrics.ObserveTaskProcessingDuration("task", "create", duration.Seconds())

	if err != nil {
		md.metrics.IncTasksTotal("task", "create_error")
	} else {
		md.metrics.IncTasksTotal("task", "create_success")
	}

	return taskID, err
}

// GetTaskByID возвращает задачу по ID с измерением метрик
func (md *MetricsDecorator) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	start := time.Now()
	task, err := md.next.GetTaskByID(ctx, id)
	duration := time.Since(start)

	md.metrics.ObserveTaskProcessingDuration("task", "get", duration.Seconds())

	if err != nil {
		md.metrics.IncTasksTotal("task", "get_error")
	} else {
		md.metrics.IncTasksTotal("task", "get_success")
	}

	return task, err
}

// UpdateTaskStatus обновляет статус задачи с измерением метрик
func (md *MetricsDecorator) UpdateTaskStatus(ctx context.Context, event TaskEvent) error {
	start := time.Now()
	err := md.next.UpdateTaskStatus(ctx, event)
	duration := time.Since(start)

	md.metrics.ObserveTaskProcessingDuration("task", "update_status", duration.Seconds())

	if err != nil {
		md.metrics.IncTasksTotal("task", "update_status_error")
	} else {
		md.metrics.IncTasksTotal("task", "update_status_success")
	}

	return err
}

// ProcessTask обрабатывает задачу с измерением метрик
func (md *MetricsDecorator) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	start := time.Now()
	err := md.next.ProcessTask(ctx, taskID)
	duration := time.Since(start)

	md.metrics.ObserveTaskProcessingDuration("task", "process", duration.Seconds())

	if err != nil {
		md.metrics.IncTasksTotal("task", "process_error")
	} else {
		md.metrics.IncTasksTotal("task", "process_success")
	}

	return err
}
