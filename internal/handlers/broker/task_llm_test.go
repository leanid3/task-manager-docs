//go:build integration

package broker

import (
	apperrors "app/internal/entity/errors"
	"app/internal/usecase"
	"app/test/mocks"
	"context"
	"strings"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

func TestTaskLLMHandle(t *testing.T) {
	//Создаем моки
	mockLogger := mocks.NewMockLogger()
	mockUsecase := mocks.NewMockTaskLLMUC(t)

	//Создаем handler
	handler := &KafkaMessageHandler{
		uc: usecase.UseCases{TaskLLMUC: mockUsecase},
		l:  mockLogger,
	}

	tests := []struct {
		name         string
		msg          *kafka.Message
		isError      bool
		expectedCode apperrors.ErrorCode
		expectedMsg  string
	}{
		{
			name: "успешно - valid task_id and status",
			msg: &kafka.Message{
				Key:   []byte(uuid.New().String()),
				Value: []byte(`{"result": {"key": "value"}}`),
				Headers: []kafka.Header{
					{Key: "status", Value: []byte("3")}, // 3 = TaskStatusCompleted
					{Key: "worker_id", Value: []byte("worker_id")},
					{Key: "trace_id", Value: []byte(uuid.New().String())},
				},
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError: false,
		},
		{
			name: "ошибка - invalid task_id format",
			msg: &kafka.Message{
				Key:            []byte("invalid_task_id"),
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError:      true,
			expectedCode: apperrors.CodeInvalidMessageFormat,
			expectedMsg:  "invalid task_id format",
		},
		{
			name: "ошибка - empty task_id",
			msg: &kafka.Message{
				Key:            []byte(""),
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError:      true,
			expectedCode: apperrors.CodeInvalidMessageFormat,
			expectedMsg:  "invalid task_id format",
		},
		{
			name: "ошибка - invalid status format",
			msg: &kafka.Message{
				Key:   []byte(uuid.New().String()),
				Value: []byte(`{"result": {"key": "value"}}`),
				Headers: []kafka.Header{
					{Key: "status", Value: []byte("invalid_status")},
				},
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError:      true,
			expectedCode: apperrors.CodeInvalidMessageFormat,
			expectedMsg:  "invalid task status format",
		},
		{
			name: "ошибка - invalid status code",
			msg: &kafka.Message{
				Key:   []byte(uuid.New().String()),
				Value: []byte(`{"result": {"key": "value"}}`),
				Headers: []kafka.Header{
					{Key: "status", Value: []byte("999")}, // несуществующий статус
				},
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError:      true,
			expectedCode: apperrors.CodeInvalidMessageFormat,
			expectedMsg:  "invalid task status",
		},
		{
			name: "ошибка - invalid json in value",
			msg: &kafka.Message{
				Key:   []byte(uuid.New().String()),
				Value: []byte(`{invalid json}`),
				Headers: []kafka.Header{
					{Key: "status", Value: []byte("3")},
				},
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError:      true,
			expectedCode: apperrors.CodeInvalidMessageFormat,
			expectedMsg:  "failed to unmarshal task status event",
		},
		{
			name: "ошибка - invalid trace_id format",
			msg: &kafka.Message{
				Key:   []byte(uuid.New().String()),
				Value: []byte(`{"result": {"key": "value"}}`),
				Headers: []kafka.Header{
					{Key: "status", Value: []byte("3")},
					{Key: "trace_id", Value: []byte("invalid_trace_id")},
				},
				TopicPartition: kafka.TopicPartition{Topic: stringPtr("tasks_llm")},
			},
			isError:      true,
			expectedCode: apperrors.CodeInvalidMessageFormat,
			expectedMsg:  "invalid trace_id format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandleTaskStatusLLM(context.Background(), tt.msg)

			if (err != nil) != tt.isError {
				t.Errorf("HandleTaskStatusLLM() error = %v, isError %v", err, tt.isError)
				return
			}

			if tt.isError && err != nil {
				appErr, ok := apperrors.IsAppError(err)
				if !ok {
					t.Errorf("HandleTaskStatusLLM() expected AppError, got %T", err)
					return
				}

				if appErr.Code != tt.expectedCode {
					t.Errorf("HandleTaskStatusLLM() error code = %v, want %v", appErr.Code, tt.expectedCode)
				}

				if !strings.Contains(appErr.Message, tt.expectedMsg) {
					t.Errorf("HandleTaskStatusLLM() error message = %v, want to contain %v", appErr.Message, tt.expectedMsg)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
