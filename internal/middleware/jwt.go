package middleware

import (
	"net/http"

	"futureEnvironsBE/internal/auth"
	"futureEnvironsBE/internal/response"

	"github.com/gin-gonic/gin"
)

const claimsContextKey = "auth_claims"

// RequireAuth validates a Bearer JWT and stores claims on the Gin context.
func RequireAuth(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := auth.ParseBearer(c.GetHeader("Authorization"))
		if err != nil {
			writeAuthError(c, err)
			return
		}

		claims, err := tokens.ParseAccessToken(raw)
		if err != nil {
			writeAuthError(c, err)
			return
		}

		c.Set(claimsContextKey, claims)
		c.Next()
	}
}

// ClaimsFromContext returns JWT claims set by RequireAuth.
func ClaimsFromContext(c *gin.Context) (*auth.Claims, bool) {
	value, ok := c.Get(claimsContextKey)
	if !ok {
		return nil, false
	}
	claims, ok := value.(*auth.Claims)
	return claims, ok
}

func writeAuthError(c *gin.Context, err error) {
	switch err {
	case auth.ErrMissingToken, auth.ErrInvalidToken:
		response.Unauthorized(c, err.Error())
	default:
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "an unexpected error occurred")
	}
	c.Abort()
}
