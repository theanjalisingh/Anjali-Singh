package user

import (
	"errors"
	"net/http"

	"futureEnvironsBE/internal/middleware"
	"futureEnvironsBE/internal/response"

	"github.com/gin-gonic/gin"
)

// Handler exposes user management HTTP endpoints.
type Handler struct {
	service *Service
}

// NewHandler constructs a user Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts protected user routes on the versioned API group.
func (h *Handler) Register(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	users := rg.Group("/users")
	users.Use(authMiddleware)
	{
		users.GET("", h.List)
		users.GET("/:id", h.GetByID)
		users.POST("", h.Create)
		users.PUT("/:id", h.Update)
		users.DELETE("/:id", h.Delete)
	}
}

// List handles GET /api/v1/users.
func (h *Handler) List(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "missing or invalid authorization header")
		return
	}

	users, err := h.service.List(c.Request.Context(), actor)
	if err != nil {
		writeUserError(c, err)
		return
	}
	response.OK(c, users)
}

// GetByID handles GET /api/v1/users/:id.
func (h *Handler) GetByID(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "missing or invalid authorization header")
		return
	}

	user, err := h.service.GetByID(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		writeUserError(c, err)
		return
	}
	response.OK(c, user)
}

// Create handles POST /api/v1/users.
func (h *Handler) Create(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "missing or invalid authorization header")
		return
	}

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user, err := h.service.Create(c.Request.Context(), actor, req)
	if err != nil {
		writeUserError(c, err)
		return
	}
	response.Created(c, user)
}

// Update handles PUT /api/v1/users/:id.
func (h *Handler) Update(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "missing or invalid authorization header")
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user, err := h.service.Update(c.Request.Context(), actor, c.Param("id"), req)
	if err != nil {
		writeUserError(c, err)
		return
	}
	response.OK(c, user)
}

// Delete handles DELETE /api/v1/users/:id.
func (h *Handler) Delete(c *gin.Context) {
	actor, ok := actorFromContext(c)
	if !ok {
		response.Unauthorized(c, "missing or invalid authorization header")
		return
	}

	result, err := h.service.Delete(c.Request.Context(), actor, c.Param("id"))
	if err != nil {
		writeUserError(c, err)
		return
	}
	response.OK(c, result)
}

func actorFromContext(c *gin.Context) (Actor, bool) {
	claims, ok := middleware.ClaimsFromContext(c)
	if !ok {
		return Actor{}, false
	}
	return ActorFromClaims(claims), true
}

func writeUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrDuplicateEmail):
		response.Fail(c, http.StatusConflict, response.CodeConflict, err.Error())
	case errors.Is(err, ErrUserNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrCannotAssignRole),
		errors.Is(err, ErrCannotAccessUser):
		response.Forbidden(c, err.Error())
	case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrInvalidOrganization):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrUnauthorized):
		response.Unauthorized(c, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "an unexpected error occurred")
	}
}
