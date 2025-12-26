package handlers

import (
	"app/config"
	"app/internal/handlers/restapi/middleware"
	v1 "app/internal/handlers/restapi/v1"
	"app/internal/usecase"
	"app/pkg/logger"
	"app/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "app/docs" // swagger docs
)

// @title Document Processing API Gateway
// @version 1.0
// @description ML-based document processing system
// @host localhost:8080
// @BasePath /api/v1
func NewRoutes(engine *gin.Engine, cfg *config.Config, uc usecase.UseCases, l logger.Interface) {
	engine.Use(middleware.RequestID())
	engine.Use(middleware.Logger(l))
	engine.Use(middleware.Recovery(l))
	// engine.Use(middelware.CORS())

	// @Summary Live check
	// @Description Live check
	// @Tags health
	// @Produce json
	// @Success 200 {object} response.SuccessResponse "Live"
	// @Router /api/v1/health/live [get]
	health := engine.Group("/health")
	{
		health.GET("/live", func(c *gin.Context) {
			c.JSON(http.StatusOK, response.Success(map[string]string{"status": "live"}, c.GetString("request_id")))
		})
		health.GET("/ready", func(c *gin.Context) {
			c.JSON(http.StatusOK, response.Success(map[string]string{"status": "ready"}, c.GetString("request_id")))
		})
	}

	//Metrics(recommended by Prometheus)
	// if cfg.Metrics.Enabled {
	// 	//TODO реализовать promhttp
	// 	engine.GET(cfg.Metrics.Path, gin.WrapH(promhttp.Handler()))
	// }

	//Swagger documentation
	if cfg.Swagger.Enabled {
		engine.GET(cfg.Swagger.Path+"/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	//API v1
	apiV1Group := engine.Group("/api/v1")
	{
		v1.NewV1Routes(apiV1Group, uc, cfg, l)
	}
}
