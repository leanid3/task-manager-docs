// pkg/logger/manager.go
package logger

import (
	"io"
	"os"
	"path/filepath"
)

type LoggerManager struct {
	loggers         map[string]Interface
	rotatingWriters []*RotatingWriter // для закрытия при завершении
}

type LoggerManagerConfig struct {
	Level        string
	Format       string
	Mode         string
	AppPath      string
	HealthPath   string
	HTTPPath     string
	KafkaPath    string
	MinioPath    string
	TaskPath     string
	S3Path       string
	MaxSize      int64 // максимальный размер файла в байтах (0 = без ограничений)
	MaxFiles     int   // максимальное количество файлов (0 = без ограничений)
	ClearOnStart bool  // очищать ли файл при старте
}

func NewManager(cfg LoggerManagerConfig) (*LoggerManager, error) {
	mgr := &LoggerManager{
		loggers:         make(map[string]Interface),
		rotatingWriters: make([]*RotatingWriter, 0),
	}

	switch cfg.Mode {
	case "files":
		if err := mgr.initFileMode(cfg); err != nil {
			return nil, err
		}
	case "stdout":
		fallthrough
	default:
		if err := mgr.initStdoutMode(cfg); err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func (mgr *LoggerManager) initStdoutMode(cfg LoggerManagerConfig) error {
	// Один логгер на stdout с category
	l := New(os.Stdout, cfg.Level, cfg.Format)
	mgr.loggers["app"] = l
	mgr.loggers["health"] = l
	mgr.loggers["http"] = l
	mgr.loggers["kafka"] = l
	mgr.loggers["minio"] = l
	mgr.loggers["task"] = l
	mgr.loggers["s3"] = l
	return nil
}

func (mgr *LoggerManager) initFileMode(cfg LoggerManagerConfig) error {
	// Используем fallback logger для логирования ошибок инициализации
	fallbackLogger := NewFallback()

	openLogWriter := func(path, defaultPath string) io.Writer {
		if path == "" {
			path = defaultPath
		}
		if path == "" {
			return os.Stdout
		}

		// Если ротация отключена (MaxSize = 0), используем обычный файл
		if cfg.MaxSize <= 0 {
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				fallbackLogger.Error("failed to create log directory", "dir", dir, "error", err)
				return os.Stdout
			}
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
			if err != nil {
				fallbackLogger.Error("failed to open log file", "path", path, "error", err)
				return os.Stdout
			}
			return file
		}

		// Используем RotatingWriter
		rw, err := NewRotatingWriter(path, cfg.MaxSize, cfg.MaxFiles, cfg.ClearOnStart)
		if err != nil {
			fallbackLogger.Error("failed to create rotating writer", "path", path, "error", err)
			return os.Stdout
		}
		mgr.rotatingWriters = append(mgr.rotatingWriters, rw)
		return rw
	}

	mgr.loggers["app"] = New(openLogWriter(cfg.AppPath, "/logs/app.log"), cfg.Level, cfg.Format)
	mgr.loggers["health"] = New(openLogWriter(cfg.HealthPath, "/logs/health.log"), "info", "text")
	mgr.loggers["http"] = New(openLogWriter(cfg.HTTPPath, "/logs/http.log"), "info", "json")
	mgr.loggers["kafka"] = New(openLogWriter(cfg.KafkaPath, "/logs/kafka.log"), "warn", "text")
	mgr.loggers["minio"] = New(openLogWriter(cfg.MinioPath, "/logs/minio.log"), "info", "json")
	mgr.loggers["task"] = New(openLogWriter(cfg.TaskPath, "/logs/task.log"), cfg.Level, cfg.Format)
	mgr.loggers["s3"] = New(openLogWriter(cfg.S3Path, "/logs/s3.log"), cfg.Level, cfg.Format)

	return nil
}

func (mgr *LoggerManager) Get(category string) Interface {
	if l, ok := mgr.loggers[category]; ok {
		return l
	}
	// Fallback на app logger
	return mgr.loggers["app"]
}

func (mgr *LoggerManager) Close() error {
	// Используем fallback logger для логирования ошибок закрытия
	fallbackLogger := NewFallback()

	// Закрываем все rotating writers
	for _, rw := range mgr.rotatingWriters {
		if err := rw.Close(); err != nil {
			fallbackLogger.Error("failed to close rotating writer", "error", err)
		}
	}
	return nil
}
