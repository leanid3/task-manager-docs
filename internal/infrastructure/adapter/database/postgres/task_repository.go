package postgres

import (
	"app/internal/entity/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TaskRepository реализует интерфейс repository.Task
type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: pool}
}

func (r *TaskRepository) Create(ctx context.Context, task domain.Task) error {
	query := `
        INSERT INTO tasks (
            task_id, status, request_id, trace_id, created_at, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `
	metadataJSON, err := json.Marshal(task.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to marshal input metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query, task.GetID(), task.GetStatus(), task.GetRequestID(), task.GetTraceID(), task.GetCreatedAt(), metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, taskID uuid.UUID) (domain.Task, error) {
	query := `
        SELECT task_id, status, metadata,
               worker_id, created_at, request_id, trace_id,
               started_at, completed_at, error_message, result
        FROM tasks
        WHERE task_id = $1
    `

	var baseTask domain.BaseTask
	var metadataJSON, resultJSON []byte
	var workerID, requestID, traceID, errorMessage *string
	var startedAt, completedAt *time.Time

	err := r.db.QueryRow(ctx, query, taskID).Scan(
		&baseTask.TaskID, &baseTask.Status, &metadataJSON,
		&workerID, &baseTask.CreatedAt, &requestID, &traceID,
		&startedAt, &completedAt, &errorMessage, &resultJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &baseTask.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input metadata: %w", err)
		}
	}

	if workerID != nil {
		baseTask.WorkerID = *workerID
	}
	baseTask.StartedAt = startedAt
	baseTask.CompletedAt = completedAt

	if requestID != nil {
		baseTask.RequestID = *requestID
	}
	if traceID != nil {
		traceIDUUID, err := uuid.Parse(*traceID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trace ID: %w", err)
		}
		baseTask.TraceID = &traceIDUUID
	}
	if errorMessage != nil {
		baseTask.ErrorMessage = *errorMessage
	}
	if len(resultJSON) > 0 {
		if err := json.Unmarshal(resultJSON, &baseTask.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result worker: %w", err)
		}
	}

	return &baseTask, nil
}

func (r *TaskRepository) UpdateWithStatus(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus) error {
	query := `
        UPDATE tasks
        SET status = $1::VARCHAR,
            started_at = CASE
                WHEN status = 'PENDING' AND $1::VARCHAR = 'PROCESSING' THEN GREATEST(NOW(), created_at)
                WHEN started_at IS NULL AND $1::VARCHAR IN ('COMPLETED', 'FAILED') THEN GREATEST(NOW(), created_at)
                ELSE started_at
            END,
            completed_at = CASE
                WHEN $1::VARCHAR IN ('COMPLETED', 'FAILED') THEN GREATEST(NOW(), created_at, COALESCE(started_at, created_at))
                ELSE completed_at
            END, worker_id = $2
        WHERE task_id = $3
    `
	result, err := r.db.Exec(ctx, query, status, worker_id, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil
}

func (r *TaskRepository) UpdateWithResult(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus, result json.RawMessage) error {
	query := `
		UPDATE tasks
		SET result = $1, status = $2, worker_id = $3,
		    completed_at = GREATEST(NOW(), created_at, COALESCE(started_at, created_at))
		WHERE task_id = $4
	`
	cmdTag, err := r.db.Exec(ctx, query, result, status, worker_id, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task result: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil
}

func (r *TaskRepository) UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error {
	query := `
		UPDATE tasks
		SET error_message = $1, status = $2,
		    completed_at = GREATEST(NOW(), created_at, COALESCE(started_at, created_at))
		WHERE task_id = $3
	`

	result, err := r.db.Exec(ctx, query, errorMessage, status, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task error: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil
}

func (r *TaskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]domain.Task, error) {
	query := `
	SELECT task_id, status, created_at
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

	var tasks []domain.Task
	for rows.Next() {
		var baseTask domain.BaseTask
		if err := rows.Scan(&baseTask.TaskID, &baseTask.Status, &baseTask.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &baseTask)
	}
	return tasks, rows.Err()
}
