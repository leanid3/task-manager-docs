package broker

import (
	"app/pkg/logger"
	"context"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaMessageHandler struct {
	registry *Registry
	l        logger.Interface
}

func NewKafkaMessageHandler(registry *Registry, l logger.Interface) *KafkaMessageHandler {
	return &KafkaMessageHandler{registry: registry, l: l}
}

func (h *KafkaMessageHandler) Handle(ctx context.Context, msg *kafka.Message) error {
	h.l.Debug("kafka message received",
		"topic", *msg.TopicPartition.Topic,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset,
	)
	return h.registry.Handle(ctx, msg)
}
