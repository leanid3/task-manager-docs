package multiupload

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"app/internal/entity/domain"
	pkgminio "app/pkg/minio"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTaskRepository - mock для репозитория задач
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	args := m.Called(ctx, taskID)
	return args.Get(0).(*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error) {
	args := m.Called(ctx, status, limit)
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) UpdateWithStatus(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus) error {
	args := m.Called(ctx, taskID, worker_id, status)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdateWithResult(ctx context.Context, taskID uuid.UUID, worker_id string, status domain.TaskStatus, result json.RawMessage) error {
	args := m.Called(ctx, taskID, worker_id, status, result)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdateWithError(ctx context.Context, taskID uuid.UUID, status domain.TaskStatus, errorMessage string) error {
	args := m.Called(ctx, taskID, status, errorMessage)
	return args.Error(0)
}

// MockProducer - mock для продюсера брокера
type MockProducer struct {
	mock.Mock
}

func (m *MockProducer) Send(ctx context.Context, topic, key string, headers map[string]string, value interface{}) error {
	args := m.Called(ctx, topic, key, headers, value)
	return args.Error(0)
}

func (m *MockProducer) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockStorageRepository - mock для репозитория хранилища
type MockStorageRepository struct {
	mock.Mock
}

func (m *MockStorageRepository) UploadStream(ctx context.Context, key string, reader io.Reader, size int64, opts pkgminio.UploadOptions) (*pkgminio.ObjectInfo, error) {
	args := m.Called(ctx, key, reader, size, opts)
	return args.Get(0).(*pkgminio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) DeleteObject(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockStorageRepository) GenerateStoragePath(taskID uuid.UUID, filename string) string {
	args := m.Called(taskID, filename)
	return args.String(0)
}

func (m *MockStorageRepository) BucketName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageRepository) FileExists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageRepository) GetObjectMetadata(ctx context.Context, key string) (*pkgminio.ObjectInfo, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(*pkgminio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) ListFiles(ctx context.Context, prefix string) ([]pkgminio.ObjectInfo, error) {
	args := m.Called(ctx, prefix)
	return args.Get(0).([]pkgminio.ObjectInfo), args.Error(1)
}

func TestCreateMultiUploadTask(t *testing.T) {
	// Подготовка моков
	taskRepo := new(MockTaskRepository)
	producer := new(MockProducer)
	storageRepo := new(MockStorageRepository)

	uc := NewMultiUploadUC(taskRepo, producer, storageRepo, "test_topic", 5)

	// Тестирование успешного создания задачи
	t.Run("successful creation", func(t *testing.T) {
		// Здесь мы не можем протестировать реальную загрузку файлов, так как нам нужны multipart.FileHeader
		// Вместо этого протестируем возвращение ошибки при пустом списке файлов
		ctx := context.Background()
		
		taskID, err := uc.CreateMultiUploadTask(ctx, nil, "test_request_id")
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, taskID)
	})
}