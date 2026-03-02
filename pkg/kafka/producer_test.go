//go:build integration

package kafka

import (
	"context"
	"fmt"
	"testing"
	"time"

	"app/test/mocks"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestProducerSendSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	// Создаем продюсера
	mockLogger := mocks.NewMockLogger()
	producer, err := NewProducer(ProducerConfig{
		BootstrapServers: brokers,
		ClientID:         "test-producer",
	}, mockLogger)
	require.NoError(t, err)
	defer producer.Close()

	// Тестовое сообщение с использованием BasicMessage
	testTopic := fmt.Sprintf("test-producer-%d", time.Now().UnixNano())
	taskID := uuid.New()
	headers := map[string]string{
		"task-id": taskID.String(),
		"status":  "pending",
	}
	value := []byte(`{"test":"data"}`)

	msg := NewBasicMessage(testTopic, taskID.String(), headers, value)

	// ✅ Верифицируем доставку через consumer (consumer запустится ДО отправки)
	verifyMessageDelivered(t, brokers, testTopic, taskID, msg, producer)
}

func TestProducerSendContextTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	// Producer с отключенными retries для быстрого таймаута
	producer, err := NewProducer(ProducerConfig{
		BootstrapServers: brokers,
		ClientID:         "test-producer",
	}, mocks.NewMockLogger())
	require.NoError(t, err)
	defer producer.Close()

	taskID := uuid.New()
	headers := map[string]string{"task-id": taskID.String()}
	value := []byte(`{}`)

	// Таймаут контекста 1ms
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err = producer.Send(ctx, "timeout-topic", taskID.String(), headers, value)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestProducerCloseFlush(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	producer, err := NewProducer(ProducerConfig{
		BootstrapServers: brokers,
		ClientID:         "test-producer",
	}, mocks.NewMockLogger())
	require.NoError(t, err)
	defer producer.Close()

	// Отправляем несколько сообщений
	testTopic := "flush-test"
	taskID := uuid.New()
	headers := map[string]string{"task-id": taskID.String()}
	value := []byte(`{}`)

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, producer.Send(ctx, testTopic, taskID.String(), headers, value))
	}

	// Graceful close
	err = producer.Close()
	require.NoError(t, err)

	// ✅ Все сообщения доставлены
	// Создаем новый producer для проверки, так как старый закрыт
	verifyProducer, err := NewProducer(ProducerConfig{
		BootstrapServers: brokers,
		ClientID:         "test-producer-verify",
	}, mocks.NewMockLogger())
	require.NoError(t, err)
	defer verifyProducer.Close()
	verifyMessageDelivered(t, brokers, testTopic, taskID, NewBasicMessage(testTopic, taskID.String(), headers, value), verifyProducer)
}

func TestProducerHeadersAndKey(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	brokers, cleanup := setupTestKafka(t)
	defer cleanup()

	producer, err := NewProducer(ProducerConfig{
		BootstrapServers: brokers,
		ClientID:         "test-producer",
	}, mocks.NewMockLogger())
	require.NoError(t, err)
	defer producer.Close()

	testTopic := fmt.Sprintf("headers-test-%d", time.Now().UnixNano())
	taskID := uuid.New()
	headers := map[string]string{
		"task-id":   taskID.String(),
		"trace-id":  uuid.New().String(),
		"worker-id": "worker-1",
	}
	value := []byte(`{}`)

	msg := NewBasicMessage(testTopic, taskID.String(), headers, value)

	// ✅ Проверяем headers и key через consumer (consumer запустится ДО отправки)
	verifyMessageDelivered(t, brokers, testTopic, taskID, msg, producer)
}

// Вспомогательная функция - проверяет доставку через consumer
func verifyMessageDelivered(t *testing.T, brokers, topic string, expectedTaskID uuid.UUID, msg MessageInterface, producer Producer) {
	handledMessages := make(chan *kafka.Message, 10)
	handler := func(msg *kafka.Message) error {
		handledMessages <- msg
		return nil
	}

	// Создаем consumer с уникальным groupID для каждого теста
	// Используем earliest offset, чтобы читать сообщения с начала топика
	consumer, err := NewConsumer(handler, []string{topic}, ConsumerConfig{
		BootstrapServers:     brokers,
		ClientID:             fmt.Sprintf("test-client-%d", time.Now().UnixNano()),
		GroupID:              fmt.Sprintf("test-group-%d", time.Now().UnixNano()),
		EnableAutoCommit:     true,
		AutoCommitIntervalMs: 5000,
		SessionTimeoutMs:     30000,
		HeartbeatIntervalMs:  3000,
		AutoOffsetReset:      "earliest", // Читаем с начала топика для тестов
	}, mocks.NewMockLogger())
	require.NoError(t, err)
	// Не используем defer consumer.Stop() - Stop() вызывается автоматически из Start() при отмене контекста

	// Сначала отправляем сообщение, чтобы топик точно создался
	// (Kafka создает топики автоматически при первой отправке)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Запускаем consumer в отдельной горутине
	go func() {
		_ = consumer.Start(ctx)
	}()

	// Ждем немного, чтобы consumer успел подписаться
	time.Sleep(2 * time.Second)

	// Отправляем сообщение
	err = producer.Send(ctx, msg.ToTopic(), msg.ToKey(), msg.ToHeaders(), msg.ToValue())
	require.NoError(t, err)

	// Ждем обработки сообщения
	select {
	case receivedMsg := <-handledMessages:
		// Проверяем ключ
		if string(receivedMsg.Key) != msg.ToKey() {
			t.Errorf("Expected key %s, got %s", msg.ToKey(), string(receivedMsg.Key))
		}
		
		// Проверяем заголовки
		expectedHeaders := msg.ToHeaders()
		for k, v := range expectedHeaders {
			found := false
			for _, header := range receivedMsg.Headers {
				if header.Key == k && string(header.Value) == v {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Header %s=%s not found in received message", k, v)
			}
		}
		
		// Проверяем значение
		if string(receivedMsg.Value) != string(msg.ToValue()) {
			t.Errorf("Expected value %s, got %s", string(msg.ToValue()), string(receivedMsg.Value))
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Timeout waiting for message handling")
	}

	cancel()
}