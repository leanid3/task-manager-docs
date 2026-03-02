//go:build integration

package kafka

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"app/test/mocks"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kafkatestcontainers "github.com/testcontainers/testcontainers-go/modules/kafka"
)

func setupTestKafka(t *testing.T) (string, func()) {
	// Используем контекст с таймаутом для скачивания образа
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Используем модуль kafka с поддержкой KRaft (без Zookeeper)
	// Модуль автоматически использует правильный образ и конфигурацию
	kafkaContainer, err := kafkatestcontainers.RunContainer(ctx,
		kafkatestcontainers.WithClusterID("test-cluster"),
	)
	require.NoError(t, err)

	brokersList, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)

	brokers := strings.Join(brokersList, ",")

	cleanup := func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer terminateCancel()
		kafkaContainer.Terminate(terminateCtx)
	}

	return brokers, cleanup
}

// TestConsumerLifecycle проверяет полный lifecycle
// Интеграционный тест - требует доступный Kafka брокер
func TestConsumerLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testTopic := fmt.Sprintf("test-%d", time.Now().UnixNano())

	// 1. Consumer
	handled := make(chan struct{})
	handler := func(msg *kafka.Message) error {
		t.Logf("✅ Handled: %s", string(msg.Value))
		close(handled)
		return nil
	}

	cons, err := NewConsumer(handler, []string{testTopic}, ConsumerConfig{
		BootstrapServers:     brokers,
		ClientID:             "test-client",
		GroupID:              "test-group",
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}, mocks.NewMockLogger())
	require.NoError(t, err)
	defer cons.Stop()

	// 2. Producer
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokers})
	require.NoError(t, err)
	defer producer.Close()

	// 3. Сначала отправляем сообщение, чтобы топик создался
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: kafka.PartitionAny},
		Value:          []byte("test"),
	}
	deliveryChan := make(chan kafka.Event, 1)
	err = producer.Produce(msg, deliveryChan)
	require.NoError(t, err)

	// Ждем подтверждения доставки
	select {
	case ev := <-deliveryChan:
		if msg, ok := ev.(*kafka.Message); ok && msg.TopicPartition.Error != nil {
			t.Fatalf("Message delivery failed: %v", msg.TopicPartition.Error)
		}
		t.Log("✅ Message delivered")
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message delivery")
	}

	// Даем время топику создаться
	time.Sleep(500 * time.Millisecond)

	// 4. Запуск consumer (он будет читать с earliest offset)
	go func() { _ = cons.Start(ctx) }()

	// Ждем подписки consumer на топик
	time.Sleep(2 * time.Second)

	// Ждем обработки сообщения consumer
	select {
	case <-handled:
		t.Log("✅ Message handled")
	case <-time.After(15 * time.Second):
		t.Fatal("Timeout waiting for message handling")
	}

	// Отменяем контекст для корректного завершения consumer
	cancel()

	// Даем время горутинам завершиться
	time.Sleep(1 * time.Second)
}

