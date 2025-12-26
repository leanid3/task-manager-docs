package v1

import (
	"app/pkg/response"
	"fmt"
	"net/http"

	apperrors "app/internal/entity/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (r *V1) handleError(c *gin.Context, err error, operation string) {
	requestID := c.GetString("request_id")

	baseArgs := []interface{}{
		"request_id", requestID,
		"path", c.Request.URL.Path,
		"method", c.Request.Method,
		"operation", operation,
	}

	switch {
	case err == nil:
		return

	case isValidationError(err):
		// Валидационные ошибки - это не ERROR, а WARN
		r.l.Warn(operation, append(baseArgs, "error", err.Error(), "type", "validation")...)

		validationErrors := parseValidationErrors(err)
		c.JSON(http.StatusBadRequest, response.ValidationErrors(validationErrors, requestID))
		return

	default:
		if appErr, ok := apperrors.IsAppError(err); ok {
			// Разные уровни логирования в зависимости от типа ошибки
			logLevel := r.getLogLevel(appErr)

			args := append(baseArgs,
				"error", err.Error(),
				"error_code", appErr.Code,
				"http_status", appErr.HTTPStatus,
			)

			// Добавляем source information из AppError
			if appErr.File != "" {
				args = append(args, "error_source", fmt.Sprintf("%s:%d", appErr.File, appErr.Line))
			}

			switch logLevel {
			case "error":
				r.l.ErrorWithSkip(1, operation, args...)
			case "warn":
				r.l.Warn(operation, args...)
			case "info":
				r.l.Info(operation, args...)
			}

			c.JSON(appErr.HTTPStatus, response.Error(appErr, requestID))
			return
		}

		// Неизвестные ошибки - всегда ERROR
		r.l.ErrorWithSkip(1, operation, append(baseArgs,
			"error", err.Error(),
			"type", "unexpected",
		)...)

		genericErr := apperrors.ErrInternalError.WithDetails(map[string]interface{}{
			"операция": operation,
		})
		c.JSON(http.StatusInternalServerError, response.Error(genericErr, requestID))
	}
}

func (r *V1) getLogLevel(err *apperrors.AppError) string {
	switch {
	case err.HTTPStatus >= 500:
		return "error" // Серверные ошибки
	case err.HTTPStatus == 404:
		return "info" // Not found - обычное явление
	case err.HTTPStatus >= 400:
		return "warn" // Клиентские ошибки
	default:
		return "info"
	}
}

// isValidationError проверяет является ли ошибка ошибкой валидации
func isValidationError(err error) bool {
	_, ok := err.(validator.ValidationErrors)
	return ok
}

// abortWithError прерывает выполнение и возвращает ошибку
func (r *V1) abortWithError(c *gin.Context, err error, operation string) {
	r.handleError(c, err, operation)
	c.Abort()
}
