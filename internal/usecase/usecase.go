package usecase

import (
	multiuploaduc "app/internal/usecase/multiupload"
)

// UseCases контейнер всех usecase приложения
type UseCases struct {
	TaskLLMUC     TaskLLMUCInterface
	MultiUploadUC multiuploaduc.MultiUploadUCInterface
}

// NewUseCases собирает контейнер usecase
func NewUseCases(taskLLMUC TaskLLMUCInterface, multiUploadUC multiuploaduc.MultiUploadUCInterface) *UseCases {
	return &UseCases{
		TaskLLMUC:     taskLLMUC,
		MultiUploadUC: multiUploadUC,
	}
}
