package broker

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/handlers/broker/extractors"
	"encoding/json"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
)

// UniversalValidator универсальный валидатор для всех типов задач
type UniversalValidator struct {
	extractor *extractors.HeaderExtractor
}

// NewUniversalValidator создает новый универсальный валидатор
func NewUniversalValidator(extractor *extractors.HeaderExtractor) *UniversalValidator {
	return &UniversalValidator{extractor: extractor}
}

// Validate валидирует сообщение и возвращает универсальное событие
func (v *UniversalValidator) Validate(msg *kafka.Message) (*domain.TaskEvent, error) {
	// 1) Key: task_id (обязательный)
	taskIDStr := string(msg.Key)
	if taskIDStr == "" {
		return nil, apperrors.New(apperrors.CodeInvalidMessageFormat, "task_id is required")
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

	// 4) Валидируем специфичный payload (Value)
	var payload interface{}
	if len(msg.Value) > 0 {
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			return nil, apperrors.Wrap(
				apperrors.CodeInvalidMessageFormat,
				"failed to unmarshal task event payload",
				err,
			)
		}
	}

	// 5) Собираем событие
	event := &domain.TaskEvent{
		Key: domain.TaskContractKey{
			TaskID: taskID,
		},
		Headers: *headers,
		Value:   payload,
	}

	return event, nil
}
