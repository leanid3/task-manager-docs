package postgres

import (
	"app/internal/entity/domain"
	"app/pkg/metrics"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TaskRepositoryWithMetrics - декоратор для TaskRepository с метриками
type TaskRepositoryWithMetrics struct {
	repo *TaskRepository
}

func NewTaskRepositoryWithMetrics(pool *pgxpool.Pool) *TaskRepositoryWithMetrics {
	return &TaskRepositoryWithMetrics{
		repo: &TaskRepository{db: pool},
	}
}

func (r *TaskRepositoryWithMetrics) Create(ctx context.Context, task *domain.Task) error {
	start := time.Now()
	err := r.repo.Create(ctx, task)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDatabaseQueryDuration("create", "tasks", duration)
		metrics.IncTasksTotal("db_create", "error")
	} else {
		metrics.ObserveDatabaseQueryDuration("create", "tasks", duration)
		metrics.IncTasksTotal("db_create", "success")
	}

	return err
}

func (r *TaskRepositoryWithMetrics) GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	start := time.Now()
	task, err := r.repo.GetByID(ctx, taskID)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDatabaseQueryDuration("select", "tasks", duration)
		if err == pgx.ErrNoRows {
			metrics.IncTasksTotal("db_get_by_id", "not_found")
		} else {
			metrics.IncTasksTotal("db_get_by_id", "error")
		}
	} else {
		metrics.ObserveDatabaseQueryDuration("select", "tasks", duration)
		metrics.IncTasksTotal("db_get_by_id", "success")
	}

	return task, err
}

func (r *TaskRepositoryWithMetrics) UpdateWithStatus(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus) error {
	start := time.Now()
	err := r.repo.UpdateWithStatus(ctx, taskID, worker_id, status)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDatabaseQueryDuration("update", "tasks", duration)
		metrics.IncTasksTotal("db_update_status", "error")
	} else {
		metrics.ObserveDatabaseQueryDuration("update", "tasks", duration)
		metrics.IncTasksTotal("db_update_status", "success")
	}

	return err
}

func (r *TaskRepositoryWithMetrics) UpdateWithResult(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus, result json.RawMessage) error {
	start := time.Now()
	err := r.repo.UpdateWithResult(ctx, taskID, worker_id, status, result)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDatabaseQueryDuration("update", "tasks", duration)
		metrics.IncTasksTotal("db_update_result", "error")
	} else {
		metrics.ObserveDatabaseQueryDuration("update", "tasks", duration)
		metrics.IncTasksTotal("db_update_result", "success")
	}

	return err
}

func (r *TaskRepositoryWithMetrics) UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error {
	start := time.Now()
	err := r.repo.UpdateWithError(ctx, taskID, status, errorMessage)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDatabaseQueryDuration("update", "tasks", duration)
		metrics.IncTasksTotal("db_update_error", "error")
	} else {
		metrics.ObserveDatabaseQueryDuration("update", "tasks", duration)
		metrics.IncTasksTotal("db_update_error", "success")
	}

	return err
}

func (r *TaskRepositoryWithMetrics) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error) {
	start := time.Now()
	tasks, err := r.repo.ListByStatus(ctx, status, limit)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.ObserveDatabaseQueryDuration("select", "tasks", duration)
		metrics.IncTasksTotal("db_list_by_status", "error")
	} else {
		metrics.ObserveDatabaseQueryDuration("select", "tasks", duration)
		metrics.IncTasksTotal("db_list_by_status", "success")
	}

	return tasks, err
}