// TestConsumer_ErrorHandling проверяет обработку ошибок handler
func TestConsumerErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testTopic := fmt.Sprintf("test-error-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	mockLogger := mocks.NewMockLogger()
	cfg := ConsumerConfig{
		BootstrapServers:     brokers,
		ClientID:             "test-client",
		GroupID:              groupID,
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}

	errorHandler := func(msg *kafka.Message) error {
		return errors.New("handler error")
	}

	cons, err := NewConsumer(errorHandler, []string{testTopic}, cfg, mockLogger)
	require.NoError(t, err)
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
		cons.Stop()
	}()

	// Сначала отправляем сообщение, чтобы топик создался
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokers})
	require.NoError(t, err)
	defer producer.Close()

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: kafka.PartitionAny},
		Value:          []byte("test error"),
	}
	deliveryChan := make(chan kafka.Event, 1)
	err = producer.Produce(msg, deliveryChan)
	require.NoError(t, err)

	// Ждем подтверждения доставки
	select {
	case ev := <-deliveryChan:
		if msg, ok := ev.(*kafka.Message); ok && msg.TopicPartition.Error != nil {
			t.Fatalf("Message delivery failed: %v", msg.TopicPartition.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message delivery")
	}

	// Даем время топику создаться и сообщению записаться
	time.Sleep(500 * time.Millisecond)

	// Запускаем consumer (он будет читать с earliest offset)
	// Handler вернет ошибку, но consumer должен продолжить работу
	go func() { _ = cons.Start(ctx) }()

	// Ждем подписки
	time.Sleep(2 * time.Second)

	// Даем время на обработку (handler вернет ошибку, но consumer должен продолжить работу)
	time.Sleep(2 * time.Second)
}

// TestConsumer_CommitBatch проверяет батч коммит
func TestConsumerCommitBatch(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testTopic := fmt.Sprintf("test-batch-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	commitMessages := make(chan *kafka.Message, 20)

	mockLogger := mocks.NewMockLogger()

	commitHandler := func(msg *kafka.Message) error {
		commitMessages <- msg
		return nil
	}

	cfg := ConsumerConfig{
		BootstrapServers:     brokers,
		ClientID:             "test-client",
		GroupID:              groupID,
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}
	cons, err := NewConsumer(commitHandler, []string{testTopic}, cfg, mockLogger)
	require.NoError(t, err)
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
		cons.Stop()
	}()

	// Сначала отправляем 15 сообщений, чтобы топик создался
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokers})
	require.NoError(t, err)
	defer producer.Close()

	for i := 0; i < 15; i++ {
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: kafka.PartitionAny},
			Value:          []byte(fmt.Sprintf("msg-%d", i)),
		}
		deliveryChan := make(chan kafka.Event, 1)
		err = producer.Produce(msg, deliveryChan)
		require.NoError(t, err)

		// Ждем подтверждения доставки
		select {
		case ev := <-deliveryChan:
			if msg, ok := ev.(*kafka.Message); ok && msg.TopicPartition.Error != nil {
				t.Fatalf("Message delivery failed: %v", msg.TopicPartition.Error)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Timeout waiting for message %d delivery", i)
		}
	}

	// Даем время топику создаться и сообщениям записаться
	time.Sleep(500 * time.Millisecond)

	// Запускаем consumer (он будет читать с earliest offset)
	go func() { _ = cons.Start(ctx) }()

	// Ждем подписки
	time.Sleep(2 * time.Second)

	// Ждем обработки всех сообщений
	timeout := time.After(10 * time.Second)
	committed := 0
	for committed < 15 {
		select {
		case <-commitMessages:
			committed++
		case <-timeout:
			t.Fatalf("Timeout waiting for messages. Received %d/15", committed)
		}
	}

	// Проверяем коммиты
	assert.Equal(t, 15, committed)
}

// TestConsumer_GracefulShutdown проверяет graceful shutdown
func TestConsumerGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testTopic := fmt.Sprintf("test-shutdown-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	cfg := ConsumerConfig{
		BootstrapServers:     brokers,
		ClientID:             "test-client",
		GroupID:              groupID,
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}

	handler := func(msg *kafka.Message) error {
		return nil
	}

	cons, err := NewConsumer(handler, []string{testTopic}, cfg, mocks.NewMockLogger())
	require.NoError(t, err)

	// Запускаем consumer в горутине
	go func() { _ = cons.Start(ctx) }()

	// Ждем подписки
	time.Sleep(2 * time.Second)

	// Отменяем контекст для graceful shutdown
	cancel()

	// Даем время на graceful shutdown
	time.Sleep(1 * time.Second)

	// Останавливаем consumer
	err = cons.Stop()
	require.NoError(t, err)
}

