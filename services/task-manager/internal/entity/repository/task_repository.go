package repository

import (
	"app/internal/entity/domain"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type Task interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error)
	ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error)
	// WithTx(tx database.DB) Task
	UpdateWithStatus(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus) error
	UpdateWithResult(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, result json.RawMessage) error
	UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error
}
