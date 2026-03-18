package broker

import (
	"context"

	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/pkg/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// UniversalValidatorFunc универсальная функция валидации для любых типов событий
type UniversalValidatorFunc func(*kafka.Message) (*domain.TaskEvent, error)

// UniversalUseCaseFunc универсальная функция обработки для любых типов событий
type UniversalUseCaseFunc func(context.Context, *domain.TaskEvent) error

// UniversalRoute универсальный обработчик для всех типов задач
type UniversalRoute struct {
	topic     string
	validator UniversalValidatorFunc
	useCase   UniversalUseCaseFunc
	logger    logger.Interface
}

// NewUniversalRoute создает универсальный маршрут для обработки любых типов задач
func NewUniversalRoute(
	topic string,
	validator UniversalValidatorFunc,
	useCase UniversalUseCaseFunc,
	logger logger.Interface,
) *UniversalRoute {
	return &UniversalRoute{
		topic:     topic,
		validator: validator,
		useCase:   useCase,
		logger:    logger,
	}
}

// Topic возвращает топик, для которого зарегистрирован обработчик
func (r *UniversalRoute) Topic() string {
	return r.topic
}

// Handle обрабатывает сообщение из Kafka
func (r *UniversalRoute) Handle(ctx context.Context, msg *kafka.Message) error {
	event, err := r.validator(msg)
	if err != nil {
		r.logger.Error("universal validation failed", "topic", r.topic, "error", err)
		return apperrors.Wrap(apperrors.CodeInvalidMessageFormat, "universal validation failed", err)
	}

	if err := r.useCase(ctx, event); err != nil {
		r.logger.Error("universal usecase failed", "topic", r.topic, "error", err)
		return err
	}

	r.logger.Debug("universal message processed", "topic", r.topic)
	return nil
}
