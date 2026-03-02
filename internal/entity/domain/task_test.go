package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTaskStatus_ToDatabaseCode(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		expected string
	}{
		{
			name:     "pending status",
			status:   TaskStatusPending,
			expected: "PENDING",
		},
		{
			name:     "processing status",
			status:   TaskStatusProcessing,
			expected: "PROCESSING",
		},
		{
			name:     "completed status",
			status:   TaskStatusCompleted,
			expected: "COMPLETED",
		},
		{
			name:     "failed status",
			status:   TaskStatusFailed,
			expected: "FAILED",
		},
		{
			name:     "cancelled status",
			status:   TaskStatusCancelled,
			expected: "CANCELLED",
		},
		{
			name:     "unknown status",
			status:   TaskStatus("UNKNOWN"),
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &BaseTask{Status: tt.status}
			result := task.ToDatabaseCode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskStatus_ToKafkaCode(t *testing.T) {
	tests := []struct {
		name     string
		status   TaskStatus
		expected int
	}{
		{
			name:     "pending status",
			status:   TaskStatusPending,
			expected: 1,
		},
		{
			name:     "processing status",
			status:   TaskStatusProcessing,
			expected: 2,
		},
		{
			name:     "completed status",
			status:   TaskStatusCompleted,
			expected: 3,
		},
		{
			name:     "failed status",
			status:   TaskStatusFailed,
			expected: 4,
		},
		{
			name:     "cancelled status",
			status:   TaskStatusCancelled,
			expected: 5,
		},
		{
			name:     "unknown status defaults to failed",
			status:   TaskStatus("UNKNOWN"),
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.ToKafkaCode()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskStatus_FromKafkaCode(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		expectedStatus TaskStatus
		expectedOK     bool
	}{
		{
			name:           "code 1 to pending",
			code:           1,
			expectedStatus: TaskStatusPending,
			expectedOK:     true,
		},
		{
			name:           "code 2 to processing",
			code:           2,
			expectedStatus: TaskStatusProcessing,
			expectedOK:     true,
		},
		{
			name:           "code 3 to completed",
			code:           3,
			expectedStatus: TaskStatusCompleted,
			expectedOK:     true,
		},
		{
			name:           "code 4 to failed",
			code:           4,
			expectedStatus: TaskStatusFailed,
			expectedOK:     true,
		},
		{
			name:           "code 5 to cancelled",
			code:           5,
			expectedStatus: TaskStatusCancelled,
			expectedOK:     true,
		},
		{
			name:           "unknown code defaults to failed",
			code:           99,
			expectedStatus: TaskStatusFailed,
			expectedOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := TaskStatus("").FromKafkaCode(tt.code)
			assert.Equal(t, tt.expectedStatus, result)
			assert.Equal(t, tt.expectedOK, ok)
		})
	}
}

func TestTask_Struct(t *testing.T) {
	now := time.Now()
	traceID := uuid.New()
	taskID := uuid.New()

	task := BaseTask{
		TaskID:       taskID,
		Status:       TaskStatusProcessing,
		Metadata:     map[string]interface{}{"key": "value"},
		WorkerID:     "worker-1",
		RequestID:    "req-123",
		TraceID:      &traceID,
		Result:       json.RawMessage(`{"result": "data"}`),
		CreatedAt:    now,
		StartedAt:    &now,
		CompletedAt:  &now,
		ErrorMessage: "error",
	}

	assert.Equal(t, taskID, task.TaskID)
	assert.Equal(t, TaskStatusProcessing, task.Status)
	assert.Equal(t, map[string]interface{}{"key": "value"}, task.Metadata)
	assert.Equal(t, "worker-1", task.WorkerID)
	assert.Equal(t, "req-123", task.RequestID)
	assert.Equal(t, &traceID, task.TraceID)
	assert.Equal(t, json.RawMessage(`{"result": "data"}`), task.Result)
	assert.Equal(t, now, task.CreatedAt)
	assert.Equal(t, &now, task.StartedAt)
	assert.Equal(t, &now, task.CompletedAt)
	assert.Equal(t, "error", task.ErrorMessage)
}
