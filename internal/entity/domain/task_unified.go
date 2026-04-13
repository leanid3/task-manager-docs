package domain

import "time"

// TaskType тип задачи
type TaskType string

const (
	TaskTypeLLM        TaskType = "llm"
	TaskTypeParsing    TaskType = "parsing"
	TaskTypeAlgorithms TaskType = "algorithms"
	TaskTypeAnalyze    TaskType = "analyze"
)

// TaskDefinition определяет спецификацию задачи
type TaskDefinition struct {
	Type        TaskType               `json:"type"`
	Name        string                 `json:"name"`
	Topic       string                 `json:"topic"`
	Timeout     time.Duration          `json:"timeout"`
	MaxRetries  int                    `json:"max_retries"`
	StoragePath string                 `json:"storage_path"`
	Enabled     bool                   `json:"enabled"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
