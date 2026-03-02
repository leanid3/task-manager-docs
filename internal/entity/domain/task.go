package domain

// @name TaskStatus
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "PENDING"
	TaskStatusProcessing TaskStatus = "PROCESSING"
	TaskStatusCompleted  TaskStatus = "COMPLETED"
	TaskStatusFailed     TaskStatus = "FAILED"
	TaskStatusCancelled  TaskStatus = "CANCELLED"
)

// type TaskType string

// const (
// 	TaskTypeParsing    TaskType = "PARSING"
// 	TaskTypeAlgorithms TaskType = "ALGORITHMS"
// 	TaskTypeLLM        TaskType = "LLM"
// 	TaskTypeAnalyze    TaskType = "ANALYZE"
// )

// MultiUploadMetadata структура для хранения метаданных задачи многофайловой загрузки
type MultiUploadMetadata struct {
	FileCount int      `json:"file_count"`
	FileKeys  []string `json:"file_keys"`
}

// ToDatabaseCode возвращает строковый код для базы данных.
func (t BaseTask) ToDatabaseCode() string {
	switch t.Status {
	case TaskStatusPending:
		return "PENDING"
	case TaskStatusProcessing:
		return "PROCESSING"
	case TaskStatusCompleted:
		return "COMPLETED"
	case TaskStatusFailed:
		return "FAILED"
	case TaskStatusCancelled:
		return "CANCELLED"
	}
	return ""
}

// ToKafkaCode возвращает числовой код для Kafka.
func (s TaskStatus) ToKafkaCode() int {
	switch s {
	case TaskStatusPending:
		return 1
	case TaskStatusProcessing:
		return 2
	case TaskStatusCompleted:
		return 3
	case TaskStatusFailed:
		return 4
	case TaskStatusCancelled:
		return 5
	default:
		return 4
	}
}

// FromKafkaCode преобразует код обратно в TaskStatus.
func (s TaskStatus) FromKafkaCode(code int) (TaskStatus, bool) {
	switch code {
	case 1:
		return TaskStatusPending, true
	case 2:
		return TaskStatusProcessing, true
	case 3:
		return TaskStatusCompleted, true
	case 4:
		return TaskStatusFailed, true
	case 5:
		return TaskStatusCancelled, true
	default:
		return TaskStatusFailed, false
	}
}
