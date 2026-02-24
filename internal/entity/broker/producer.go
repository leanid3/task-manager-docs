package broker

import (
	"context"
)

type Producer interface {
	Send(ctx context.Context, topic string, key string, headers map[string]string, value []byte) error
	Close() error
}

type Consumer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
