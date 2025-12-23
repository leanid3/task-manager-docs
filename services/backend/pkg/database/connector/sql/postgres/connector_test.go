//go:build integration

package postgres

import (
	"app/pkg/logger"
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB создает PostgreSQL контейнер и возвращает pool, порт и cleanup функцию
func setupTestDB(t *testing.T) (*pgxpool.Pool, int, func()) {
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

	portInt, err := strconv.Atoi(port.Port())
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

	// Cleanup функция
	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}

	return pool, portInt, cleanup
}

func TestNewConnectorSuccess(t *testing.T) {
	_, port, cleanup := setupTestDB(t) // поднимаем testcontainers Postgres
	defer cleanup()

	cfg := &Config{
		User:           "postgres",
		Password:       "password",
		Host:           "localhost",
		Port:           port,
		Database:       "testdb",
		MaxConnections: 5,
		MinConnections: 1,
	}

	log := logger.NewMockLogger()
	log.On("Info", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	connector, err := NewConnector(cfg, log)
	require.NoError(t, err)
	require.NotNil(t, connector)

	// Проверяем, что Pool рабочий
	err = connector.HealthCheck(context.Background())
	require.NoError(t, err)
}
