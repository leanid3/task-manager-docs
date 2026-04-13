package usecase

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/entity/repository"
	"app/internal/service"
	pkgminio "app/pkg/minio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// TaskLLMUCInterface интерфейс для TaskLLMUC
type TaskLLMUCInterface interface {
	CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error)
	UpdateTaskStatus(ctx context.Context, evt domain.BrokerCommand[domain.TaskLLMStatusEventPayload]) error
}

type TaskLLMUC struct {
	taskRepo    repository.Task
	producer    service.Broker
	storageRepo service.Storage
	topic       string
}

func NewTaskLLMUC(taskRepo repository.Task, producer service.Broker, storageRepo service.Storage, topic string) *TaskLLMUC {
	return &TaskLLMUC{
		taskRepo:    taskRepo,
		producer:    producer,
		storageRepo: storageRepo,
		topic:       topic,
	}
}

// CreateTask - create new task
func (uc *TaskLLMUC) CreateTask(
	ctx context.Context,
	reader io.Reader,
	filename string,
	filesize int64,
	requestID string,
) (uuid.UUID, error) {
	if reader == nil {
		return uuid.Nil, apperrors.New(
			apperrors.CodeValidationFailed,
			"reader cannot be nil",
		).WithStatus(http.StatusBadRequest)
	}
	if filename == "" {
		return uuid.Nil, apperrors.New(
			apperrors.CodeValidationFailed,
			"filename cannot be empty",
		).WithStatus(http.StatusBadRequest)
	}
	if filesize <= 0 {
		return uuid.Nil, apperrors.New(
			apperrors.CodeValidationFailed,
			"filesize must be greater than zero",
		).WithStatus(http.StatusBadRequest)
	}

	traceID := uuid.New()
	taskID := uuid.New()
	storagePath := uc.storageRepo.GenerateStoragePath(taskID, filename)

	// Преобразуем сохранение metadataдля в Task.Metadata
	llmMetadata := domain.LLMMetadata{
		Filename:    filename,
		ContentType: domain.ContentTypeProtocol,
		Filesize:    filesize,
		StoragePath: storagePath,
	}
	metadataJSON, err := json.Marshal(llmMetadata)
	if err != nil {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeValidationFailed,
			"failed to marshal metadata",
			err,
		).WithStatus(http.StatusInternalServerError)
	}
	var metadataMap map[string]interface{}
	if err := json.Unmarshal(metadataJSON, &metadataMap); err != nil {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeValidationFailed,
			"failed to unmarshal metadata",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	taskLLM := &domain.TaskLLM{
		BaseTask: domain.BaseTask{
			TaskID:    taskID,
			Status:    domain.TaskStatusPending,
			CreatedAt: time.Now(),
			TraceID:   &traceID,
			RequestID: requestID,
			Metadata:  metadataMap,
		},
		Metadata:    llmMetadata,
		StoragePath: storagePath,
		StorageSize: filesize,
	}

	opts := pkgminio.UploadOptions{
		ContentType: domain.ContentTypeProtocol,
	}

	// Загружаем файл в хранилище
	_, err = uc.storageRepo.UploadStream(ctx, storagePath, reader, filesize, opts)
	if err != nil {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeStorageError,
			"failed to upload file to storage",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	// Сохраняем задачу в базе данных
	if err := uc.taskRepo.Create(ctx, taskLLM); err != nil {
		// Если не удалось сохранить задачу в БД, удаляем загруженный файл
		uc.storageRepo.DeleteObject(ctx, storagePath) // игнорируем ошибку удаления
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeDatabaseError,
			"failed to create task in database",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	// Отправляем задачу в очередь
	taskData, err := json.Marshal(taskLLM)
	if err != nil {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeInternalError,
			"failed to serialize task",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	// Создаём Kafka-команду по контракту
	cmd := domain.NewTaskLLMCommand(taskLLM)
	taskData, err = json.Marshal(cmd.Value)
	if err != nil {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeInternalError,
			"failed to serialize task command for kafka",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	if err := uc.producer.Send(ctx, uc.topic, cmd.Key.TaskID.String(), cmd.Headers.ToMap(), taskData); err != nil {
		// Если не удалось отправить задачу в очередь, обновляем статус задачи на FAILED
		if updateErr := uc.taskRepo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, err.Error()); updateErr != nil {
			// Логируем ошибку обновления статуса, но не возвращаем её как основную
		}

		// Также удаляем файл из хранилища, так как задача не будет обработана
		if deleteErr := uc.storageRepo.DeleteObject(ctx, storagePath); deleteErr != nil {
			// Логируем ошибку удаления, но не возвращаем её как основную
		}

		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeKafkaError,
			"failed to send task to broker",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	return taskID, nil
}

// GetTaskByID - get task by ID
func (uc *TaskLLMUC) GetTaskByID(ctx context.Context, taskID uuid.UUID) (domain.Task, error) {
	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, apperrors.Wrap(
			apperrors.CodeDatabaseError,
			"failed to get task from database",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	return task, nil
}

func (uc *TaskLLMUC) UpdateTaskStatus(ctx context.Context, evt domain.BrokerCommand[domain.TaskLLMStatusEventPayload]) error {
	//TODO также как и в handle - сделать Middelware для преобразования кодов или перейти на коды статусов
	taskStatus, ok := domain.TaskStatus("").FromKafkaCode(evt.Headers.Status)
	if !ok {
		return apperrors.New(apperrors.CodeInvalidMessageFormat, "unknown task status")
	}

	var result error
	switch taskStatus {
	case domain.TaskStatusProcessing:
		//TODO заменить на кеш задач вместо хождения в базу
		task, err := uc.taskRepo.GetByID(ctx, evt.Key.TaskID)
		if err != nil {
			result = apperrors.Wrap(apperrors.CodeDatabaseError, "failed to get task", err)
		} else if task.GetStatus() == domain.TaskStatusProcessing {
			result = apperrors.New(apperrors.CodeTaskAlreadyProcessing, "task already processing")
		} else {
			result = uc.taskRepo.UpdateWithStatus(ctx, evt.Key.TaskID, evt.Headers.WorkerID, domain.TaskStatusProcessing)
		}
	case domain.TaskStatusCompleted:
		task, err := uc.taskRepo.GetByID(ctx, evt.Key.TaskID)
		if err != nil {
			result = apperrors.Wrap(apperrors.CodeDatabaseError, "failed to get task", err)
		} else if task.GetStatus() == domain.TaskStatusCompleted {
			result = apperrors.New(apperrors.CodeTaskAlreadyCompleted, "task already completed")
		} else {
			result = uc.taskRepo.UpdateWithResult(ctx, evt.Key.TaskID, evt.Headers.WorkerID, domain.TaskStatusCompleted, evt.Value.Result)
		}
	case domain.TaskStatusFailed:
		task, err := uc.taskRepo.GetByID(ctx, evt.Key.TaskID)
		if err != nil {
			result = apperrors.Wrap(apperrors.CodeDatabaseError, "failed to get task", err)
		} else if task.GetStatus() == domain.TaskStatusFailed {
			result = apperrors.New(apperrors.CodeTaskAlreadyFailed, "task already failed")
		} else {
			result = uc.taskRepo.UpdateWithError(ctx, evt.Key.TaskID, domain.TaskStatusFailed, evt.Value.ErrorMessage)
		}
	case domain.TaskStatusPending:
		result = nil
	case domain.TaskStatusCancelled:
		result = nil
	default:
		result = apperrors.New(apperrors.CodeInvalidMessageFormat, "unknown task status")
	}

	return result
}
