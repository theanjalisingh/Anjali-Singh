package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// loginCursorName is transaction-scoped; concurrent logins use separate connections.
const loginCursorName = "login_details_cur"

const loginOpType = 1

// Store loads login rows from PostgreSQL. Tests replace this with a fake.
type Store interface {
	GetLoginDetails(ctx context.Context, emailID string) ([]LoginRow, error)
	RecordSuccessfulLogin(ctx context.Context, userID string) error
}

// Repository calls the existing identity.sp_get_login_details procedure.
// It does not create identity tables or duplicate login SQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository returns a PostgreSQL-backed Store.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetLoginDetails opens a transaction, CALLs identity.sp_get_login_details
// (p_optype = 1), and FETCHes all rows from the refcursor.
func (r *Repository) GetLoginDetails(ctx context.Context, emailID string) ([]LoginRow, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin login transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// INOUT refcursor is passed as the cursor name. FETCH must run on the same transaction.
	_, err = tx.Exec(ctx, "CALL identity.sp_get_login_details($1, $2::refcursor, $3)", emailID, loginCursorName, loginOpType)
	if err != nil {
		return nil, fmt.Errorf("call identity.sp_get_login_details: %w", err)
	}

	rows, err := tx.Query(ctx, "FETCH ALL FROM "+loginCursorName)
	if err != nil {
		return nil, fmt.Errorf("fetch login details cursor: %w", err)
	}
	defer rows.Close()

	var result []LoginRow
	for rows.Next() {
		row, mapErr := mapLoginRow(rows)
		if mapErr != nil {
			return nil, mapErr
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read login details rows: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit login transaction: %w", err)
	}

	return result, nil
}

// RecordSuccessfulLogin updates last_login_on on the existing credentials table.
func (r *Repository) RecordSuccessfulLogin(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("parse user_id: %w", err)
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE identity.user_credentials
		SET last_login_on = NOW(),
		    failed_login_count = 0,
		    modified_on = NOW()
		WHERE user_id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("update last_login_on: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("user_credentials row not found")
	}
	return nil
}

func mapLoginRow(rows pgx.Rows) (LoginRow, error) {
	values, err := rows.Values()
	if err != nil {
		return LoginRow{}, fmt.Errorf("read login row values: %w", err)
	}

	fields := make(map[string]any, len(values))
	for i, fd := range rows.FieldDescriptions() {
		if i < len(values) {
			fields[strings.ToLower(string(fd.Name))] = values[i]
		}
	}

	var validTill *time.Time
	if raw, ok := fields["valid_till"]; ok {
		if ts, parsed := asTime(raw); parsed {
			validTill = ts
		}
	}

	return LoginRow{
		UserID:           asString(first(fields, "user_id")),
		UserName:         asString(first(fields, "user_name")),
		EmailID:          asString(first(fields, "email_id")),
		ContactNo:        asString(first(fields, "contact_no")),
		OrganizationID:   asString(first(fields, "organization_id")),
		OrganizationName: asString(first(fields, "organization_name")),
		IsActive:         asBool(first(fields, "is_active")),
		IsApproved:       asBool(first(fields, "is_approved")),
		ValidTill:        validTill,
		PasswordHash:     asString(first(fields, "password_hash")),
		RoleID:           asString(first(fields, "role_id")),
		RoleCode:         asString(first(fields, "role_code")),
		RoleName:         asString(first(fields, "role_name")),
	}, nil
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
