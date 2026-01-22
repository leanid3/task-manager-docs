package kafka

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"app/pkg/logger"
	"app/pkg/metrics"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type Consumer interface {
	Start(ctx context.Context) error
	Stop() error
}

type MessageHandler func(msg *kafka.Message) error

// Handler интерфейс для обработчиков с context
type Handler interface {
	Handle(ctx context.Context, msg *kafka.Message) error
}

// ConsumerConfig содержит конфигурацию для создания consumer
type ConsumerConfig struct {
	BootstrapServers     string
	ClientID             string
	GroupID              string
	EnableAutoCommit     bool
	AutoCommitIntervalMs int
	SessionTimeoutMs     int
	HeartbeatIntervalMs  int
	AutoOffsetReset      string // earliest, latest, none (по умолчанию latest)
}

type consumer struct {
	client      *kafka.Consumer
	topics      []string
	groupID     string
	handler     MessageHandler
	cfg         *kafka.ConfigMap
	commitChain chan *kafka.Message
	wg          sync.WaitGroup
	once        sync.Once
	l           logger.Interface
	// Состояние для отслеживания недоступности брокеров
	brokerDownCount   int
	brokerDownStart   time.Time
	lastBrokerDownLog time.Time
}

func NewConsumer(handler MessageHandler, topics []string, config ConsumerConfig, l logger.Interface) (Consumer, error) {
	cfg := kafka.ConfigMap{
		"bootstrap.servers":       config.BootstrapServers,
		"client.id":               config.ClientID,
		"group.id":                config.GroupID,
		"enable.auto.commit":      config.EnableAutoCommit,
		"auto.commit.interval.ms": config.AutoCommitIntervalMs,
		"session.timeout.ms":      config.SessionTimeoutMs,
		"heartbeat.interval.ms":   config.HeartbeatIntervalMs,
	}

	// Устанавливаем auto.offset.reset если указано
	if config.AutoOffsetReset != "" {
		cfg["auto.offset.reset"] = config.AutoOffsetReset
	}

	c, err := kafka.NewConsumer(&cfg)
	if err != nil {
		l.Error("failed to create kafka consumer", "error", err, "config", &cfg)
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &consumer{
		client:      c,
		topics:      topics,
		groupID:     config.GroupID,
		handler:     handler,
		cfg:         &cfg,
		commitChain: make(chan *kafka.Message, 1000),
		l:           l,
	}, nil
}

// NewConsumerWithHandler создает consumer с Handler (с context)
func NewConsumerWithHandler(ctx context.Context, handler Handler, topics []string, config ConsumerConfig, l logger.Interface) (Consumer, error) {
	// Адаптер для преобразования Handler (с context) в MessageHandler (без context)
	messageHandler := func(msg *kafka.Message) error {
		return handler.Handle(ctx, msg)
	}

	return NewConsumer(messageHandler, topics, config, l)
}

func (c *consumer) Start(ctx context.Context) error {
	err := c.client.SubscribeTopics(c.topics, nil)
	if err != nil {
		c.l.Error("failed to subscribe to topics", "error", err, "topics", strings.Join(c.topics, ","))
		return fmt.Errorf("failed to subscribe to topics: %w", err)
	}

	c.wg.Add(2)
	go c.pollLoop(ctx)
	go c.commitLoop(ctx)

	<-ctx.Done()
	c.Stop()
	return nil
}

func (c *consumer) Stop() error {
	c.once.Do(func() {
		close(c.commitChain)
		c.client.Close()
		c.wg.Wait()
	})
	return nil
}

func (c *consumer) pollLoop(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := c.client.ReadMessage(100 * time.Millisecond)
			if err != nil {
				if kafkaErr, ok := err.(kafka.Error); ok {
					switch kafkaErr.Code() {
					case kafka.ErrTimedOut:
						// Сбрасываем счетчик при успешном таймауте (брокеры доступны)
						if c.brokerDownCount > 0 {
							duration := time.Since(c.brokerDownStart)
							c.l.Info("kafka brokers recovered",
								"downtime", duration,
								"retry_attempts", c.brokerDownCount)
							c.brokerDownCount = 0
						}
						continue // Нормально
					case kafka.ErrAllBrokersDown, kafka.ErrBrokerNotAvailable:
						// Отслеживаем начало недоступности
						if c.brokerDownCount == 0 {
							c.brokerDownStart = time.Now()
							c.lastBrokerDownLog = time.Now()
						}
						c.brokerDownCount++

						// Логируем только раз в минуту или при первой попытке
						shouldLog := c.brokerDownCount == 1 ||
							time.Since(c.lastBrokerDownLog) >= 60*time.Second

						if shouldLog {
							duration := time.Since(c.brokerDownStart)
							c.l.Warn("kafka brokers unavailable",
								"error", err,
								"retry_attempt", c.brokerDownCount,
								"downtime", duration,
								"retrying...")
							c.lastBrokerDownLog = time.Now()
						}

						// Экспоненциальный backoff с максимумом 30 секунд
						backoff := time.Duration(c.brokerDownCount) * 2 * time.Second
						if backoff > 30*time.Second {
							backoff = 30 * time.Second
						}

						// Проверяем контекст перед ожиданием
						select {
						case <-ctx.Done():
							return
						case <-time.After(backoff):
							continue
						}
					default:
						c.l.Error("kafka read error", "code", kafkaErr.Code(), "error", err)
						// Проверяем контекст перед ожиданием
						select {
						case <-ctx.Done():
							return
						case <-time.After(1 * time.Second):
							continue
						}
					}
				}
				c.l.Error("failed to read message", "error", err)
				// Проверяем контекст перед ожиданием
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					continue
				}
			}

			// Сбрасываем счетчик при успешном чтении сообщения
			if c.brokerDownCount > 0 {
				duration := time.Since(c.brokerDownStart)
				c.l.Info("kafka brokers recovered",
					"downtime", duration,
					"retry_attempts", c.brokerDownCount)
				c.brokerDownCount = 0
			}

			// Обработка сообщения...
			start := time.Now()
			if err := c.handler(msg); err != nil {
				c.l.Error("failed to handle message", "error", err)

				// Регистрируем метрики ошибки
				if msg.TopicPartition.Topic != nil {
					metrics.IncKafkaMessagesConsumed(*msg.TopicPartition.Topic, fmt.Sprintf("%d", msg.TopicPartition.Partition))
				}
			} else {
				// Регистрируем успешное потребление сообщения
				if msg.TopicPartition.Topic != nil {
					metrics.IncKafkaMessagesConsumed(*msg.TopicPartition.Topic, fmt.Sprintf("%d", msg.TopicPartition.Partition))
				}
				c.commitChain <- msg
			}

			// Регистрируем время обработки сообщения
			if msg.TopicPartition.Topic != nil {
				duration := time.Since(start).Seconds()
				metrics.ObserveTaskProcessingDuration("kafka_message", "processed", duration)
			}
		}
	}
}

// commitLoop - батч коммит каждые 10 сообщений или 5 сек
func (c *consumer) commitLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	batch := make([]*kafka.Message, 0, 10)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			if len(batch) > 0 {
				c.drainCommit(batch)
			}
			return
		case msg, ok := <-c.commitChain:
			if !ok {
				ticker.Stop()
				c.drainCommit(batch)
				return
			}
			batch = append(batch, msg)
			if len(batch) >= 10 {
				c.drainCommit(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				c.drainCommit(batch)
				batch = batch[:0]
			}
		}
	}
}

func (c *consumer) drainCommit(batch []*kafka.Message) {
	for _, msg := range batch {
		if _, err := c.client.CommitMessage(msg); err != nil {
			c.l.Error("failed to commit message", "error", err, "message", msg)
		}
	}
}
