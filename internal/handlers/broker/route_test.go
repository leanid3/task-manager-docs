package broker

import (
	"context"
	"errors"
	"testing"

	apperrors "app/internal/entity/errors"
	"app/test/mocks"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
)

// SimpleTestStruct for testing the Route with a concrete type
type SimpleTestStruct struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestRoute_Topic(t *testing.T) {
	logger := mocks.NewMockLogger()

	validator := func(msg *kafka.Message) (*SimpleTestStruct, error) {
		return &SimpleTestStruct{ID: "1", Name: "test"}, nil
	}

	useCase := func(ctx context.Context, data *SimpleTestStruct) error {
		return nil
	}

	route := NewRoute("test-topic", validator, useCase, logger)

	topic := route.Topic()

	assert.Equal(t, "test-topic", topic)
}

func TestRoute_Handle_ValidationSuccess(t *testing.T) {
	logger := mocks.NewMockLogger()

	validator := func(msg *kafka.Message) (*SimpleTestStruct, error) {
		return &SimpleTestStruct{ID: "1", Name: "test"}, nil
	}

	var receivedData *SimpleTestStruct
	useCase := func(ctx context.Context, data *SimpleTestStruct) error {
		receivedData = data
		return nil
	}

	route := NewRoute("test-topic", validator, useCase, logger)

	msg := &kafka.Message{}

	err := route.Handle(context.Background(), msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", receivedData.ID)
	assert.Equal(t, "test", receivedData.Name)
}

func TestRoute_Handle_ValidationFailure(t *testing.T) {
	logger := mocks.NewMockLogger()

	validationError := errors.New("validation failed")
	validator := func(msg *kafka.Message) (*SimpleTestStruct, error) {
		return nil, validationError
	}

	useCase := func(ctx context.Context, data *SimpleTestStruct) error {
		// This should not be called
		t.Error("useCase should not be called when validation fails")
		return nil
	}

	route := NewRoute("test-topic", validator, useCase, logger)

	msg := &kafka.Message{}

	err := route.Handle(context.Background(), msg)

	assert.Error(t, err)
	appErr, ok := err.(*apperrors.AppError)
	assert.True(t, ok)
	assert.Equal(t, apperrors.CodeInvalidMessageFormat, appErr.Code)
	assert.Contains(t, appErr.Message, "validation failed")
}

func TestRoute_Handle_UseCaseFailure(t *testing.T) {
	logger := mocks.NewMockLogger()

	validator := func(msg *kafka.Message) (*SimpleTestStruct, error) {
		return &SimpleTestStruct{ID: "1", Name: "test"}, nil
	}

	useCaseError := errors.New("use case failed")
	useCase := func(ctx context.Context, data *SimpleTestStruct) error {
		return useCaseError
	}

	route := NewRoute("test-topic", validator, useCase, logger)

	msg := &kafka.Message{}

	err := route.Handle(context.Background(), msg)

	assert.Error(t, err)
	assert.Equal(t, useCaseError, err)
}

func TestRoute_Handle_WithRealContext(t *testing.T) {
	logger := mocks.NewMockLogger()

	var receivedCtx context.Context
	validator := func(msg *kafka.Message) (*SimpleTestStruct, error) {
		return &SimpleTestStruct{ID: "ctx-test", Name: "context"}, nil
	}

	useCase := func(ctx context.Context, data *SimpleTestStruct) error {
		receivedCtx = ctx
		return nil
	}

	route := NewRoute("test-topic", validator, useCase, logger)

	msg := &kafka.Message{}

	ctx := context.WithValue(context.Background(), "test-key", "test-value")
	err := route.Handle(ctx, msg)

	assert.NoError(t, err)
	assert.Equal(t, "test-value", receivedCtx.Value("test-key"))
}