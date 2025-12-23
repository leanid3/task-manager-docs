//go:build integration

package broker

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/usecase"
	"context"
	"io"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

func TestTaskLLMHandleTaskStatusLLM(t *testing.T) {
	// Создаем mock logger и usecase для handler
	mockLogger := &mockLogger{}
	mockUsecase := &mockTaskLLMUC{}

	handler := &KafkaMessageHandler{
		uc: usecase.UseCases{TaskLLMUC: mockUsecase},
		l:  mockLogger,
	}

	tests := []struct {
		name    string
		msg     *kafka.Message
		wantErr bool
	}{
		{
			name: "success - valid task_id and status",
			msg: &kafka.Message{
				Key:   []byte(uuid.New().String()),
				Value: []byte(`{"result": {"key": "value"}}`),
				Headers: []kafka.Header{
					{Key: "status", Value: []byte("COMPLETED")},
				},
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("test-topic")},
			},
			wantErr: false,
		},
		{
			name: "error - invalid task_id format",
			msg: &kafka.Message{
				Key:            []byte("invalid_task_id"),
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("test-topic")},
			},
			wantErr: true,
		},
		{
			name: "error - empty task_id",
			msg: &kafka.Message{
				Key:            []byte(""),
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("test-topic")},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandleTaskStatusLLM(context.Background(), tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTaskStatusLLM() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				// Проверяем, что ошибка правильного типа
				if appErr, ok := apperrors.IsAppError(err); ok {
					if appErr.Code != apperrors.CodeInvalidMessageFormat {
						t.Errorf("HandleTaskStatusLLM() error code = %v, want %v", appErr.Code, apperrors.CodeInvalidMessageFormat)
					}
				} else {
					t.Errorf("HandleTaskStatusLLM() expected AppError, got %T", err)
				}
			}
		})
	}
}

// Вспомогательные функции и моки

func stringPtr(s string) *string {
	return &s
}

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...interface{})                   {}
func (m *mockLogger) Info(msg string, args ...interface{})                    {}
func (m *mockLogger) Warn(msg string, args ...interface{})                    {}
func (m *mockLogger) Error(msg string, args ...interface{})                   {}
func (m *mockLogger) ErrorWithSkip(skip int, msg string, args ...interface{}) {}
func (m *mockLogger) Fatal(msg string, args ...interface{})                   {}

type mockTaskLLMUC struct{}

func (m *mockTaskLLMUC) CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockTaskLLMUC) GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	return nil, nil
}

func (m *mockTaskLLMUC) UpdateTaskStatusProcessing(ctx context.Context, evt domain.TaskStatusEvent) error {
	return nil
}

func (m *mockTaskLLMUC) UpdateTaskStatusCompleted(ctx context.Context, evt domain.TaskStatusEvent) error {
	return nil
}

func (m *mockTaskLLMUC) UpdateTaskStatusFailed(ctx context.Context, evt domain.TaskStatusEvent) error {
	return nil
}
