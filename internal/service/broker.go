// Package service содержит инфраструктурные сервисы — абстракции над внешними системами.
// UseCase-ы зависят от этих интерфейсов, а не от конкретных адаптеров.
package service

import (
	"context"
)

// Broker — абстракция над брокером сообщений (Kafka).
// UseCase отправляет команды через этот интерфейс, не зная о Kafka.
type Broker interface {
	// Send отправляет сообщение в топик
	Send(ctx context.Context, topic string, key string, headers map[string]string, value []byte) error
	// Close закрывает соединение
	Close() error
}
