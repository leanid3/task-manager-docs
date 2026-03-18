package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Interface interface {
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	ErrorWithSkip(skip int, msg string, args ...interface{})
	Fatal(msg string, args ...interface{})

	// Методы с поддержкой контекста для передачи корреляционных ID
	InfoCtx(ctx context.Context, msg string, args ...interface{})
	DebugCtx(ctx context.Context, msg string, args ...interface{})
	WarnCtx(ctx context.Context, msg string, args ...interface{})
	ErrorCtx(ctx context.Context, msg string, args ...interface{})
}

type Logger struct {
	logger *slog.Logger
}

var _ Interface = (*Logger)(nil)

// NewFallback создает простой логгер для использования до инициализации основного менеджера
// Используется в критических местах, где логгер еще не инициализирован
func NewFallback() Interface {
	return New(os.Stderr, "info", "text")
}

func New(w io.Writer, level string, format string) *Logger {
	var l slog.Level

	// Определяем и устанавливаем уровень логирования
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "info":
		l = slog.LevelInfo
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	case "fatal":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetLogLoggerLevel(l)

	// Создаем handler с опциями
	opts := &slog.HandlerOptions{
		Level:     l,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				// ISO8601 формат для time
				return slog.String("timestamp", a.Value.Time().Format(time.RFC3339))
			}
			if a.Key == slog.SourceKey {
				if source, ok := a.Value.Any().(*slog.Source); ok {
					// Укорачиваем путь
					source.File = filepath.Base(source.File)
				}
			}
			return a
		},
	}

	var handler slog.Handler
	switch strings.ToLower(format) {
	case "color":
		fallthrough
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	case "text", "colored":
		if isTerminal(w) {
			handler = NewColorTextHandler(w, opts)
		} else {
			handler = slog.NewTextHandler(w, opts)
		}
	default:
		handler = slog.NewTextHandler(w, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		logger: logger,
	}
}

func (l *Logger) Info(msg string, args ...interface{}) {
	l.log(context.Background(), slog.LevelInfo, msg, args...)
}

func (l *Logger) Debug(msg string, args ...interface{}) {
	l.log(context.Background(), slog.LevelDebug, msg, args...)
}

func (l *Logger) Warn(msg string, args ...interface{}) {
	l.log(context.Background(), slog.LevelWarn, msg, args...)
}

func (l *Logger) Error(msg string, args ...interface{}) {
	l.log(context.Background(), slog.LevelError, msg, args...)
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.log(context.Background(), slog.LevelError, msg, args...)
	os.Exit(1)
}

// ErrorWithSkip версия Error для обработчика ошибок в handler
func (l *Logger) ErrorWithSkip(skip int, msg string, args ...interface{}) {
	l.logWithSkip(context.Background(), slog.LevelError, skip, msg, args...)
}

// Внутренний метод с правильным caller tracking
func (l *Logger) log(ctx context.Context, level slog.Level, msg string, args ...interface{}) {
	if !l.logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip 3: Callers, log, и вызывающий метод (Info/Debug/etc)

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	// Конвертируем args в атрибуты
	if len(args) > 0 {
		attrs := make([]slog.Attr, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				attrs = append(attrs, slog.Any(key, args[i+1]))
			}
		}
		r.AddAttrs(attrs...)
	}

	_ = l.logger.Handler().Handle(ctx, r)
}

// для handler
func (l *Logger) logWithSkip(ctx context.Context, level slog.Level, skip int, msg string, args ...interface{}) {
	if !l.logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(skip+3, pcs[:]) // +3 для базового skip

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	if len(args) > 0 {
		attrs := make([]slog.Attr, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				attrs = append(attrs, slog.Any(key, args[i+1]))
			}
		}
		r.AddAttrs(attrs...)
	}

	_ = l.logger.Handler().Handle(ctx, r)
}

// Методы с поддержкой контекста
func (l *Logger) InfoCtx(ctx context.Context, msg string, args ...interface{}) {
	l.logWithContext(ctx, slog.LevelInfo, msg, args...)
}

func (l *Logger) DebugCtx(ctx context.Context, msg string, args ...interface{}) {
	l.logWithContext(ctx, slog.LevelDebug, msg, args...)
}

func (l *Logger) WarnCtx(ctx context.Context, msg string, args ...interface{}) {
	l.logWithContext(ctx, slog.LevelWarn, msg, args...)
}

func (l *Logger) ErrorCtx(ctx context.Context, msg string, args ...interface{}) {
	l.logWithContext(ctx, slog.LevelError, msg, args...)
}

// Внутренний метод для логирования с контекстом
func (l *Logger) logWithContext(ctx context.Context, level slog.Level, msg string, args ...interface{}) {
	if !l.logger.Enabled(ctx, level) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip 3: Callers, logWithContext, и вызывающий метод

	r := slog.NewRecord(time.Now(), level, msg, pcs[0])

	// Добавляем атрибуты из контекста
	contextAttrs := extractContextAttributes(ctx)
	r.AddAttrs(contextAttrs...)

	// Конвертируем args в атрибуты
	if len(args) > 0 {
		attrs := make([]slog.Attr, 0, len(args)/2)
		for i := 0; i < len(args)-1; i += 2 {
			if key, ok := args[i].(string); ok {
				attrs = append(attrs, slog.Any(key, args[i+1]))
			}
		}
		r.AddAttrs(attrs...)
	}

	_ = l.logger.Handler().Handle(ctx, r)
}

// Ключи для контекста
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	TraceIDKey   contextKey = "trace_id"
	TaskIDKey    contextKey = "task_id"
	WorkerIDKey  contextKey = "worker_id"
)

// extractContextAttributes извлекает атрибуты из контекста
func extractContextAttributes(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr

	if reqID := ctx.Value(RequestIDKey); reqID != nil {
		if reqIDStr, ok := reqID.(string); ok && reqIDStr != "" {
			attrs = append(attrs, slog.String("request_id", reqIDStr))
		}
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if traceIDStr, ok := traceID.(string); ok && traceIDStr != "" {
			attrs = append(attrs, slog.String("trace_id", traceIDStr))
		}
	}

	if taskID := ctx.Value(TaskIDKey); taskID != nil {
		if taskIDStr, ok := taskID.(string); ok && taskIDStr != "" {
			attrs = append(attrs, slog.String("task_id", taskIDStr))
		}
	}

	if workerID := ctx.Value(WorkerIDKey); workerID != nil {
		if workerIDStr, ok := workerID.(string); ok && workerIDStr != "" {
			attrs = append(attrs, slog.String("worker_id", workerIDStr))
		}
	}

	return attrs
}
