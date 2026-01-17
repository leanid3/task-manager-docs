package errors

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrorType тип ошибки
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeKafka      ErrorType = "kafka"
	ErrorTypeStorage    ErrorType = "storage"
	ErrorTypeNetwork    ErrorType = "network"
	ErrorTypeBusiness   ErrorType = "business"
	ErrorTypeSystem     ErrorType = "system"
)

// ErrorSeverity уровень серьезности ошибки
type ErrorSeverity string

const (
	ErrorSeverityLow    ErrorSeverity = "low"
	ErrorSeverityMedium ErrorSeverity = "medium"
	ErrorSeverityHigh   ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// ErrorClassifier интерфейс для классификации ошибок
type ErrorClassifier interface {
	Classify(err error) (ErrorType, ErrorSeverity)
	IsRetryable(err error) bool
	GetRetryDelay(attempt int) time.Duration
}

// StandardErrorClassifier стандартный классификатор ошибок
type StandardErrorClassifier struct{}

// NewStandardErrorClassifier создает новый стандартный классификатор ошибок
func NewStandardErrorClassifier() *StandardErrorClassifier {
	return &StandardErrorClassifier{}
}

// Classify классифицирует ошибку
func (sec *StandardErrorClassifier) Classify(err error) (ErrorType, ErrorSeverity) {
	errStr := err.Error()

	// Проверяем на основе текста ошибки
	switch {
	case containsAny(errStr, "validation", "invalid", "format"):
		return ErrorTypeValidation, ErrorSeverityHigh
	case containsAny(errStr, "database", "sql", "connection", "pool"):
		return ErrorTypeDatabase, ErrorSeverityHigh
	case containsAny(errStr, "kafka", "producer", "consumer", "topic", "partition"):
		return ErrorTypeKafka, ErrorSeverityMedium
	case containsAny(errStr, "storage", "upload", "download", "file", "minio"):
		return ErrorTypeStorage, ErrorSeverityMedium
	case containsAny(errStr, "timeout", "connection refused", "network"):
		return ErrorTypeNetwork, ErrorSeverityMedium
	case containsAny(errStr, "business", "constraint", "rule"):
		return ErrorTypeBusiness, ErrorSeverityHigh
	default:
		return ErrorTypeSystem, ErrorSeverityMedium
	}
}

// IsRetryable определяет, можно ли повторить операцию
func (sec *StandardErrorClassifier) IsRetryable(err error) bool {
	errStr := err.Error()

	// Ошибки, которые можно повторить
	retryableErrors := []string{
		"timeout", "connection refused", "connection reset", "network",
		"database connection", "kafka unavailable", "temporary", "retry",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// GetRetryDelay возвращает задержку перед повторной попыткой
func (sec *StandardErrorClassifier) GetRetryDelay(attempt int) time.Duration {
	// Экспоненциальная задержка с jitter
	baseDelay := time.Second * time.Duration(attempt*attempt)
	maxDelay := time.Minute * 5

	if baseDelay > maxDelay {
		baseDelay = maxDelay
	}

	// Добавляем jitter (до 25% от базовой задержки)
	jitter := time.Duration(float64(baseDelay) * 0.25)
	delay := baseDelay + time.Duration(getRandomInt(int(jitter)))

	return delay
}

// contains проверяет, содержит ли строка подстроку
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

// containsInMiddle проверяет, содержится ли подстрока в середине строки
func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// containsAny проверяет, содержит ли строка любую из подстрок
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

// getRandomInt возвращает случайное число (для упрощения возвращаем фиксированное значение)
func getRandomInt(max int) int {
	// В реальной реализации использовать crypto/rand
	return max / 2
}

// ErrorHandler интерфейс для обработки ошибок
type ErrorHandler interface {
	Handle(ctx context.Context, taskID uuid.UUID, err error) error
	LogError(ctx context.Context, taskID uuid.UUID, err error, severity ErrorSeverity)
	Notify(ctx context.Context, taskID uuid.UUID, err error, severity ErrorSeverity)
}

// StandardErrorHandler стандартный обработчик ошибок
type StandardErrorHandler struct {
	classifier ErrorClassifier
	logger     interface{} // logger.Interface, но чтобы избежать циклической зависимости
}

// NewStandardErrorHandler создает новый стандартный обработчик ошибок
func NewStandardErrorHandler(classifier ErrorClassifier) *StandardErrorHandler {
	return &StandardErrorHandler{
		classifier: classifier,
	}
}

// Handle обрабатывает ошибку
func (seh *StandardErrorHandler) Handle(ctx context.Context, taskID uuid.UUID, err error) error {
	_, severity := seh.classifier.Classify(err)

	// Логируем ошибку
	seh.LogError(ctx, taskID, err, severity)

	// В зависимости от типа и серьезности ошибки принимаем решение
	switch severity {
	case ErrorSeverityCritical, ErrorSeverityHigh:
		// Для критических ошибок отправляем уведомление
		seh.Notify(ctx, taskID, err, severity)
		return err
	case ErrorSeverityMedium:
		// Для средних ошибок можем попробовать повторить
		if seh.classifier.IsRetryable(err) {
			// Здесь может быть логика повторных попыток
			return fmt.Errorf("retryable error: %w", err)
		}
		return err
	default:
		// Для низких ошибок просто логируем
		return err
	}
}

// LogError логирует ошибку
func (seh *StandardErrorHandler) LogError(ctx context.Context, taskID uuid.UUID, err error, severity ErrorSeverity) {
	// В реальной реализации использовать logger
	fmt.Printf("ERROR [TaskID: %s, Severity: %s]: %v\n", taskID, severity, err)
}

// Notify отправляет уведомление об ошибке
func (seh *StandardErrorHandler) Notify(ctx context.Context, taskID uuid.UUID, err error, severity ErrorSeverity) {
	// В реальной реализации отправлять уведомления (email, slack, etc.)
	fmt.Printf("NOTIFICATION [TaskID: %s, Severity: %s]: %v\n", taskID, severity, err)
}

// RetryPolicy политика повторных попыток
type RetryPolicy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Backoff     func(attempt int, baseDelay time.Duration) time.Duration
}

// DefaultRetryPolicy возвращает стандартную политику повторных попыток
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    time.Minute,
		Backoff:     ExponentialBackoff,
	}
}

// ExponentialBackoff экспоненциальная задержка
func ExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	delay := baseDelay * time.Duration(1<<uint(attempt-1)) // 2^(attempt-1)
	if delay > time.Minute {
		delay = time.Minute
	}
	return delay
}

// LinearBackoff линейная задержка
func LinearBackoff(attempt int, baseDelay time.Duration) time.Duration {
	delay := baseDelay * time.Duration(attempt)
	if delay > time.Minute {
		delay = time.Minute
	}
	return delay
}