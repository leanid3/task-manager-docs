//go:build integration

package usecase

import (
	"app/config"
	"app/internal/entity/domain"
	"app/internal/entity/repository"
	"app/internal/infrastructure/adapter/database/postgres"
	"app/internal/infrastructure/adapter/storage/minio"
	appkafka "app/pkg/kafka"
	"app/pkg/logger"
	appminio "app/pkg/minio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	apperrors "app/internal/entity/errors"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	kafkatestcontainers "github.com/testcontainers/testcontainers-go/modules/kafka"
	tminio "github.com/testcontainers/testcontainers-go/modules/minio"
	postgrestestcontainers "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestTaskUCIntegrationCreateTask(t *testing.T) {
	ctx := context.Background()

	// 🗄️ Поднимаем Postgres
	postgresContainer, err := postgrestestcontainers.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgrestestcontainers.WithDatabase("testdb"),
		postgrestestcontainers.WithUsername("testuser"),
		postgrestestcontainers.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	defer postgresContainer.Terminate(ctx)

	pgURL, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// 🗃️ Поднимаем MinIO
	minioContainer, err := tminio.Run(ctx, "minio/minio:RELEASE.2024-01-16T16-07-38Z")
	require.NoError(t, err)
	defer minioContainer.Terminate(ctx)

	minioConnectionString, err := minioContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// 📨 Поднимаем Kafka
	kafkaContainer, err := kafkatestcontainers.RunContainer(ctx,
		kafkatestcontainers.WithClusterID("test-cluster"),
	)
	require.NoError(t, err)
	defer kafkaContainer.Terminate(ctx)

	kafkaBootstrapServers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)

	// Инициализируем репозитории с реальными сервисами
	taskRepo := setupTaskRepo(t, ctx, pgURL)
	storageRepo := setupStorageRepo(t, ctx, minioConnectionString, minioContainer)
	producer := setupKafkaProducer(t, kafkaBootstrapServers)
	defer producer.Close()

	// Создаем TaskUC с реальными зависимостями
	l := logger.NewMockLogger()
	uc := NewTaskLLMUC(taskRepo, producer, storageRepo, l)

	t.Run("CreateTask success", func(t *testing.T) {
		filename := "test.pdf"
		body := bytes.NewBufferString("fake-pdf-content")
		requestID := "req-test-123"

		taskID, err := uc.CreateTask(ctx, body, filename, int64(body.Len()), requestID)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, taskID)

		// ✅ Проверяем, что задача создана в DB
		task, err := uc.GetTaskByID(ctx, taskID)
		require.NoError(t, err)
		require.Equal(t, domain.TaskStatusPending, task.Status)
		require.Equal(t, filename, task.Metadata["filename"])
		require.Equal(t, "application/pdf", task.Metadata["content_type"])

		// ✅ Проверяем, что файл в MinIO
		exists, err := storageRepo.FileExists(ctx, task.StoragePath)
		require.NoError(t, err)
		require.True(t, exists)

		// ✅ Проверяем, что сообщение в Kafka (через consumer)
		msg := waitForKafkaMessage(t, ctx, kafkaBootstrapServers, "tasks_llm", taskID.String(), 10*time.Second)
		require.NotNil(t, msg)
		require.Contains(t, string(msg.Value), taskID.String())
	})

	t.Run("CreateTask validation errors", func(t *testing.T) {
		_, err := uc.CreateTask(ctx, nil, "test.pdf", 10, "req-123")
		require.Error(t, err)
		appErr, ok := apperrors.IsAppError(err)
		require.True(t, ok)
		require.Equal(t, apperrors.CodeValidationFailed, appErr.Code)
		require.Equal(t, http.StatusBadRequest, appErr.HTTPStatus)

		_, err = uc.CreateTask(ctx, bytes.NewBufferString("data"), "", 10, "req-123")
		require.Error(t, err)
		appErr, ok = apperrors.IsAppError(err)
		require.True(t, ok)
		require.Equal(t, apperrors.CodeValidationFailed, appErr.Code)

		_, err = uc.CreateTask(ctx, bytes.NewBufferString("data"), "test.pdf", 0, "req-123")
		require.Error(t, err)
		appErr, ok = apperrors.IsAppError(err)
		require.True(t, ok)
		require.Equal(t, apperrors.CodeValidationFailed, appErr.Code)
	})

	t.Run("GetTaskByID success", func(t *testing.T) {
		// Создаем задачу
		taskID, err := uc.CreateTask(ctx, bytes.NewBufferString("data"), "test.pdf", 4, "req-123")
		require.NoError(t, err)

		// Получаем
		task, err := uc.GetTaskByID(ctx, taskID)
		require.NoError(t, err)
		require.Equal(t, taskID, task.TaskID)
	})

	t.Run("GetTaskByID not found", func(t *testing.T) {
		task, err := uc.GetTaskByID(ctx, uuid.New())
		require.Error(t, err)
		require.Nil(t, task)
		appErr, ok := apperrors.IsAppError(err)
		require.True(t, ok)
		require.Equal(t, apperrors.CodeDatabaseError, appErr.Code)
	})
}

