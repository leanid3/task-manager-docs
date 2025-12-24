package middleware

import (
	"time"

	"app/pkg/logger"

	"github.com/gin-gonic/gin"
)

func Logger(l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		args := []interface{}{
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"path", path,
			"status", statusCode,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if query != "" {
			args = append(args, "query", query)
		}

		if statusCode < 400 {
			l.Info("request completed", args...)
		}
	}
}
