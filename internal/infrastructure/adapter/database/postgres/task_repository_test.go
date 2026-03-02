//go:build integration

package postgres

import (
	"app/internal/entity/domain"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB создает PostgreSQL контейнер и возвращает pool
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	// Создаем PostgreSQL контейнер
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "password",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort(nat.Port("5432/tcp")),
		).WithDeadline(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	// Получаем порт
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Подключаемся к БД с retry
	connString := fmt.Sprintf(
		"postgresql://postgres:password@localhost:%s/testdb?sslmode=disable",
		port.Port(),
	)

	var pool *pgxpool.Pool
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.New(ctx, connString)
		if err == nil {
			// Проверяем подключение
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		if i < maxRetries-1 {
			time.Sleep(500 * time.Millisecond * time.Duration(i+1))
		}
	}
	require.NoError(t, err, "failed to connect to database after %d retries", maxRetries)

	// Запускаем миграции
	err = runMigrations(pool)
	require.NoError(t, err)

	// Cleanup функция
	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}

	return pool, cleanup
}

// runMigrations выполняет SQL миграции
func runMigrations(pool *pgxpool.Pool) error {
	ctx := context.Background()

	// Ваша миграция для tasks таблицы
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

// TestTaskRepositoryCreate проверяет создание задачи
func TestTaskRepositoryCreate(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Подготовка данных
	taskID := uuid.New()
	traceID := uuid.New()
	task := &domain.BaseTask{
		TaskID:    taskID,
		Status:    domain.TaskStatusPending,
		RequestID: "req-123",
		TraceID:   &traceID,
		CreatedAt: time.Now(),
		Metadata:  nil,
	}

	// Act
	err := repo.Create(ctx, task)

	// Assert
	require.NoError(t, err)

	// Проверяем, что задача действительно создана в БД
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM tasks WHERE task_id = $1", taskID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Task должна быть в БД")

	// Проверяем поля
	var status string
	var traceIDUUID uuid.UUID
	err = pool.QueryRow(ctx,
		"SELECT status, trace_id FROM tasks WHERE task_id = $1",
		taskID,
	).Scan(&status, &traceIDUUID)
	require.NoError(t, err)
	assert.Equal(t, string(domain.TaskStatusPending), status)
	assert.Equal(t, traceID, traceIDUUID)
}

