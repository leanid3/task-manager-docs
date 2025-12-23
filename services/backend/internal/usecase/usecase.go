package usecase

type UseCases struct {
	TaskLLMUC TaskLLMUC
}

func NewUseCases(taskLLMUC TaskLLMUC) *UseCases {
	return &UseCases{
		TaskLLMUC: taskLLMUC,
	}
}
