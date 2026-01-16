package cache

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryCache реализация in-memory кеша
type InMemoryCache struct {
	data map[string]*cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// NewInMemoryCache создает новый in-memory кеш
func NewInMemoryCache() *InMemoryCache {
	cache := &InMemoryCache{
		data: make(map[string]*cacheItem),
	}

	// Запускаем горутину для очистки просроченных элементов
	go cache.cleanupExpired()

	return cache
}

// Get получает значение из кеша
func (c *InMemoryCache) Get(ctx context.Context, key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.data[key]
	if !exists || time.Now().After(item.expiration) {
		delete(c.data, key) // Удаляем просроченный элемент
		return nil, nil
	}

	return item.value, nil
}

// Set устанавливает значение в кеш
func (c *InMemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiration := time.Now().Add(ttl)
	c.data[key] = &cacheItem{
		value:      value,
		expiration: expiration,
	}

	return nil
}

// Delete удаляет значение из кеша
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, key)
	return nil
}

// Exists проверяет существование ключа в кеша
func (c *InMemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.data[key]
	if !exists {
		return false, nil
	}

	if time.Now().After(item.expiration) {
		delete(c.data, key) // Удаляем просроченный элемент
		return false, nil
	}

	return true, nil
}

// cleanupExpired очищает просроченные элементы
func (c *InMemoryCache) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.data {
			if now.After(item.expiration) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

// TaskCacheImpl реализация кеширования задач
type TaskCacheImpl struct {
	cache Cache
}

// NewTaskCache создает новый кеш задач
func NewTaskCache(cache Cache) *TaskCacheImpl {
	return &TaskCacheImpl{
		cache: cache,
	}
}

// GetTask получает задачу из кеша
func (tc *TaskCacheImpl) GetTask(ctx context.Context, taskID uuid.UUID) (interface{}, error) {
	key := "task:" + taskID.String()
	return tc.cache.Get(ctx, key)
}

// SetTask устанавливает задачу в кеш
func (tc *TaskCacheImpl) SetTask(ctx context.Context, taskID uuid.UUID, task interface{}, ttl time.Duration) error {
	key := "task:" + taskID.String()
	return tc.cache.Set(ctx, key, task, ttl)
}

// DeleteTask удаляет задачу из кеша
func (tc *TaskCacheImpl) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	key := "task:" + taskID.String()
	return tc.cache.Delete(ctx, key)
}

// InvalidateTask инвалидирует кеш задачи
func (tc *TaskCacheImpl) InvalidateTask(ctx context.Context, taskID uuid.UUID) error {
	return tc.DeleteTask(ctx, taskID)
}