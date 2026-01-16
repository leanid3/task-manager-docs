package llm

import (
	"encoding/json"
	"errors"

	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/handlers/broker"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TaskLLMValidator - валидатор для LLM задач
type TaskLLMValidator struct {
	base *broker.BaseValidator
}

func NewTaskLLMValidator(base *broker.BaseValidator) *TaskLLMValidator {
	return &TaskLLMValidator{base: base}
}

// Validate - полная валидация TaskLLMStatusEvent
func (v *TaskLLMValidator) Validate(msg *kafka.Message) (*domain.TaskLLMStatusEvent, error) {
	if v.base == nil {
		return nil, errors.New("base is not initialized")
	}

	// 1) Валидируем общую часть (Key + Headers)
	contract, err := v.base.ValidateContract(msg)
	if err != nil {
		return nil, err
	}

	// 2) Валидируем специфичный payload (Value)
	var payload domain.TaskLLMStatusEventPayload
	if len(msg.Value) > 0 {
		if err := json.Unmarshal(msg.Value, &payload); err != nil {
			return nil, apperrors.Wrap(
				apperrors.CodeInvalidMessageFormat,
				"failed to unmarshal task status event",
				err,
			)
		}
	}

	// 3) Собираем событие через композицию
	event := &domain.TaskLLMStatusEvent{
		Key: domain.TaskContractKey{
			TaskID: contract.TaskID,
		},
		Headers: domain.TaskContractHeaders{
			Status:   contract.Status,
			WorkerID: contract.WorkerID,
			TraceID:  contract.TraceID,
		},
		Value: payload,
	}

	return event, nil
}
