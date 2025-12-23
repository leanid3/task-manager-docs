package logger

import (
	"os"
)

// logger/multi_writer.go
type MultiWriterLogger struct {
	loggers map[string]Interface
}

func NewMultiWriterLogger() *MultiWriterLogger {
	return &MultiWriterLogger{
		loggers: make(map[string]Interface),
	}
}

func (mw *MultiWriterLogger) Get(category string) Interface {
	if l, ok := mw.loggers[category]; ok {
		return l
	}
	// Создаем новый логгер с префиксом
	return New(os.Stdout, "info", "json")
}

func (mw *MultiWriterLogger) Add(category string, l Interface) {
	mw.loggers[category] = l
}
