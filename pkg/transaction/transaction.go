package transaction

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// TransactionManager интерфейс для управления транзакциями
type TransactionManager interface {
	// Begin начинает транзакцию
	Begin(ctx context.Context) error
	// Commit фиксирует транзакцию
	Commit(ctx context.Context) error
	// Abort откатывает транзакцию
	Abort(ctx context.Context) error
	// SendMessage отправляет сообщение в рамках транзакции
	SendMessage(ctx context.Context, topic string, key string, value []byte, headers map[string]string) error
	// SendMessages отправляет несколько сообщений в рамках одной транзакции
	SendMessages(ctx context.Context, messages []TransactionMessage) error
}

// TransactionMessage сообщение для транзакции
type TransactionMessage struct {
	Topic   string
	Key     string
	Value   []byte
	Headers map[string]string
}

// KafkaTransactionManager реализация транзакционного менеджера для Kafka
type KafkaTransactionManager struct {
	producer *kafka.Producer
}

// NewKafkaTransactionManager создает новый транзакционный менеджер для Kafka
func NewKafkaTransactionManager(producer *kafka.Producer) *KafkaTransactionManager {
	return &KafkaTransactionManager{
		producer: producer,
	}
}

// Begin начинает транзакцию
func (ktm *KafkaTransactionManager) Begin(ctx context.Context) error {
	return ktm.producer.InitTransactions(ctx)
}

// Commit фиксирует транзакцию
func (ktm *KafkaTransactionManager) Commit(ctx context.Context) error {
	return ktm.producer.CommitTransaction(ctx)
}

// Abort откатывает транзакцию
func (ktm *KafkaTransactionManager) Abort(ctx context.Context) error {
	return ktm.producer.AbortTransaction(ctx)
}

// SendMessage отправляет сообщение в рамках транзакции
func (ktm *KafkaTransactionManager) SendMessage(ctx context.Context, topic string, key string, value []byte, headers map[string]string) error {
	// Подготовка сообщения
	kafkaMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          value,
	}

	// Добавление заголовков
	for k, v := range headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{
			Key:   k,
			Value: []byte(v),
		})
	}

	// Отправка сообщения в транзакции
	return ktm.producer.Produce(kafkaMsg, nil)
}

// SendMessages отправляет несколько сообщений в рамках одной транзакции
func (ktm *KafkaTransactionManager) SendMessages(ctx context.Context, messages []TransactionMessage) error {
	for _, msg := range messages {
		if err := ktm.SendMessage(ctx, msg.Topic, msg.Key, msg.Value, msg.Headers); err != nil {
			return fmt.Errorf("failed to send message to topic %s: %w", msg.Topic, err)
		}
	}
	return nil
}

// TransactionCoordinator интерфейс для координации транзакций
type TransactionCoordinator interface {
	// ExecuteInTransaction выполняет функцию в транзакции
	ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	// ExecuteInTransactionWithResult выполняет функцию в транзакции с возвратом результата
	ExecuteInTransactionWithResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error)
}

// KafkaTransactionCoordinator реализация координатора транзакций для Kafka
type KafkaTransactionCoordinator struct {
	transactionManager TransactionManager
}

// NewKafkaTransactionCoordinator создает новый координатор транзакций
func NewKafkaTransactionCoordinator(transactionManager TransactionManager) *KafkaTransactionCoordinator {
	return &KafkaTransactionCoordinator{
		transactionManager: transactionManager,
	}
}

// ExecuteInTransaction выполняет функцию в транзакции
func (ktc *KafkaTransactionCoordinator) ExecuteInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := ktc.transactionManager.Begin(ctx); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			// В случае паники откатываем транзакцию
			_ = ktc.transactionManager.Abort(ctx)
			panic(r)
		}
	}()

	if err := fn(ctx); err != nil {
		// При ошибке откатываем транзакцию
		if abortErr := ktc.transactionManager.Abort(ctx); abortErr != nil {
			return fmt.Errorf("transaction failed: %v, abort error: %w", err, abortErr)
		}
		return err
	}

	// При успехе фиксируем транзакцию
	if err := ktc.transactionManager.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecuteInTransactionWithResult выполняет функцию в транзакции с возвратом результата
func (ktc *KafkaTransactionCoordinator) ExecuteInTransactionWithResult(ctx context.Context, fn func(ctx context.Context) (interface{}, error)) (interface{}, error) {
	var result interface{}

	err := ktc.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		res, err := fn(ctx)
		if err != nil {
			return err
		}
		result = res
		return nil
	})

	return result, err
}

// TaskTransactionManager интерфейс для транзакционного управления задачами
type TaskTransactionManager interface {
	// CreateTaskInTransaction создает задачу в транзакции
	CreateTaskInTransaction(ctx context.Context, taskData interface{}) error
	// UpdateTaskStatusInTransaction обновляет статус задачи в транзакции
	UpdateTaskStatusInTransaction(ctx context.Context, taskID string, status string) error
	// PublishTaskResultInTransaction публикует результат задачи в транзакции
	PublishTaskResultInTransaction(ctx context.Context, taskID string, result interface{}) error
}

// DefaultTaskTransactionManager реализация транзакционного менеджера задач
type DefaultTaskTransactionManager struct {
	coordinator TransactionCoordinator
}

// NewDefaultTaskTransactionManager создает новый транзакционный менеджер задач
func NewDefaultTaskTransactionManager(coordinator TransactionCoordinator) *DefaultTaskTransactionManager {
	return &DefaultTaskTransactionManager{
		coordinator: coordinator,
	}
}

// CreateTaskInTransaction создает задачу в транзакции
func (dttm *DefaultTaskTransactionManager) CreateTaskInTransaction(ctx context.Context, taskData interface{}) error {
	return dttm.coordinator.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		// Здесь будет логика создания задачи в базе данных
		// и публикации сообщения в Kafka в одной транзакции
		fmt.Printf("Creating task in transaction: %+v\n", taskData)
		return nil
	})
}

// UpdateTaskStatusInTransaction обновляет статус задачи в транзакции
func (dttm *DefaultTaskTransactionManager) UpdateTaskStatusInTransaction(ctx context.Context, taskID string, status string) error {
	return dttm.coordinator.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		// Здесь будет логика обновления статуса задачи в базе данных
		// и публикации сообщения в Kafka в одной транзакции
		fmt.Printf("Updating task status in transaction: %s -> %s\n", taskID, status)
		return nil
	})
}

// PublishTaskResultInTransaction публикует результат задачи в транзакции
func (dttm *DefaultTaskTransactionManager) PublishTaskResultInTransaction(ctx context.Context, taskID string, result interface{}) error {
	return dttm.coordinator.ExecuteInTransaction(ctx, func(ctx context.Context) error {
		// Здесь будет логика обновления результата задачи в базе данных
		// и публикации сообщения в Kafka в одной транзакции
		fmt.Printf("Publishing task result in transaction: %s -> %+v\n", taskID, result)
		return nil
	})
}
