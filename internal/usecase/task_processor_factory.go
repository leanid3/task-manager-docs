package usecase

import (
	"app/internal/entity/domain"
	"fmt"
)

// TaskProcessorFactoryImpl реализация фабрики обработчиков задач
type TaskProcessorFactoryImpl struct {
	processors map[domain.TaskType]TaskProcessor
}

// NewTaskProcessorFactory создает новую фабрику обработчиков задач
func NewTaskProcessorFactory() TaskProcessorFactory {
	return &TaskProcessorFactoryImpl{
		processors: make(map[domain.TaskType]TaskProcessor),
	}
}

// Create создает обработчик задачи по типу
func (f *TaskProcessorFactoryImpl) Create(taskType domain.TaskType) (TaskProcessor, error) {
	processor, exists := f.processors[taskType]
	if !exists {
		return nil, fmt.Errorf("processor for task type '%s' not registered", taskType)
	}
	return processor, nil
}

// Register регистрирует обработчик задачи
func (f *TaskProcessorFactoryImpl) Register(taskType domain.TaskType, processor TaskProcessor) error {
	f.processors[taskType] = processor
	return nil
}
