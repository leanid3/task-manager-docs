package middleware

import (
	apperrors "app/internal/entity/errors"
	"app/pkg/logger"
	"app/pkg/response"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func buildPanicMessage(c *gin.Context, err interface{}) string {
	return fmt.Sprintf(
		"%s - %s %s PANIC: %v\n%s",
		c.ClientIP(),
		c.Request.Method,
		c.Request.URL.Path,
		err,
		debug.Stack(),
	)
}

func Recovery(l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()

				l.Error("panic recovered",
					"error", err,
					"stack", string(stack),
					"request_id", c.GetString("request_id"),
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError,
					response.Error(apperrors.ErrInternalError, c.GetString("request_id")))
			}
		}()
		c.Next()
	}
}
