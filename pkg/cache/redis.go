package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// RedisCache реализация кеша с использованием Redis
type RedisCache struct {
	client *redis.Client
}

// RedisConfig конфигурация Redis кеша
type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

// NewRedisCache создает новый Redis кеш
func NewRedisCache(config RedisConfig) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Password: config.Password,
		DB:       config.DB,
	})

	return &RedisCache{
		client: client,
	}
}

// Get получает значение из кеша
func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // ключ не существует
		}
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// Set устанавливает значение в кеш
func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, bytes, ttl).Err()
}

// Delete удаляет значение из кеша
func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	return rc.client.Del(ctx, key).Err()
}

// Exists проверяет существование ключа в кеше
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}