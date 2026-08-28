package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newManageUserCursor() string {
	return "manage_user_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// Store loads and mutates users in PostgreSQL.
type Store interface {
	GetByID(ctx context.Context, userID string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Create(ctx context.Context, input createUserInput) (*User, error)
	Update(ctx context.Context, input manageUserInput) (*User, error)
	Delete(ctx context.Context, userID, deletedBy string) (*DeleteUserResponse, error)
	GetRoleByID(ctx context.Context, roleID string) (*roleRecord, error)
	OrganizationExists(ctx context.Context, organizationID string) (bool, error)
	EmailExists(ctx context.Context, emailID string, excludeUserID string) (bool, error)
	SetUserRole(ctx context.Context, userID, roleID, actorID string) error
}

// Repository calls identity.sp_manage_user and related identity tables.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed Store.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetByID loads one user via sp_manage_user (optype 1).
func (r *Repository) GetByID(ctx context.Context, userID string) (*User, error) {
	users, err := r.callManageUser(ctx, manageUserInput{
		UserID: &userID,
		OpType: opGetUserByID,
	})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	user := users[0]
	roles, err := r.loadRoles(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return &user, nil
}

// List loads all active users via sp_manage_user (optype 5).
func (r *Repository) List(ctx context.Context) ([]User, error) {
	users, err := r.callManageUser(ctx, manageUserInput{OpType: opGetAllUsers})
	if err != nil {
		return nil, err
	}

	for i := range users {
		roles, err := r.loadRoles(ctx, users[i].UserID)
		if err != nil {
			return nil, err
		}
		users[i].Roles = roles
	}
	return users, nil
}

// Create inserts users, credentials, and role assignment in one transaction.
func (r *Repository) Create(ctx context.Context, input createUserInput) (*User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create user transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	users, err := r.callManageUserTx(ctx, tx, input.manageUserInput)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, errors.New("create user returned no rows")
	}
	created := users[0]

	userUUID, err := uuid.Parse(created.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse created user_id: %w", err)
	}
	roleUUID, err := uuid.Parse(input.RoleID)
	if err != nil {
		return nil, fmt.Errorf("parse role_id: %w", err)
	}
	var actorUUID *uuid.UUID
	if input.CreatedBy != nil && strings.TrimSpace(*input.CreatedBy) != "" {
		parsed, parseErr := uuid.Parse(*input.CreatedBy)
		if parseErr != nil {
			return nil, fmt.Errorf("parse created_by: %w", parseErr)
		}
		actorUUID = &parsed
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO identity.user_credentials (user_id, password_hash, created_on)
		VALUES ($1, $2, NOW())
	`, userUUID, input.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("insert user_credentials: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO identity.user_role_assignment (user_id, role_id, created_by, created_on, is_active)
		VALUES ($1, $2, $3, NOW(), TRUE)
	`, userUUID, roleUUID, actorUUID)
	if err != nil {
		return nil, fmt.Errorf("insert user_role_assignment: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create user transaction: %w", err)
	}

	roles, err := r.loadRoles(ctx, created.UserID)
	if err != nil {
		return nil, err
	}
	created.Roles = roles
	return &created, nil
}

// Update updates a user via sp_manage_user (optype 3).
func (r *Repository) Update(ctx context.Context, input manageUserInput) (*User, error) {
	users, err := r.callManageUser(ctx, input)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	user := users[0]
	roles, err := r.loadRoles(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles
	return &user, nil
}

// Delete soft-deletes a user via sp_manage_user (optype 4).
func (r *Repository) Delete(ctx context.Context, userID, deletedBy string) (*DeleteUserResponse, error) {
	users, err := r.callManageUser(ctx, manageUserInput{
		UserID:    &userID,
		CreatedBy: &deletedBy,
		OpType:    opDeleteUser,
	})
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrUserNotFound
	}

	row := users[0]
	return &DeleteUserResponse{
		UserID:    row.UserID,
		UserName:  row.UserName,
		EmailID:   row.EmailID,
		IsActive:  row.IsActive,
		IsDeleted: row.IsDeleted,
	}, nil
}

// GetRoleByID loads an active role by ID.
func (r *Repository) GetRoleByID(ctx context.Context, roleID string) (*roleRecord, error) {
	id, err := uuid.Parse(roleID)
	if err != nil {
		return nil, fmt.Errorf("parse role_id: %w", err)
	}

	var record roleRecord
	err = r.pool.QueryRow(ctx, `
		SELECT role_id, role_code, role_name
		FROM identity.role
		WHERE role_id = $1
		  AND is_active = TRUE
		  AND is_deleted = FALSE
	`, id).Scan(&id, &record.RoleCode, &record.RoleName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidRole
		}
		return nil, fmt.Errorf("get role: %w", err)
	}
	record.RoleID = id.String()
	return &record, nil
}

// OrganizationExists reports whether an organization is active.
func (r *Repository) OrganizationExists(ctx context.Context, organizationID string) (bool, error) {
	id, err := uuid.Parse(organizationID)
	if err != nil {
		return false, fmt.Errorf("parse organization_id: %w", err)
	}

	var exists bool
	err = r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM identity.organization
			WHERE organization_id = $1
			  AND is_active = TRUE
			  AND is_deleted = FALSE
		)
	`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check organization: %w", err)
	}
	return exists, nil
}

// EmailExists checks for another active user with the same email.
func (r *Repository) EmailExists(ctx context.Context, emailID, excludeUserID string) (bool, error) {
	emailID = strings.ToLower(strings.TrimSpace(emailID))
	args := []any{emailID}
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM identity.users
			WHERE LOWER(email_id) = $1
			  AND is_deleted = FALSE
	`
	if strings.TrimSpace(excludeUserID) != "" {
		query += ` AND user_id <> $2`
		id, err := uuid.Parse(excludeUserID)
		if err != nil {
			return false, fmt.Errorf("parse exclude user_id: %w", err)
		}
		args = append(args, id)
	}
	query += `)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email: %w", err)
	}
	return exists, nil
}

// SetUserRole replaces the active role assignment for a user.
func (r *Repository) SetUserRole(ctx context.Context, userID, roleID, actorID string) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("parse user_id: %w", err)
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("parse role_id: %w", err)
	}
	var actorUUID *uuid.UUID
	if strings.TrimSpace(actorID) != "" {
		parsed, parseErr := uuid.Parse(actorID)
		if parseErr != nil {
			return fmt.Errorf("parse actor_id: %w", parseErr)
		}
		actorUUID = &parsed
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin set user role transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		UPDATE identity.user_role_assignment
		SET is_active = FALSE,
		    modified_by = $2,
		    modified_on = NOW()
		WHERE user_id = $1
		  AND is_active = TRUE
	`, userUUID, actorUUID)
	if err != nil {
		return fmt.Errorf("deactivate user roles: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO identity.user_role_assignment (user_id, role_id, created_by, created_on, is_active)
		VALUES ($1, $2, $3, NOW(), TRUE)
		ON CONFLICT (user_id, role_id) DO UPDATE
		SET is_active = TRUE,
		    modified_by = EXCLUDED.created_by,
		    modified_on = NOW()
	`, userUUID, roleUUID, actorUUID)
	if err != nil {
		return fmt.Errorf("assign user role: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) callManageUser(ctx context.Context, input manageUserInput) ([]User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin manage user transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	users, err := r.callManageUserTx(ctx, tx, input)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit manage user transaction: %w", err)
	}
	return users, nil
}

func (r *Repository) callManageUserTx(ctx context.Context, tx pgx.Tx, input manageUserInput) ([]User, error) {
	cursorName := newManageUserCursor()

	_, err := tx.Exec(ctx, `
		CALL identity.sp_manage_user(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::refcursor
		)`,
		nullableUUID(input.UserID),
		nullableString(input.UserName),
		nullableString(input.ContactNo),
		nullableString(input.EmailID),
		nullableString(input.Address),
		nullableUUID(input.OrganizationID),
		input.ValidTill,
		input.IsActive,
		input.IsApproved,
		input.IsDeleted,
		nullableUUID(input.CreatedBy),
		input.OpType,
		cursorName,
	)
	if err != nil {
		if isDuplicateEmailError(err) {
			return nil, ErrDuplicateEmail
		}
		if isUserNotFoundError(err) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("call identity.sp_manage_user: %w", err)
	}

	rows, err := tx.Query(ctx, "FETCH ALL FROM "+cursorName)
	if err != nil {
		return nil, fmt.Errorf("fetch manage user cursor: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		user, mapErr := mapUserRow(rows)
		if mapErr != nil {
			return nil, mapErr
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read manage user rows: %w", err)
	}

	if _, err := tx.Exec(ctx, "CLOSE "+cursorName); err != nil {
		return nil, fmt.Errorf("close manage user cursor: %w", err)
	}
	return users, nil
}

func (r *Repository) loadRoles(ctx context.Context, userID string) ([]Role, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user_id: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT r.role_id, r.role_code, r.role_name
		FROM identity.user_role_assignment ura
		JOIN identity.role r ON r.role_id = ura.role_id
		WHERE ura.user_id = $1
		  AND ura.is_active = TRUE
		  AND r.is_active = TRUE
		  AND r.is_deleted = FALSE
		ORDER BY r.role_code
	`, id)
	if err != nil {
		return nil, fmt.Errorf("load user roles: %w", err)
	}
	defer rows.Close()

	roles := make([]Role, 0)
	for rows.Next() {
		var roleID uuid.UUID
		var role Role
		if err := rows.Scan(&roleID, &role.RoleCode, &role.RoleName); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}
		role.RoleID = roleID.String()
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read user roles: %w", err)
	}
	return roles, nil
}

