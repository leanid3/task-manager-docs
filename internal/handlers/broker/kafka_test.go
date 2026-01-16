package broker

import (
	"context"
	"errors"
	"testing"

	"app/test/mocks"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRegistry - mock для Registry
type MockRegistry struct {
	mock.Mock
}

func (m *MockRegistry) Handle(ctx context.Context, msg *kafka.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockRegistry) Register(handler TaskHandler) error {
	args := m.Called(handler)
	return args.Error(0)
}

func TestKafkaMessageHandler_Handle(t *testing.T) {
	logger := mocks.NewMockLogger()
	registry := NewRegistry(logger)

	handler := NewKafkaMessageHandler(registry, logger)

	// Create a mock handler to register with the registry
	mockTaskHandler := new(MockTaskHandler)
	mockTaskHandler.On("Topic").Return("test-topic")
	mockTaskHandler.On("Handle", mock.Anything, mock.AnythingOfType("*kafka.Message")).Return(nil)

	err := registry.Register(mockTaskHandler)
	assert.NoError(t, err)

	t.Run("successful message handling", func(t *testing.T) {
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &[]string{"test-topic"}[0],
				Partition: 0,
				Offset:    1,
			},
			Key:   []byte("test-key"),
			Value: []byte("test-value"),
		}

		err := handler.Handle(context.Background(), msg)

		assert.NoError(t, err)
		mockTaskHandler.AssertExpectations(t)
	})

	// For testing error cases, we'd need a different approach
	// since the registry.Handle method is internal to the registry
	// Let's test with a handler that returns an error
	t.Run("handler returns error", func(t *testing.T) {
		// Create a new registry for this test to avoid conflicts
		testRegistry := NewRegistry(logger)
		errorHandler := new(MockTaskHandler)
		errorHandler.On("Topic").Return("error-topic")
		errorHandler.On("Handle", mock.Anything, mock.AnythingOfType("*kafka.Message")).Return(errors.New("handler error"))

		err := testRegistry.Register(errorHandler)
		assert.NoError(t, err)

		testHandler := NewKafkaMessageHandler(testRegistry, logger)

		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic: &[]string{"error-topic"}[0],
			},
		}

		err = testHandler.Handle(context.Background(), msg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler error")
		errorHandler.AssertExpectations(t)
	})
}