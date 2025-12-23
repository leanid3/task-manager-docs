package main

import (
	"app/config"
	brokerhandlers "app/internal/handlers/broker"
	handlers "app/internal/handlers/restapi"
	"app/internal/infrastructure/adapter/database/postgres"
	"app/internal/infrastructure/adapter/storage/minio"
	"app/internal/usecase"
	database "app/pkg/database/connector/sql/postgres"
	"app/pkg/httpserver"
	"app/pkg/kafka"
	"app/pkg/logger"
	pkgminio "app/pkg/minio"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Инициализация логгеров
	// Конвертируем MaxSize из мегабайт в байты (если указано)
	maxSizeBytes := cfg.Logger.MaxSize * 1024 * 1024
	logMgr, err := logger.NewManager(logger.LoggerManagerConfig{
		Level:        cfg.Logger.Level,
		Format:       cfg.Logger.Format,
		Mode:         cfg.Logger.Mode,
		AppPath:      cfg.Logger.AppPath,
		HealthPath:   cfg.Logger.HealthPath,
		HTTPPath:     cfg.Logger.HTTPPath,
		KafkaPath:    cfg.Logger.KafkaPath,
		MinioPath:    cfg.Logger.MinioPath,
		TaskPath:     cfg.Logger.TaskPath,
		S3Path:       cfg.Logger.S3Path,
		MaxSize:      maxSizeBytes,
		MaxFiles:     cfg.Logger.MaxFiles,
		ClearOnStart: cfg.Logger.ClearOnStart,
	})
	if err != nil {
		slog.Error("failed to init log manager", "error", err)
		os.Exit(1)
	}
	defer logMgr.Close()

	l := logMgr.Get("app")
	l.Info("logger manager initialized", "mode", cfg.Logger.Mode)
	// Подключение к базе данных
	postgresConnector, err := database.NewConnector(&cfg.Database, logMgr.Get("health"))
	if err != nil {
		l.Error("failed to create postgres connector", "error", err)
		os.Exit(1)
	}

	minioConnector, err := pkgminio.NewConnector(&cfg.Minio, logMgr.Get("minio"))
	if err != nil {
		l.Error("failed to create minio connector", "error", err)
		os.Exit(1)
	}

	kafkaProducer, err := kafka.NewProducer(kafka.ProducerConfig{
		BootstrapServers:          cfg.Broker.BootstrapService,
		ClientID:                  cfg.Broker.ClientID,
		ProducerAcks:              cfg.Broker.ProducerAcks,
		ProducerEnableIdempotence: cfg.Broker.ProducerEnableIdempotence,
		ProducerCompressionType:   cfg.Broker.ProducerCompressionType,
		ProducerRetries:           cfg.Broker.ProducerRetries,
	}, logMgr.Get("kafka"))
	if err != nil {
		l.Error("failed to create kafka connector", "error", err)
		os.Exit(1)
	}

	// Репозитории
	taskRepo := postgres.NewTaskRepository(postgresConnector.Pool())
	logMgr.Get("health").Info("Task repo created")

	storageRepo := minio.NewMinioAdapter(minioConnector, logMgr.Get("minio"))
	logMgr.Get("health").Info("minio repo created")

	// Usecases
	taskLLMUC := usecase.NewTaskLLMUC(taskRepo, kafkaProducer, storageRepo, cfg.Broker.Topics[0], logMgr.Get("task"))
	logMgr.Get("health").Info("task llm usecase created")

	usecases := usecase.NewUseCases(*taskLLMUC)
	logMgr.Get("health").Info("task llm usecases created")

	// Health check
	if err := postgresConnector.HealthCheck(context.Background()); err != nil {
		logMgr.Get("health").Error("postgres health check failed", "error", err)
		os.Exit(1)
	}
	logMgr.Get("health").Info("postgres health check passed")

	//TODO расширить для нескольких типов tasks
	kafkaHandler := brokerhandlers.NewKafkaMessageHandler(*usecases, logMgr.Get("kafka"))

	cons, err := kafka.NewConsumerWithHandler(ctx, kafkaHandler, cfg.Broker.Topics, kafka.ConsumerConfig{
		BootstrapServers:     cfg.Broker.BootstrapService,
		ClientID:             cfg.Broker.ClientID,
		GroupID:              cfg.Broker.ConsumerGroupID,
		EnableAutoCommit:     cfg.Broker.ConsumerEnableAutoCommit,
		AutoCommitIntervalMs: cfg.Broker.ConsumerAutoCommitIntervalMs,
		SessionTimeoutMs:     cfg.Broker.ConsumerSessionTimeoutMs,
		HeartbeatIntervalMs:  cfg.Broker.ConsumerHeartbeatIntervalMs,
	}, logMgr.Get("kafka"))
	if err != nil {
		l.Error("failed to create kafka consumer", "error", err)
		os.Exit(1)
	}

	// ✅ Запускаем CONSUMER в фоне
	go func() {
		l.Info("🚀 Starting Kafka consumer...")
		if err := cons.Start(ctx); err != nil {
			l.Error("kafka consumer failed", "error", err)
		}
		l.Info("🛑 Kafka consumer stopped")
	}()

	// ✅ HTTP сервер (главный поток)
	httpServer := httpserver.New(logMgr.Get("app"), httpserver.Port(cfg.Server.Port), httpserver.ReadTimeout(cfg.Server.ReadTimeout))
	handlers.NewRoutes(httpServer.Engine(), cfg, *usecases, logMgr.Get("http"))

	l.Info("🚀 Starting HTTP server", "port", cfg.Server.Port)
	httpServer.Start() // ← Теперь запустится!

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("🛑 Shutting down...")
	cancel() // Остановит consumer
	httpServer.Shutdown()
	cons.Stop()
	postgresConnector.Close()

	l.Info("✅ Graceful shutdown complete")
}
