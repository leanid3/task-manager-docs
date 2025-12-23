package errors

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
)

type ErrorCode string

const (
	// Validation errors
	CodeValidationFailed     ErrorCode = "VALIDATION_FAILED"
	CodeInvalidTaskID        ErrorCode = "INVALID_TASK_ID"
	CodeInvalidFileFormat    ErrorCode = "INVALID_FILE_FORMAT"
	CodeFileTooLarge         ErrorCode = "FILE_TOO_LARGE"
	CodeInvalidMetadata      ErrorCode = "INVALID_METADATA" //TODO убрать
	CodeInvalidFilename      ErrorCode = "INVALID_FILENAME"
	CodeInvalidRequest       ErrorCode = "INVALID_REQUEST"
	CodeInvalidMessageFormat ErrorCode = "INVALID_MESSAGE_FORMAT"
	CodeUnknownMessageType   ErrorCode = "UNKNOWN_MESSAGE_TYPE"
	// Resource errors
	CodeTaskNotFound ErrorCode = "TASK_NOT_FOUND"
	CodeFileNotFound ErrorCode = "FILE_NOT_FOUND"

	// Infrastructure errors
	CodeDatabaseError ErrorCode = "DATABASE_ERROR"
	CodeStorageError  ErrorCode = "STORAGE_ERROR"
	CodeKafkaError    ErrorCode = "KAFKA_ERROR"

	// Business logic errors
	CodeTaskCreationFailed ErrorCode = "TASK_CREATION_FAILED"
	CodeInvalidTaskType    ErrorCode = "INVALID_TASK_TYPE"

	// Internal errors
	CodeInternalError ErrorCode = "INTERNAL_ERROR"

	// Processing Task
	CodeTaskAlreadyCompleted  ErrorCode = "TASK_ALREADY_COMPLETED"
	CodeTaskAlreadyFailed     ErrorCode = "TASK_ALREADY_FAILED"
	CodeTaskAlreadyProcessing ErrorCode = "TASK_ALREADY_PROCESSING"
)

type AppError struct {
	Code       ErrorCode
	Message    string
	Internal   error `json:"-"`
	HTTPStatus int
	Details    map[string]interface{}

	File     string `json:"-"`
	Line     int    `json:"-"`
	Function string `json:"-"`
}

func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Internal
}

// Wrap оборачивает внутреннюю ошибку
func Wrap(code ErrorCode, message string, err error) *AppError {
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)

	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
		Internal:   err,
		File:       filepath.Base(file),
		Line:       line,
		Function:   fn.Name(),
	}
}

// WithStatus устанавливает HTTP статус
func (e *AppError) WithStatus(status int) *AppError {
	e.HTTPStatus = status
	return e
}

// WithDetails добавляет детали к ошибке
func (e *AppError) WithDetails(details map[string]interface{}) *AppError {
	e.Details = details
	return e
}

var (
	//Validation errors
	ErrValidationFailed = New(CodeValidationFailed, "ошибка валидации").
				WithStatus(http.StatusBadRequest)

	//Client errors
	ErrInvalidTaskID = New(CodeInvalidTaskID, "неверный формат ID задачи").
				WithStatus(http.StatusBadRequest)

	ErrFileTooLarge = New(CodeFileTooLarge, "размер файла превышает 1GB").
			WithStatus(http.StatusBadRequest)

	ErrInvalidMetadata = New(CodeInvalidMetadata, "неверный формат метаданных").
				WithStatus(http.StatusBadRequest)

	ErrTaskNotFound = New(CodeTaskNotFound, "задача не найдена").
			WithStatus(http.StatusNotFound)

	ErrFileNotFound = New(CodeFileNotFound, "файл не найден в запросе").
			WithStatus(http.StatusBadRequest)

	//Adapter errors
	ErrDatabaseError = New(CodeDatabaseError, "ошибка операции с базой данных").
				WithStatus(http.StatusInternalServerError)

	ErrStorageError = New(CodeStorageError, "ошибка операции с хранилищем").
			WithStatus(http.StatusInternalServerError)

	ErrKafkaError = New(CodeKafkaError, "ошибка операции с брокером сообщений").
			WithStatus(http.StatusInternalServerError)

	//Server errors
	ErrInternalError = New(CodeInternalError, "внутренняя ошибка сервера").
				WithStatus(http.StatusInternalServerError)
)

// IsAppError проверяет является ли ошибка AppError
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError

	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
