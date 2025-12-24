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
}

type Logger struct {
	logger *slog.Logger
}

var _ Interface = (*Logger)(nil)

func New(w io.Writer, level string, format string) *Logger {
	var l slog.Level

	//Оределяем и устанваливаем уровень логирования
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
