package broker

import (
	apperrors "app/internal/entity/errors"
	"app/pkg/logger"
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type TaskHandler interface {
	Handle(ctx context.Context, msg *kafka.Message) error
	Topic() string
}

type Registry struct {
	handlers map[string]TaskHandler
	logger   logger.Interface
}

func NewRegistry(logger logger.Interface) *Registry {
	return &Registry{
		handlers: make(map[string]TaskHandler),
		logger:   logger,
	}
}

func (r *Registry) Register(handler TaskHandler) error {
	topic := handler.Topic()
	if _, exists := r.handlers[topic]; exists {
		return fmt.Errorf("handler for topic %s already registered", topic)
	}
	r.handlers[topic] = handler
	r.logger.Info("handler registered", "topic", topic)
	return nil
}

func (r *Registry) Handle(ctx context.Context, msg *kafka.Message) error {
	topic := *msg.TopicPartition.Topic
	handler, ok := r.handlers[topic]
	if !ok {
		return apperrors.New(apperrors.CodeKafkaError,
			fmt.Sprintf("no handler for topic: %s", topic))
	}
	return handler.Handle(ctx, msg)
}
