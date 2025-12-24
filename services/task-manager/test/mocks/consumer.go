package mocks

import (
	"context"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// MockConsumer для тестирования без реального Kafka
type MockConsumer struct {
	messages     chan *kafka.Message
	errors       chan error
	subscribeErr error
	commitErr    error
	closed       bool
	mu           sync.Mutex
}

func NewMockConsumer() *MockConsumer {
	return &MockConsumer{
		messages: make(chan *kafka.Message, 10),
		errors:   make(chan error, 10),
	}
}

func (m *MockConsumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	select {
	case msg := <-m.messages:
		return msg, nil
	case err := <-m.errors:
		return nil, err
	case <-time.After(timeout):
		return nil, kafka.NewError(kafka.ErrTimedOut, "timeout", false)
	}
}

func (m *MockConsumer) SubscribeTopics(topics []string, rebalanceCb kafka.RebalanceCb) error {
	return m.subscribeErr
}

func (m *MockConsumer) CommitMessage(msg *kafka.Message) error {
	return m.commitErr
}

func (m *MockConsumer) SendMessage(ctx context.Context, topic string, msg *kafka.Message) {
	select {
	case m.messages <- msg:
	case <-ctx.Done():
	}
}

func (m *MockConsumer) Close() {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	close(m.messages)
	close(m.errors)
}
