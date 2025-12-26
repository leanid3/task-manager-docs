package repository

import (
	"app/internal/entity/domain"
	"context"
)

type Broker interface {
	//PublishTaskCreated публикует событие создания задачи в Kafka
	PublishTaskCreated(ctx context.Context, task *domain.Task) error

	Health(ctx context.Context) error
}
