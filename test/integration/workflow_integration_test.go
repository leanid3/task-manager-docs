package integration_test

import (
	"app/internal/entity/domain"
	"app/internal/entity/repository"
	"app/internal/infrastructure/adapter/database/postgres"
	"app/internal/infrastructure/adapter/storage/minio"
	"app/internal/usecase"
	pg_connector "app/pkg/database/connector/sql/postgres"
	"app/pkg/logger"
	minio_pkg "app/pkg/minio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestComponents struct {
	TaskRepo    repository.Task
	StorageRepo repository.Storage
	Logger      logger.Interface
	DBContainer testcontainers.Container
	MinioCont   testcontainers.Container
}

func setupTestEnvironment(ctx context.Context, t *testing.T) *TestComponents {
	// Create a network for our containers
	net, err := network.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := net.Remove(ctx); err != nil {
			t.Logf("failed to remove network: %v", err)
		}
	})

	// Start PostgreSQL container
	reqDB := testcontainers.ContainerRequest{
		Image: "postgres:15-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       "test_db",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5 * time.Minute),
		),
	}
	dbContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: reqDB,
		Started:          true,
	})
	require.NoError(t, err)

	// Get connection string
	host, err := dbContainer.Host(ctx)
	require.NoError(t, err)
	port, err := dbContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Start MinIO container
	reqMinIO := testcontainers.ContainerRequest{
		Image: "minio/minio:RELEASE.2024-01-16T16-07-38Z",
		Env: map[string]string{
			"MINIO_ROOT_USER":     "minioadmin",
			"MINIO_ROOT_PASSWORD": "minioadmin",
		},
		Cmd:          []string{"server", "/data", "--console-address", ":9001"},
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		WaitingFor:   wait.ForLog("Server startup"),
	}
	minioContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: reqMinIO,
		Started:          true,
	})
	require.NoError(t, err)

	// Get MinIO endpoint
	minioHost, err := minioContainer.Host(ctx)
	require.NoError(t, err)
	minioPort, err := minioContainer.MappedPort(ctx, "9000")
	require.NoError(t, err)
	minioEndpoint := fmt.Sprintf("%s:%s", minioHost, minioPort.Port())

	// Setup database connection
	dbConfig := &pg_connector.Config{
		Host:            host,
		Port:            int(port.Int()),
		User:            "test_user",
		Password:        "test_password",
		Database:        "test_db",
		MaxConnections:  10,
		MinConnections:  5,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 30 * time.Minute,
	}
	dbLogger := logger.New(os.Stdout, "debug", "text")
	dbConn, err := pg_connector.NewConnector(dbConfig, dbLogger)
	require.NoError(t, err)

	// Setup MinIO connection
	minioConfig := &minio_pkg.Config{
		Endpoint:  minioEndpoint,
		AccessKey: "minioadmin",
		SecretKey: "minioadmin",
		UseSSL:    false,
		Region:    "us-east-1",
		Bucket:    "test-bucket",
		Timeout:   10 * time.Second,
	}
	minioConn, err := minio_pkg.NewConnector(minioConfig, dbLogger)
	require.NoError(t, err)

	// Create repositories
	taskRepo := postgres.NewTaskRepository(dbConn.Pool())
	storageRepo := minio.NewMinioAdapter(minioConn, dbLogger)

	return &TestComponents{
		TaskRepo:    taskRepo,
		StorageRepo: storageRepo,
		Logger:      dbLogger,
		DBContainer: dbContainer,
		MinioCont:   minioContainer,
	}
}

func TestTaskWorkflow_Integration(t *testing.T) {
	ctx := context.Background()

	components := setupTestEnvironment(ctx, t)
	defer func() {
		if err := components.DBContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate DB container: %v", err)
		}
		if err := components.MinioCont.Terminate(ctx); err != nil {
			t.Logf("failed to terminate MinIO container: %v", err)
		}
	}()

	// Initialize the unified usecase
	taskProcessorFactory := usecase.NewTaskProcessorFactory()
	unifiedTaskUC := usecase.NewUnifiedTaskUC(components.TaskRepo, components.StorageRepo, taskProcessorFactory, components.Logger)

	// Step 1: Create a task
	taskID := uuid.New()
	taskInput := domain.TaskInput{
		TaskID:    taskID,
		Filename:  "test-document.pdf",
		Filesize:  1024,
		RequestID: "req-" + uuid.New().String(),
		Metadata: map[string]interface{}{
			"task_type": "LLM",
			"priority":  "high",
		},
	}

	createdTaskID, err := unifiedTaskUC.CreateTask(ctx, taskInput)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, createdTaskID)

	// Verify the task was created in the database
	createdTask, err := unifiedTaskUC.GetTaskByID(ctx, createdTaskID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusPending, createdTask.Status)
	assert.Equal(t, taskInput.RequestID, createdTask.RequestID)
	assert.Equal(t, taskInput.Metadata, createdTask.Metadata)

	// Step 2: Simulate task processing by updating status to processing
	traceID := uuid.New()
	eventProcessing := domain.TaskEvent{
		Key: domain.TaskContractKey{
			TaskID: createdTaskID,
		},
		Headers: domain.TaskContractHeaders{
			Status:   domain.TaskStatusProcessing.ToKafkaCode(), // 2
			WorkerID: "worker-1",
			TraceID:  &traceID,
		},
		Value: map[string]interface{}{},
	}

	err = unifiedTaskUC.UpdateTaskStatus(ctx, eventProcessing)
	require.NoError(t, err)

	// Verify the task status was updated to processing
	updatedTask, err := unifiedTaskUC.GetTaskByID(ctx, createdTaskID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusProcessing, updatedTask.Status)
	assert.Equal(t, "worker-1", updatedTask.WorkerID)

	// Step 3: Simulate task completion with result
	resultData := map[string]interface{}{
		"summary": "This is a summary of the document",
		"pages":   10,
		"words":   1500,
	}
	jsonResult, err := json.Marshal(resultData)
	require.NoError(t, err)

	traceID2 := uuid.New()
	eventCompleted := domain.TaskEvent{
		Key: domain.TaskContractKey{
			TaskID: createdTaskID,
		},
		Headers: domain.TaskContractHeaders{
			Status:   domain.TaskStatusCompleted.ToKafkaCode(), // 3
			WorkerID: "worker-1",
			TraceID:  &traceID2,
		},
		Value: resultData,
	}

	err = unifiedTaskUC.UpdateTaskStatus(ctx, eventCompleted)
	require.NoError(t, err)

	// Verify the task status was updated to completed with result
	completedTask, err := unifiedTaskUC.GetTaskByID(ctx, createdTaskID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusCompleted, completedTask.Status)
	assert.JSONEq(t, string(jsonResult), string(completedTask.Result))
}