// Helpers для настройки реальных репозиториев
func setupTaskRepo(t *testing.T, ctx context.Context, pgURL string) repository.Task {
	t.Helper()
	pool, err := pgxpool.New(ctx, pgURL)
	require.NoError(t, err)

	// Запускаем миграции
	err = runMigrations(pool)
	require.NoError(t, err)

	return postgres.NewTaskRepository(pool)
}

func runMigrations(pool *pgxpool.Pool) error {
	ctx := context.Background()
	migration := `
	CREATE TABLE IF NOT EXISTS tasks (
		task_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		status        VARCHAR(50) NOT NULL DEFAULT 'PENDING',
		storage_path  TEXT NOT NULL,
		storage_size  BIGINT NOT NULL,
		metadata      JSONB,
		worker_id     UUID,
		created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
		started_at    TIMESTAMP,
		completed_at  TIMESTAMP,
		error_message TEXT,
		request_id    VARCHAR(100),
		trace_id      VARCHAR(100),
		result_worker JSONB,
		
		CONSTRAINT tasks_status_check 
			CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED')),
		
		CONSTRAINT tasks_date_check 
			CHECK (
				(started_at IS NULL OR started_at >= created_at) AND
				(completed_at IS NULL OR completed_at >= created_at)
			)
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status) WHERE status IN ('PENDING', 'PROCESSING');
	CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);
	CREATE INDEX IF NOT EXISTS idx_tasks_trace_id ON tasks(trace_id) WHERE trace_id IS NOT NULL;
	`
	_, err := pool.Exec(ctx, migration)
	return err
}

func setupStorageRepo(t *testing.T, ctx context.Context, minioConnectionString string, container *tminio.MinioContainer) repository.Storage {
	t.Helper()
	accessKey := container.Username
	secretKey := container.Password

	cfg := &config.MinioConfig{
		Endpoint:  minioConnectionString,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    false,
		Region:    "us-east-1",
		Bucket:    "test-bucket",
		Timeout:   5 * time.Second,
	}

	connector, err := appminio.NewConnector(cfg, logger.NewMockLogger())
	require.NoError(t, err)

	return minio.NewMinioAdapter(connector, logger.NewMockLogger())
}

func setupKafkaProducer(t *testing.T, bootstrapServers []string) appkafka.Producer {
	t.Helper()
	brokers := strings.Join(bootstrapServers, ",")
	producer, err := appkafka.NewProducer(appkafka.ProducerConfig{
		BootstrapServers:          brokers,
		ClientID:                  "test-client",
		ProducerAcks:              "-1",
		ProducerEnableIdempotence: true,
		ProducerCompressionType:   "snappy",
		ProducerRetries:           3,
	}, logger.NewMockLogger())
	require.NoError(t, err)
	return producer
}

func waitForKafkaMessage(t *testing.T, ctx context.Context, brokers []string, topic, taskID string, timeout time.Duration) *kafka.Message {
	t.Helper()
	brokersStr := strings.Join(brokers, ",")
	cfg := &kafka.ConfigMap{
		"bootstrap.servers": brokersStr,
		"group.id":          fmt.Sprintf("test-consumer-%d", time.Now().UnixNano()),
		"auto.offset.reset": "earliest",
	}

	consumer, err := kafka.NewConsumer(cfg)
	require.NoError(t, err)
	defer consumer.Close()

	err = consumer.SubscribeTopics([]string{topic}, nil)
	require.NoError(t, err)

	// Даем время consumer присоединиться к группе
	time.Sleep(1 * time.Second)

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		msg, err := consumer.ReadMessage(5 * time.Second)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if strings.Contains(string(msg.Value), taskID) {
			return msg
		}
	}
	t.Fatal("kafka message timeout")
	return nil
}
