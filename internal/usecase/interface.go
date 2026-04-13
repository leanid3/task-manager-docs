// Package usecase содержит бизнес-логику приложения.
// Каждый usecase — независимый модуль со своим интерфейсом.
package usecase

import (
	"app/internal/entity/domain"
	"context"
	"io"

	"github.com/google/uuid"
)

// TaskLLMUCInterface интерфейс для TaskLLMUC
type TaskLLMUCInterface interface {
	CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, evt domain.BrokerCommand[domain.TaskLLMStatusEventPayload]) error
}