func TestTaskWorkflow_WithFileStorage_Integration(t *testing.T) {
	ctx := context.Background()

	components := setupTestEnvironment(ctx, t)
	defer func() {
		if err := components.DBContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate DB container: %v", err)
		}
		if err := components.MinioCont.Terminate(ctx); err != nil {
			t.Logf("failed to terminate MinIO container: %v", err)
		}
	}()

	// Initialize the unified usecase
	taskProcessorFactory := usecase.NewTaskProcessorFactory()
	unifiedTaskUC := usecase.NewUnifiedTaskUC(components.TaskRepo, components.StorageRepo, taskProcessorFactory, components.Logger)

	// Create a sample document to upload
	docContent := []byte("This is a test document for the task manager system.")
	docReader := bytes.NewReader(docContent)

	// Step 1: Create a task with file upload
	taskID := uuid.New()
	storagePath := components.StorageRepo.GenerateStoragePath(taskID, "test-document.pdf")

	// Upload the document to MinIO
	uploadOpts := minio_pkg.UploadOptions{
		ContentType: "application/pdf",
		Metadata: map[string]string{
			"task_id": taskID.String(),
			"purpose": "testing",
		},
	}
	uploadInfo, err := components.StorageRepo.UploadStream(ctx, storagePath, docReader, int64(len(docContent)), uploadOpts)
	require.NoError(t, err)
	require.NotNil(t, uploadInfo)

	// Verify the file exists in storage
	exists, err := components.StorageRepo.FileExists(ctx, storagePath)
	require.NoError(t, err)
	assert.True(t, exists)

	// Step 2: Create the task in the database referencing the stored file
	taskInput := domain.TaskInput{
		TaskID:    taskID,
		Filename:  "test-document.pdf",
		Filesize:  int64(len(docContent)),
		RequestID: "req-" + uuid.New().String(),
		Metadata: map[string]interface{}{
			"task_type": "PARSING",
			"file_path": storagePath,
		},
	}

	createdTaskID, err := unifiedTaskUC.CreateTask(ctx, taskInput)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, createdTaskID)

	// Step 3: Process the task (simulated)
	traceID3 := uuid.New()
	eventProcessing := domain.TaskEvent{
		Key: domain.TaskContractKey{
			TaskID: createdTaskID,
		},
		Headers: domain.TaskContractHeaders{
			Status:   domain.TaskStatusProcessing.ToKafkaCode(), // 2
			WorkerID: "worker-2",
			TraceID:  &traceID3,
		},
		Value: map[string]interface{}{},
	}

	err = unifiedTaskUC.UpdateTaskStatus(ctx, eventProcessing)
	require.NoError(t, err)

	// Step 4: Complete the task with results
	resultData := map[string]interface{}{
		"parsed_text": string(docContent),
		"word_count":  len(bytes.Fields(docContent)),
		"page_count":  1,
	}

	traceID4 := uuid.New()
	eventCompleted := domain.TaskEvent{
		Key: domain.TaskContractKey{
			TaskID: createdTaskID,
		},
		Headers: domain.TaskContractHeaders{
			Status:   domain.TaskStatusCompleted.ToKafkaCode(), // 3
			WorkerID: "worker-2",
			TraceID:  &traceID4,
		},
		Value: resultData,
	}

	err = unifiedTaskUC.UpdateTaskStatus(ctx, eventCompleted)
	require.NoError(t, err)

	// Step 5: Verify final state
	finalTask, err := unifiedTaskUC.GetTaskByID(ctx, createdTaskID)
	require.NoError(t, err)
	assert.Equal(t, domain.TaskStatusCompleted, finalTask.Status)
	assert.Equal(t, "worker-2", finalTask.WorkerID)
	assert.Contains(t, string(finalTask.Result), "test document for the task manager")
}
