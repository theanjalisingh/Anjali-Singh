package auth

import "time"

// LoginRequest is the JSON body for POST /api/v1/auth/login.
type LoginRequest struct {
	EmailID  string `json:"email_id"`
	Password string `json:"password"`
}

// Role is a single assigned application role.
type Role struct {
	RoleID   string `json:"role_id"`
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
}

// UserProfile is the authenticated user payload returned to the client.
// Password hashes are never included.
type UserProfile struct {
	UserID           string `json:"user_id"`
	UserName         string `json:"user_name"`
	EmailID          string `json:"email_id"`
	ContactNo        string `json:"contact_no,omitempty"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	Roles            []Role `json:"roles"`
}

// LoginResponse is returned after successful authentication.
type LoginResponse struct {
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
	User        UserProfile `json:"user"`
}

// LogoutResponse is returned after a successful logout.
type LogoutResponse struct {
	Message string `json:"message"`
}

// LoginRow is one row from identity.sp_get_login_details.
// A user with multiple roles produces multiple rows that share the same user_id.
type LoginRow struct {
	UserID           string
	UserName         string
	EmailID          string
	ContactNo        string
	OrganizationID   string
	OrganizationName string
	IsActive         bool
	IsApproved       bool
	ValidTill        *time.Time
	PasswordHash     string
	RoleID           string
	RoleCode         string
	RoleName         string
}
