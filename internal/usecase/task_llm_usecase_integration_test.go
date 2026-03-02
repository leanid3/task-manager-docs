//go:build integration

package usecase

import (
	"app/internal/entity/domain"
	apperrors "app/internal/entity/errors"
	"app/internal/entity/repository"
	"app/internal/infrastructure/adapter/database/postgres"
	"app/internal/infrastructure/adapter/storage/minio"
	appkafka "app/pkg/kafka"
	appminio "app/pkg/minio"
	"app/test/mocks"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

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

	// Создаем TaskUC
	uc := NewTaskLLMUC(taskRepo, producer, storageRepo, "tasks_llm")

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
		require.Equal(t, domain.TaskStatusPending, task.GetStatus())
		metadata := task.GetMetadata()
		require.Equal(t, filename, metadata["filename"])
		require.Equal(t, "application/octet-stream", metadata["content_type"])

		// ✅ Проверяем, что файл в MinIO
		// Генерируем путь заново, так как репозиторий возвращает только Task, а не TaskLLM
		storagePath := storageRepo.GenerateStoragePath(taskID, filename)
		exists, err := storageRepo.FileExists(ctx, storagePath)
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
		require.Equal(t, taskID, task.GetID())
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

func TestTaskUCIntegrationUpdateTaskStatus(t *testing.T) {
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

	// Создаем TaskUC
	uc := NewTaskLLMUC(taskRepo, producer, storageRepo, "tasks_llm")

	t.Run("UpdateTaskStatus success", func(t *testing.T) {
		taskID, err := uc.CreateTask(ctx, bytes.NewBufferString("data"), "test.pdf", 1, "req-123")
		require.NoError(t, err)

		err = uc.UpdateTaskStatus(ctx, domain.TaskLLMStatusEvent{
			Key: domain.TaskContractKey{
				TaskID: taskID,
			},
			Headers: domain.TaskContractHeaders{
				Status: domain.TaskStatusProcessing.ToKafkaCode(),
			},
		})
		require.NoError(t, err)
	})

	t.Run("UpdateTaskStatus failed", func(t *testing.T) {
		taskID, err := uc.CreateTask(ctx, bytes.NewBufferString("data"), "test.pdf", 1, "req-123")
		require.NoError(t, err)

		err = uc.UpdateTaskStatus(ctx, domain.TaskLLMStatusEvent{
			Key: domain.TaskContractKey{
				TaskID: taskID,
			},
		})

		err = uc.UpdateTaskStatus(ctx, domain.TaskLLMStatusEvent{
			Key: domain.TaskContractKey{
				TaskID: taskID,
			},
			Headers: domain.TaskContractHeaders{
				Status: domain.TaskStatusProcessing.ToKafkaCode(),
			},
		})
		require.NoError(t, err)
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
	-- ============================================================================
	-- Базовая таблица tasks (общая для всех типов задач)
	-- ============================================================================
	CREATE TABLE tasks (
		-- Идентификация
		task_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		-- Состояние
		status        VARCHAR(50) NOT NULL DEFAULT 'PENDING',
		-- PENDING → PROCESSING → COMPLETED | FAILED
		result JSONB,
		-- Worker info (заполняется при обработке)
		worker_id     VARCHAR(100),  -- ID pod'а или worker instance
		-- Временные метки (для метрик и SLA)
		created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
		started_at    TIMESTAMP,      -- когда worker начал обработку
		completed_at  TIMESTAMP,      -- когда завершилась (успех или ошибка)
		
		-- Ошибки
		error_message TEXT,
		
		-- Трейсинг (observability)
		request_id    VARCHAR(100),   -- HTTP request ID (для логов gateway)
		trace_id      VARCHAR(100),   -- Distributed tracing ID (сквозной)
		metadata JSONB,
		-- Constraints
		CONSTRAINT tasks_status_check 
			CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED')),
		
		CONSTRAINT tasks_date_check 
			CHECK (
				(started_at IS NULL OR started_at >= created_at) AND
				(completed_at IS NULL OR completed_at >= created_at)
			)
	);

	-- Индексы
	CREATE INDEX idx_tasks_status ON tasks(status) WHERE status IN ('PENDING', 'PROCESSING');
	CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);
	CREATE INDEX idx_tasks_trace_id ON tasks(trace_id) WHERE trace_id IS NOT NULL;

	COMMENT ON TABLE tasks IS 'Базовая таблица задач (общая для всех типов)';
	COMMENT ON COLUMN tasks.request_id IS 'ID HTTP запроса клиента (для логов)';
	COMMENT ON COLUMN tasks.trace_id IS 'Distributed tracing ID (OpenTelemetry/Jaeger)';



	`
	_, err := pool.Exec(ctx, migration)
	return err
}

func setupStorageRepo(t *testing.T, ctx context.Context, minioConnectionString string, container *tminio.MinioContainer) repository.Storage {
	t.Helper()
	accessKey := container.Username
	secretKey := container.Password

	cfgMinio := &appminio.Config{
		Endpoint:  minioConnectionString,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    false,
		Region:    "us-east-1",
		Bucket:    "test-bucket",
		Timeout:   5 * time.Second,
	}

	l := mocks.NewMockLogger()
	connector, err := appminio.NewConnector(cfgMinio, l)
	require.NoError(t, err)

	return minio.NewMinioAdapter(connector, l)
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
	}, mocks.NewMockLogger())
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
