package extractors

import (
	"testing"

	"app/test/mocks"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHeaderExtractor_ExtractStatus(t *testing.T) {
	logger := mocks.NewMockLogger()
	extractor := NewHeaderExtractor(logger)

	t.Run("successful status extraction", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "status", Value: []byte("200")},
		}

		status, err := extractor.ExtractStatus(headers)

		assert.NoError(t, err)
		assert.Equal(t, 200, status)
	})

	t.Run("status not found", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "other_header", Value: []byte("value")},
		}

		status, err := extractor.ExtractStatus(headers)

		assert.Error(t, err)
		assert.Equal(t, 0, status)
		assert.Contains(t, err.Error(), "status header not found")
	})

	t.Run("invalid status format", func(t *testing.T) {
		// This test is not applicable since Atoi will handle invalid format
		headers := []kafka.Header{
			{Key: "status", Value: []byte("invalid")},
		}

		status, err := extractor.ExtractStatus(headers)

		assert.Error(t, err)
		assert.Equal(t, 0, status)
	})

	t.Run("multiple headers with status", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "other", Value: []byte("value")},
			{Key: "status", Value: []byte("404")},
			{Key: "another", Value: []byte("value")},
		}

		status, err := extractor.ExtractStatus(headers)

		assert.NoError(t, err)
		assert.Equal(t, 404, status)
	})
}

func TestHeaderExtractor_ExtractWorkerID(t *testing.T) {
	logger := mocks.NewMockLogger()
	extractor := NewHeaderExtractor(logger)

	t.Run("successful worker_id extraction", func(t *testing.T) {
		workerID := "worker-123"
		headers := []kafka.Header{
			{Key: "worker_id", Value: []byte(workerID)},
		}

		result, err := extractor.ExtractWorkerID(headers)

		assert.NoError(t, err)
		assert.Equal(t, &workerID, result)
	})

	t.Run("worker_id not found", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "other_header", Value: []byte("value")},
		}

		result, err := extractor.ExtractWorkerID(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "worker_id header not found")
	})

	t.Run("worker_id with special characters", func(t *testing.T) {
		workerID := "worker-service-456"
		headers := []kafka.Header{
			{Key: "worker_id", Value: []byte(workerID)},
		}

		result, err := extractor.ExtractWorkerID(headers)

		assert.NoError(t, err)
		assert.Equal(t, &workerID, result)
	})
}

func TestHeaderExtractor_ExtractTraceID(t *testing.T) {
	logger := mocks.NewMockLogger()
	extractor := NewHeaderExtractor(logger)

	t.Run("successful trace_id extraction", func(t *testing.T) {
		traceID := uuid.New()
		headers := []kafka.Header{
			{Key: "trace_id", Value: []byte(traceID.String())},
		}

		result, err := extractor.ExtractTraceID(headers)

		assert.NoError(t, err)
		assert.Equal(t, &traceID, result)
	})

	t.Run("trace_id not found", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "other_header", Value: []byte("value")},
		}

		result, err := extractor.ExtractTraceID(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trace_id header not found")
	})

	t.Run("invalid trace_id format", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "trace_id", Value: []byte("invalid-uuid")},
		}

		result, err := extractor.ExtractTraceID(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "invalid trace_id format")
	})

	t.Run("malformed trace_id", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "trace_id", Value: []byte("not-a-uuid-at-all")},
		}

		result, err := extractor.ExtractTraceID(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestHeaderExtractor_ExtractAll(t *testing.T) {
	logger := mocks.NewMockLogger()
	extractor := NewHeaderExtractor(logger)

	t.Run("successful extraction of all headers", func(t *testing.T) {
		traceID := uuid.New()
		workerID := "worker-123"
		
		headers := []kafka.Header{
			{Key: "status", Value: []byte("200")},
			{Key: "worker_id", Value: []byte(workerID)},
			{Key: "trace_id", Value: []byte(traceID.String())},
		}

		result, err := extractor.ExtractAll(headers)

		assert.NoError(t, err)
		assert.Equal(t, 200, result.Status)
		assert.Equal(t, workerID, result.WorkerID)
		assert.Equal(t, &traceID, result.TraceID)
	})

	t.Run("missing status header", func(t *testing.T) {
		traceID := uuid.New()
		workerID := "worker-123"
		
		headers := []kafka.Header{
			{Key: "worker_id", Value: []byte(workerID)},
			{Key: "trace_id", Value: []byte(traceID.String())},
		}

		result, err := extractor.ExtractAll(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "status header not found")
	})

	t.Run("missing worker_id header", func(t *testing.T) {
		traceID := uuid.New()
		
		headers := []kafka.Header{
			{Key: "status", Value: []byte("200")},
			{Key: "trace_id", Value: []byte(traceID.String())},
		}

		result, err := extractor.ExtractAll(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "worker_id header not found")
	})

	t.Run("missing trace_id header", func(t *testing.T) {
		workerID := "worker-123"
		
		headers := []kafka.Header{
			{Key: "status", Value: []byte("200")},
			{Key: "worker_id", Value: []byte(workerID)},
		}

		result, err := extractor.ExtractAll(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "trace_id header not found")
	})

	t.Run("all headers missing", func(t *testing.T) {
		headers := []kafka.Header{
			{Key: "other_header", Value: []byte("value")},
		}

		result, err := extractor.ExtractAll(headers)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}