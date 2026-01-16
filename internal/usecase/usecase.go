package usecase

import "app/internal/entity/domain"

type UseCases struct {
	TaskLLMUC     TaskLLMUCInterface
	UnifiedTaskUC domain.TaskManager
}

func NewUseCases(taskLLMUC TaskLLMUCInterface, unifiedTaskUC domain.TaskManager) *UseCases {
	return &UseCases{
		TaskLLMUC:     taskLLMUC,
		UnifiedTaskUC: unifiedTaskUC,
	}
}
