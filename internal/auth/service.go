package auth

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Service implements login/logout business rules.
type Service struct {
	store  Store
	tokens *TokenManager
	logger *slog.Logger
}

// NewService wires repository and JWT helpers.
func NewService(store Store, tokens *TokenManager, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{store: store, tokens: tokens, logger: logger}
}

// Login loads identity details via the stored procedure, verifies the password
// hash in Go, and issues a JWT. The password hash is never returned.
func (s *Service) Login(ctx context.Context, emailID, password string) (*LoginResponse, error) {
	emailID = normalizeEmail(emailID)
	if emailID == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	rows, err := s.store.GetLoginDetails(ctx, emailID)
	if err != nil {
		s.logger.Error("get login details failed", "error", err)
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrInvalidCredentials
	}

	primary := rows[0]
	if err := checkAccount(primary); err != nil {
		return nil, err
	}
	if strings.TrimSpace(primary.PasswordHash) == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(primary.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	user := assembleProfile(rows)
	token, err := s.tokens.IssueAccessToken(user)
	if err != nil {
		s.logger.Error("issue access token failed", "error", err)
		return nil, err
	}

	if err := s.store.RecordSuccessfulLogin(ctx, user.UserID); err != nil {
		// Authentication already succeeded; do not fail the login for an audit update.
		s.logger.Error("record successful login failed", "user_id", user.UserID, "error", err)
	}

	return &LoginResponse{
		AccessToken: token,
		TokenType:   tokenTypeBearer,
		ExpiresIn:   int64(s.tokens.TTL().Seconds()),
		User:        user,
	}, nil
}

// Logout validates the access token. Session revocation is deferred until
// identity.refresh_session exists; clients must discard the token locally.
func (s *Service) Logout(_ context.Context, authorizationHeader string) error {
	raw, err := ParseBearer(authorizationHeader)
	if err != nil {
		return err
	}
	if _, err := s.tokens.ParseAccessToken(raw); err != nil {
		return ErrInvalidToken
	}
	return nil
}

func checkAccount(row LoginRow) error {
	if !row.IsActive {
		return ErrAccountInactive
	}
	if !row.IsApproved {
		return ErrAccountNotApproved
	}
	if row.ValidTill != nil && row.ValidTill.Before(time.Now().UTC()) {
		return ErrAccountExpired
	}
	return nil
}

func assembleProfile(rows []LoginRow) UserProfile {
	primary := rows[0]
	roles := make([]Role, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))

	for _, row := range rows {
		if row.RoleID == "" && row.RoleCode == "" {
			continue
		}
		key := row.RoleID + "|" + row.RoleCode
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		roles = append(roles, Role{
			RoleID:   row.RoleID,
			RoleCode: row.RoleCode,
			RoleName: row.RoleName,
		})
	}

	return UserProfile{
		UserID:           primary.UserID,
		UserName:         primary.UserName,
		EmailID:          primary.EmailID,
		ContactNo:        primary.ContactNo,
		OrganizationID:   primary.OrganizationID,
		OrganizationName: primary.OrganizationName,
		Roles:            roles,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
