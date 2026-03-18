package metrics

// Interface определяет интерфейс для работы с метриками
type Interface interface {
	ObserveTaskProcessingDuration(taskType, status string, duration float64)
	IncTasksTotal(taskType, status string)
	SetTaskQueueSize(taskType string, size float64)

	// Другие методы метрик могут быть добавлены по необходимости
}
