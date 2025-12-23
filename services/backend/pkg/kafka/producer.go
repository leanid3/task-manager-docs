package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"app/pkg/logger"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// Producer интерфейс для DI и тестов
type Producer interface {
	Send(ctx context.Context, topic string, key string, headers map[string]string, value interface{}) error
	Close() error
}

// producer реализация
type producer struct {
	client *kafka.Producer
	l      logger.Interface
	once   sync.Once
}

type ProducerConfig struct {
	BootstrapServers          string
	ClientID                  string
	ProducerAcks              string
	ProducerEnableIdempotence bool
	ProducerCompressionType   string
	ProducerRetries           int
}

// NewProducer создает продюсера
func NewProducer(config ProducerConfig, l logger.Interface) (Producer, error) {

	//TODO декомпозировать на несколько producer при создании новых типов tasks
	cfg := kafka.ConfigMap{
		"bootstrap.servers": config.BootstrapServers,
		"client.id":         config.ClientID,
	}

	// Устанавливаем значения по умолчанию для опциональных полей
	if config.ProducerAcks != "" {
		cfg["acks"] = config.ProducerAcks
	} else {
		cfg["acks"] = "-1" // all (по умолчанию)
	}

	if config.ProducerCompressionType != "" {
		cfg["compression.type"] = config.ProducerCompressionType
	}

	if config.ProducerRetries > 0 {
		cfg["retries"] = config.ProducerRetries
	}

	// enable.idempotence устанавливаем только если явно указано
	cfg["enable.idempotence"] = config.ProducerEnableIdempotence

	p, err := kafka.NewProducer(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	return &producer{
		client: p,
		l:      l,
	}, nil
}

// Send отправляет TaskCommand в Kafka
func (p *producer) Send(ctx context.Context, topic string, key string, cmdHeaders map[string]string, cmdValue interface{}) error {
	p.l.Debug("Producer.Send - start",
		"key", key,
		"Headers", cmdHeaders,
		"Value", cmdValue,
		"topic", topic,
	)
	data, err := json.Marshal(cmdValue)
	if err != nil {
		return fmt.Errorf("failed to marshal task command: %w", err)
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Headers:        p.headersToBroker(cmdHeaders),
		Value:          data,
	}
	p.l.Debug("Producer message: ", "msg", msg)
	deliveryChan := make(chan kafka.Event, 1)

	p.client.Produce(msg, deliveryChan)

	select {
	case <-ctx.Done():
		// Drain канал асинхронно без блокировки
		go func() {
			select {
			case ev := <-deliveryChan:
				if msg, ok := ev.(*kafka.Message); ok && msg.TopicPartition.Error != nil {
					p.l.Warn("message delivery failed after ctx cancel",
						"task_id", key, "error", msg.TopicPartition.Error)
				}
			case <-time.After(100 * time.Millisecond): // timeout drain
			}
		}()
		return ctx.Err()
	case ev := <-deliveryChan:
		switch e := ev.(type) {
		case *kafka.Message:
			if e.TopicPartition.Error != nil {
				p.l.Error("message delivery failed",
					"task_id", key,
					"error", e.TopicPartition.Error,
					"topic", topic)
				return fmt.Errorf("delivery failed: %w", e.TopicPartition.Error)
			}
			p.l.Debug("message delivered",
				"task_id", key,
				"topic", topic,
				"partition", e.TopicPartition.Partition,
				"offset", e.TopicPartition.Offset)
			return nil
		}
	}
	return nil
}

func (p *producer) headersToBroker(headers map[string]string) []kafka.Header {
	headersMessage := make([]kafka.Header, 0, len(headers))
	for key, value := range headers {
		headersMessage = append(headersMessage, kafka.Header{Key: key, Value: []byte(value)})
	}
	return headersMessage
}

// Close graceful shutdown с flush
func (p *producer) Close() error {
	var err error
	p.once.Do(func() {
		// Flush pending messages
		p.client.Flush(15 * 1000)

		// Drain events чтобы избежать memory leak
		drained := false
		for i := 0; i < 1000 && !drained; i++ { // max pending messages
			select {
			case ev := <-p.client.Events():
				if msg, ok := ev.(*kafka.Message); ok && msg.TopicPartition.Error != nil {
					p.l.Warn("undelivered message on close", "error", msg.TopicPartition.Error)
				}
			default:
				drained = true
			}
		}
		p.client.Close()
	})
	return err
}
