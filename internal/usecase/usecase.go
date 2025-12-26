package usecase

type UseCases struct {
	TaskLLMUC TaskLLMUCInterface
}

func NewUseCases(taskLLMUC TaskLLMUCInterface) *UseCases {
	return &UseCases{
		TaskLLMUC: taskLLMUC,
	}
}
