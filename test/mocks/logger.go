package mocks

import "context"

type Logger struct{}

func NewMockLogger() *Logger {
	return &Logger{}
}
func (m *Logger) Debug(msg string, args ...interface{})                         {}
func (m *Logger) Info(msg string, args ...interface{})                          {}
func (m *Logger) Warn(msg string, args ...interface{})                          {}
func (m *Logger) Error(msg string, args ...interface{})                         {}
func (m *Logger) ErrorWithSkip(skip int, msg string, args ...interface{})       {}
func (m *Logger) Fatal(msg string, args ...interface{})                         {}
func (m *Logger) InfoCtx(ctx context.Context, msg string, args ...interface{})  {}
func (m *Logger) DebugCtx(ctx context.Context, msg string, args ...interface{}) {}
func (m *Logger) WarnCtx(ctx context.Context, msg string, args ...interface{})  {}
func (m *Logger) ErrorCtx(ctx context.Context, msg string, args ...interface{}) {}
