package middleware

import (
	"context"
	"time"

	"app/pkg/logger"

	"github.com/gin-gonic/gin"
)

func Logger(l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Создаем контекст с request_id
		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, c.GetString("request_id"))
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		args := []interface{}{
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
			l.InfoCtx(c.Request.Context(), "request completed", args...)
		} else {
			l.WarnCtx(c.Request.Context(), "request completed with warning", args...)
		}
	}
}
