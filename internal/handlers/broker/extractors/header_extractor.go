// internal/handlers/broker/extractors/header_extractor.go
package extractors

import (
	"app/internal/entity/domain"
	"app/pkg/logger"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

const (
	HeaderStatus   = "status"
	HeaderWorkerID = "worker_id"
	HeaderTraceID  = "trace_id"
)

type HeaderExtractor struct {
	logger logger.Interface
}

func NewHeaderExtractor(logger logger.Interface) *HeaderExtractor {
	return &HeaderExtractor{logger: logger}
}

// ExtractStatus - парсинг статуса (обязательный)
func (e *HeaderExtractor) ExtractStatus(headers []kafka.Header) (int, error) {
	for _, header := range headers {
		if string(header.Key) == HeaderStatus {
			return strconv.Atoi(string(header.Value))
		}
	}
	return 0, fmt.Errorf("status header not found")
}

// ExtractWorkerID - парсинг worker_id (теперь обязательный)
func (e *HeaderExtractor) ExtractWorkerID(headers []kafka.Header) (*string, error) {
	for _, header := range headers {
		if string(header.Key) == HeaderWorkerID {
			val := string(header.Value)
			return &val, nil
		}
	}
	return nil, fmt.Errorf("worker_id header not found")
}

// ExtractTraceID - парсинг trace_id (теперь обязательный)
func (e *HeaderExtractor) ExtractTraceID(headers []kafka.Header) (*uuid.UUID, error) {
	for _, header := range headers {
		if string(header.Key) == HeaderTraceID {
			traceIDStr := string(header.Value)
			id, err := uuid.Parse(traceIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid trace_id format: %w", err)
			}
			return &id, nil
		}
	}
	return nil, fmt.Errorf("trace_id header not found")
}

// ExtractAll - единый вызов для всех обязательных заголовков
func (e *HeaderExtractor) ExtractAll(headers []kafka.Header) (*domain.TaskContractHeaders, error) {
	status, err := e.ExtractStatus(headers)
	if err != nil {
		return nil, err
	}

	workerID, err := e.ExtractWorkerID(headers)
	if err != nil {
		return nil, err
	}

	traceID, err := e.ExtractTraceID(headers)
	if err != nil {
		return nil, err
	}

	return &domain.TaskContractHeaders{
		Status:   status,
		WorkerID: *workerID,
		TraceID:  traceID,
	}, nil
}
