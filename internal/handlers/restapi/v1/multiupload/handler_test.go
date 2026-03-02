package multiupload

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/internal/entity/domain"
	"app/pkg/logger"
	"app/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMultiUploadUCInterface - mock для интерфейса юзкейса
type MockMultiUploadUCInterface struct {
	mock.Mock
}

func (m *MockMultiUploadUCInterface) CreateMultiUploadTask(ctx context.Context, files []*multipart.FileHeader, requestID string) (uuid.UUID, error) {
	args := m.Called(ctx, files, requestID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *MockMultiUploadUCInterface) GetTaskByID(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(domain.Task), args.Error(1)
}

func TestCreateMultiUploadTaskHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Подготовка мока юзкейса
	mockUC := new(MockMultiUploadUCInterface)

	// Подготовка конфигурации
	config := &Config{
		Server: struct {
			Timeout int `mapstructure:"timeout"`
		}{
			Timeout: 30,
		},
		MaxFileSize:  10485760, // 10MB
		MaxFileCount: 10,
	}

	// Подготовка логгера
	log := logger.New(io.Discard, "debug", "text")

	handler := New(mockUC, config, log)

	t.Run("successful multi-upload task creation", func(t *testing.T) {
		// Создаем multipart/form-data запрос
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Добавляем несколько файлов
		for i := 0; i < 2; i++ {
			part, err := writer.CreateFormFile("files", fmt.Sprintf("test%d.txt", i))
			assert.NoError(t, err)

			_, err = part.Write([]byte(fmt.Sprintf("content of file %d", i)))
			assert.NoError(t, err)
		}

		writer.Close()

		req, _ := http.NewRequest("POST", "/multiupload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req

		// Устанавливаем request_id в контексте
		c.Set("request_id", "test_request_id")

		// Ожидаем, что юзкейс вернет UUID и nil ошибку
		expectedTaskID := uuid.New()
		mockUC.On("CreateMultiUploadTask", mock.Anything, mock.Anything, "test_request_id").Return(expectedTaskID, nil).Once()

		handler.CreateMultiUploadTask(c)

		assert.Equal(t, http.StatusAccepted, w.Code)

		var response response.SuccessData[CreateMultiUploadTaskResponse]
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Проверяем, что в ответе содержится ожидаемый task_id
		assert.Equal(t, expectedTaskID, response.Data.TaskID)
		assert.Equal(t, domain.TaskStatusPending, response.Data.Status)

		mockUC.AssertExpectations(t)
	})

	t.Run("no files provided", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		writer.Close()

		req, _ := http.NewRequest("POST", "/multiupload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("request_id", "test_request_id")

		handler.CreateMultiUploadTask(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response response.ErrorResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "no files provided", response.Error)
	})

	t.Run("file too large", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)

		// Добавляем файл, который превышает лимит
		part, err := writer.CreateFormFile("files", "large_file.txt")
		assert.NoError(t, err)

		// Записываем больше данных, чем разрешено
		largeData := make([]byte, config.MaxFileSize+1)
		_, err = part.Write(largeData)
		assert.NoError(t, err)

		writer.Close()

		req, _ := http.NewRequest("POST", "/multiupload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = req
		c.Set("request_id", "test_request_id")

		handler.CreateMultiUploadTask(c)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response response.ErrorResponse
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response.Error, "file too large")
	})
}
