package auth

import "errors"

var (
	// ErrInvalidCredentials is returned for unknown emails or bad passwords.
	// The message is intentionally generic so callers cannot enumerate accounts.
	ErrInvalidCredentials = errors.New("invalid email or password")

	// ErrAccountInactive is returned when identity.users.is_active is false.
	ErrAccountInactive = errors.New("account is inactive")

	// ErrAccountNotApproved is returned when identity.users.is_approved is false.
	ErrAccountNotApproved = errors.New("account is not approved")

	// ErrAccountExpired is returned when identity.users.valid_till is in the past.
	ErrAccountExpired = errors.New("account has expired")

	// ErrMissingToken is returned when logout is called without a Bearer token.
	ErrMissingToken = errors.New("missing or invalid authorization header")

	// ErrInvalidToken is returned when a JWT cannot be parsed or has expired.
	ErrInvalidToken = errors.New("invalid or expired token")
)
