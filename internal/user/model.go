package user

import "time"

const (
	roleSuperAdmin = "SUPER_ADMIN"
	roleOrgAdmin   = "ORG_ADMIN"
)

const (
	opGetUserByID  = 1
	opInsertUser   = 2
	opUpdateUser   = 3
	opDeleteUser   = 4
	opGetAllUsers  = 5
)

// Role is an assigned application role returned with user payloads.
type Role struct {
	RoleID   string `json:"role_id"`
	RoleCode string `json:"role_code"`
	RoleName string `json:"role_name"`
}

// User is the public user profile. Password hashes are never included.
type User struct {
	UserID           string     `json:"user_id"`
	UserName         string     `json:"user_name"`
	ContactNo        string     `json:"contact_no,omitempty"`
	EmailID          string     `json:"email_id"`
	Address          string     `json:"address,omitempty"`
	OrganizationID   string     `json:"organization_id"`
	OrganizationName string     `json:"organization_name,omitempty"`
	ValidTill        *time.Time `json:"valid_till,omitempty"`
	IsActive         bool       `json:"is_active"`
	IsApproved       bool       `json:"is_approved"`
	IsDeleted        bool       `json:"is_deleted,omitempty"`
	CreatedBy        string     `json:"created_by,omitempty"`
	CreatedOn        *time.Time `json:"created_on,omitempty"`
	ModifiedBy       string     `json:"modified_by,omitempty"`
	ModifiedOn       *time.Time `json:"modified_on,omitempty"`
	Roles            []Role     `json:"roles,omitempty"`
}

// CreateUserRequest is the JSON body for POST /api/v1/users.
type CreateUserRequest struct {
	UserName       string     `json:"user_name"`
	ContactNo      string     `json:"contact_no"`
	EmailID        string     `json:"email_id"`
	Address        string     `json:"address"`
	OrganizationID string     `json:"organization_id"`
	ValidTill      *time.Time `json:"valid_till"`
	IsActive       *bool      `json:"is_active"`
	IsApproved     *bool      `json:"is_approved"`
	Password       string     `json:"password"`
	RoleID         string     `json:"role_id"`
}

// UpdateUserRequest is the JSON body for PUT /api/v1/users/:id.
type UpdateUserRequest struct {
	UserName       *string    `json:"user_name"`
	ContactNo      *string    `json:"contact_no"`
	EmailID        *string    `json:"email_id"`
	Address        *string    `json:"address"`
	OrganizationID *string    `json:"organization_id"`
	ValidTill      *time.Time `json:"valid_till"`
	IsActive       *bool      `json:"is_active"`
	IsApproved     *bool      `json:"is_approved"`
	RoleID         *string    `json:"role_id"`
}

// DeleteUserResponse is returned after a soft delete.
type DeleteUserResponse struct {
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	EmailID   string `json:"email_id"`
	IsActive  bool   `json:"is_active"`
	IsDeleted bool   `json:"is_deleted"`
}

// manageUserInput is passed to identity.sp_manage_user.
type manageUserInput struct {
	UserID         *string
	UserName       *string
	ContactNo      *string
	EmailID        *string
	Address        *string
	OrganizationID *string
	ValidTill      *time.Time
	IsActive       bool
	IsApproved     bool
	IsDeleted      bool
	CreatedBy      *string
	OpType         int
}

// createUserInput is used by the repository transaction for user creation.
type createUserInput struct {
	manageUserInput
	PasswordHash string
	RoleID       string
}

// roleRecord is a row from identity.role.
type roleRecord struct {
	RoleID   string
	RoleCode string
	RoleName string
}