// TestTaskRepositoryGetByID проверяет получение задачи
func TestTaskRepositoryGetByID(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем задачу напрямую в БД
	taskID := uuid.New()
	metadata := `{"filename": "test.pdf", "filesize": 1024}`
	traceID := uuid.New()

	_, err := pool.Exec(ctx, `
        INSERT INTO tasks (task_id, status, metadata, request_id, trace_id, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, taskID, "PENDING", metadata, "req-123", &traceID, time.Now())
	require.NoError(t, err)

	// Act
	task, err := repo.GetByID(ctx, taskID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, taskID, task.GetID())
	assert.Equal(t, domain.TaskStatusPending, task.GetStatus())
	assert.Equal(t, "req-123", task.GetRequestID())
	assert.Equal(t, &traceID, task.GetTraceID())
}

// TestTaskRepositoryGetByIDNotFound проверяет ошибку "не найдено"
func TestTaskRepositoryGetByIDNotFound(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Act
	task, err := repo.GetByID(ctx, uuid.New())

	// Assert
	assert.Error(t, err)
	assert.Nil(t, task)
	assert.Contains(t, err.Error(), "task not found")
}

// TestTaskRepositoryUpdateStatus проверяет обновление статуса
func TestTaskRepositoryUpdateStatus(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем задачу
	taskID := uuid.New()
	traceID := uuid.New()
	worker_id := "worker-1"
	metadata := map[string]interface{}{
		"filename": "test.pdf",
		"filesize": 1024,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		require.NoError(t, err)
	}
	_, err = pool.Exec(ctx, `
        INSERT INTO tasks (task_id, status, metadata, request_id, trace_id, created_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, taskID, "PENDING", metadataJSON, "req-123", &traceID, time.Now())

	// Act: Обновляем статус на PROCESSING
	err = repo.UpdateWithStatus(ctx, taskID, worker_id, domain.TaskStatusProcessing)

	// Assert
	require.NoError(t, err)

	// Проверяем, что статус обновился
	var status string
	var startedAt *time.Time
	err = pool.QueryRow(ctx,
		"SELECT status, started_at FROM tasks WHERE task_id = $1",
		taskID,
	).Scan(&status, &startedAt)
	require.NoError(t, err)
	assert.Equal(t, string(domain.TaskStatusProcessing), status)
	assert.NotNil(t, startedAt, "started_at должен быть установлен при переходе в PROCESSING")
}

// TestTaskRepositoryUpdateStatusCompletedSetsTimestamp проверяет автоматическую установку completed_at
func TestTaskRepositoryUpdateStatusCompletedSetsTimestamp(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()
	worker_id := "worker-1"

	// Создаем задачу
	taskID := uuid.New()
	_, err := pool.Exec(ctx, `
        INSERT INTO tasks (task_id, status, created_at)
        VALUES ($1, $2, $3)
    `, taskID, "PENDING", time.Now())
	require.NoError(t, err)

	// Act: Обновляем статус на COMPLETED
	err = repo.UpdateWithStatus(ctx, taskID, worker_id, domain.TaskStatusCompleted)

	// Assert
	require.NoError(t, err)

	// Проверяем, что completed_at установлен
	var completedAt *time.Time
	err = pool.QueryRow(ctx,
		"SELECT completed_at FROM tasks WHERE task_id = $1",
		taskID,
	).Scan(&completedAt)
	require.NoError(t, err)
	assert.NotNil(t, completedAt, "completed_at должен быть установлен при статусе COMPLETED")
}

// TestTaskRepositoryUpdateError проверяет сохранение ошибки
func TestTaskRepositoryUpdateError(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем задачу
	taskID := uuid.New()
	_, err := pool.Exec(ctx, `
        INSERT INTO tasks (task_id, status, created_at)
        VALUES ($1, $2, $3)
    `, taskID, "PENDING", time.Now())
	require.NoError(t, err)

	// Act
	errorMsg := "kafka broker unavailable"
	err = repo.UpdateWithError(ctx, taskID, domain.TaskStatusFailed, errorMsg)

	// Assert
	require.NoError(t, err)

	// Проверяем, что ошибка сохранена
	var savedError string
	err = pool.QueryRow(ctx,
		"SELECT error_message FROM tasks WHERE task_id = $1",
		taskID,
	).Scan(&savedError)
	require.NoError(t, err)
	assert.Equal(t, errorMsg, savedError)
}

// TestTaskRepositoryListByStatus проверяет получение списка задач
func TestTaskRepositoryListByStatus(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем несколько задач с разными статусами
	now := time.Now()

	// 3 PENDING задачи
	for i := 0; i < 3; i++ {
		_, err := pool.Exec(ctx, `
            INSERT INTO tasks (task_id, status, created_at)
            VALUES ($1, $2, $3)
        `, uuid.New(), "PENDING", now.Add(time.Duration(i)*time.Second))
		require.NoError(t, err)
	}

	// 2 COMPLETED задачи
	for i := 0; i < 2; i++ {
		_, err := pool.Exec(ctx, `
            INSERT INTO tasks (task_id, status,  created_at)
            VALUES ($1, $2, $3)
        `, uuid.New(), "COMPLETED", now)
		require.NoError(t, err)
	}

	// Act: Получаем PENDING задачи
	tasks, err := repo.ListByStatus(ctx, domain.TaskStatusPending, 10)

	// Assert
	require.NoError(t, err)
	assert.Len(t, tasks, 3, "Должно быть 3 PENDING задачи")

	// Проверяем сортировку (по created_at DESC)
	for i := 0; i < len(tasks)-1; i++ {
		assert.True(t,
			tasks[i].GetCreatedAt().After(tasks[i+1].GetCreatedAt()) || tasks[i].GetCreatedAt().Equal(tasks[i+1].GetCreatedAt()),
			"Задачи должны быть отсортированы по created_at DESC",
		)
	}
}

// TestTaskRepositoryListByStatusLimit проверяет ограничение количества
func TestTaskRepositoryListByStatusLimit(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем 10 задач
	for i := 0; i < 10; i++ {
		_, err := pool.Exec(ctx, `
            INSERT INTO tasks (task_id, status, created_at)
            VALUES ($1, $2, $3)
        `, uuid.New(), "PENDING", time.Now())
		require.NoError(t, err)
	}

	// Act: Получаем только 5 задач
	tasks, err := repo.ListByStatus(ctx, domain.TaskStatusPending, 5)

	// Assert
	require.NoError(t, err)
	assert.Len(t, tasks, 5, "Должно быть ровно 5 задач из-за лимита")
}

// TestTaskRepositoryCreateJSONMetadata проверяет сериализацию сложного JSON
func TestTaskRepositoryCreateJSONMetadata(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Сложная metadata с вложенными структурами
	taskID := uuid.New()
	metadata := map[string]interface{}{
		"filename": "test.pdf",
		"filesize": 1024,
	}

	// Act - создаем задачу через репозиторий
	err := repo.Create(ctx, &domain.BaseTask{
		TaskID:    taskID,
		Status:    domain.TaskStatusPending,
		CreatedAt: time.Now(),
		Metadata:  metadata,
	})
	require.NoError(t, err)

	// Получаем задачу обратно
	retrieved, err := repo.GetByID(ctx, taskID)
	require.NoError(t, err)

	// Assert: Проверяем десериализацию сложной metadata
	// JSON unmarshal преобразует числа в float64, поэтому сравниваем значения отдельно
	metadata = retrieved.GetMetadata()
	assert.Equal(t, "test.pdf", metadata["filename"])
	assert.Equal(t, float64(1024), metadata["filesize"])
}

func TestTaskRepositoryUpdateWithStatus(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем задачу
	taskID := uuid.New()
	worker_id := "worker-1"
	_, err := pool.Exec(ctx, `
		INSERT INTO tasks (task_id, status, created_at)
		VALUES ($1, $2, $3)
	`, taskID, "PENDING", time.Now())
	require.NoError(t, err)

	// Act: Обновляем статус на PROCESSING
	err = repo.UpdateWithStatus(ctx, taskID, worker_id, domain.TaskStatusProcessing)

	// Assert
	require.NoError(t, err)

	// Проверяем, что статус обновился
	var status string
	var startedAt *time.Time
	err = pool.QueryRow(ctx,
		"SELECT status, started_at FROM tasks WHERE task_id = $1",
		taskID,
	).Scan(&status, &startedAt)
	require.NoError(t, err)
	assert.Equal(t, string(domain.TaskStatusProcessing), status)
	assert.NotNil(t, startedAt, "started_at должен быть установлен при переходе в PROCESSING")
}

func TestTaskRepositoryUpdateWithResult(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewTaskRepository(pool)
	ctx := context.Background()

	// Создаем задачу
	taskID := uuid.New()
	worker_id := "worker-1"
	_, err := pool.Exec(ctx, `
		INSERT INTO tasks (task_id, status,  created_at)
		VALUES ($1, $2, $3)
	`, taskID, "PENDING", time.Now())
	require.NoError(t, err)

	// Act: Обновляем результат
	err = repo.UpdateWithResult(ctx, taskID, worker_id, domain.TaskStatusCompleted, []byte(`{"result": "success"}`))

	// Assert
	require.NoError(t, err)

	// Проверяем, что результат сохранен
	var result string
	err = pool.QueryRow(ctx,
		"SELECT result FROM tasks WHERE task_id = $1",
		taskID,
	).Scan(&result)
	require.NoError(t, err)
	assert.Equal(t, `{"result": "success"}`, result)
}
