// Package response provides a consistent JSON envelope for API success and error replies.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the standard shape returned by all HTTP handlers.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError describes a client-visible failure.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Common machine-readable error codes.
const (
	CodeBadRequest   = "BAD_REQUEST"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL_ERROR"
)

// JSON writes a successful response with the given HTTP status and payload.
func JSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, Envelope{
		Success: true,
		Data:    data,
	})
}

// OK is a convenience helper for HTTP 200 responses.
func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, data)
}

// Created is a convenience helper for HTTP 201 responses.
func Created(c *gin.Context, data interface{}) {
	JSON(c, http.StatusCreated, data)
}

// Fail writes an error response with an explicit code and message.
func Fail(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, Envelope{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

// BadRequest writes an HTTP 400 error response.
func BadRequest(c *gin.Context, message string) {
	Fail(c, http.StatusBadRequest, CodeBadRequest, message)
}

// Unauthorized writes an HTTP 401 error response.
func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

// Forbidden writes an HTTP 403 error response.
func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, CodeForbidden, message)
}

// NotFound writes an HTTP 404 error response.
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, CodeNotFound, message)
}

// Internal writes an HTTP 500 error response.
func Internal(c *gin.Context, message string) {
	Fail(c, http.StatusInternalServerError, CodeInternal, message)
}
