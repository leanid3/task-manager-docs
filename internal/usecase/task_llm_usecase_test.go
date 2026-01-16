package usecase

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	pkgminio "app/pkg/minio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"app/test/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskUCCreateTask(t *testing.T) {
	tests := []struct {
		name          string
		reader        io.Reader
		filename      string
		filesize      int64
		requestID     string
		setupMocks    func(*mocks.MockRepository, *mocks.MockProducer, *mocks.MockStorageRepository, *mocks.Logger)
		expectedError bool
		errorCode     apperrors.ErrorCode
	}{
		{
			name:      "задача успешно создана",
			reader:    bytes.NewReader([]byte("test content")),
			filename:  "test.pdf",
			filesize:  100,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, logger *mocks.Logger) {
				// Мокируем BucketName
				storage.On("GenerateStoragePath", mock.Anything, mock.Anything).Return("test-bucket/test-id/test.pdf")

				//Мокируем загрузку файла, успшное сохранение
				storage.On("UploadStream", mock.Anything, mock.Anything, mock.Anything, int64(100), mock.Anything).
					Return(&pkgminio.ObjectInfo{}, nil)

				//Мокируем создание в БД
				//! без транзакции
				repo.On("Create", mock.Anything, mock.MatchedBy(func(task *domain.Task) bool {
					return task.Status == domain.TaskStatusPending
				})).
					Return(nil)

				//Мокируем публикацию в очередь
				producer.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
			},
			expectedError: false,
		},
		{
			name:      "ошибка валидации: nil reader",
			reader:    nil,
			filename:  "test.pdf",
			filesize:  100,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, log *mocks.Logger) {
			},
			expectedError: true,
			errorCode:     apperrors.CodeValidationFailed,
		},
		{
			name:      "ошибка валидации: не передан filename",
			reader:    bytes.NewReader([]byte("test")),
			filename:  "",
			filesize:  100,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, log *mocks.Logger) {
			},
			expectedError: true,
			errorCode:     apperrors.CodeValidationFailed,
		}, {

			name:      "ошибка валидации: filesize <= 0 или не передан",
			reader:    bytes.NewReader([]byte("test")),
			filename:  "test.pdf",
			filesize:  0,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, log *mocks.Logger) {
			},
			expectedError: true,
			errorCode:     apperrors.CodeValidationFailed,
		},
		{
			name:      "ошибка загрузки файла в хранилище",
			reader:    bytes.NewReader([]byte("test content")),
			filename:  "test.pdf",
			filesize:  100,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, log *mocks.Logger) {
				storage.On("GenerateStoragePath", mock.Anything, mock.Anything).
					Return("test-bucket/test-id/test.pdf")
				storage.On("UploadStream", mock.Anything, mock.Anything, mock.Anything, int64(100), mock.Anything).
					Return(nil, errors.New("не удалось подключиться к хранилищу"))
			},
			expectedError: true,
			errorCode:     apperrors.CodeStorageError,
		},
		{
			name:      "ошибка создания задачи в БД",
			reader:    bytes.NewReader([]byte("test content")),
			filename:  "test.pdf",
			filesize:  100,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, logger *mocks.Logger) {
				storage.On("GenerateStoragePath", mock.Anything, mock.Anything).
					Return("test-bucket/test-id/test.pdf")
				storage.On("UploadStream", mock.Anything, mock.Anything, mock.Anything, int64(100), mock.Anything).
					Return(&pkgminio.ObjectInfo{}, nil)
				storage.On("DeleteObject", mock.Anything, mock.Anything).
					Return(nil)
				repo.On("Create", mock.Anything, mock.Anything).
					Return(errors.New("database error"))
			},
			expectedError: true,
			errorCode:     apperrors.CodeDatabaseError,
		},
		{
			name:      "ошибка публикации в очередь",
			reader:    bytes.NewReader([]byte("test content")),
			filename:  "test.pdf",
			filesize:  100,
			requestID: "req-123",
			setupMocks: func(repo *mocks.MockRepository, producer *mocks.MockProducer, storage *mocks.MockStorageRepository, logger *mocks.Logger) {
				storage.On("GenerateStoragePath", mock.Anything, mock.Anything).
					Return("test-bucket/test-id/test.pdf")
				storage.On("UploadStream", mock.Anything, mock.Anything, mock.Anything, int64(100), mock.Anything).
					Return(&pkgminio.ObjectInfo{}, nil)
				storage.On("DeleteObject", mock.Anything, mock.Anything).
					Return(nil)
				repo.On("Create", mock.Anything, mock.Anything).
					Return(nil)
				repo.On("UpdateWithError", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil)
				producer.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("ошибка публикации в очередь"))
			},
			expectedError: true,
			errorCode:     apperrors.CodeKafkaError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := new(mocks.MockStorageRepository)
			mockRepo := new(mocks.MockRepository)
			mockProducer := new(mocks.MockProducer)
			mockLogger := new(mocks.Logger)

			tt.setupMocks(mockRepo, mockProducer, mockStorage, mockLogger)

			uc := NewTaskLLMUC(mockRepo, mockProducer, mockStorage, "tasks_llm", mockLogger)

			taskID, err := uc.CreateTask(context.Background(), tt.reader, tt.filename, tt.filesize, tt.requestID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorCode != "" {
					if appErr, ok := apperrors.IsAppError(err); ok {
						assert.Equal(t, tt.errorCode, appErr.Code)
					}
				}
				assert.Equal(t, uuid.Nil, taskID)
			} else {
				assert.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, taskID)
			}

			mockStorage.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
			mockProducer.AssertExpectations(t)
		})
	}

}

