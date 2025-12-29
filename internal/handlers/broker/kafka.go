package broker

import (
	"app/internal/usecase"
	"app/pkg/logger"
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Handler интерфейс для диспетчеризации
type Handler interface {
	Handle(ctx context.Context, msg *kafka.Message) error
}

// KafkaMessageHandler диспетчер handlers по типу сообщения
type KafkaMessageHandler struct {
	uc usecase.UseCases
	l  logger.Interface
}

func NewKafkaMessageHandler(uc usecase.UseCases, l logger.Interface) *KafkaMessageHandler {
	return &KafkaMessageHandler{uc: uc, l: l}
}

// Handle обрабатывает сообщение из Kafka, проверяет тип сообщения и передает его в соответствующий обработчик
// TODO сделать общий регистратор обработчиков для разных consumers
func (h *KafkaMessageHandler) Handle(ctx context.Context, msg *kafka.Message) error {

	//TODO сделать общию валидацию сообщения
	//TODO при добавлении новых типов сообщений, добавить обработку в этот switch

	h.l.Debug("kafka message received",
		"topic", *msg.TopicPartition.Topic,
		"partition", msg.TopicPartition.Partition,
		"offset", msg.TopicPartition.Offset,
		"key", string(msg.Key),
		"value_length", len(msg.Value),
	)
	//TODO вынести генерацию маршрутов в конфиг
	switch topic := *msg.TopicPartition.Topic; topic {
	case "tasks_llm":
		return h.HandleTaskStatusLLM(ctx, msg)
	case "tasks_analysis":
		// return h.HandleTaskAnalysis(ctx, msg)
	default:
		return fmt.Errorf("unknown topic: %s", topic)
	}
	return nil
}
