package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusProcessing TaskStatus = "PROCESSING"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusFailed     TaskStatus = "FAILED"
)

type TaskType string

const (
	TaskTypeParsing   TaskType = "parsing"
	TaskTypeAlgorithm TaskType = "algorithm"
	TaskTypeLLM       TaskType = "llm"
)

type Task struct {
	TaskID         uuid.UUID              `json:"task_id"`
	TaskType       TaskType               `json:"task_type"`
	Status         TaskStatus             `json:"status"`
	InputMetadata  map[string]interface{} `json:"input_metadata"`
	OutputMetadata map[string]interface{} `json:"output_metadata"`
	S3InputPath    string                 `json:"s3_input_path"`
	S3OutputPath   string                 `json:"s3_output_path"`
	WorkerID       *uuid.UUID             `json:"worker_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	RetryCount     int                    `json:"retry_count"`
	MaxRetries     int                    `json:"max_retries"`
}
