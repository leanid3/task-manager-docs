package broker

import (
	"context"

	apperrors "app/internal/entity/errors"
	"app/pkg/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type ValidatorFunc[T any] func(*kafka.Message) (T, error)
type UseCaseFunc[T any] func(context.Context, T) error

// Route – стандартный handler для topic'а
type Route[T any] struct {
	topic     string
	validator ValidatorFunc[T]
	useCase   UseCaseFunc[T]
	logger    logger.Interface
}

func NewRoute[T any](
	topic string,
	validator ValidatorFunc[T],
	useCase UseCaseFunc[T],
	logger logger.Interface,
) *Route[T] {
	return &Route[T]{topic: topic, validator: validator, useCase: useCase, logger: logger}
}

func (r *Route[T]) Topic() string { return r.topic }

func (r *Route[T]) Handle(ctx context.Context, msg *kafka.Message) error {
	event, err := r.validator(msg)
	if err != nil {
		r.logger.Error("validation failed", "topic", r.topic, "error", err)
		return apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "validation failed", err)
	}

	if err := r.useCase(ctx, event); err != nil {
		r.logger.Error("usecase failed", "topic", r.topic, "error", err)
		return err
	}

	r.logger.Debug("message processed", "topic", r.topic)
	return nil
}