// TestConsumerWithHandler интерфейсный handler с context
func TestConsumerInterfaceHandlerWithContext(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testTopic := fmt.Sprintf("test-ctx-handler-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	mockLogger := mocks.NewMockLogger()

	handledCtx := false
	ctxHandler := &mockHandler{
		handleFunc: func(ctx context.Context, msg *kafka.Message) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				handledCtx = true
				return nil
			}
		},
	}

	consumerConfig := ConsumerConfig{
		BootstrapServers:     brokers,
		ClientID:             "test-client",
		GroupID:              groupID,
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}
	cons, err := NewConsumerWithHandler(ctx, ctxHandler, []string{testTopic}, consumerConfig, mockLogger)
	require.NoError(t, err)
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
		cons.Stop()
	}()

	// Сначала отправляем сообщение, чтобы топик создался
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": brokers})
	require.NoError(t, err)
	defer producer.Close()

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &testTopic, Partition: kafka.PartitionAny},
		Value:          []byte("test ctx handler"),
	}
	deliveryChan := make(chan kafka.Event, 1)
	err = producer.Produce(msg, deliveryChan)
	require.NoError(t, err)

	// Ждем подтверждения доставки
	select {
	case ev := <-deliveryChan:
		if msg, ok := ev.(*kafka.Message); ok && msg.TopicPartition.Error != nil {
			t.Fatalf("Message delivery failed: %v", msg.TopicPartition.Error)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for message delivery")
	}

	// Даем время топику создаться и сообщению записаться
	time.Sleep(500 * time.Millisecond)

	// Запускаем consumer (он будет читать с earliest offset)
	go func() { _ = cons.Start(ctx) }()

	// Ждем подписки
	time.Sleep(2 * time.Second)

	// Ждем обработки сообщения
	timeout := time.After(10 * time.Second)
	for !handledCtx {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for message handling")
		case <-time.After(100 * time.Millisecond):
			// Продолжаем ждать
		}
	}

	assert.True(t, handledCtx)
}

// Вспомогательные типы
type mockHandler struct {
	handleFunc func(ctx context.Context, msg *kafka.Message) error
}

func (m *mockHandler) Handle(ctx context.Context, msg *kafka.Message) error {
	return m.handleFunc(ctx, msg)
}

func stringPtr(s string) *string {
	return &s
}

