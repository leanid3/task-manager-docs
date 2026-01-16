package limits

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TimeoutManager интерфейс для управления таймаутами
type TimeoutManager interface {
	// WithTimeout создает контекст с таймаутом для задачи
	WithTimeout(ctx context.Context, taskID uuid.UUID, taskType string) (context.Context, context.CancelFunc)
	// GetTimeout возвращает таймаут для типа задачи
	GetTimeout(taskType string) time.Duration
	// SetTimeout устанавливает таймаут для типа задачи
	SetTimeout(taskType string, timeout time.Duration)
}

// RateLimiter интерфейс для ограничения частоты
type RateLimiter interface {
	// Allow проверяет, разрешен ли запрос
	Allow(key string) bool
	// AllowN проверяет, разрешено ли N запросов
	AllowN(key string, n int) bool
}

// ResourceLimiter интерфейс для ограничения ресурсов
type ResourceLimiter interface {
	// Acquire пытается получить ресурс
	Acquire(ctx context.Context, resourceType string) error
	// Release освобождает ресурс
	Release(resourceType string)
	// GetAvailable возвращает количество доступных ресурсов
	GetAvailable(resourceType string) int
}

// DefaultTimeoutManager реализация менеджера таймаутов
type DefaultTimeoutManager struct {
	timeouts map[string]time.Duration
}

// NewDefaultTimeoutManager создает новый менеджер таймаутов
func NewDefaultTimeoutManager() *DefaultTimeoutManager {
	return &DefaultTimeoutManager{
		timeouts: map[string]time.Duration{
			"llm":       300 * time.Second,  // 5 минут
			"parsing":   120 * time.Second,  // 2 минуты
			"algorithms": 60 * time.Second,  // 1 минута
			"analyze":   180 * time.Second,  // 3 минуты
		},
	}
}

// WithTimeout создает контекст с таймаутом для задачи
func (dtm *DefaultTimeoutManager) WithTimeout(ctx context.Context, taskID uuid.UUID, taskType string) (context.Context, context.CancelFunc) {
	timeout := dtm.GetTimeout(taskType)
	return context.WithTimeout(ctx, timeout)
}

// GetTimeout возвращает таймаут для типа задачи
func (dtm *DefaultTimeoutManager) GetTimeout(taskType string) time.Duration {
	if timeout, exists := dtm.timeouts[taskType]; exists {
		return timeout
	}
	// Возвращаем стандартный таймаут, если тип не найден
	return 300 * time.Second
}

// SetTimeout устанавливает таймаут для типа задачи
func (dtm *DefaultTimeoutManager) SetTimeout(taskType string, timeout time.Duration) {
	dtm.timeouts[taskType] = timeout
}

// MemoryRateLimiter реализация ограничителя частоты в памяти
type MemoryRateLimiter struct {
	limits map[string]*rateBucket
}

type rateBucket struct {
	lastReset time.Time
	count     int
	limit     int
	window    time.Duration
}

// NewMemoryRateLimiter создает новый ограничитель частоты
func NewMemoryRateLimiter() *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limits: make(map[string]*rateBucket),
	}
}

// Allow проверяет, разрешен ли запрос
func (mrl *MemoryRateLimiter) Allow(key string) bool {
	return mrl.AllowN(key, 1)
}

// AllowN проверяет, разрешено ли N запросов
func (mrl *MemoryRateLimiter) AllowN(key string, n int) bool {
	now := time.Now()
	
	bucket, exists := mrl.limits[key]
	if !exists {
		// Устанавливаем стандартные лимиты (например, 10 запросов в минуту)
		bucket = &rateBucket{
			lastReset: now,
			count:     0,
			limit:     10,
			window:    time.Minute,
		}
		mrl.limits[key] = bucket
	} else {
		// Сброс счетчика, если прошло достаточно времени
		if now.Sub(bucket.lastReset) >= bucket.window {
			bucket.count = 0
			bucket.lastReset = now
		}
	}

	// Проверяем, не превышает ли количество запросов лимит
	if bucket.count+n > bucket.limit {
		return false
	}

	bucket.count += n
	return true
}

