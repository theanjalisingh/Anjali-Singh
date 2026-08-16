// Package router wires HTTP routes and shared middleware.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"futureEnvironsBE/internal/config"
	"futureEnvironsBE/internal/middleware"
	"futureEnvironsBE/internal/response"

	"github.com/gin-gonic/gin"
)

// New builds the Gin engine with foundation middleware and routes.
func New(cfg *config.Config, logger *slog.Logger) *gin.Engine {
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Foundation middleware (order matters).
	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.RequestLogger(logger))

	// Load balancer / ops probe — kept outside /api/v1.
	r.GET("/health", healthHandler(cfg))

	// Versioned API root. Domain modules will register under this group later.
	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", healthHandler(cfg))
	}

	// Consistent JSON 404 for unknown routes.
	r.NoRoute(func(c *gin.Context) {
		response.NotFound(c, "route not found")
	})

	r.NoMethod(func(c *gin.Context) {
		response.Fail(c, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})

	return r
}

func healthHandler(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.OK(c, gin.H{
			"status":    "ok",
			"service":   cfg.AppName,
			"env":       cfg.AppEnv,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}
