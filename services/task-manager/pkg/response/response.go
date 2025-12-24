package response

import (
	apperrors "app/internal/entity/errors"
	"time"
)

// ErrorResponse структура ответа с ошибкой
// @name ErrorResponse
type ErrorResponse struct {
	Error     string                 `json:"error"`
	Code      string                 `json:"code"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id"`
	Timestamp string                 `json:"timestamp"`
}

// SuccessResponse структура успешного ответа
type SuccessResponse struct {
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// SuccessData структура успешного ответа
// @name SuccessData
type SuccessData[T any] struct {
	Data      T      `json:"data"`
	RequestID string `json:"request_id,omitempty"`
	Timestamp string `json:"timestamp"`
}

// ValidationError структура для ошибок валидации
// @name ValidationError
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Tag     string `json:"tag,omitempty"`
}

// ValidationErrorResponse ответ с ошибками валидации
// @name ValidationErrorResponse
type ValidationErrorResponse struct {
	Error      string            `json:"error"`
	Code       string            `json:"code"`
	Validation []ValidationError `json:"validation"`
	RequestID  string            `json:"request_id"`
	Timestamp  string            `json:"timestamp"`
}

// Error создает ErrorResponse из AppError
func Error(err *apperrors.AppError, requestID string) ErrorResponse {
	return ErrorResponse{
		Error:     err.Message,
		Code:      string(err.Code),
		Details:   err.Details,
		RequestID: requestID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// simpleError создает ErrorResponse из простого сообщения и кода ошибки
func SimpleError(message, code, requestID string) ErrorResponse {
	return ErrorResponse{
		Error:     message,
		Code:      string(code),
		RequestID: requestID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// TODO удалить
// Success создает сообщение об успехе
func OLDSuccess(data interface{}, requestID string) SuccessResponse {
	return SuccessResponse{
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

// Success создает типизированное сообщение об успехе
// @name Success
func Success[T any](data T, requestID string) SuccessData[T] {
	return SuccessData[T]{
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func ValidationErrors(validationErrors []ValidationError, requestID string) ValidationErrorResponse {
	return ValidationErrorResponse{
		Error:      "ошибка валидации",
		Code:       string(apperrors.CodeValidationFailed),
		Validation: validationErrors,
		RequestID:  requestID,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}
