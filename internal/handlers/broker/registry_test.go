package broker

import (
	"context"
	"errors"
	"testing"

	apperrors "app/internal/entity/errors"
	"app/test/mocks"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTaskHandler - mock для TaskHandler интерфейса
type MockTaskHandler struct {
	mock.Mock
}

func (m *MockTaskHandler) Handle(ctx context.Context, msg *kafka.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockTaskHandler) Topic() string {
	args := m.Called()
	return args.String(0)
}

func TestRegistry_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		logger := mocks.NewMockLogger()
		registry := NewRegistry(logger)
		mockHandler := new(MockTaskHandler)
		mockHandler.On("Topic").Return("test-topic")

		err := registry.Register(mockHandler)

		assert.NoError(t, err)
		assert.Len(t, registry.handlers, 1)
		assert.Contains(t, registry.handlers, "test-topic")
		mockHandler.AssertExpectations(t)
	})

	t.Run("duplicate registration should fail", func(t *testing.T) {
		logger := mocks.NewMockLogger()
		registry := NewRegistry(logger)
		mockHandler1 := new(MockTaskHandler)
		mockHandler1.On("Topic").Return("test-topic")

		mockHandler2 := new(MockTaskHandler)
		mockHandler2.On("Topic").Return("test-topic")

		// Register first handler
		err := registry.Register(mockHandler1)
		assert.NoError(t, err)

		// Attempt to register second handler with same topic
		err = registry.Register(mockHandler2)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
		mockHandler1.AssertExpectations(t)
		mockHandler2.AssertExpectations(t)
	})
}

func TestRegistry_Handle(t *testing.T) {
	t.Run("successful handling", func(t *testing.T) {
		logger := mocks.NewMockLogger()
		registry := NewRegistry(logger)
		mockHandler := new(MockTaskHandler)
		mockHandler.On("Topic").Return("test-topic")
		mockHandler.On("Handle", mock.Anything, mock.AnythingOfType("*kafka.Message")).Return(nil)

		err := registry.Register(mockHandler)
		assert.NoError(t, err)

		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic: &[]string{"test-topic"}[0],
			},
		}

		err = registry.Handle(context.Background(), msg)

		assert.NoError(t, err)
		mockHandler.AssertExpectations(t)
	})

	t.Run("no handler for topic", func(t *testing.T) {
		logger := mocks.NewMockLogger()
		registry := NewRegistry(logger)
		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic: &[]string{"nonexistent-topic"}[0],
			},
		}

		err := registry.Handle(context.Background(), msg)

		assert.Error(t, err)
		appErr, ok := err.(*apperrors.AppError)
		assert.True(t, ok)
		assert.Equal(t, apperrors.CodeKafkaError, appErr.Code)
		assert.Contains(t, appErr.Message, "no handler for topic")
	})

	t.Run("handler returns error", func(t *testing.T) {
		logger := mocks.NewMockLogger()
		registry := NewRegistry(logger)
		mockHandler := new(MockTaskHandler)
		mockHandler.On("Topic").Return("test-topic")
		mockHandler.On("Handle", mock.Anything, mock.AnythingOfType("*kafka.Message")).Return(errors.New("handler error"))

		err := registry.Register(mockHandler)
		assert.NoError(t, err)

		msg := &kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic: &[]string{"test-topic"}[0],
			},
		}

		err = registry.Handle(context.Background(), msg)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler error")
		mockHandler.AssertExpectations(t)
	})
}
