package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// CentralizedValidator централизованный валидатор
type CentralizedValidator struct {
	validator *validator.Validate
}

// NewCentralizedValidator создает новый централизованный валидатор
func NewCentralizedValidator() *CentralizedValidator {
	return &CentralizedValidator{
		validator: validator.New(),
	}
}

// Validate валидирует структуру
func (cv *CentralizedValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}

// ValidateField валидирует отдельное поле
func (cv *CentralizedValidator) ValidateField(field interface{}, tag string) error {
	return cv.validator.Var(field, tag)
}

// RegisterValidation регистрирует пользовательскую валидацию
func (cv *CentralizedValidator) RegisterValidation(tag string, fn validator.Func) error {
	return cv.validator.RegisterValidation(tag, fn)
}

// ValidateTaskInput валидирует входные данные задачи
func (cv *CentralizedValidator) ValidateTaskInput(filename string, filesize int64, requestID string) error {
	var errors []string

	// Проверка имени файла
	if filename == "" {
		errors = append(errors, "filename cannot be empty")
	} else {
		// Проверка на безопасность имени файла
		if !isValidFilename(filename) {
			errors = append(errors, "filename contains invalid characters")
		}
	}

	// Проверка размера файла
	if filesize <= 0 {
		errors = append(errors, "filesize must be greater than zero")
	}

	// Проверка requestID
	if requestID == "" {
		errors = append(errors, "requestID cannot be empty")
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %s", strings.Join(errors, ", "))
	}

	return nil
}

// isValidFilename проверяет, является ли имя файла безопасным
func isValidFilename(filename string) bool {
	// Проверяем, что имя файла не содержит опасные символы
	// и не пытается выйти за пределы директории
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return false
	}

	// Проверяем, что имя файла соответствует допустимому формату
	validName := regexp.MustCompile(`^[a-zA-Z0-9._-]+\.[a-zA-Z0-9]+$`)
	return validName.MatchString(filename)
}

// ValidateTaskID валидирует ID задачи
func (cv *CentralizedValidator) ValidateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	// Проверяем формат UUID
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(taskID) {
		return fmt.Errorf("invalid task ID format: %s", taskID)
	}

	return nil
}

// ValidateContentType валидирует тип контента
func (cv *CentralizedValidator) ValidateContentType(contentType string) error {
	allowedTypes := []string{
		"application/pdf",
		"text/plain",
		"text/csv",
		"application/json",
		"application/xml",
		"image/jpeg",
		"image/png",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/octet-stream",
	}

	for _, allowedType := range allowedTypes {
		if contentType == allowedType {
			return nil
		}
	}

	return fmt.Errorf("unsupported content type: %s", contentType)
}

// ValidateTaskType валидирует тип задачи
func (cv *CentralizedValidator) ValidateTaskType(taskType string) error {
	allowedTypes := []string{"llm", "parsing", "algorithms", "analyze"}

	for _, allowedType := range allowedTypes {
		if taskType == allowedType {
			return nil
		}
	}

	return fmt.Errorf("unsupported task type: %s", taskType)
}

// ValidateWorkerID валидирует ID воркера
func (cv *CentralizedValidator) ValidateWorkerID(workerID string) error {
	if workerID == "" {
		return fmt.Errorf("worker ID cannot be empty")
	}

	// Проверяем формат worker ID (допускаем только буквы, цифры, дефисы и подчеркивания)
	workerIDRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !workerIDRegex.MatchString(workerID) {
		return fmt.Errorf("invalid worker ID format: %s", workerID)
	}

	return nil
}