func mapUserRow(rows pgx.Rows) (User, error) {
	values, err := rows.Values()
	if err != nil {
		return User{}, fmt.Errorf("read user row values: %w", err)
	}

	fields := make(map[string]any, len(values))
	for i, fd := range rows.FieldDescriptions() {
		if i < len(values) {
			fields[strings.ToLower(string(fd.Name))] = values[i]
		}
	}

	var validTill, createdOn, modifiedOn *time.Time
	if raw, ok := fields["valid_till"]; ok {
		if ts, parsed := asTime(raw); parsed {
			validTill = ts
		}
	}
	if raw, ok := fields["created_on"]; ok {
		if ts, parsed := asTime(raw); parsed {
			createdOn = ts
		}
	}
	if raw, ok := fields["modified_on"]; ok {
		if ts, parsed := asTime(raw); parsed {
			modifiedOn = ts
		}
	}

	return User{
		UserID:           asString(first(fields, "user_id")),
		UserName:         asString(first(fields, "user_name")),
		ContactNo:        asString(first(fields, "contact_no")),
		EmailID:          asString(first(fields, "email_id")),
		Address:          asString(first(fields, "address")),
		OrganizationID:   asString(first(fields, "organization_id")),
		OrganizationName: asString(first(fields, "organization_name")),
		ValidTill:        validTill,
		IsActive:         asBool(first(fields, "is_active")),
		IsApproved:       asBool(first(fields, "is_approved")),
		IsDeleted:        asBool(first(fields, "is_deleted")),
		CreatedBy:        asString(first(fields, "created_by")),
		CreatedOn:        createdOn,
		ModifiedBy:       asString(first(fields, "modified_by")),
		ModifiedOn:       modifiedOn,
	}, nil
}

func isDuplicateEmailError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "P0001" && strings.Contains(strings.ToLower(pgErr.Message), "already exists")
	}
	return strings.Contains(strings.ToLower(err.Error()), "already exists")
}

func isUserNotFoundError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "P0001" && strings.Contains(strings.ToLower(pgErr.Message), "user not found")
	}
	return strings.Contains(strings.ToLower(err.Error()), "user not found")
}

func nullableUUID(value *string) any {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	id, err := uuid.Parse(strings.TrimSpace(*value))
	if err != nil {
		return nil
	}
	return id
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func first(fields map[string]any, key string) any {
	return fields[key]
}

func asString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case [16]byte:
		return uuid.UUID(t).String()
	case uuid.UUID:
		return t.String()
	case fmt.Stringer:
		return t.String()
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func asBool(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case int32:
		return t != 0
	case int64:
		return t != 0
	case int:
		return t != 0
	default:
		s := strings.ToLower(strings.TrimSpace(fmt.Sprint(t)))
		return s == "true" || s == "t" || s == "1"
	}
}

func asTime(v any) (*time.Time, bool) {
	switch t := v.(type) {
	case nil:
		return nil, false
	case time.Time:
		tt := t
		return &tt, true
	default:
		return nil, false
	}
}
