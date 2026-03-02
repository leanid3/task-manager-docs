package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"app/internal/entity/domain"
	"app/internal/usecase"
	"app/pkg/deduplication"
	"app/pkg/errors"
	"app/pkg/limits"
	pkgminio "app/pkg/minio"
	"app/pkg/validation"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock для репозитория задач
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, taskID uuid.UUID) (domain.Task, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(domain.Task), args.Error(1)
}

func (m *MockTaskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]domain.Task, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Task), args.Error(1)
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

// Mock для репозитория хранилища
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

// Тест интеграции всех компонентов
func TestUnifiedTaskIntegration(t *testing.T) {
	// Создаем моки
	mockTaskRepo := new(MockTaskRepository)
	mockStorageRepo := new(MockStorageRepository)

	// Создаем компоненты
	validator := validation.NewCentralizedValidator()
	deduplicator := deduplication.NewInMemoryDeduplicator(24 * time.Hour)
	errorClassifier := errors.NewStandardErrorClassifier()
	timeoutManager := limits.NewDefaultTimeoutManager()
	resourceLimiter := limits.NewSemaphoreResourceLimiter()
	resourceLimiter.SetLimit("tasks", 10)

	// Создаем фабрику процессоров задач
	taskProcessorFactory := usecase.NewTaskProcessorFactory()

	// Создаем универсальный usecase
	unifiedTaskUC := usecase.NewUnifiedTaskUC(mockTaskRepo, mockStorageRepo, taskProcessorFactory)

	// Подготовка тестовых данных
	taskID := uuid.New()
	testTask := &domain.BaseTask{
		TaskID:    taskID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now(),
	}

	// Настройка моков
	mockTaskRepo.On("GetByID", mock.Anything, taskID).Return(testTask, nil)
	mockTaskRepo.On("UpdateWithStatus", mock.Anything, taskID, mock.Anything, domain.TaskStatusProcessing).Return(nil)

	// Настройка моков для создания задачи
	mockTaskRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// Тест создания задачи
	ctx := context.Background()
	input := domain.TaskInput{
		TaskID:    taskID,
		Filename:  "test.pdf",
		Filesize:  1024,
		RequestID: "req-123",
		Metadata: map[string]interface{}{
			"task_type": string(domain.TaskTypeLLM),
		},
	}

	// Проверяем валидацию
	err := validator.ValidateTaskInput(input.Filename, input.Filesize, input.RequestID)
	assert.NoError(t, err)

	// Проверяем дедупликацию
	taskHash := deduplicator.GenerateTaskHash(input.TaskID, input.Filename, input.Filesize, input.RequestID)
	isDuplicate, err := deduplicator.IsDuplicate(ctx, taskHash)
	assert.NoError(t, err)
	assert.False(t, isDuplicate)

	// Создаем задачу
	createdTaskID, err := unifiedTaskUC.CreateTask(ctx, input)
	assert.NoError(t, err)
	assert.Equal(t, taskID, createdTaskID)

	// Проверяем, что задача была помечена как обработанная в дедупликаторе
	err = deduplicator.MarkAsProcessed(ctx, taskHash, time.Hour)
	assert.NoError(t, err)

	// Проверяем обновление статуса задачи
	event := domain.TaskEvent{
		Key: domain.TaskContractKey{TaskID: taskID},
		Headers: domain.TaskContractHeaders{
			Status:   domain.TaskStatusProcessing.ToKafkaCode(),
			WorkerID: "worker-1",
		},
		Value: map[string]interface{}{
			"result": "processing",
		},
	}

	err = unifiedTaskUC.UpdateTaskStatus(ctx, event)
	assert.NoError(t, err)

	// Проверяем обработку задачи
	err = unifiedTaskUC.ProcessTask(ctx, taskID)
	assert.NoError(t, err)

	// Проверяем ограничения ресурсов
	err = resourceLimiter.Acquire(ctx, "tasks")
	assert.NoError(t, err)

	// Освобождаем ресурс
	resourceLimiter.Release("tasks")

	// Проверяем классификацию ошибок
	testErr := assert.AnError
	errType, severity := errorClassifier.Classify(testErr)
	assert.NotEqual(t, "", string(errType))
	assert.NotEqual(t, "", string(severity))

	// Проверяем таймаут
	taskCtx, cancel := timeoutManager.WithTimeout(ctx, taskID, "llm")
	defer cancel()

	select {
	case <-taskCtx.Done():
		t.Log("Task context completed")
	default:
		t.Log("Task context still active")
	}

	// Проверяем, что все моки были вызваны
	mockTaskRepo.AssertExpectations(t)
}
