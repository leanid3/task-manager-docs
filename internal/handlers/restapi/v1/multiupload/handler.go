package multiupload

import (
	"app/internal/entity/domain"
	"app/pkg/response"
	"context"
	"errors"
	"net/http"
	"time"

	apperrors "app/internal/entity/errors"
	"mime/multipart"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @name CreateMultiUploadTaskResponse
type CreateMultiUploadTaskResponse struct {
	TaskID uuid.UUID         `json:"task_id"`
	Status domain.TaskStatus `json:"status"`
}

type MultiUploadUCInterface interface {
	CreateMultiUploadTask(ctx context.Context, files []*multipart.FileHeader, requestID string) (uuid.UUID, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
}

type Handler struct {
	uc  MultiUploadUCInterface
	cfg *Config
	l   Logger
}

type Config struct {
	Server struct {
		Timeout int `mapstructure:"timeout"`
	} `mapstructure:"server"`
	MaxFileSize int64 `mapstructure:"max_file_size"` // максимальный размер файла в байтах
	MaxFileCount int  `mapstructure:"max_file_count"` // максимальное количество файлов
}

type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

func New(uc MultiUploadUCInterface, cfg *Config, l Logger) *Handler {
	return &Handler{
		uc:  uc,
		cfg: cfg,
		l:   l,
	}
}

// createMultiUploadTask создает задачу для загрузки нескольких файлов
// @Summary      Создать задачу для загрузки нескольких файлов
// @Description  Загружает несколько файлов и создает задачу на их обработку.
// @Tags         multiupload
// @Accept       multipart/form-data
// @Produce      json
// @Param        files        formData  file  true  "Файлы для загрузки (множественная загрузка)"
// @Success      202  {object}  CreateMultiUploadTaskResponse      "Задача успешно создана"
// @Failure      400  {object}  response.ErrorResponse  "Неверный формат запроса"
// @Failure      500  {object}  response.ErrorResponse  "Ошибка при создании задачи"
// @Router       /api/v1/multiupload [post]
func (h *Handler) CreateMultiUploadTask(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.cfg.Server.Timeout)*time.Second)
	defer cancel()

	// Ограничиваем размер тела запроса
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.cfg.MaxFileSize*int64(h.cfg.MaxFileCount))

	// Парсим multipart форму
	err := c.Request.ParseMultipartForm(int64(h.cfg.MaxFileSize) * int64(h.cfg.MaxFileCount))
	if err != nil && !errors.Is(err, http.ErrNotMultipart) {
		h.handleError(c, apperrors.Wrap(
			apperrors.CodeInvalidRequest,
			"unable to parse form",
			err,
		).WithStatus(http.StatusBadRequest), "parse_multipart_form")
		return
	}

	form := c.Request.MultipartForm
	files := form.File["files"] // "files[]" — массив файлов в форме

	if len(files) == 0 {
		h.handleError(c, apperrors.New(
			apperrors.CodeInvalidRequest,
			"no files provided",
		).WithStatus(http.StatusBadRequest), "no_files_provided")
		return
	}

	if len(files) > h.cfg.MaxFileCount {
		h.handleError(c, apperrors.New(
			apperrors.CodeInvalidRequest,
			"too many files provided",
		).WithStatus(http.StatusBadRequest), "too_many_files")
		return
	}

	// Проверяем размер каждого файла
	for _, fileHeader := range files {
		if fileHeader.Size > h.cfg.MaxFileSize {
			h.handleError(c, apperrors.New(
				apperrors.CodeInvalidRequest,
				"file too large",
			).WithStatus(http.StatusBadRequest), "file_too_large")
			return
		}
	}

	requestID := c.GetString("request_id")

	h.l.Info("creating multi-upload task",
		"file_count", len(files),
		"request_id", requestID,
	)

	// создание задачи
	taskID, err := h.uc.CreateMultiUploadTask(ctx, files, requestID)
	if err != nil {
		h.handleError(c, err, "create_multi_upload_task")
		return
	}

	h.l.Info("multi-upload task created successfully",
		"task_id", taskID,
		"status", domain.TaskStatusPending,
		"request_id", requestID,
	)

	c.JSON(http.StatusAccepted, response.Success(CreateMultiUploadTaskResponse{
		TaskID: taskID,
		Status: domain.TaskStatusPending,
	}, requestID))
}

func (h *Handler) handleError(c *gin.Context, err error, op string) {
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		appErr = apperrors.Wrap(
			apperrors.CodeInternalError,
			"internal error",
			err,
		).WithStatus(http.StatusInternalServerError)
	}

	h.l.Error("handler error",
		"operation", op,
		"error", appErr.Error(),
		"status", appErr.HTTPStatus,
	)

	c.JSON(appErr.HTTPStatus, response.Error(appErr, c.GetString("request_id")))
}