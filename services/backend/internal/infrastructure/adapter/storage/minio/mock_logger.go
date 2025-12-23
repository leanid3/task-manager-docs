package minio

import "github.com/stretchr/testify/mock"

// MockLogger mock реализация Logger
type MockLogger struct {
	mock.Mock
}

// NewMockLogger создает новый mock logger
func NewMockLogger() *MockLogger {
	return &MockLogger{}
}

func (m *MockLogger) Debug(msg string, args ...interface{}) {
	// Для интеграционных тестов: если нет настроенных ожиданий, игнорируем вызов
	if len(m.ExpectedCalls) > 0 {
		m.Called(append([]interface{}{msg}, args...)...)
	}
}

func (m *MockLogger) Info(msg string, args ...interface{}) {
	// Для интеграционных тестов: если нет настроенных ожиданий, игнорируем вызов
	if len(m.ExpectedCalls) > 0 {
		m.Called(append([]interface{}{msg}, args...)...)
	}
}

func (m *MockLogger) Warn(msg string, args ...interface{}) {
	// Для интеграционных тестов: если нет настроенных ожиданий, игнорируем вызов
	if len(m.ExpectedCalls) > 0 {
		m.Called(append([]interface{}{msg}, args...)...)
	}
}

func (m *MockLogger) Error(msg string, args ...interface{}) {
	// Для интеграционных тестов: если нет настроенных ожиданий, игнорируем вызов
	if len(m.ExpectedCalls) > 0 {
		m.Called(append([]interface{}{msg}, args...)...)
	}
}

func (m *MockLogger) ErrorWithSkip(skip int, msg string, args ...interface{}) {
	// Для интеграционных тестов: если нет настроенных ожиданий, игнорируем вызов
	if len(m.ExpectedCalls) > 0 {
		m.Called(append([]interface{}{skip, msg}, args...)...)
	}
}

func (m *MockLogger) Fatal(msg string, args ...interface{}) {
	// Для интеграционных тестов: если нет настроенных ожиданий, игнорируем вызов
	if len(m.ExpectedCalls) > 0 {
		m.Called(append([]interface{}{msg}, args...)...)
	}
}

// Вспомогательные методы для тестов
func (m *MockLogger) ExpectDebug() *mock.Call {
	return m.On("Debug", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func (m *MockLogger) ExpectError() *mock.Call {
	return m.On("Error", mock.Anything, mock.Anything, mock.Anything)
}
