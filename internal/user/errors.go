package user

import "errors"

var (
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("user with this email already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidRole        = errors.New("invalid role")
	ErrInvalidOrganization = errors.New("invalid organization")
	ErrCannotAssignRole   = errors.New("cannot assign this role")
	ErrCannotAccessUser   = errors.New("cannot access user outside your organization")
)
