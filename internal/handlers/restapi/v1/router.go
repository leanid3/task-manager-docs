package v1

import (
	"app/config"
	"app/internal/handlers/restapi/v1/multiupload"
	"app/internal/usecase"
	"app/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// V1 содержит обработчики API v1
type V1 struct {
	taskLLMUC     usecase.TaskLLMUCInterface
	multiUploadUC multiupload.MultiUploadUCInterface
	l             logger.Interface
	cfg           config.Config
	v             *validator.Validate
}

// NewV1Routes настраивает маршруты API v1
func NewV1Routes(group *gin.RouterGroup, uc usecase.UseCases, cfg *config.Config, l logger.Interface) {
	r := &V1{
		taskLLMUC:     uc.TaskLLMUC,
		multiUploadUC: uc.MultiUploadUC,
		l:             l,
		cfg:           *cfg,
		v:             validator.New(validator.WithRequiredStructEnabled()),
	}

	tasks := group.Group("/tasks")
	{
		tasks.POST("/:filename", r.createTask)
		tasks.GET("/:task_id", r.getTaskByID)
	}

	// Многофайловая загрузка
	muConfig := &multiupload.Config{
		Server: struct {
			Timeout int `mapstructure:"timeout"`
		}{
			Timeout: int(cfg.Server.Timeout.Seconds()),
		},
		MaxFileSize:  cfg.Server.MaxFileSize,
		MaxFileCount: cfg.Server.MaxFileCount,
	}

	muHandler := multiupload.New(r.multiUploadUC, muConfig, l)

	multiUploadGroup := group.Group("/multiupload")
	{
		multiUploadGroup.POST("", muHandler.CreateMultiUploadTask)
	}
}
