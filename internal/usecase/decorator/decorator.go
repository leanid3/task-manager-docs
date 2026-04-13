// Package decorator предоставляет универсальные декораторы для любого usecase.
// Каждый декоратор реализует паттерн Decorator — добавляет поведение (логирование, метрики)
// без изменения основной логики usecase.
package decorator

import (
	"app/internal/entity/domain"
	"app/internal/usecase"
	"app/pkg/logger"
	"app/pkg/metrics"
	"context"
	"time"

	"github.com/google/uuid"
)

// UseCase — интерфейс любого usecase, который поддерживает стандартные операции
type UseCase interface {
	CreateTask(ctx context.Context, input usecase.TaskInput) (uuid.UUID, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, event usecase.TaskEvent) error
	ProcessTask(ctx context.Context, taskID uuid.UUID) error
}

// Logging — декоратор логирования для любого UseCase
type Logging struct {
	next   UseCase
	logger logger.Interface
}

func NewLogging(next UseCase, l logger.Interface) *Logging {
	return &Logging{next: next, logger: l}
}

func (d *Logging) CreateTask(ctx context.Context, input usecase.TaskInput) (uuid.UUID, error) {
	d.logger.Info("creating task", "request_id", input.RequestID)
	taskID, err := d.next.CreateTask(ctx, input)
	if err != nil {
		d.logger.Error("failed to create task", "error", err, "request_id", input.RequestID)
	} else {
		d.logger.Info("task created", "task_id", taskID)
	}
	return taskID, err
}

func (d *Logging) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	d.logger.Debug("getting task", "task_id", id)
	task, err := d.next.GetTaskByID(ctx, id)
	if err != nil {
		d.logger.Error("failed to get task", "error", err, "task_id", id)
	}
	return task, err
}

func (d *Logging) UpdateTaskStatus(ctx context.Context, event usecase.TaskEvent) error {
	d.logger.Debug("updating task status", "task_id", event.Key.TaskID)
	err := d.next.UpdateTaskStatus(ctx, event)
	if err != nil {
		d.logger.Error("failed to update status", "error", err, "task_id", event.Key.TaskID)
	}
	return err
}

func (d *Logging) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	d.logger.Info("processing task", "task_id", taskID)
	start := time.Now()
	err := d.next.ProcessTask(ctx, taskID)
	d.logger.Info("task processed", "task_id", taskID, "duration", time.Since(start))
	return err
}

// Metrics — декоратор метрик для любого UseCase
type Metrics struct {
	next    UseCase
	metrics metrics.Interface
	name    string // имя usecase для метрик (например "unified", "llm")
}

func NewMetrics(next UseCase, m metrics.Interface, name string) *Metrics {
	return &Metrics{next: next, metrics: m, name: name}
}

func (d *Metrics) CreateTask(ctx context.Context, input usecase.TaskInput) (uuid.UUID, error) {
	start := time.Now()
	taskID, err := d.next.CreateTask(ctx, input)
	d.record("create", time.Since(start), err)
	return taskID, err
}

func (d *Metrics) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	start := time.Now()
	task, err := d.next.GetTaskByID(ctx, id)
	d.record("get", time.Since(start), err)
	return task, err
}

func (d *Metrics) UpdateTaskStatus(ctx context.Context, event usecase.TaskEvent) error {
	start := time.Now()
	err := d.next.UpdateTaskStatus(ctx, event)
	d.record("update_status", time.Since(start), err)
	return err
}

func (d *Metrics) ProcessTask(ctx context.Context, taskID uuid.UUID) error {
	start := time.Now()
	err := d.next.ProcessTask(ctx, taskID)
	d.record("process", time.Since(start), err)
	return err
}

func (d *Metrics) record(operation string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}
	d.metrics.ObserveTaskProcessingDuration(d.name, operation, duration.Seconds())
	d.metrics.IncTasksTotal(d.name, status)
}
