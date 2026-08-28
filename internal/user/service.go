package user

import (
	"context"
	"log/slog"
	"strings"

	"futureEnvironsBE/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

// Actor is the authenticated caller performing a user operation.
type Actor struct {
	UserID         string
	OrganizationID string
	Roles          []string
}

// ActorFromClaims builds an Actor from JWT claims.
func ActorFromClaims(claims *auth.Claims) Actor {
	if claims == nil {
		return Actor{}
	}
	return Actor{
		UserID:         claims.UserID,
		OrganizationID: claims.OrganizationID,
		Roles:          append([]string(nil), claims.Roles...),
	}
}

// Service implements user management business rules.
type Service struct {
	store  Store
	logger *slog.Logger
}

// NewService wires the user repository.
func NewService(store Store, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, logger: logger}
}

// List returns users visible to the actor.
func (s *Service) List(ctx context.Context, actor Actor) ([]User, error) {
	if err := requireUserManager(actor); err != nil {
		return nil, err
	}

	users, err := s.store.List(ctx)
	if err != nil {
		s.logger.Error("list users failed", "error", err)
		return nil, err
	}

	if isSuperAdmin(actor) {
		return users, nil
	}

	filtered := make([]User, 0, len(users))
	for _, user := range users {
		if user.OrganizationID == actor.OrganizationID {
			filtered = append(filtered, user)
		}
	}
	return filtered, nil
}

