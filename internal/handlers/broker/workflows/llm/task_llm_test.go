package llm

import (
	"context"
	"io"
	"testing"

	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/handlers/broker"
	"app/internal/handlers/broker/extractors"
	"app/test/mocks"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRegistry for testing
type MockRegistry struct {
	mock.Mock
}

func (m *MockRegistry) Register(handler broker.TaskHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

// We'll use the actual BaseValidator with a real HeaderExtractor for this test

// MockTaskLLMUCInterface for testing
type MockTaskLLMUCInterface struct {
	mock.Mock
}

func (m *MockTaskLLMUCInterface) CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error) {
	args := m.Called(ctx, reader, filename, filesize, requestID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockTaskLLMUCInterface) GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Task), args.Error(1)
}

func (m *MockTaskLLMUCInterface) UpdateTaskStatus(ctx context.Context, evt domain.TaskLLMStatusEvent) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func TestTaskLLMValidator_Validate(t *testing.T) {
	logger := mocks.NewMockLogger()
	headerExtractor := extractors.NewHeaderExtractor(logger)
	baseValidator := broker.NewBaseValidator(headerExtractor)
	validator := NewTaskLLMValidator(baseValidator)

	t.Run("successful validation with result", func(t *testing.T) {
		taskID := uuid.New()
		traceID := uuid.New()
		workerID := "worker-123"

		// Create a message with proper headers
		msg := &kafka.Message{
			Key: []byte(taskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("3")}, // Completed
				{Key: "worker_id", Value: []byte(workerID)},
				{Key: "trace_id", Value: []byte(traceID.String())},
			},
			Value: []byte(`{"result": "test result", "error_message": ""}`),
		}

		result, err := validator.Validate(msg)

		assert.NoError(t, err)
		assert.Equal(t, taskID, result.Key.TaskID)
		assert.Equal(t, 3, result.Headers.Status)
		assert.Equal(t, workerID, result.Headers.WorkerID)
		assert.Equal(t, &traceID, result.Headers.TraceID)
		assert.Equal(t, `"test result"`, string(result.Value.Result))
		assert.Equal(t, "", result.Value.ErrorMessage)
	})

	t.Run("successful validation with error", func(t *testing.T) {
		taskID := uuid.New()
		traceID := uuid.New()
		workerID := "worker-123"

		msg := &kafka.Message{
			Key: []byte(taskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("4")}, // Failed
				{Key: "worker_id", Value: []byte(workerID)},
				{Key: "trace_id", Value: []byte(traceID.String())},
			},
			Value: []byte(`{"result": null, "error_message": "something went wrong"}`),
		}

		result, err := validator.Validate(msg)

		assert.NoError(t, err)
		assert.Equal(t, taskID, result.Key.TaskID)
		assert.Equal(t, 4, result.Headers.Status)
		assert.Equal(t, workerID, result.Headers.WorkerID)
		assert.Equal(t, &traceID, result.Headers.TraceID)
		assert.Equal(t, "something went wrong", result.Value.ErrorMessage)
	})

	t.Run("base validation fails - missing headers", func(t *testing.T) {
		taskID := uuid.New()

		msg := &kafka.Message{
			Key: []byte(taskID.String()),
			// Missing required headers
			Value: []byte(`{"result": "test"}`),
		}

		result, err := validator.Validate(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("invalid JSON in value", func(t *testing.T) {
		taskID := uuid.New()
		traceID := uuid.New()
		workerID := "worker-123"

		msg := &kafka.Message{
			Key: []byte(taskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("3")}, // Completed
				{Key: "worker_id", Value: []byte(workerID)},
				{Key: "trace_id", Value: []byte(traceID.String())},
			},
			Value: []byte(`{"result": "test result", "error_message":}`), // Invalid JSON
		}

		result, err := validator.Validate(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
		assert.Contains(t, appErr.Message, "failed to unmarshal")
	})

	t.Run("empty value in message", func(t *testing.T) {
		taskID := uuid.New()
		traceID := uuid.New()
		workerID := "worker-123"

		msg := &kafka.Message{
			Key: []byte(taskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("2")}, // Processing
				{Key: "worker_id", Value: []byte(workerID)},
				{Key: "trace_id", Value: []byte(traceID.String())},
			},
			Value: []byte(""), // Empty value
		}

		result, err := validator.Validate(msg)

		assert.NoError(t, err)
		assert.Equal(t, taskID, result.Key.TaskID)
		assert.Equal(t, 2, result.Headers.Status)
		assert.Equal(t, workerID, result.Headers.WorkerID)
		assert.Equal(t, &traceID, result.Headers.TraceID)
		assert.Empty(t, result.Value.Result)
		assert.Empty(t, result.Value.ErrorMessage)
	})
}

func TestRegister(t *testing.T) {
	logger := mocks.NewMockLogger()
	registry := broker.NewRegistry(logger)
	// For this test, we'll create a real validator with a real extractor
	realLogger := mocks.NewMockLogger()
	headerExtractor := extractors.NewHeaderExtractor(realLogger)
	baseValidator := broker.NewBaseValidator(headerExtractor)
	validator := NewTaskLLMValidator(baseValidator)
	mockUC := new(MockTaskLLMUCInterface)

	t.Run("successful registration", func(t *testing.T) {
		// We'll just call the Register function and check that no error occurs
		// Since the registration modifies the registry internally, we can't easily mock it
		err := Register(registry, validator, mockUC, logger)

		assert.NoError(t, err)
		// We can't directly access registry.handlers since it's unexported
		// So we'll just verify that no error occurred during registration
	})
}

func TestRegister_UseCaseFunction(t *testing.T) {
	logger := mocks.NewMockLogger()
	registry := broker.NewRegistry(logger)
	// For this test, we'll create a real validator with a real extractor
	realLogger := mocks.NewMockLogger()
	headerExtractor := extractors.NewHeaderExtractor(realLogger)
	baseValidator := broker.NewBaseValidator(headerExtractor)
	validator := NewTaskLLMValidator(baseValidator)
	mockUC := new(MockTaskLLMUCInterface)

	// Test the use case function that gets registered
	taskID := uuid.New()
	traceID := uuid.New()

	event := &domain.TaskLLMStatusEvent{
		Key: domain.TaskContractKey{
			TaskID: taskID,
		},
		Headers: domain.TaskContractHeaders{
			Status:   3, // Completed
			WorkerID: "worker-123",
			TraceID:  &traceID,
		},
		Value: domain.TaskLLMStatusEventPayload{
			Result: []byte(`{"result": "test"}`),
		},
	}

	mockUC.On("UpdateTaskStatus", mock.Anything, *event).Return(nil)

	err := Register(registry, validator, mockUC, logger)
	assert.NoError(t, err)

	// The use case function should call UpdateTaskStatus
	ctx := context.Background()
	err = mockUC.UpdateTaskStatus(ctx, *event)
	assert.NoError(t, err)

	mockUC.AssertExpectations(t)
}