func TestTaskUCGetTaskByID(t *testing.T) {
	tests := []struct {
		name          string
		taskID        uuid.UUID
		setupMocks    func(*mocks.MockRepository, *mocks.Logger)
		expectedError bool
		errorCode     apperrors.ErrorCode
	}{
		{
			name:   "задача успешно получена",
			taskID: uuid.New(),
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
					}, nil)
			},
			expectedError: false,
		},
		{
			name:   "ошибка получения задачи из БД",
			taskID: uuid.New(),
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, errors.New("не удалось получить запись из базы данных"))
			},
			expectedError: true,
			errorCode:     apperrors.CodeDatabaseError,
		},
		{
			name:   "задача не найдена",
			taskID: uuid.New(),
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(nil, errors.New("задача не найдена"))
			},
			expectedError: true,
			errorCode:     apperrors.CodeDatabaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockRepository)
			mockLogger := new(mocks.Logger)
			tt.setupMocks(mockRepo, mockLogger)

			uc := NewTaskLLMUC(mockRepo, nil, nil, "tasks_llm", mockLogger)

			task, err := uc.GetTaskByID(context.Background(), tt.taskID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, task)
				if tt.errorCode != "" {
					if appErr, ok := apperrors.IsAppError(err); ok {
						assert.Equal(t, tt.errorCode, appErr.Code)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTaskStatusUCUpdateTaskStatus(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*mocks.MockRepository, *mocks.Logger)
		executeMethod func(*TaskLLMUC, context.Context, domain.TaskLLMStatusEvent) error
		event         domain.TaskLLMStatusEvent
		expectedError bool
		errorCode     apperrors.ErrorCode
	}{
		{
			name: "задача успешно обновлена",
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
						Status: domain.TaskStatusPending,
					}, nil)
				repo.On("UpdateWithStatus", mock.Anything, mock.Anything, mock.Anything, domain.TaskStatusProcessing).
					Return(nil)
			},
			executeMethod: func(uc *TaskLLMUC, ctx context.Context, evt domain.TaskLLMStatusEvent) error {
				return uc.UpdateTaskStatus(ctx, evt)
			},
			event: domain.TaskLLMStatusEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status: domain.TaskStatusProcessing.ToKafkaCode(),
				},
			},
			expectedError: false,
		},
		{
			name: "добавлен результат",
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
						Status: domain.TaskStatusPending,
					}, nil)
				repo.On("UpdateWithResult", mock.Anything, mock.Anything, mock.Anything, domain.TaskStatusCompleted, mock.Anything).
					Return(nil)
			},
			executeMethod: func(uc *TaskLLMUC, ctx context.Context, evt domain.TaskLLMStatusEvent) error {
				return uc.UpdateTaskStatus(ctx, evt)
			},
			event: domain.TaskLLMStatusEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status: domain.TaskStatusCompleted.ToKafkaCode(),
				},
				Value: domain.TaskLLMStatusEventPayload{
					Result: json.RawMessage(`{"result": "test"}`),
				},
			},
			expectedError: false,
		},
		{
			name: "добавлена ошибка",
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
						Status: domain.TaskStatusPending,
					}, nil)
				repo.On("UpdateWithError", mock.Anything, mock.Anything, domain.TaskStatusFailed, mock.Anything).
					Return(nil)
			},
			executeMethod: func(uc *TaskLLMUC, ctx context.Context, evt domain.TaskLLMStatusEvent) error {
				return uc.UpdateTaskStatus(ctx, evt)
			},
			event: domain.TaskLLMStatusEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status: domain.TaskStatusFailed.ToKafkaCode(),
				},
				Value: domain.TaskLLMStatusEventPayload{
					ErrorMessage: "test error",
				},
			},
			expectedError: false,
		},
		{
			name: "задача не найдена",
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
						Status: domain.TaskStatusPending,
					}, nil)
				repo.On("UpdateWithStatus", mock.Anything, mock.Anything, mock.Anything, domain.TaskStatusProcessing).
					Return(fmt.Errorf("задача не найдена"))
			},
			executeMethod: func(uc *TaskLLMUC, ctx context.Context, evt domain.TaskLLMStatusEvent) error {
				return uc.UpdateTaskStatus(ctx, evt)
			},
			event: domain.TaskLLMStatusEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status: domain.TaskStatusProcessing.ToKafkaCode(),
				},
				Value: domain.TaskLLMStatusEventPayload{
					Result: json.RawMessage(`{"result": "test"}`),
				},
			},
			expectedError: true,
			errorCode:     apperrors.CodeDatabaseError,
		},
		{
			name: "задача не найдена при добавлении результата",
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
						Status: domain.TaskStatusPending,
					}, nil)
				repo.On("UpdateWithResult", mock.Anything, mock.Anything, mock.Anything, domain.TaskStatusCompleted, mock.Anything).
					Return(fmt.Errorf("задача не найдена"))
			},
			executeMethod: func(uc *TaskLLMUC, ctx context.Context, evt domain.TaskLLMStatusEvent) error {
				return uc.UpdateTaskStatus(ctx, evt)
			},
			event: domain.TaskLLMStatusEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status: domain.TaskStatusCompleted.ToKafkaCode(),
				},
				Value: domain.TaskLLMStatusEventPayload{
					Result: json.RawMessage(`{"result": "test"}`),
				},
			},
			expectedError: true,
			errorCode:     apperrors.CodeDatabaseError,
		},
		{
			name: "задача не найдена при добавлении ошибки",
			setupMocks: func(repo *mocks.MockRepository, logger *mocks.Logger) {
				repo.On("GetByID", mock.Anything, mock.Anything).
					Return(&domain.Task{
						TaskID: uuid.New(),
						Status: domain.TaskStatusPending,
					}, nil)
				repo.On("UpdateWithError", mock.Anything, mock.Anything, domain.TaskStatusFailed, mock.Anything).
					Return(fmt.Errorf("задача не найдена"))
			},
			executeMethod: func(uc *TaskLLMUC, ctx context.Context, evt domain.TaskLLMStatusEvent) error {
				return uc.UpdateTaskStatus(ctx, evt)
			},
			event: domain.TaskLLMStatusEvent{
				Key: domain.TaskContractKey{
					TaskID: uuid.New(),
				},
				Headers: domain.TaskContractHeaders{
					Status: domain.TaskStatusFailed.ToKafkaCode(),
				},
				Value: domain.TaskLLMStatusEventPayload{
					ErrorMessage: "test error",
				},
			},
			expectedError: true,
			errorCode:     apperrors.CodeDatabaseError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mocks.MockRepository)
			logger := new(mocks.Logger)
			tt.setupMocks(repo, logger)
			uc := NewTaskLLMUC(repo, nil, nil, "tasks_llm", logger)

			err := tt.executeMethod(uc, context.Background(), tt.event)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorCode != apperrors.ErrorCode("") {
					if appErr, ok := apperrors.IsAppError(err); ok {
						assert.Equal(t, tt.errorCode, appErr.Code)
					}
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
