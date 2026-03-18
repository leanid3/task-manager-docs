package deduplication

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Deduplicator интерфейс для дедупликации задач
type Deduplicator interface {
	// IsDuplicate проверяет, является ли задача дубликатом
	IsDuplicate(ctx context.Context, taskHash string) (bool, error)
	// MarkAsProcessed отмечает задачу как обработанную
	MarkAsProcessed(ctx context.Context, taskHash string, ttl time.Duration) error
	// GenerateTaskHash генерирует уникальный хеш для задачи
	GenerateTaskHash(taskID uuid.UUID, filename string, filesize int64, requestID string) string
}

// InMemoryDeduplicator реализация дедупликации в памяти
type InMemoryDeduplicator struct {
	processed map[string]time.Time
	ttl       time.Duration
}

// NewInMemoryDeduplicator создает новый in-memory дедупликатор
func NewInMemoryDeduplicator(ttl time.Duration) *InMemoryDeduplicator {
	dedup := &InMemoryDeduplicator{
		processed: make(map[string]time.Time),
		ttl:       ttl,
	}

	// Запускаем горутину для очистки старых записей
	go dedup.cleanupExpired()

	return dedup
}

// IsDuplicate проверяет, является ли задача дубликатом
func (id *InMemoryDeduplicator) IsDuplicate(ctx context.Context, taskHash string) (bool, error) {
	now := time.Now()
	if processedTime, exists := id.processed[taskHash]; exists {
		if now.Sub(processedTime) <= id.ttl {
			return true, nil
		}
		// Удаляем просроченную запись
		delete(id.processed, taskHash)
	}

	return false, nil
}

// MarkAsProcessed отмечает задачу как обработанную
func (id *InMemoryDeduplicator) MarkAsProcessed(ctx context.Context, taskHash string, ttl time.Duration) error {
	id.processed[taskHash] = time.Now()
	return nil
}

// GenerateTaskHash генерирует уникальный хеш для задачи
func (id *InMemoryDeduplicator) GenerateTaskHash(taskID uuid.UUID, filename string, filesize int64, requestID string) string {
	data := fmt.Sprintf("%s|%s|%d|%s", taskID.String(), filename, filesize, requestID)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// cleanupExpired очищает просроченные записи
func (id *InMemoryDeduplicator) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		for hash, processedTime := range id.processed {
			if now.Sub(processedTime) > id.ttl {
				delete(id.processed, hash)
			}
		}
	}
}

// RedisDeduplicator реализация дедупликации с использованием Redis
type RedisDeduplicator struct {
	// Здесь будет реализация с использованием Redis
	// Для упрощения пока не реализуем, но оставим интерфейс
}

// NewRedisDeduplicator создает новый Redis дедупликатор
func NewRedisDeduplicator(redisAddress string) *RedisDeduplicator {
	return &RedisDeduplicator{}
}

// IsDuplicate проверяет, является ли задача дубликатом
func (rd *RedisDeduplicator) IsDuplicate(ctx context.Context, taskHash string) (bool, error) {
	// Реализация с использованием Redis
	return false, nil
}

// MarkAsProcessed отмечает задачу как обработанную
func (rd *RedisDeduplicator) MarkAsProcessed(ctx context.Context, taskHash string, ttl time.Duration) error {
	// Реализация с использованием Redis
	return nil
}

// GenerateTaskHash генерирует уникальный хеш для задачи
func (rd *RedisDeduplicator) GenerateTaskHash(taskID uuid.UUID, filename string, filesize int64, requestID string) string {
	data := fmt.Sprintf("%s|%s|%d|%s", taskID.String(), filename, filesize, requestID)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
