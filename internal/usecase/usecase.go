package usecase

import (
	"app/internal/entity/domain"
	multiuploaduc "app/internal/usecase/multiupload"
)

type UseCases struct {
	TaskLLMUC     TaskLLMUCInterface
	UnifiedTaskUC domain.TaskManager
	MultiUploadUC multiuploaduc.MultiUploadUCInterface
}

func NewUseCases(taskLLMUC TaskLLMUCInterface, unifiedTaskUC domain.TaskManager, multiUploadUC multiuploaduc.MultiUploadUCInterface) *UseCases {
	return &UseCases{
		TaskLLMUC:     taskLLMUC,
		UnifiedTaskUC: unifiedTaskUC,
		MultiUploadUC: multiUploadUC,
	}
}
