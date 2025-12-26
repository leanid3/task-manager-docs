package postgres

import (
	"app/internal/entity/domain"
	"app/pkg/database"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TODO обудмать добавление транзакции
type TaskRepository struct {
	db database.DB
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: pool}
}

func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) error {
	query := `
        INSERT INTO tasks (
            task_id, status, request_id, trace_id, created_at, metadata
        ) VALUES ($1, $2, $3, $4, $5, $6)
    `
	metadataJSON, err := json.Marshal(task.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal input metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query, task.TaskID, task.Status, task.RequestID, task.TraceID, task.CreatedAt, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	query := `
        SELECT task_id, status, metadata,
               worker_id, created_at, request_id, trace_id,
               started_at, completed_at, error_message, result
        FROM tasks
        WHERE task_id = $1
    `

	var task domain.Task
	var metadataJSON []byte
	var workerID *string
	var startedAt, completedAt *time.Time
	var requestID, traceID, errorMessage *string
	var resultJSON []byte

	err := r.db.QueryRow(ctx, query, taskID).Scan(
		&task.TaskID, &task.Status, &metadataJSON,
		&workerID, &task.CreatedAt, &requestID, &traceID,
		&startedAt, &completedAt, &errorMessage, &resultJSON,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &task.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal input metadata: %w", err)
		}
	}

	if workerID != nil {
		task.WorkerID = *workerID
	}
	task.StartedAt = startedAt
	task.CompletedAt = completedAt

	if requestID != nil {
		task.RequestID = *requestID
	}
	if traceID != nil {
		traceIDUUID, err := uuid.Parse(*traceID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse trace ID: %w", err)
		}
		task.TraceID = &traceIDUUID
	}
	if errorMessage != nil {
		task.ErrorMessage = *errorMessage
	}
	if len(resultJSON) > 0 {
		if err := json.Unmarshal(resultJSON, &task.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result worker: %w", err)
		}
	}

	return &task, nil
}

func (r *TaskRepository) UpdateWithStatus(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus) error {
	statusStr := string(status)
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
            END
        WHERE task_id = $2
    `
	result, err := r.db.Exec(ctx, query, statusStr, taskID)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("task not found: %s", taskID)
	}
	return nil

}

func (r *TaskRepository) UpdateWithResult(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, result json.RawMessage) error {
	query := `
		UPDATE tasks 
		SET result = $1, status = $2,  
		    completed_at = GREATEST(NOW(), created_at, COALESCE(started_at, created_at))
		WHERE task_id = $3 
	`
	cmdTag, err := r.db.Exec(ctx, query, result, status, taskID)
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

func (r *TaskRepository) UpdateWithWorker(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, workerID string) error {
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

func (r *TaskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error) {
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

	var tasks []*domain.Task
	for rows.Next() {
		var task domain.Task
		if err := rows.Scan(&task.TaskID, &task.Status, &task.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, &task)
	}
	return tasks, rows.Err()
}