// GetByID returns one user if the actor is allowed to view it.
func (s *Service) GetByID(ctx context.Context, actor Actor, userID string) (*User, error) {
	if err := requireUserManager(actor); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.store.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureActorCanAccessUser(actor, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Create registers a new user with credentials and role assignment.
func (s *Service) Create(ctx context.Context, actor Actor, req CreateUserRequest) (*User, error) {
	if err := requireUserManager(actor); err != nil {
		return nil, err
	}

	organizationID := strings.TrimSpace(req.OrganizationID)
	if !isSuperAdmin(actor) {
		organizationID = actor.OrganizationID
	}
	if err := validateCreateRequest(req, organizationID); err != nil {
		return nil, err
	}

	exists, err := s.store.OrganizationExists(ctx, organizationID)
	if err != nil {
		s.logger.Error("check organization failed", "error", err)
		return nil, err
	}
	if !exists {
		return nil, ErrInvalidOrganization
	}

	role, err := s.store.GetRoleByID(ctx, req.RoleID)
	if err != nil {
		return nil, err
	}
	if err := ensureActorCanAssignRole(actor, role.RoleCode); err != nil {
		return nil, err
	}

	emailID := normalizeEmail(req.EmailID)
	duplicate, err := s.store.EmailExists(ctx, emailID, "")
	if err != nil {
		s.logger.Error("check duplicate email failed", "error", err)
		return nil, err
	}
	if duplicate {
		return nil, ErrDuplicateEmail
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	isApproved := false
	if req.IsApproved != nil {
		isApproved = *req.IsApproved
	}

	userName := strings.TrimSpace(req.UserName)
	contactNo := strings.TrimSpace(req.ContactNo)
	address := strings.TrimSpace(req.Address)
	actorID := actor.UserID

	created, err := s.store.Create(ctx, createUserInput{
		manageUserInput: manageUserInput{
			UserName:       &userName,
			ContactNo:      stringPtr(contactNo),
			EmailID:        &emailID,
			Address:        stringPtr(address),
			OrganizationID: &organizationID,
			ValidTill:      req.ValidTill,
			IsActive:       isActive,
			IsApproved:     isApproved,
			CreatedBy:      &actorID,
			OpType:         opInsertUser,
		},
		PasswordHash: string(hash),
		RoleID:       req.RoleID,
	})
	if err != nil {
		s.logger.Error("create user failed", "error", err)
		return nil, err
	}
	return created, nil
}

// Update changes an existing user profile and optionally the role.
func (s *Service) Update(ctx context.Context, actor Actor, userID string, req UpdateUserRequest) (*User, error) {
	if err := requireUserManager(actor); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidInput
	}

	existing, err := s.store.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureActorCanAccessUser(actor, existing); err != nil {
		return nil, err
	}

	merged := mergeUpdate(existing, req)

	if !isSuperAdmin(actor) {
		merged.OrganizationID = existing.OrganizationID
	} else if req.OrganizationID != nil && strings.TrimSpace(*req.OrganizationID) != "" {
		exists, orgErr := s.store.OrganizationExists(ctx, merged.OrganizationID)
		if orgErr != nil {
			s.logger.Error("check organization failed", "error", orgErr)
			return nil, orgErr
		}
		if !exists {
			return nil, ErrInvalidOrganization
		}
	}

	if req.EmailID != nil {
		duplicate, emailErr := s.store.EmailExists(ctx, merged.EmailID, userID)
		if emailErr != nil {
			s.logger.Error("check duplicate email failed", "error", emailErr)
			return nil, emailErr
		}
		if duplicate {
			return nil, ErrDuplicateEmail
		}
	}

	if req.RoleID != nil && strings.TrimSpace(*req.RoleID) != "" {
		role, roleErr := s.store.GetRoleByID(ctx, *req.RoleID)
		if roleErr != nil {
			return nil, roleErr
		}
		if err := ensureActorCanAssignRole(actor, role.RoleCode); err != nil {
			return nil, err
		}
		if err := s.store.SetUserRole(ctx, userID, *req.RoleID, actor.UserID); err != nil {
			s.logger.Error("set user role failed", "error", err)
			return nil, err
		}
	}

	userName := merged.UserName
	contactNo := merged.ContactNo
	emailID := merged.EmailID
	address := merged.Address
	organizationID := merged.OrganizationID
	actorID := actor.UserID

	updated, err := s.store.Update(ctx, manageUserInput{
		UserID:         &userID,
		UserName:       &userName,
		ContactNo:      stringPtr(contactNo),
		EmailID:        &emailID,
		Address:        stringPtr(address),
		OrganizationID: &organizationID,
		ValidTill:      merged.ValidTill,
		IsActive:       merged.IsActive,
		IsApproved:     merged.IsApproved,
		CreatedBy:      &actorID,
		OpType:         opUpdateUser,
	})
	if err != nil {
		s.logger.Error("update user failed", "error", err)
		return nil, err
	}
	return updated, nil
}

// Delete soft-deletes a user.
func (s *Service) Delete(ctx context.Context, actor Actor, userID string) (*DeleteUserResponse, error) {
	if err := requireUserManager(actor); err != nil {
		return nil, err
	}
	if strings.TrimSpace(userID) == "" {
		return nil, ErrInvalidInput
	}
	if actor.UserID == userID {
		return nil, ErrForbidden
	}

	existing, err := s.store.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if err := ensureActorCanAccessUser(actor, existing); err != nil {
		return nil, err
	}

	result, err := s.store.Delete(ctx, userID, actor.UserID)
	if err != nil {
		s.logger.Error("delete user failed", "error", err)
		return nil, err
	}
	return result, nil
}

func validateCreateRequest(req CreateUserRequest, organizationID string) error {
	if strings.TrimSpace(req.UserName) == "" ||
		strings.TrimSpace(req.EmailID) == "" ||
		strings.TrimSpace(req.Password) == "" ||
		strings.TrimSpace(req.RoleID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(organizationID) == "" {
		return ErrInvalidInput
	}
	if len(req.Password) < 8 {
		return ErrInvalidInput
	}
	return nil
}

func mergeUpdate(existing *User, req UpdateUserRequest) User {
	merged := *existing
	if req.UserName != nil {
		merged.UserName = strings.TrimSpace(*req.UserName)
	}
	if req.ContactNo != nil {
		merged.ContactNo = strings.TrimSpace(*req.ContactNo)
	}
	if req.EmailID != nil {
		merged.EmailID = normalizeEmail(*req.EmailID)
	}
	if req.Address != nil {
		merged.Address = strings.TrimSpace(*req.Address)
	}
	if req.OrganizationID != nil {
		merged.OrganizationID = strings.TrimSpace(*req.OrganizationID)
	}
	if req.ValidTill != nil {
		merged.ValidTill = req.ValidTill
	}
	if req.IsActive != nil {
		merged.IsActive = *req.IsActive
	}
	if req.IsApproved != nil {
		merged.IsApproved = *req.IsApproved
	}
	return merged
}

func requireUserManager(actor Actor) error {
	if isSuperAdmin(actor) || isOrgAdmin(actor) {
		return nil
	}
	return ErrForbidden
}

func ensureActorCanAccessUser(actor Actor, user *User) error {
	if isSuperAdmin(actor) {
		return nil
	}
	if user.OrganizationID != actor.OrganizationID {
		return ErrCannotAccessUser
	}
	return nil
}

func ensureActorCanAssignRole(actor Actor, roleCode string) error {
	roleCode = strings.ToUpper(strings.TrimSpace(roleCode))
	if isSuperAdmin(actor) {
		return nil
	}
	if roleCode == roleSuperAdmin {
		return ErrCannotAssignRole
	}
	if isOrgAdmin(actor) {
		return nil
	}
	return ErrForbidden
}

func isSuperAdmin(actor Actor) bool {
	return hasRole(actor.Roles, roleSuperAdmin)
}

func isOrgAdmin(actor Actor) bool {
	return hasRole(actor.Roles, roleOrgAdmin)
}

func hasRole(roles []string, target string) bool {
	for _, role := range roles {
		if strings.EqualFold(role, target) {
			return true
		}
	}
	return false
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	v := strings.TrimSpace(value)
	return &v
}
