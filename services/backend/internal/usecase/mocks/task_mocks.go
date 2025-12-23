package mocks

import (
	"app/internal/entity/domain"
	pkgminio "app/pkg/minio"
	"context"
	"encoding/json"
	"io"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, task *domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Task), args.Error(1)
}

func (m *MockRepository) UpdateWithStatus(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus) error {
	args := m.Called(ctx, taskID, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateWithResult(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, result json.RawMessage) error {
	args := m.Called(ctx, taskID, status, result)
	return args.Error(0)
}

func (m *MockRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockRepository) UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error {
	args := m.Called(ctx, taskID, status, errorMessage)
	return args.Error(0)
}

type MockStorageRepository struct {
	mock.Mock
}

func (m *MockStorageRepository) UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts pkgminio.UploadOptions) (*pkgminio.ObjectInfo, error) {
	args := m.Called(ctx, objectName, reader, objectSize, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgminio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) DeleteObject(ctx context.Context, objectName string) error {
	args := m.Called(ctx, objectName)
	return args.Error(0)
}

func (m *MockStorageRepository) GetObjectMetadata(ctx context.Context, objectName string) (*pkgminio.ObjectInfo, error) {
	args := m.Called(ctx, objectName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pkgminio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) FileExists(ctx context.Context, objectName string) (bool, error) {
	args := m.Called(ctx, objectName)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageRepository) ListFiles(ctx context.Context, prefix string) ([]pkgminio.ObjectInfo, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]pkgminio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) BucketName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageRepository) GenerateStoragePath(taskID uuid.UUID, filename string) string {
	args := m.Called(taskID, filename)
	return args.String(0)
}

type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Send(ctx context.Context, topic string, key string, headers map[string]string, value interface{}) error {
	args := m.Called(ctx, topic, key, headers, value)
	return args.Error(0)
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	allArgs := append([]interface{}{msg}, args...)
	m.Called(allArgs...)
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}

func (m *MockLogger) ErrorWithSkip(skip int, msg string, args ...interface{}) {
	m.Called(append([]interface{}{skip, msg}, args...)...)
}

func (m *MockLogger) Fatal(msg string, args ...interface{}) {
	m.Called(append([]interface{}{msg}, args...)...)
}
