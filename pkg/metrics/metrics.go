package metrics

import (
	"runtime"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// TODO возможно вынести в метод
var (
	// Global registry for metrics
	registry = prometheus.NewRegistry()
	autoReg  = promauto.With(registry)
)

// HTTP metrics
var (
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsTotal   *prometheus.CounterVec
)

// Task metrics
var (
	TaskProcessingDuration *prometheus.HistogramVec
	TasksTotal             *prometheus.CounterVec
	TaskQueueSize          *prometheus.GaugeVec
)

// Kafka metrics
var (
	KafkaMessagesConsumed *prometheus.CounterVec
	KafkaMessagesProduced *prometheus.CounterVec
	KafkaConsumerLag      *prometheus.GaugeVec
)

// Database metrics
var (
	DatabaseQueryDuration *prometheus.HistogramVec
	DatabaseConnections   prometheus.Gauge
)

// MinIO metrics
var (
	MinIOOperationDuration *prometheus.HistogramVec
	MinIOBytesTransferred  *prometheus.CounterVec
)

// System metrics
var (
	GoRoutines        prometheus.GaugeFunc
	GoMemoryAllocated prometheus.GaugeFunc
)

// init функция для инициализации метрик
func init() {
	// HTTP metrics
	HTTPRequestDuration = autoReg.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestsTotal = autoReg.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// Task metrics
	TaskProcessingDuration = autoReg.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "task_processing_duration_seconds",
			Help: "Task processing duration in seconds",
		},
		[]string{"task_type", "status"},
	)

	TasksTotal = autoReg.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasks_total",
			Help: "Total number of processed tasks",
		},
		[]string{"task_type", "status"},
	)

	TaskQueueSize = autoReg.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "task_queue_size",
			Help: "Current size of task queue",
		},
		[]string{"task_type"},
	)

	// Kafka metrics
	KafkaMessagesConsumed = autoReg.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total number of consumed Kafka messages",
		},
		[]string{"topic", "partition"},
	)

	KafkaMessagesProduced = autoReg.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total number of produced Kafka messages",
		},
		[]string{"topic"},
	)

	KafkaConsumerLag = autoReg.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Current lag of Kafka consumer",
		},
		[]string{"topic", "partition"},
	)

	// Database metrics
	DatabaseQueryDuration = autoReg.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "database_query_duration_seconds",
			Help: "Database query duration in seconds",
		},
		[]string{"operation", "table"},
	)

	DatabaseConnections = autoReg.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Current number of database connections",
		},
	)

	// MinIO metrics
	MinIOOperationDuration = autoReg.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "minio_operation_duration_seconds",
			Help: "MinIO operation duration in seconds",
		},
		[]string{"operation", "bucket"},
	)

	MinIOBytesTransferred = autoReg.NewCounterVec(
		prometheus.CounterOpts{
			Name: "minio_bytes_transferred_total",
			Help: "Total bytes transferred to/from MinIO",
		},
		[]string{"operation", "bucket"},
	)

	// System metrics
	GoRoutines = autoReg.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "go_goroutines",
			Help: "Number of goroutines that currently exist",
		},
		func() float64 {
			return float64(runtime.NumGoroutine())
		},
	)

	GoMemoryAllocated = autoReg.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "go_memstats_alloc_bytes",
			Help: "Number of bytes allocated and still in use",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.Alloc)
		},
	)
}

// Default глобальный экземпляр интерфейса метрик
var Default Interface = &defaultMetrics{}

// defaultMetrics реализация интерфейса метрик по умолчанию
type defaultMetrics struct{}

func (dm *defaultMetrics) ObserveTaskProcessingDuration(taskType, status string, duration float64) {
	ObserveTaskProcessingDuration(taskType, status, duration)
}

func (dm *defaultMetrics) IncTasksTotal(taskType, status string) {
	IncTasksTotal(taskType, status)
}

func (dm *defaultMetrics) SetTaskQueueSize(taskType string, size float64) {
	SetTaskQueueSize(taskType, size)
}

// GetRegistry возвращает реестр метрик
func GetRegistry() prometheus.Gatherer {
	return registry
}

// InitMetrics инициализирует метрики
func InitMetrics() {
	// Для обратной совместимости
}

// HTTP Metrics Functions
func ObserveHTTPRequestDuration(method, endpoint, status string, duration float64) {
	HTTPRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)
}

func IncHTTPRequestsTotal(method, endpoint, status string) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
}

// Task Metrics Functions
func ObserveTaskProcessingDuration(taskType, status string, duration float64) {
	TaskProcessingDuration.WithLabelValues(taskType, status).Observe(duration)
}

func IncTasksTotal(taskType, status string) {
	TasksTotal.WithLabelValues(taskType, status).Inc()
}

func SetTaskQueueSize(taskType string, size float64) {
	TaskQueueSize.WithLabelValues(taskType).Set(size)
}

// Kafka Metrics Functions
func IncKafkaMessagesConsumed(topic, partition string) {
	KafkaMessagesConsumed.WithLabelValues(topic, partition).Inc()
}

func IncKafkaMessagesProduced(topic string) {
	KafkaMessagesProduced.WithLabelValues(topic).Inc()
}

func SetKafkaConsumerLag(topic, partition string, lag float64) {
	KafkaConsumerLag.WithLabelValues(topic, partition).Set(lag)
}

// Database Metrics Functions
func ObserveDatabaseQueryDuration(operation, table string, duration float64) {
	DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

func SetDatabaseConnections(count float64) {
	DatabaseConnections.Set(count)
}

// MinIO Metrics Functions
func ObserveMinIOOperationDuration(operation, bucket string, duration float64) {
	MinIOOperationDuration.WithLabelValues(operation, bucket).Observe(duration)
}

func IncMinIOBytesTransferred(operation, bucket string, bytes float64) {
	MinIOBytesTransferred.WithLabelValues(operation, bucket).Add(bytes)
}
