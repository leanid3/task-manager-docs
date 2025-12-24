package mocks

type Logger struct{}

func NewMockLogger() *Logger {
	return &Logger{}
}
func (m *Logger) Debug(msg string, args ...interface{})                   {}
func (m *Logger) Info(msg string, args ...interface{})                    {}
func (m *Logger) Warn(msg string, args ...interface{})                    {}
func (m *Logger) Error(msg string, args ...interface{})                   {}
func (m *Logger) ErrorWithSkip(skip int, msg string, args ...interface{}) {}
func (m *Logger) Fatal(msg string, args ...interface{})                   {}
