//go:build integration

package minio

import (
	"app/test/mocks/logger"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tminio "github.com/testcontainers/testcontainers-go/modules/minio"
)

func setupMinioContainer(t *testing.T) (*Config, func()) {
	t.Helper()

	ctx := context.Background()

	// Запускаем MinIO контейнер
	container, err := tminio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	require.NoError(t, err)

	// Достаём endpoint и креды из контейнера
	// ConnectionString возвращает строку вида "host:port"
	connectionString, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	// MinioContainer имеет поля Username и Password напрямую
	accessKey := container.Username
	secretKey := container.Password

	// Готовим конфиг для нашего Connector
	cfg := &Config{
		Endpoint:  connectionString, // формат host:port
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    false,
		Region:    "us-east-1",
		Bucket:    "test-bucket",
		Timeout:   5 * time.Second,
	}

	cleanup := func() {
		_ = container.Terminate(ctx)
	}

	return cfg, cleanup
}

func TestConnectorLifecycle(t *testing.T) {
	cfg, cleanup := setupMinioContainer(t)
	defer cleanup()

	// Используем mock logger для тестов
	l := logger.NewMockLogger()

	// Act: создаём коннектор
	connector, err := NewConnector(cfg, l)

	// Assert: коннектор создался и обеспечил существование бакета
	require.NoError(t, err)
	require.NotNil(t, connector)

	client := connector.Client()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Проверяем, что бакет действительно существует
	exists, err := client.BucketExists(ctx, connector.BucketName())
	require.NoError(t, err)
	require.True(t, exists, "bucket must exist after NewConnector")

	// Проверяем Health
	err = connector.Health(ctx)
	require.NoError(t, err, "Health must succeed for running minio and existing bucket")
}