// SemaphoreResourceLimiter реализация ограничителя ресурсов с помощью семафора
type SemaphoreResourceLimiter struct {
	semaphores map[string]chan struct{}
	limits     map[string]int
}

// NewSemaphoreResourceLimiter создает новый ограничитель ресурсов
func NewSemaphoreResourceLimiter() *SemaphoreResourceLimiter {
	return &SemaphoreResourceLimiter{
		semaphores: make(map[string]chan struct{}),
		limits:     make(map[string]int),
	}
}

// SetLimit устанавливает лимит для типа ресурса
func (srl *SemaphoreResourceLimiter) SetLimit(resourceType string, limit int) {
	srl.limits[resourceType] = limit
	srl.semaphores[resourceType] = make(chan struct{}, limit)
}

// Acquire пытается получить ресурс
func (srl *SemaphoreResourceLimiter) Acquire(ctx context.Context, resourceType string) error {
	semaphore, exists := srl.semaphores[resourceType]
	if !exists {
		return fmt.Errorf("resource type '%s' not configured", resourceType)
	}

	select {
	case semaphore <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release освобождает ресурс
func (srl *SemaphoreResourceLimiter) Release(resourceType string) {
	semaphore, exists := srl.semaphores[resourceType]
	if exists {
		select {
		case <-semaphore:
		default:
			// Не должно происходить, но на всякий случай
		}
	}
}

// GetAvailable возвращает количество доступных ресурсов
func (srl *SemaphoreResourceLimiter) GetAvailable(resourceType string) int {
	semaphore, exists := srl.semaphores[resourceType]
	if !exists {
		return 0
	}
	return cap(semaphore) - len(semaphore)
}

// TaskLimitsChecker интерфейс для проверки ограничений задач
type TaskLimitsChecker interface {
	// CheckLimits проверяет, удовлетворяет ли задача ограничениям
	CheckLimits(ctx context.Context, taskID uuid.UUID, taskType string, userID string) error
	// UpdateUsage обновляет использование ресурсов
	UpdateUsage(taskID uuid.UUID, taskType string, userID string, duration time.Duration)
}

// DefaultTaskLimitsChecker реализация проверки ограничений задач
type DefaultTaskLimitsChecker struct {
	userLimits map[string]*userUsage
	taskLimits map[string]int64 // максимальный размер файла в байтах
}

type userUsage struct {
	taskCount int
	totalSize int64
	lastReset time.Time
}

// NewDefaultTaskLimitsChecker создает новый проверяльщик ограничений
func NewDefaultTaskLimitsChecker() *DefaultTaskLimitsChecker {
	return &DefaultTaskLimitsChecker{
		userLimits: make(map[string]*userUsage),
		taskLimits: map[string]int64{
			"llm":       100 * 1024 * 1024,      // 100MB
			"parsing":   50 * 1024 * 1024,       // 50MB
			"algorithms": 25 * 1024 * 1024,      // 25MB
			"analyze":   75 * 1024 * 1024,       // 75MB
		},
	}
}

// CheckLimits проверяет, удовлетворяет ли задача ограничениям
func (dtlc *DefaultTaskLimitsChecker) CheckLimits(ctx context.Context, taskID uuid.UUID, taskType string, userID string) error {
	now := time.Now()
	usage, exists := dtlc.userLimits[userID]
	
	if !exists || now.Sub(usage.lastReset) >= 24*time.Hour {
		// Сброс дневной статистики
		usage = &userUsage{
			taskCount: 0,
			totalSize: 0,
			lastReset: now,
		}
		dtlc.userLimits[userID] = usage
	}

	// Проверяем ограничения (например, не более 100 задач в день)
	if usage.taskCount >= 100 {
		return fmt.Errorf("daily task limit exceeded for user %s", userID)
	}

	return nil
}

// UpdateUsage обновляет использование ресурсов
func (dtlc *DefaultTaskLimitsChecker) UpdateUsage(taskID uuid.UUID, taskType string, userID string, duration time.Duration) {
	usage, exists := dtlc.userLimits[userID]
	if !exists {
		return // Это может быть ошибка, но для упрощения игнорируем
	}
	
	usage.taskCount++
}