package usecase

import (
	"app/internal/entity/broker"
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/entity/repository"
	"app/pkg/logger"
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
	GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	UpdateTaskStatus(ctx context.Context, evt domain.TaskLLMStatusEvent) error
}

type TaskLLMUC struct {
	taskRepo    repository.Task
	producer    broker.Producer
	storageRepo repository.Storage
	topic       string
	l           logger.Interface
}

func NewTaskLLMUC(taskRepo repository.Task, producer broker.Producer, storageRepo repository.Storage, topic string, l logger.Interface) *TaskLLMUC {
	return &TaskLLMUC{
		taskRepo:    taskRepo,
		producer:    producer,
		storageRepo: storageRepo,
		topic:       topic,
		l:           l,
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
		Task: domain.Task{
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

	uc.l.Info("uploading file to storage",
		"task_id", taskID,
		"storage_path", storagePath,
		"size", filesize,
		"filename", filename,
		"request_id", requestID,
	)

	// Storage - загрузка файла в хранилище
	if _, err := uc.storageRepo.UploadStream(ctx, storagePath, reader, filesize, opts); err != nil {
		uc.l.Error("failed to upload file to storage",
			"error", err,
			"task_id", taskID,
			"storage_path", storagePath,
			"filename", filename,
			"request_id", requestID,
		)

		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeStorageError,
			"не удалось загрузить файл в хранилище.",
			err,
		).WithDetails(map[string]interface{}{
			"task_id":      taskID.String(),
			"storage_path": storagePath,
		}).WithStatus(http.StatusInternalServerError)
	}
	// Удаление файла из хранилища в случае ошибки загрузки
	needsCleanup := true
	defer func() {
		if needsCleanup {
			if err := uc.storageRepo.DeleteObject(context.Background(), storagePath); err != nil {
				uc.l.Error("failed to cleanup storage after error",
					"error", err,
					"storage_path", storagePath,
					"task_id", taskID,
					"request_id", requestID,
				)
			}
		}
	}()

	// Database - создание задачи в базе данных
	if err := uc.taskRepo.Create(ctx, &taskLLM.Task); err != nil {
		uc.l.Error("failed to create task in database",
			"error", err,
			"task_id", taskID,
			"request_id", requestID,
		)
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeDatabaseError,
			"не удалось создать задачу в базе данных",
			err,
		).WithDetails(map[string]interface{}{
			"task_id": taskID.String(),
		})
	}

	cmd := domain.NewTaskLLMCommand(taskLLM)
	headers := cmd.Headers.ToMap()
	if err := uc.producer.Send(ctx, uc.topic, taskID.String(), headers, cmd.Value); err != nil {
		uc.l.Error("failed to publish task to queue",
			"error", err,
			"task_id", taskID,
			"topic", uc.topic,
			"request_id", requestID,
		)

		if updateErr := uc.taskRepo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, err.Error()); updateErr != nil {
			uc.l.Error("failed to update task with error status",
				"error", updateErr,
				"task_id", taskID,
				"original_error", err.Error(),
			)
		}

		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeKafkaError,
			"failed to publish task to queue",
			err,
		).WithDetails(map[string]interface{}{
			"task_id": taskID.String(),
			"topic":   uc.topic,
		})
	}

	uc.l.Info("task created and published successfully",
		"task_id", taskID,
		"status", domain.TaskStatusPending,
		"topic", uc.topic,
		"request_id", requestID,
	)

	// Удаление файла из хранилища в случае успешного создания задачи не требуется
	needsCleanup = false

	return taskID, nil
}

// GetTaskByID - get task by ID
func (uc *TaskLLMUC) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
	uc.l.Debug("getting task by ID", "task_id", taskID)

	task, err := uc.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		uc.l.Error("failed to get task from database",
			"error", err,
			"task_id", taskID,
		)
		return nil, apperrors.Wrap(
			apperrors.CodeDatabaseError,
			"не удалось получить задачу из базы данных",
			err,
		).WithDetails(map[string]interface{}{
			"task_id": taskID.String(),
		}).WithStatus(http.StatusInternalServerError)
	}

	uc.l.Debug("task retrieved successfully", "task_id", taskID, "status", task.Status)
	return task, nil
}

func (uc *TaskLLMUC) UpdateTaskStatus(ctx context.Context, evt domain.TaskLLMStatusEvent) error {
	uc.l.Debug("updating task status",
		"task_id", evt.Key.TaskID,
		"status_code", evt.Headers.Status,
	)

	//TODO также как и в handle - сделать Middelware для преобразования кодов или перейти на коды статусов
	taskStatus, ok := domain.TaskStatus("").FromKafkaCode(evt.Headers.Status)
	if !ok {
		uc.l.Error("unknown task status code",
			"task_id", evt.Key.TaskID,
			"status_code", evt.Headers.Status,
		)
		return apperrors.New(apperrors.CodeInvalidMessageFormat, "unknown task status")
	}

	switch taskStatus {
	case domain.TaskStatusProcessing:
		//TODO заменить на кеш задач вместо хождения в базу
		task, err := uc.taskRepo.GetByID(ctx, evt.Key.TaskID)
		if err != nil {
			return apperrors.Wrap(apperrors.CodeDatabaseError, "failed to get task", err)
		}
		if task.Status == domain.TaskStatusProcessing {
			return apperrors.New(apperrors.CodeTaskAlreadyProcessing, "task already processing")
		}
		return uc.taskRepo.UpdateWithStatus(ctx, evt.Key.TaskID, evt.Headers.WorkerID, domain.TaskStatusProcessing)
	case domain.TaskStatusCompleted:
		task, err := uc.taskRepo.GetByID(ctx, evt.Key.TaskID)
		if err != nil {
			return apperrors.Wrap(apperrors.CodeDatabaseError, "failed to get task", err)
		}
		if task.Status == domain.TaskStatusCompleted {
			return apperrors.New(apperrors.CodeTaskAlreadyCompleted, "task already completed")
		}
		return uc.taskRepo.UpdateWithResult(ctx, evt.Key.TaskID, evt.Headers.WorkerID, domain.TaskStatusCompleted, evt.Value.Result)
	case domain.TaskStatusFailed:
		task, err := uc.taskRepo.GetByID(ctx, evt.Key.TaskID)
		if err != nil {
			return apperrors.Wrap(apperrors.CodeDatabaseError, "failed to get task", err)
		}
		if task.Status == domain.TaskStatusFailed {
			return apperrors.New(apperrors.CodeTaskAlreadyFailed, "task already failed")
		}
		return uc.taskRepo.UpdateWithError(ctx, evt.Key.TaskID, domain.TaskStatusFailed, evt.Value.ErrorMessage)
	case domain.TaskStatusPending:
		uc.l.Debug("task status is pending, no update needed", "task_id", evt.Key.TaskID)
		return nil
	case domain.TaskStatusCancelled:
		uc.l.Debug("task status is cancelled, no update needed", "task_id", evt.Key.TaskID)
		return nil
	default:
		uc.l.Error("unknown task status",
			"task_id", evt.Key.TaskID,
			"status", taskStatus,
			"status_code", evt.Headers.Status,
		)
		return apperrors.New(apperrors.CodeInvalidMessageFormat, "unknown task status")
	}
}
