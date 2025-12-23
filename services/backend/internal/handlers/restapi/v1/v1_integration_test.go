package v1

import (
	"app/config"
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/handlers/restapi/middleware"
	"app/internal/usecase"
	"app/pkg/logger"
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRoutes_CreateTask_HTTP(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		headers    map[string]string
		body       io.Reader
		wantStatus int
	}{
		{
			name:       "POST /api/v1/tasks/test.pdf - success",
			method:     http.MethodPost,
			path:       "/api/v1/tasks/test.pdf",
			headers:    map[string]string{"Content-Length": "17"},
			body:       bytes.NewBufferString("fake-file-content"),
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "POST /api/v1/tasks/ - 404 (no filename)",
			method:     http.MethodPost,
			path:       "/api/v1/tasks/", // Gin не найдет роут
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/v1/tasks/uuid - success",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/" + uuid.New().String(),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем новый mock для каждого теста
			taskMock := &TaskLLMUCMock{}

			// Настраиваем mock для успешных тестов
			if tt.wantStatus == http.StatusAccepted {
				taskMock.On("CreateTask",
					mock.Anything,                 // context.Context
					mock.Anything,                 // io.Reader
					mock.AnythingOfType("string"), // filename
					mock.AnythingOfType("int64"),  // size
					mock.AnythingOfType("string"), // requestID
				).Return(uuid.New(), nil)
			} else if tt.wantStatus == http.StatusOK {
				// Для GET запросов - парсим UUID из пути
				pathSuffix := tt.path[len("/api/v1/tasks/"):]
				if taskID, err := uuid.Parse(pathSuffix); err == nil {
					taskMock.On("GetTaskByID",
						mock.Anything, // context.Context
						taskID,
					).Return(&domain.Task{TaskID: taskID, Status: domain.TaskStatusPending}, nil)
				}
			}

			// Собираем роутер с настроенным mock'ом
			router := setupTestRouterWithMock(taskMock)

			req := httptest.NewRequest(tt.method, tt.path, tt.body)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req) // ★ Gin обрабатывает ВСЕ сам!

			require.Equal(t, tt.wantStatus, w.Code)

			// Проверяем ожидания mock'а только для успешных тестов
			if tt.wantStatus == http.StatusAccepted || tt.wantStatus == http.StatusOK {
				taskMock.AssertExpectations(t)
			}
		})
	}
}

func TestRoutes_GetTaskByID_HTTP(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "GET /api/v1/tasks/uuid - success",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/" + uuid.New().String(),
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET /api/v1/tasks/invalid-uuid - 400",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/invalid-uuid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "GET /api/v1/tasks/uuid-not-found - 404",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/" + uuid.New().String(),
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET /api/v1/tasks/ - 404",
			method:     http.MethodGet,
			path:       "/api/v1/tasks/",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем новый mock для каждого теста
			taskMock := &TaskLLMUCMock{}

			// Настраиваем mock в зависимости от ожидаемого статуса
			pathSuffix := tt.path[len("/api/v1/tasks/"):]
			if taskID, err := uuid.Parse(pathSuffix); err == nil {
				// Валидный UUID - настраиваем mock
				if tt.wantStatus == http.StatusOK {
					// Успешный GET запрос
					taskMock.On("GetTaskByID",
						mock.Anything, // context.Context
						taskID,
					).Return(&domain.Task{TaskID: taskID, Status: domain.TaskStatusPending}, nil)
				} else if tt.wantStatus == http.StatusNotFound {
					// Task not found - возвращаем ошибку
					taskMock.On("GetTaskByID",
						mock.Anything, // context.Context
						taskID,
					).Return(nil, apperrors.Wrap(apperrors.CodeTaskNotFound, "task not found", nil).WithStatus(http.StatusNotFound))
				}
			}
			// Для невалидного UUID и пустого пути mock не нужен - будет ошибка валидации

			router := setupTestRouterWithMock(taskMock)
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tt.wantStatus, w.Code)

			// Проверяем ожидания mock'а только для тестов, где был вызов usecase
			if tt.wantStatus == http.StatusOK || (tt.wantStatus == http.StatusNotFound && tt.path != "/api/v1/tasks/") {
				taskMock.AssertExpectations(t)
			}
		})
	}
}

func setupTestRouterWithMock(taskMock *TaskLLMUCMock) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Добавляем middleware для request_id
	r.Use(middleware.RequestID())

	cfg := &config.Config{Swagger: config.SwaggerConfig{Enabled: false}}
	uc := usecase.UseCases{TaskLLMUC: taskMock}
	l := logger.NewMockLogger()

	// Используем NewV1Routes напрямую, чтобы избежать циклической зависимости
	apiV1Group := r.Group("/api/v1")
	NewV1Routes(apiV1Group, uc, cfg, l)
	return r
}
