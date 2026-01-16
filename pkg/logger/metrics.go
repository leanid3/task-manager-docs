package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MetricType тип метрики
type MetricType string

const (
	RequestMetric    MetricType = "request"
	TaskMetric       MetricType = "task"
	SystemMetric     MetricType = "system"
	ErrorMetric      MetricType = "error"
	PerformanceMetric MetricType = "performance"
)

// Metric представляет собой структурированную метрику
type Metric struct {
	Timestamp   time.Time    `json:"timestamp"`
	Type        MetricType   `json:"type"`
	Name        string       `json:"name"`
	Value       float64      `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Description string       `json:"description,omitempty"`
	Unit        string       `json:"unit,omitempty"`
}

// MetricsLogger предоставляет интерфейс для логирования метрик
type MetricsLogger struct {
	logger Interface
}

// NewMetricsLogger создает новый логгер метрик
func NewMetricsLogger(logger Interface) *MetricsLogger {
	return &MetricsLogger{
		logger: logger,
	}
}

// LogRequest логирует метрики запроса
func (ml *MetricsLogger) LogRequest(ctx context.Context, name string, duration time.Duration, statusCode int, labels map[string]string) {
	value := float64(duration.Milliseconds())
	labels["status_code"] = fmt.Sprintf("%d", statusCode)
	
	metric := Metric{
		Timestamp: time.Now(),
		Type:      RequestMetric,
		Name:      name,
		Value:     value,
		Labels:    labels,
		Unit:      "milliseconds",
		Description: "Request processing time",
	}
	
	ml.logMetric(ctx, metric)
}

// LogTask логирует метрики задачи
func (ml *MetricsLogger) LogTask(ctx context.Context, taskID, taskType string, duration time.Duration, status string, labels map[string]string) {
	value := float64(duration.Milliseconds())
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["task_id"] = taskID
	labels["task_type"] = taskType
	labels["status"] = status
	
	metric := Metric{
		Timestamp: time.Now(),
		Type:      TaskMetric,
		Name:      "task_processing_time",
		Value:     value,
		Labels:    labels,
		Unit:      "milliseconds",
		Description: "Task processing time",
	}
	
	ml.logMetric(ctx, metric)
}

// LogSystem логирует системные метрики
func (ml *MetricsLogger) LogSystem(ctx context.Context, name string, value float64, unit string, labels map[string]string) {
	metric := Metric{
		Timestamp: time.Now(),
		Type:      SystemMetric,
		Name:      name,
		Value:     value,
		Labels:    labels,
		Unit:      unit,
		Description: "System metric",
	}
	
	ml.logMetric(ctx, metric)
}

// LogError логирует метрики ошибок
func (ml *MetricsLogger) LogError(ctx context.Context, errorType, errorMessage string, count int, labels map[string]string) {
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["error_type"] = errorType
	labels["error_message"] = errorMessage
	
	metric := Metric{
		Timestamp: time.Now(),
		Type:      ErrorMetric,
		Name:      "error_count",
		Value:     float64(count),
		Labels:    labels,
		Unit:      "count",
		Description: "Error occurrence count",
	}
	
	ml.logMetric(ctx, metric)
}

// LogPerformance логирует метрики производительности
func (ml *MetricsLogger) LogPerformance(ctx context.Context, name string, value float64, unit string, labels map[string]string) {
	metric := Metric{
		Timestamp: time.Now(),
		Type:      PerformanceMetric,
		Name:      name,
		Value:     value,
		Labels:    labels,
		Unit:      unit,
		Description: "Performance metric",
	}
	
	ml.logMetric(ctx, metric)
}

// logMetric внутренний метод для логирования метрики
func (ml *MetricsLogger) logMetric(ctx context.Context, metric Metric) {
	// Преобразуем метрику в JSON для структурированного логирования
	jsonData, err := json.Marshal(metric)
	if err != nil {
		// Если не удалось сериализовать метрику, логируем ошибку
		ml.logger.ErrorCtx(ctx, "failed to serialize metric", "error", err, "metric_name", metric.Name)
		return
	}
	
	// Логируем метрику как структурированное сообщение
	ml.logger.InfoCtx(ctx, "metric", "data", string(jsonData))
}