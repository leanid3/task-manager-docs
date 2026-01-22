package middleware

import (
	"app/pkg/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(c.Writer.Status())

		// Обновляем метрики
		metrics.ObserveHTTPRequestDuration(c.Request.Method, c.FullPath(), statusCode, duration)
		metrics.IncHTTPRequestsTotal(c.Request.Method, c.FullPath(), statusCode)
	}
}