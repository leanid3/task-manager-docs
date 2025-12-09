package postgres

import (
	"app/internal/entity/domain"
	"app/internal/repository"
	"app/pkg/database"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TaskPostgresAdapter struct {
	db database.DB
}

func NewTaskPostgresAdapter(pool *pgxpool.Pool) repository.TaskRepository {
	return &TaskPostgresAdapter{
		db: pool,
	}
}

func NewTaskPostgresAdapterWithDB(db database.DB) repository.TaskRepository {
	return &TaskPostgresAdapter{
		db: db,
	}
}

func (r *TaskPostgresAdapter) Create(ctx context.Context, task *domain.Task) error {
	query := `
        INSERT INTO tasks (
            task_id, task_type, status, input_metadata, 
            s3_input_path, created_at, retry_count, max_retries
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
	inputMetadata, err := json.Marshal(task.InputMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal input metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query, task.TaskID, task.TaskType, task.Status, inputMetadata, task.S3InputPath, task.CreatedAt, task.RetryCount, task.MaxRetries)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	return nil
}

func (r *TaskPostgresAdapter) GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	query := `
        SELECT task_id, task_type, status, input_metadata, output_metadata,
               s3_input_path, s3_output_path, worker_id, created_at, 
               started_at, completed_at, error_message, retry_count, max_retries
        FROM tasks
        WHERE task_id = $1
    `

	var task domain.Task
	var inputMetadataJSON, outputMetadataJSON []byte
	var workerID *uuid.UUID
	var StartedAt, CompletedAt *time.Time

	err := r.db.QueryRow(ctx, query, taskID).Scan(
		&task.TaskID, &task.TaskType, &task.Status, &inputMetadataJSON, &outputMetadataJSON,
		&task.S3InputPath, &task.S3OutputPath, &workerID, &task.CreatedAt,
		&StartedAt, &CompletedAt, &task.ErrorMessage, &task.RetryCount, &task.MaxRetries,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if len(inputMetadataJSON) > 0 {
		if err := json.Unmarshal(inputMetadataJSON, &task.InputMetadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input metadata: %w", err)
		}
	}
	if len(outputMetadataJSON) > 0 {
		if err := json.Unmarshal(outputMetadataJSON, &task.OutputMetadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal output metadata: %w", err)
		}
	}

	task.WorkerID = workerID
	task.StartedAt = StartedAt
	task.CompletedAt = CompletedAt

	return &task, nil
}

func (r *TaskPostgresAdapter) UpdateStatus(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus) error {
	query := `
        UPDATE tasks 
        SET status = $1, 
            started_at = CASE WHEN status = 'PENDING' AND $1 = 'PROCESSING' THEN NOW() ELSE started_at END,
            completed_at = CASE WHEN $1 IN ('COMPLETED', 'FAILED') THEN NOW() ELSE completed_at END
        WHERE task_id = $2
    `
	result, err := r.db.Exec(ctx, query, status, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil

}

func (r *TaskPostgresAdapter) UpdateWithWorker(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, workerID uuid.UUID) error {
	query := `
        UPDATE tasks 
        SET status = $1, worker_id = $2, started_at = NOW()
        WHERE task_id = $3
    `

	result, err := r.db.Exec(ctx, query, status, workerID, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task with worker: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil
}

func (r *TaskPostgresAdapter) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error) {
	query := `
	SELECT task_id, task_type, status, s3_input_path, created_at
	FROM tasks
	WHERE status = $1
	ORDER BY created_at DESC
	LIMIT $2
`
	rows, err := r.db.Query(ctx, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks by status: %w", err)
	}
	defer rows.Close()

	var tasks []*domain.Task
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.TaskID, &task.TaskType, &task.Status, &task.S3InputPath, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}
	return tasks, rows.Err()
}
