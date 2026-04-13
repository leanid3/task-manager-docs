// Package repository содержит интерфейсы доступа к данным (БД).
// Интерфейс Storage перенесён в package service — это инфраструктурный сервис.
package repository

import (
	"app/internal/entity/domain"
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// Task интерфейс репозитория задач (PostgreSQL)
type Task interface {
	Create(ctx context.Context, task domain.Task) error
	GetByID(ctx context.Context, taskID uuid.UUID) (domain.Task, error)
	ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]domain.Task, error)
	UpdateWithStatus(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus) error
	UpdateWithResult(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus, result json.RawMessage) error
	UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error
}
