package usecase

import (
	multiuploaduc "app/internal/usecase/multiupload"
)

// UseCases контейнер всех usecase приложения
type UseCases struct {
	TaskLLMUC     TaskLLMUCInterface
	UnifiedTaskUC TaskManager
	MultiUploadUC multiuploaduc.MultiUploadUCInterface
}

// NewUseCases собирает контейнер usecase
func NewUseCases(taskLLMUC TaskLLMUCInterface, unifiedTaskUC TaskManager, multiUploadUC multiuploaduc.MultiUploadUCInterface) *UseCases {
	return &UseCases{
		TaskLLMUC:     taskLLMUC,
		UnifiedTaskUC: unifiedTaskUC,
		MultiUploadUC: multiUploadUC,
	}
}
