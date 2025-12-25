package v1

import (
	"app/config"
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/usecase"
	"app/test/mocks"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper для создания тестового контекста
func newTestContext(method, path string, body io.Reader, headers map[string]string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	c.Request = req
	return c, w
}

// Helper для создания V1 с моками
func newTestV1() *V1 {
	taskMock := &mocks.TaskLLMUC{}
	uc := usecase.UseCases{TaskLLMUC: taskMock}
	cfg := config.Config{
		Server: config.ServerConfig{Timeout: time.Second * 5},
	}
	l := mocks.NewMockLogger()

	return &V1{
		uc:  uc,
		l:   l,
		cfg: cfg,
		v:   validator.New(validator.WithRequiredStructEnabled()),
	}
}

func TestCreateTask(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		filename      string
		body          io.Reader
		contentLength string
		requestID     string
		mockError     error
		mockTaskID    uuid.UUID
		wantStatus    int
		wantContains  string // для проверки тела ответа
	}{
		{
			name:          "success case",
			filename:      "test.pdf",
			body:          bytes.NewBufferString("fake-file-content"),
			contentLength: "17", // точная длина body
			requestID:     "req-123",
			mockTaskID:    uuid.New(),
			wantStatus:    http.StatusAccepted,
			wantContains:  `"task_id"`,
		},
		{
			name:          "empty filename",
			filename:      "",
			body:          bytes.NewBufferString("fake-file-content"),
			contentLength: "17",
			requestID:     "req-123",
			wantStatus:    http.StatusBadRequest,
			wantContains:  `"неверный формат запроса"`,
		},
		{
			name:          "invalid content-length",
			filename:      "test.pdf",
			body:          bytes.NewBufferString("fake-file-content"),
			contentLength: "invalid", // не число
			requestID:     "req-123",
			wantStatus:    http.StatusBadRequest,
			wantContains:  `"неверный формат запроса"`,
		},
		{
			name:          "zero content-length",
			filename:      "test.pdf",
			body:          bytes.NewBufferString(""),
			contentLength: "0",
			requestID:     "req-123",
			wantStatus:    http.StatusBadRequest,
			wantContains:  `"неверный формат запроса"`,
		},
		{
			name:          "usecase returns 404 error",
			filename:      "test.pdf",
			body:          bytes.NewBufferString("fake-file-content"),
			contentLength: "17",
			requestID:     "req-123",
			mockError:     apperrors.Wrap(apperrors.CodeTaskNotFound, "not found", nil).WithStatus(http.StatusNotFound),
			mockTaskID:    uuid.Nil,
			wantStatus:    http.StatusNotFound,
			wantContains:  `"not found"`,
		},
		{
			name:          "usecase returns 500 error",
			filename:      "test.pdf",
			body:          bytes.NewBufferString("fake-file-content"),
			contentLength: "17",
			requestID:     "req-123",
			mockError:     apperrors.Wrap(apperrors.CodeInternalError, "internal error", nil).WithStatus(http.StatusInternalServerError),
			mockTaskID:    uuid.Nil,
			wantStatus:    http.StatusInternalServerError,
			wantContains:  `"internal error"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем свежие моки для каждого теста
			v1Handler := newTestV1()
			taskMock := v1Handler.uc.TaskLLMUC.(*mocks.TaskLLMUC)

			body := tt.body
			if body == nil {
				body = bytes.NewBufferString("")
			}
			contentLengthInt := int64(0)
			if tt.contentLength != "" {
				contentLengthInt, _ = strconv.ParseInt(tt.contentLength, 10, 64)
			}

			// Настраиваем мок только если ожидается вызов usecase
			if tt.mockTaskID != uuid.Nil || tt.mockError != nil {
				taskMock.On("CreateTask",
					mock.Anything, // context.Context
					mock.Anything, // io.Reader (может быть io.nopCloserWriterTo или другой тип)
					tt.filename,
					contentLengthInt,
					tt.requestID,
				).Return(tt.mockTaskID, tt.mockError)
			}

			// Создаем контекст
			headers := map[string]string{}
			if tt.contentLength != "" {
				headers["Content-Length"] = tt.contentLength
			}

			c, w := newTestContext(http.MethodPost, "/api/v1/tasks/"+tt.filename, body, headers)
			c.Params = gin.Params{{Key: "filename", Value: tt.filename}}
			c.Set("request_id", tt.requestID)

			// Вызываем хендлер
			v1Handler.createTask(c)

			// Проверяем статус
			require.Equal(t, tt.wantStatus, w.Code, "unexpected status code")

			// Проверяем тело ответа
			bodyStr := w.Body.String()
			if tt.wantContains != "" {
				require.Contains(t, bodyStr, tt.wantContains, "response body mismatch")
			}

			// Проверяем вызов мока
			if tt.mockTaskID != uuid.Nil || tt.mockError != nil {
				taskMock.AssertExpectations(t)
			} else {
				taskMock.AssertNotCalled(t, "CreateTask")
			}
		})
	}
}

func TestGetTaskByID(t *testing.T) {
	tests := []struct {
		name         string
		taskIDStr    string
		mockTask     *domain.Task
		mockError    error
		wantStatus   int
		wantContains string // ✅ ДОБАВИЛ
	}{
		{
			name:         "success",
			taskIDStr:    uuid.New().String(),
			mockTask:     &domain.Task{TaskID: uuid.New(), Status: domain.TaskStatusPending},
			mockError:    nil,
			wantStatus:   http.StatusOK,
			wantContains: `"status"`, // проверяем JSON
		},
		{
			name:         "invalid uuid",
			taskIDStr:    "invalid-uuid",
			mockTask:     nil,
			mockError:    nil,
			wantStatus:   http.StatusBadRequest,
			wantContains: `"неверный формат ID задачи"`, // из твоего кода
		},
		{
			name:         "task not found",
			taskIDStr:    uuid.New().String(),
			mockTask:     nil,
			mockError:    apperrors.Wrap(apperrors.CodeTaskNotFound, "not found", nil).WithStatus(http.StatusNotFound),
			wantStatus:   http.StatusNotFound,
			wantContains: `"not found"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)

			v1Handler := newTestV1()
			taskMock := v1Handler.uc.TaskLLMUC.(*mocks.TaskLLMUC)

			if tt.mockTask != nil || tt.mockError != nil {
				validUUID := uuid.MustParse(tt.taskIDStr)
				taskMock.On("GetTaskByID", mock.Anything, validUUID).
					Return(tt.mockTask, tt.mockError)
			}

			c, w := newTestContext(http.MethodGet, "/api/v1/tasks/"+tt.taskIDStr, nil, nil)
			c.Params = gin.Params{{Key: "task_id", Value: tt.taskIDStr}}
			c.Set("request_id", "req-123")

			v1Handler.getTaskByID(c) // предполагаем публичный метод

			require.Equal(t, tt.wantStatus, w.Code, "unexpected status code")

			bodyStr := w.Body.String()
			if tt.wantContains != "" {
				require.Contains(t, bodyStr, tt.wantContains)
			}

			if tt.mockTask != nil || tt.mockError != nil {
				taskMock.AssertExpectations(t)
			} else {
				taskMock.AssertNumberOfCalls(t, "GetTaskByID", 0) // для invalid uuid
			}
		})
	}
}
