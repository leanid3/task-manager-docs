package mocks

import (
	"app/internal/entity/domain"
	"context"
	"encoding/json"
	"io"
	"testing"

	pkgminio "app/pkg/minio"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// MockRepository - мок для Repository
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

func (m *MockRepository) UpdateWithStatus(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus) error {
	args := m.Called(ctx, taskID, worker_id, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateWithResult(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus, result json.RawMessage) error {
	args := m.Called(ctx, taskID, worker_id, status, result)
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

// MockRepository - мок для Repository
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

// MockProducer - мок для Producer
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

// MockTaskLLMUC - мок для TaskLLMUC
type TaskLLMUC struct {
	mock.Mock
}

func (m *TaskLLMUC) CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error) {
	args := m.Called(ctx, reader, filename, filesize, requestID)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *TaskLLMUC) GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Task), args.Error(1)
}

func (m *TaskLLMUC) UpdateTaskStatus(ctx context.Context, evt domain.TaskLLMStatusEvent) error {
	args := m.Called(ctx, evt)
	return args.Error(0)
}

func NewMockTaskLLMUC(t *testing.T) *TaskLLMUC {
	return new(TaskLLMUC)
}
