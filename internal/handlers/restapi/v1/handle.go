package v1

import (
	"app/internal/entity/domain"
	"app/pkg/response"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	apperrors "app/internal/entity/errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @name CreateTaskResponse
type CreateTaskResponse struct {
	TaskID uuid.UUID         `json:"task_id"`
	Status domain.TaskStatus `json:"status"`
}

// @name GetTaskResponse
type GetTaskResponse struct {
	TaskID       uuid.UUID         `json:"task_id"`
	Status       domain.TaskStatus `json:"status"`
	TaskResults  json.RawMessage   `json:"task_results,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
}

type TaskLLMUCInterface interface {
	CreateTask(ctx context.Context, reader io.Reader, filename string, filesize int64, requestID string) (uuid.UUID, error)
	GetTaskByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
}

// createTask создает задачу обработки файла с потоковой загрузкой
// @Summary      Создать задачу обработки файла (streaming upload)
// @Description  Загружает файл в бинарном формате и создает задачу на обработку.
// @Tags         tasks
// @Accept       application/octet-stream
// @Produce      json
// @Param        filename       path      string  true   "Имя файла с расширением. Пример: document.pdf"
// @Param        Content-Length header    int     true  "Размер файла в байтах. Устанавливается автоматически при использовании curl --data-binary"
// @Param        file           body      string  true   "Бинарное содержимое файла"
// @Success      202  {object}  SwaggerCreateTaskSuccess      "Задача успешно создана"
// @Failure      400  {object}  response.ErrorResponse  "Неверный формат запроса"
// @Failure      500  {object}  response.ErrorResponse  "Ошибка при создании задачи"
// @Router       /api/v1/tasks/{filename} [post]
func (r *V1) createTask(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), r.cfg.Server.Timeout)
	defer cancel()

	//TODO Рассмотреть вынести валидацию в domain или middleware
	//валидацию загаловков и параметров запроса
	filename := c.Param("filename")
	if filename == "" {
		r.handleError(c, apperrors.Wrap(
			apperrors.CodeInvalidRequest,
			"неверный формат запроса",
			nil,
		).WithStatus(http.StatusBadRequest), "invalid_filename")
		return
	}

	contentLength, err := strconv.ParseInt(c.GetHeader("Content-Length"), 10, 64)
	if err != nil || contentLength <= 0 {
		r.handleError(c, apperrors.Wrap(
			apperrors.CodeInvalidRequest,
			"неверный формат запроса",
			nil,
		).WithStatus(http.StatusBadRequest), "invalid_content_length")
		return
	}
	requestID := c.GetString("request_id")

	r.l.Info("начало создания задачи",
		"filename", filename,
		"size", contentLength,
		"request_id", requestID,
	)

	//создание задачи
	taskID, err := r.uc.TaskLLMUC.CreateTask(ctx, c.Request.Body, filename, contentLength, requestID)
	if err != nil {
		r.handleError(c, err, "create_task")
		return
	}

	// Success лог на границе
	r.l.Info("задача создана",
		"task_id", taskID,
		"status", domain.TaskStatusPending,
		"request_id", requestID,
	)

	c.JSON(http.StatusAccepted, response.Success(CreateTaskResponse{
		TaskID: taskID,
		Status: domain.TaskStatusPending,
	}, requestID))
}

// @Summary Get task by ID
// @Description Get task by ID

// @Tags tasks
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 200 {object} SwaggerGetTaskSuccess
// @Failure 400 {object} response.ErrorResponse "Bad Request"
// @Failure 404 {object} response.ErrorResponse "Not Found"
// @Failure 500 {object} response.ErrorResponse "Internal Server Error"
// @Router /api/v1/tasks/{task_id} [get]
func (r *V1) getTaskByID(c *gin.Context) {

	//TODO Рассмотреть вынести валидацию в domain или middleware
	//валидацию загаловков и параметров запроса
	taskIDStr := c.Param("task_id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		r.handleError(c, apperrors.Wrap(
			apperrors.CodeInvalidTaskID,
			"неверный формат ID задачи",
			err,
		).WithStatus(http.StatusBadRequest), "parse_task_id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), r.cfg.Server.Timeout)
	defer cancel()

	task, err := r.uc.TaskLLMUC.GetTaskByID(ctx, taskID)
	if err != nil {
		r.handleError(c, err, "get_task_by_id")
		return
	}

	requestID := c.GetString("request_id")
	c.JSON(http.StatusOK, response.Success(GetTaskResponse{
		TaskID:       task.TaskID,
		Status:       task.Status,
		TaskResults:  task.Result,
		ErrorMessage: task.ErrorMessage,
	}, requestID))
}
