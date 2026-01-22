package usecase

import (
	"app/internal/entity/domain"
	"app/pkg/minio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTaskRepository - мок для repository.Task
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Create(ctx context.Context, task *domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) GetByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	args := m.Called(ctx, taskID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Task), args.Error(1)
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

func (m *MockTaskRepository) ListByStatus(ctx context.Context, status domain.TaskStatus, limit int) ([]*domain.Task, error) {
	args := m.Called(ctx, status, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

// MockStorageRepository - мок для repository.Storage
type MockStorageRepository struct {
	mock.Mock
}

func (m *MockStorageRepository) UploadStream(ctx context.Context, objectName string, reader io.Reader, objectSize int64, opts minio.UploadOptions) (*minio.ObjectInfo, error) {
	args := m.Called(ctx, objectName, reader, objectSize, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*minio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) DeleteObject(ctx context.Context, objectName string) error {
	args := m.Called(ctx, objectName)
	return args.Error(0)
}

func (m *MockStorageRepository) GetObjectMetadata(ctx context.Context, objectName string) (*minio.ObjectInfo, error) {
	args := m.Called(ctx, objectName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*minio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) FileExists(ctx context.Context, objectName string) (bool, error) {
	args := m.Called(ctx, objectName)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorageRepository) ListFiles(ctx context.Context, prefix string) ([]minio.ObjectInfo, error) {
	args := m.Called(ctx, prefix)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]minio.ObjectInfo), args.Error(1)
}

func (m *MockStorageRepository) BucketName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorageRepository) GenerateStoragePath(taskID uuid.UUID, filename string) string {
	args := m.Called(taskID, filename)
	return args.String(0)
}

// MockTaskProcessorFactory - мок для domain.TaskProcessorFactory
type MockTaskProcessorFactory struct {
	mock.Mock
}

func (m *MockTaskProcessorFactory) Create(taskType domain.TaskType) (domain.TaskProcessor, error) {
	args := m.Called(taskType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(domain.TaskProcessor), args.Error(1)
}

func (m *MockTaskProcessorFactory) Register(taskType domain.TaskType, processor domain.TaskProcessor) error {
	args := m.Called(taskType, processor)
	return args.Error(0)
}

// MockTaskProcessor - мок для domain.TaskProcessor
type MockTaskProcessor struct {
	mock.Mock
}

func (m *MockTaskProcessor) Process(ctx context.Context, task *domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskProcessor) GetType() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockTaskProcessor) Validate(task *domain.Task) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *MockTaskProcessor) GetTimeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

func TestUnifiedTaskUC_CreateTask(t *testing.T) {
	tests := []struct {
		name          string
		input         domain.TaskInput
		setupMocks    func(*MockTaskRepository, *MockStorageRepository)
		expectedError bool
	}{
		{
			name: "successful task creation",
			input: domain.TaskInput{
				TaskID:    uuid.New(),
				Filename:  "test.pdf",
				Filesize:  1024,
				RequestID: "req-123",
				Metadata:  map[string]interface{}{"task_type": "LLM"},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Task")).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "task creation with database error",
			input: domain.TaskInput{
				TaskID:    uuid.New(),
				Filename:  "test.pdf",
				Filesize:  1024,
				RequestID: "req-123",
				Metadata:  map[string]interface{}{"task_type": "LLM"},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Task")).Return(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			mockStorageRepo := new(MockStorageRepository)
			tt.setupMocks(mockRepo, mockStorageRepo)

			uc := NewUnifiedTaskUC(mockRepo, mockStorageRepo, nil)

			taskID, err := uc.CreateTask(context.Background(), tt.input)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, uuid.Nil, taskID)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, taskID)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUnifiedTaskUC_GetTaskByID(t *testing.T) {
	tests := []struct {
		name          string
		taskID        uuid.UUID
		setupMocks    func(*MockTaskRepository, *MockStorageRepository)
		expectedError bool
	}{
		{
			name:   "successful task retrieval",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
				}, nil)
			},
			expectedError: false,
		},
		{
			name:   "task retrieval with database error",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			mockStorageRepo := new(MockStorageRepository)
			tt.setupMocks(mockRepo, mockStorageRepo)

			uc := NewUnifiedTaskUC(mockRepo, mockStorageRepo, nil)

			task, err := uc.GetTaskByID(context.Background(), tt.taskID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUnifiedTaskUC_UpdateTaskStatus(t *testing.T) {
	tests := []struct {
		name          string
		event         domain.TaskEvent
		setupMocks    func(*MockTaskRepository, *MockStorageRepository)
		expectedError bool
	}{
		{
			name: "update to processing status",
			event: domain.TaskEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status:   2, // Processing
					WorkerID: "worker-1",
				},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
				}, nil)
				repo.On("UpdateWithStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), "worker-1", domain.TaskStatusProcessing).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "update to completed status with result",
			event: domain.TaskEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status:   3, // Completed
					WorkerID: "worker-1",
				},
				Value: map[string]interface{}{
					"result": "test result",
				},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
				}, nil)
				repo.On("UpdateWithResult", mock.Anything, mock.AnythingOfType("uuid.UUID"), "worker-1", domain.TaskStatusCompleted, mock.AnythingOfType("json.RawMessage")).Return(nil)
			},
			expectedError: false,
		},
		{
			name: "update to failed status with error",
			event: domain.TaskEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status:   4, // Failed
					WorkerID: "worker-1",
				},
				Value: map[string]interface{}{
					"error_message": "test error",
				},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
				}, nil)
				repo.On("UpdateWithError", mock.Anything, mock.AnythingOfType("uuid.UUID"), domain.TaskStatusFailed, "test error").Return(nil)
			},
			expectedError: false,
		},
		{
			name: "unknown status code",
			event: domain.TaskEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status:   99, // Unknown
					WorkerID: "worker-1",
				},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				// No calls should be made for unknown status codes
			},
			expectedError: false, // Function returns nil for unknown status codes
		},
		{
			name: "get task error",
			event: domain.TaskEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status:   2, // Processing
					WorkerID: "worker-1",
				},
			},
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository) {
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			mockStorageRepo := new(MockStorageRepository)
			tt.setupMocks(mockRepo, mockStorageRepo)

			uc := NewUnifiedTaskUC(mockRepo, mockStorageRepo, nil)

			err := uc.UpdateTaskStatus(context.Background(), tt.event)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUnifiedTaskUC_ProcessTask(t *testing.T) {
	tests := []struct {
		name          string
		taskID        uuid.UUID
		setupMocks    func(*MockTaskRepository, *MockStorageRepository, *MockTaskProcessorFactory, *MockTaskProcessor)
		expectedError bool
	}{
		{
			name:   "successful task processing",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository, factory *MockTaskProcessorFactory, processor *MockTaskProcessor) {
				// Get task
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
					Metadata: map[string]interface{}{
						"task_type": "LLM",
					},
				}, nil)

				// Update to processing
				repo.On("UpdateWithStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), "", domain.TaskStatusProcessing).Return(nil)

				// Factory create
				factory.On("Create", domain.TaskType("LLM")).Return(processor, nil)

				// Processor process
				processor.On("Process", mock.Anything, mock.AnythingOfType("*domain.Task")).Return(nil)
			},
			expectedError: false,
		},
		{
			name:   "task processing with processor error",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository, factory *MockTaskProcessorFactory, processor *MockTaskProcessor) {
				// Get task
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
					Metadata: map[string]interface{}{
						"task_type": "LLM",
					},
				}, nil)

				// Update to processing
				repo.On("UpdateWithStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), "", domain.TaskStatusProcessing).Return(nil)

				// Factory create
				factory.On("Create", domain.TaskType("LLM")).Return(processor, nil)

				// Processor process - returns error
				processor.On("Process", mock.Anything, mock.AnythingOfType("*domain.Task")).Return(errors.New("processing error"))

				// Update to failed due to processing error
				repo.On("UpdateWithError", mock.Anything, mock.AnythingOfType("uuid.UUID"), domain.TaskStatusFailed, "processing error").Return(nil)
			},
			expectedError: true,
		},
		{
			name:   "task without task type in metadata",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository, factory *MockTaskProcessorFactory, processor *MockTaskProcessor) {
				// Get task without task_type in metadata
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
					Metadata: map[string]interface{}{}, // No task_type
				}, nil)
			},
			expectedError: false, // Function returns nil when task type is not specified
		},
		{
			name:   "task with invalid processor",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository, factory *MockTaskProcessorFactory, processor *MockTaskProcessor) {
				// Get task
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&domain.Task{
					TaskID: uuid.New(),
					Status: domain.TaskStatusPending,
					Metadata: map[string]interface{}{
						"task_type": "LLM",
					},
				}, nil)

				// Factory create - returns error
				factory.On("Create", domain.TaskType("LLM")).Return(nil, errors.New("processor not found"))
			},
			expectedError: true,
		},
		{
			name:   "get task error",
			taskID: uuid.New(),
			setupMocks: func(repo *MockTaskRepository, storageRepo *MockStorageRepository, factory *MockTaskProcessorFactory, processor *MockTaskProcessor) {
				// Get task - returns error
				repo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(nil, errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockTaskRepository)
			mockStorageRepo := new(MockStorageRepository)
			mockFactory := new(MockTaskProcessorFactory)
			mockProcessor := new(MockTaskProcessor)
			tt.setupMocks(mockRepo, mockStorageRepo, mockFactory, mockProcessor)

			uc := NewUnifiedTaskUC(mockRepo, mockStorageRepo, mockFactory)

			err := uc.ProcessTask(context.Background(), tt.taskID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockFactory.AssertExpectations(t)
			if mockProcessor != nil {
				mockProcessor.AssertExpectations(t)
			}
		})
	}
}