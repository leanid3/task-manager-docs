package usecase

import (
	"app/internal/domain"
	"app/internal/repository"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo repository.TaskRepository
}

func NewTaskService(taskRepo repository.TaskRepository) *TaskService {
	return &TaskService{
		taskRepo: taskRepo,
	}
}

func (s *TaskService) CreateTask(ctx context.Context, taskType domain.TaskType, s3Path string, metadata map[string]interface{}) (*domain.Task, error) {
	slog.Debug("creating task", "taskType", taskType, "s3Path", s3Path, "metadata", metadata)
	task := &domain.Task{
		TaskID:        uuid.New(),
		TaskType:      taskType,
		Status:        domain.TaskStatusPending,
		InputMetadata: metadata,
		S3InputPath:   s3Path,
		CreatedAt:     time.Now(),
		RetryCount:    0,
		MaxRetries:    3,
	}
	if err := s.taskRepo.Create(ctx, task); err != nil {
		slog.Error("failed to create task", "error", err)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	slog.Debug("task created", "task", task)
	return task, nil
}

func (s *TaskService) GetTask(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	slog.Debug("getting task", "taskID", taskID)
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		slog.Error("failed to get task", "error", err)
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	slog.Debug("task found", "task", task)
	return task, nil
}
