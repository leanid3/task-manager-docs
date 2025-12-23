package v1

import (
	"app/config"
	"app/internal/usecase"
	"app/pkg/logger"

	"github.com/go-playground/validator/v10"
)

type V1 struct {
	uc  usecase.UseCases
	l   logger.Interface
	cfg config.Config
	v   *validator.Validate
}
