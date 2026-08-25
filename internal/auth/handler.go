package auth

import (
	"errors"
	"net/http"
	"strings"

	"futureEnvironsBE/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler exposes authentication HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler constructs an auth Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts auth routes on the versioned API group.
func (h *Handler) Register(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
	}
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "email_id and password are required")
		return
	}
	if strings.TrimSpace(req.EmailID) == "" || req.Password == "" {
		response.BadRequest(c, "email_id and password are required")
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.EmailID, req.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, result)
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(c *gin.Context) {
	if err := h.service.Logout(c.Request.Context(), c.GetHeader("Authorization")); err != nil {
		writeAuthError(c, err)
		return
	}

	response.OK(c, LogoutResponse{Message: "logged out"})
}

func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, ErrMissingToken), errors.Is(err, ErrInvalidToken):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, ErrAccountInactive),
		errors.Is(err, ErrAccountNotApproved),
		errors.Is(err, ErrAccountExpired):
		response.Forbidden(c, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "an unexpected error occurred")
	}
}