// TestConsumer_RealKafka интеграционный тест с реальным Kafka
// Для запуска требуется доступный Kafka брокер (по умолчанию localhost:9092)
// Можно пропустить тест, установив переменную окружения SKIP_INTEGRATION_TESTS=true
func TestConsumerRealKafka(t *testing.T) {
	// Проверяем, не нужно ли пропустить интеграционные тесты
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Получаем адрес Kafka из переменной окружения или используем дефолтный
	kafkaBrokers := "localhost:9092"
	if brokers := os.Getenv("KAFKA_BROKERS"); brokers != "" {
		kafkaBrokers = brokers
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Создаем уникальный топик для теста
	testTopic := fmt.Sprintf("test-real-kafka-%d", time.Now().UnixNano())
	groupID := fmt.Sprintf("test-group-%d", time.Now().UnixNano())

	// Конфигурация для producer (без consumer-специфичных параметров)
	producerCfg := &kafka.ConfigMap{
		"bootstrap.servers": kafkaBrokers,
		"client.id":         "test-producer",
	}

	// Конфигурация для consumer
	consumerCfg := ConsumerConfig{
		BootstrapServers:     kafkaBrokers,
		ClientID:             "test-client",
		GroupID:              groupID,
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}

	// Создаем logger (можно использовать mock или реальный)
	mockLogger := mocks.NewMockLogger()

	// Создаем producer для отправки сообщения
	producer, err := kafka.NewProducer(producerCfg)
	if err != nil {
		t.Skipf("Kafka is not available at %s: %v. Skipping integration test.", kafkaBrokers, err)
	}
	defer producer.Close()

	// Проверяем доступность Kafka, пытаясь отправить тестовое сообщение
	// Это более надежная проверка, чем просто создание producer
	testCheckCtx, testCheckCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer testCheckCancel()

	checkTopic := "__test_connection_check__"
	testMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &checkTopic,
			Partition: kafka.PartitionAny,
		},
		Value: []byte("connection check"),
	}

	checkChan := make(chan kafka.Event, 1)
	producer.Produce(testMsg, checkChan)

	select {
	case <-testCheckCtx.Done():
		t.Skipf("Kafka is not available at %s: connection timeout. Skipping integration test.", kafkaBrokers)
	case ev := <-checkChan:
		switch e := ev.(type) {
		case *kafka.Message:
			if e.TopicPartition.Error != nil {
				// Приводим error к kafka.Error для проверки кода
				if kafkaErr, ok := e.TopicPartition.Error.(kafka.Error); ok {
					// Если ошибка связана с отсутствием топика, это нормально - Kafka доступен
					if kafkaErr.Code() == kafka.ErrUnknownTopicOrPart {
						// Kafka доступен, но топик не существует - это нормально для проверки
						break
					}
					// Другие ошибки могут означать недоступность Kafka
					if kafkaErr.Code() == kafka.ErrNetworkException ||
						kafkaErr.Code() == kafka.ErrAllBrokersDown {
						t.Skipf("Kafka is not available at %s: %v. Skipping integration test.", kafkaBrokers, kafkaErr)
					}
				}
			}
		case kafka.Error:
			if e.Code() == kafka.ErrNetworkException || e.Code() == kafka.ErrAllBrokersDown {
				t.Skipf("Kafka is not available at %s: %v. Skipping integration test.", kafkaBrokers, e)
			}
		}
	}

	// Создаем consumer
	handledMessages := make(chan *kafka.Message, 10)
	handler := func(msg *kafka.Message) error {
		handledMessages <- msg
		return nil
	}

	cons, err := NewConsumer(handler, []string{testTopic}, consumerCfg, mockLogger)
	require.NoError(t, err, "Failed to create consumer")
	defer cons.Stop()

	// Сначала отправляем тестовое сообщение, чтобы топик создался
	testMessage := "test message from real kafka"
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &testTopic,
			Partition: kafka.PartitionAny,
		},
		Value: []byte(testMessage),
		Key:   []byte("test-key"),
	}

	deliveryChan := make(chan kafka.Event)
	producer.Produce(msg, deliveryChan)

	// Ждем подтверждения доставки
	select {
	case <-ctx.Done():
		t.Fatal("Timeout waiting for message delivery")
	case ev := <-deliveryChan:
		switch e := ev.(type) {
		case *kafka.Message:
			if e.TopicPartition.Error != nil {
				t.Fatalf("Message delivery failed: %v", e.TopicPartition.Error)
			}
		case kafka.Error:
			if e.Code() != kafka.ErrNoError {
				t.Fatalf("Producer error: %v", e)
			}
		}
	}

	// Даем время топику создаться и сообщению записаться
	time.Sleep(500 * time.Millisecond)

	// Запускаем consumer в отдельной горутине (он будет читать с earliest offset)
	consumerCtx, consumerCancel := context.WithCancel(context.Background())
	defer consumerCancel()

	var consumerErr error
	go func() {
		consumerErr = cons.Start(consumerCtx)
	}()

	// Даем время consumer'у подписаться на топик
	time.Sleep(2 * time.Second)

	// Ждем получения сообщения consumer'ом
	select {
	case <-ctx.Done():
		t.Fatal("Timeout waiting for message consumption")
	case receivedMsg := <-handledMessages:
		assert.Equal(t, testMessage, string(receivedMsg.Value))
		assert.Equal(t, testTopic, *receivedMsg.TopicPartition.Topic)
	}

	// Останавливаем consumer
	consumerCancel()
	time.Sleep(500 * time.Millisecond)

	// Проверяем, что consumer завершился без ошибок
	if consumerErr != nil && consumerErr != context.Canceled {
		t.Logf("Consumer error (expected on cancel): %v", consumerErr)
	}
}
