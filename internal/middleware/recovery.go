package middleware

import (
	"log/slog"
	"net/http"

	"futureEnvironsBE/internal/response"

	"github.com/gin-gonic/gin"
)

// Recovery catches panics from handlers, logs them, and returns a safe JSON error.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID, _ := c.Get(RequestIDContextKey)

				logger.Error("panic recovered",
					"error", recovered,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
					"request_id", requestID,
				)

				response.Fail(
					c,
					http.StatusInternalServerError,
					response.CodeInternal,
					"an unexpected error occurred",
				)
			}
		}()

		c.Next()
	}
}
