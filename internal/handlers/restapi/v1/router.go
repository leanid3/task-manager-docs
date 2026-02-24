package v1

import (
	"app/config"
	handlermultiupload "app/internal/handlers/restapi/v1/multiupload"
	"app/internal/usecase"
	"app/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func NewV1Routes(group *gin.RouterGroup, uc usecase.UseCases, cfg *config.Config, l logger.Interface) {
	r := &V1{
		uc:  uc,
		l:   l,
		cfg: *cfg,
		v:   validator.New(validator.WithRequiredStructEnabled()),
	}

	tasks := group.Group("/tasks")
	{
		tasks.POST("/:filename", r.createTask)
		tasks.GET("/:task_id", r.getTaskByID)
	}

	// Многофайловая загрузка
	multiUploadConfig := &handlermultiupload.Config{
		Server: struct {
			Timeout int `mapstructure:"timeout"`
		}{
			Timeout: int(cfg.Server.Timeout.Seconds()), // преобразуем время в секунды
		},
		MaxFileSize:  cfg.Server.MaxFileSize,
		MaxFileCount: cfg.Server.MaxFileCount,
	}

	multiUploadHandler := handlermultiupload.New(
		uc.MultiUploadUC,
		multiUploadConfig,
		l,
	)

	multiUploadGroup := group.Group("/multiupload")
	{
		multiUploadGroup.POST("", multiUploadHandler.CreateMultiUploadTask)
	}
}
