package repository

import (
	"app/internal/entity/domain"
	"context"

	"github.com/google/uuid"
)

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error)
	UpdateStatus(ctx context.Context, taskIDchan uuid.UUID, status domain.TaskStatus) error
	UpdateWithWorker(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, workerID uuid.UUID) error
	ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error)
}
