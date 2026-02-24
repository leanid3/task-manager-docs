package multiupload

import (
	"app/internal/entity/broker"
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/entity/repository"
	pkgminio "app/pkg/minio"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MultiUploadUCInterface интерфейс для MultiUploadUC
type MultiUploadUCInterface interface {
	CreateMultiUploadTask(ctx context.Context, files []*multipart.FileHeader, requestID string) (uuid.UUID, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
}

type MultiUploadUC struct {
	taskRepo    repository.Task
	producer    broker.Producer
	storageRepo repository.Storage
	topic       string
	semaphore   chan struct{} // семафор для ограничения числа одновременных загрузок
}

func NewMultiUploadUC(taskRepo repository.Task, producer broker.Producer, storageRepo repository.Storage, topic string, maxConcurrentUploads int) *MultiUploadUC {
	return &MultiUploadUC{
		taskRepo:    taskRepo,
		producer:    producer,
		storageRepo: storageRepo,
		topic:       topic,
		semaphore:   make(chan struct{}, maxConcurrentUploads),
	}
}

type UploadResult struct {
	Key string
	Err error
}

// CreateMultiUploadTask создает задачу для загрузки нескольких файлов
func (uc *MultiUploadUC) CreateMultiUploadTask(
	ctx context.Context,
	files []*multipart.FileHeader,
	requestID string,
) (uuid.UUID, error) {
	totalFiles := len(files)
	if totalFiles == 0 {
		return uuid.Nil, apperrors.New(
			apperrors.CodeValidationFailed,
			"no files provided",
		).WithStatus(http.StatusBadRequest)
	}

	// Канал для результатов
	results := make(chan UploadResult, totalFiles)
	var wg sync.WaitGroup

	for _, fileHeader := range files {
		wg.Add(1)

		go func(fh *multipart.FileHeader) {
			defer wg.Done()

			// Ограничение числа одновременных загрузок
			uc.semaphore <- struct{}{}
			defer func() { <-uc.semaphore }()

			// Открываем файл
			file, err := fh.Open()
			if err != nil {
				results <- UploadResult{Err: fmt.Errorf("failed to open file: %w", err)}
				return
			}
			defer file.Close()

			// Генерируем уникальный ключ
			key := generateFileKey(fh.Filename) // используем имя файла в ключе

			// Загружаем в MinIO
			opts := pkgminio.UploadOptions{
				ContentType: fh.Header.Get("Content-Type"),
			}

			_, err = uc.storageRepo.UploadStream(ctx, key, file, fh.Size, opts)
			if err != nil {
				results <- UploadResult{Err: fmt.Errorf("failed to upload to storage: %w", err)}
				return
			}

			results <- UploadResult{Key: key}
		}(fileHeader)
	}

	// Закрываем канал результатов после завершения всех горутин
	go func() {
		wg.Wait()
		close(results)
	}()

	// Собираем результаты
	var uploadedKeys []string
	var errorsList []error

	for res := range results {
		if res.Err != nil {
			errorsList = append(errorsList, res.Err)
		} else {
			uploadedKeys = append(uploadedKeys, res.Key)
		}
	}

	if len(errorsList) > 0 {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeStorageError,
			fmt.Sprintf("some uploads failed: %v", errorsList),
			nil,
		).WithStatus(http.StatusInternalServerError)
	}

	// Создаём задачу с ключами файлов
	traceID := uuid.New()
	taskID := uuid.New()

	// Подготовим метаданные для задачи
	multiUploadMetadata := domain.MultiUploadMetadata{
		FileCount: len(uploadedKeys),
		FileKeys:  uploadedKeys,
	}

	metadataJSON, err := json.Marshal(multiUploadMetadata)
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

	task := &domain.Task{
		TaskID:    taskID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now(),
		TraceID:   &traceID,
		RequestID: requestID,
		Metadata:  metadataMap,
	}

	// Сохраняем задачу в базе данных
	if err := uc.taskRepo.Create(ctx, task); err != nil {
		// Если не удалось сохранить задачу в БД, удаляем загруженные файлы
		for _, key := range uploadedKeys {
			uc.storageRepo.DeleteObject(ctx, key) // игнорируем ошибку удаления
		}
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeDatabaseError,
			"failed to create task in database",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	// Отправляем задачу в очередь
	taskData, err := json.Marshal(task)
	if err != nil {
		return uuid.Nil, apperrors.Wrap(
			apperrors.CodeInternalError,
			"failed to serialize task",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	// Подготовим заголовки для отправки
	headers := map[string]string{
		"trace_id":   traceID.String(),
		"request_id": requestID,
	}

	if err := uc.producer.Send(ctx, uc.topic, taskID.String(), headers, taskData); err != nil {
		// Если не удалось отправить задачу в очередь, обновляем статус задачи на FAILED
		if updateErr := uc.taskRepo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, err.Error()); updateErr != nil {
			// Логируем ошибку обновления статуса, но не возвращаем её как основную
		}

		// Также удаляем файлы из хранилища, так как задача не будет обработана
		for _, key := range uploadedKeys {
			if deleteErr := uc.storageRepo.DeleteObject(ctx, key); deleteErr != nil {
				// Логируем ошибку удаления, но не возвращаем её как основную
			}
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
func (uc *MultiUploadUC) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*domain.Task, error) {
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

func generateFileKey(filename string) string {
	return fmt.Sprintf("%s/%s", uuid.NewString(), filename)
}