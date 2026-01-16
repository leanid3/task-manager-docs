package broker

import (
	"strings"

	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/handlers/broker/extractors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

// BaseValidator - валидирует общую часть контракта (Key + Headers)
type BaseValidator struct {
	extractor *extractors.HeaderExtractor
}

func NewBaseValidator(extractor *extractors.HeaderExtractor) *BaseValidator {
	return &BaseValidator{extractor: extractor}
}

// ValidatedContract - результат валидации общей части
type ValidatedContract struct {
	domain.TaskContractKey
	domain.TaskContractHeaders
}

func (v *BaseValidator) ValidateContract(msg *kafka.Message) (*ValidatedContract, error) {
	// 1) Key: task_id (обязательный)
	taskIDStr := strings.TrimSpace(string(msg.Key))
	if taskIDStr == "" {
		return nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task_id format")
	}

	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "invalid task_id format", err)
	}

	// 2) Headers: используем extractor для всех заголовков
	headers, err := v.extractor.ExtractAll(msg.Headers)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "failed to extract headers", err)
	}

	// 3) Валидация статуса (должен быть валидным кодом)
	if _, ok := domain.TaskStatus("").FromKafkaCode(headers.Status); !ok {
		return nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "invalid task status")
	}

	// 4) worker_id - теперь обязательный
	if headers.WorkerID == "" {
		return nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "worker_id is required")
	}

	// 5) trace_id - теперь обязательный
	if headers.TraceID == nil {
		return nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "trace_id is required")
	}
	c := ValidatedContract{}
	c.TaskID = taskID
	c.TraceID = headers.TraceID
	c.WorkerID = headers.WorkerID
	c.Status = headers.Status
	return &c, nil

}
