package handlers

import (
	"app/config"
	"app/internal/handlers/restapi/middleware"
	v1 "app/internal/handlers/restapi/v1"
	"app/internal/usecase"
	"app/pkg/logger"

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

	// Health check
	// health := engine.Group("/health")
	// {
	// 	health.GET("/live", func(c *gin.Context) {
	// 		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	// 	})
	// 	health.GET("/ready", func(c *gin.Context) {
	// 		// check database, broker, storage
	// 		//TODO определить response, что это за сущность
	// 		if err := uc.TaskUC.CheckHealth(c.Request.Context()); err != nil {
	// 			c.JSON(http.StatusServiceUnavailable, response.Error(err.Error()))
	// 			return
	// 		}
	// 		c.JSON(http.StatusOK, response.Success("ready"))
	// 	})
	// }

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
