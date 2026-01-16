package llm

import (
	"app/internal/entity/domain"
	"app/internal/handlers/broker"
	"app/internal/usecase"
	"app/pkg/logger"
	"context"
)

func Register(
	r *broker.Registry,
	v *TaskLLMValidator,
	uc usecase.TaskLLMUCInterface,
	l logger.Interface,
) error {
	useCaseFn := func(ctx context.Context, evt *domain.TaskLLMStatusEvent) error {
		return uc.UpdateTaskStatus(ctx, *evt)
	}

	route := broker.NewRoute("tasks_llm", v.Validate, useCaseFn, l)
	return r.Register(route)
}
