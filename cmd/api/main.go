package main

import (
	"app/config"
	brokerhandlers "app/internal/handlers/broker"
	"app/internal/handlers/broker/extractors"
	workflowLlm "app/internal/handlers/broker/workflows/llm"
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
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Используем fallback logger до инициализации основного менеджера
	fallbackLogger := logger.NewFallback()

	cfg, err := config.Load(fallbackLogger)
	if err != nil {
		fallbackLogger.Error("failed to load config", "error", err)
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
		fallbackLogger.Error("failed to init log manager", "error", err)
		os.Exit(1)
	}
	defer logMgr.Close()

	l := logMgr.Get("app")
	l.Info("logger manager initialized", "mode", cfg.Logger.Mode, "level", cfg.Logger.Level)
	// Подключение к базе данных
	dbCfg := &database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		Database:        cfg.Database.Database,
		MaxConnections:  cfg.Database.MaxConnections,
		MinConnections:  cfg.Database.MinConnections,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
	}
	postgresConnector, err := database.NewConnector(dbCfg, logMgr.Get("health"))
	if err != nil {
		l.Error("failed to create postgres connector", "error", err)
		os.Exit(1)
	}

	minioCfg := &pkgminio.Config{
		Endpoint:  cfg.Minio.Endpoint,
		AccessKey: cfg.Minio.AccessKey,
		SecretKey: cfg.Minio.SecretKey,
		Bucket:    cfg.Minio.Bucket,
		Prefix:    cfg.Minio.Prefix,
		UseSSL:    cfg.Minio.UseSSL,
		Region:    cfg.Minio.Region,
		Timeout:   cfg.Minio.Timeout,
	}
	minioConnector, err := pkgminio.NewConnector(minioCfg, logMgr.Get("minio"))
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
	storageRepo := minio.NewMinioAdapter(minioConnector, logMgr.Get("minio"))

	// Usecases
	taskLLMUC := usecase.NewTaskLLMUC(taskRepo, kafkaProducer, storageRepo, cfg.Broker.Topics[0], logMgr.Get("task"))

	// Создаем фабрику процессоров задач
	taskProcessorFactory := usecase.NewTaskProcessorFactory()

	// Создаем универсальный usecase
	unifiedTaskUC := usecase.NewUnifiedTaskUC(taskRepo, storageRepo, taskProcessorFactory, logMgr.Get("task"))

	usecases := usecase.NewUseCases(taskLLMUC, unifiedTaskUC)

	l.Info("application components initialized",
		"task_repo", "created",
		"storage_repo", "created",
		"task_usecase", "created")

	// Health check
	if err := postgresConnector.HealthCheck(context.Background()); err != nil {
		l.Error("postgres health check failed", "error", err)
		os.Exit(1)
	}
	l.Info("postgres health check passed")

	//## Consumer
	// 1. Создаем registry - маршрутизатор, estractor - парсер, общая валидатор
	registry := brokerhandlers.NewRegistry(l)
	extractor := extractors.NewHeaderExtractor(l)
	baseValidator := brokerhandlers.NewBaseValidator(extractor)

	// 2. Конкретные маршруты и валидиация
	llmValidator := workflowLlm.NewTaskLLMValidator(baseValidator)
	workflowLlm.Register(registry, llmValidator, usecases.TaskLLMUC, l)

	//Создаем consumer
	kafkaHandler := brokerhandlers.NewKafkaMessageHandler(registry, logMgr.Get("kafka"))
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

	// Запускаем consumer в фоне
	go func() {
		l.Info("starting kafka consumer", "topics", cfg.Broker.Topics, "group_id", cfg.Broker.ConsumerGroupID)
		if err := cons.Start(ctx); err != nil {
			l.Error("kafka consumer failed", "error", err)
		}
		l.Info("kafka consumer stopped")
	}()

	//## HTTP сервер (главный поток)
	httpServer := httpserver.New(logMgr.Get("app"), httpserver.Port(cfg.Server.Port), httpserver.ReadTimeout(cfg.Server.ReadTimeout))
	handlers.NewRoutes(httpServer.Engine(), cfg, *usecases, logMgr.Get("http"))

	l.Info("starting HTTP server", "port", cfg.Server.Port, "host", cfg.Server.Host)
	httpServer.Start()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("shutting down application")
	cancel() // Остановит consumer
	httpServer.Shutdown()
	cons.Stop()
	postgresConnector.Close()

	l.Info("graceful shutdown complete")
}
