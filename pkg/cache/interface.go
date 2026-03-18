package cache

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Cache интерфейс для кеширования
type Cache interface {
	// Get получает значение из кеша
	Get(ctx context.Context, key string) (interface{}, error)
	// Set устанавливает значение в кеш
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Delete удаляет значение из кеша
	Delete(ctx context.Context, key string) error
	// Exists проверяет существование ключа в кеше
	Exists(ctx context.Context, key string) (bool, error)
}

// TaskCache интерфейс для кеширования задач
type TaskCache interface {
	// GetTask получает задачу из кеша
	GetTask(ctx context.Context, taskID uuid.UUID) (interface{}, error)
	// SetTask устанавливает задачу в кеш
	SetTask(ctx context.Context, taskID uuid.UUID, task interface{}, ttl time.Duration) error
	// DeleteTask удаляет задачу из кеша
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
	// InvalidateTask инвалидирует кеш задачи
	InvalidateTask(ctx context.Context, taskID uuid.UUID) error
}
