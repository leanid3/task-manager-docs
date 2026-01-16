package broker

import (
	"testing"

	apperrors "app/internal/entity/errors"
	"app/internal/handlers/broker/extractors"
	"app/test/mocks"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBaseValidator_ValidateContract(t *testing.T) {
	validTaskID := uuid.New()
	validTraceID := uuid.New()
	validWorkerID := "worker-123"

	logger := mocks.NewMockLogger()
	headerExtractor := extractors.NewHeaderExtractor(logger)
	baseValidator := NewBaseValidator(headerExtractor)

	t.Run("успешная валидация всех полей", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(validTaskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("2")}, // PROCESSING status
				{Key: "worker_id", Value: []byte(validWorkerID)},
				{Key: "trace_id", Value: []byte(validTraceID.String())},
			},
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.NoError(t, err)
		assert.Equal(t, validTaskID, result.TaskID)
		assert.Equal(t, 2, result.Status)
		assert.Equal(t, validWorkerID, result.WorkerID)
		assert.Equal(t, &validTraceID, result.TraceID)
	})

	t.Run("пустой task_id", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(""),
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
		assert.Contains(t, appErr.Message, "invalid task_id format")
	})

	t.Run("task_id только пробелы", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte("   "),
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
		assert.Contains(t, appErr.Message, "invalid task_id format")
	})

	t.Run("невалидный UUID формат task_id", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte("not-a-uuid"),
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
		assert.Contains(t, appErr.Message, "invalid task_id format")
	})

	t.Run("отсутствует status header", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(validTaskID.String()),
			Headers: []kafka.Header{
				{Key: "worker_id", Value: []byte(validWorkerID)},
				{Key: "trace_id", Value: []byte(validTraceID.String())},
			},
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
		assert.Contains(t, appErr.Message, "failed to extract headers")
	})

	t.Run("отсутствует worker_id header", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(validTaskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("2")},
				{Key: "trace_id", Value: []byte(validTraceID.String())},
			},
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		// The error will come from the header extractor when ExtractAll is called
		// which will fail to extract the required worker_id header
	})

	t.Run("worker_id пустая строка", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(validTaskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("2")},
				{Key: "worker_id", Value: []byte("   ")},
				{Key: "trace_id", Value: []byte(validTraceID.String())},
			},
		}

		result, err := baseValidator.ValidateContract(msg)

		// The error should occur because the worker_id is only spaces and gets trimmed to empty
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("отсутствует trace_id header", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(validTaskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("2")},
				{Key: "worker_id", Value: []byte(validWorkerID)},
			},
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		// The error will occur because the trace_id header is missing
	})

	t.Run("невалидный status код", func(t *testing.T) {
		msg := &kafka.Message{
			Key: []byte(validTaskID.String()),
			Headers: []kafka.Header{
				{Key: "status", Value: []byte("999")}, // невалидный код
				{Key: "worker_id", Value: []byte(validWorkerID)},
				{Key: "trace_id", Value: []byte(validTraceID.String())},
			},
		}

		result, err := baseValidator.ValidateContract(msg)

		assert.Error(t, err)
		assert.Nil(t, result)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
		assert.Contains(t, appErr.Message, "invalid task status")
	})
}