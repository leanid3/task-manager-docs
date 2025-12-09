package main

import (
	"app/config"
	"app/internal/adapter/postgres"
	"app/internal/handlers"
	"app/internal/usecase"
	database "app/pkg/database/connector/sql/postgres"
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	// Подключение и разрыв соединения с базой данных
	connector, err := database.NewConnector(cfg.Database.ToPostgresConfig())
	slog.Info("creating connector", "config", cfg.Database.ToPostgresConfig())
	if err != nil {
		slog.Error("failed to create connector", "error", err)
		os.Exit(1)
	}
	defer connector.Close()

	// Инициализация репозиториев
	taskRepo := postgres.NewTaskPostgresAdapter(connector.Pool())
	slog.Info("task repo created", "taskRepo", taskRepo)

	// Инициализация сервисов
	taskService := usecase.NewTaskService(taskRepo)
	slog.Info("task usecase created", "taskService", taskService)

	// проверка подключения к базе данных
	if err := connector.HealthCheck(context.Background()); err != nil {
		slog.Error("failed to check connector health", "error", err)
		os.Exit(1)
	}
	slog.Info("connector health checked successfully")

	// Настройка роутеров
	//TODO вынести в отдельный файл
	//TODO добавить валидацию
	//TODO добавить middleware для логирования запросов
	//TODO изменить формат request и response
	mux := http.NewServeMux()

	// Регистрируем handlers
	mux.HandleFunc("/", handlers.HomeHandler)
	mux.HandleFunc("/health", handlers.HealthHandler)
	mux.HandleFunc("/api/users", handlers.UsersHandler)
	mux.HandleFunc("/api/users/", handlers.UserByIDHandler)

	//TODO добавить middleware для логирования запросов
	// Конфигурация сервера
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	//TODO улучшить graceful shutdown
	// Graceful shutdown
	go func() {
		log.Printf("🚀 Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited properly")
}

// TODO пренести в  слой handler
// Middleware для логирования запросов
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// TODO адаптивровать слой config, убрать этот метод
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
