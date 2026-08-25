package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const tokenTypeBearer = "Bearer"

// Claims are embedded in the access JWT. Roles are role_code values.
type Claims struct {
	UserID         string   `json:"user_id"`
	EmailID        string   `json:"email_id"`
	OrganizationID string   `json:"organization_id"`
	Roles          []string `json:"roles"`
	jwt.RegisteredClaims
}

// TokenManager issues and validates HMAC-signed access tokens.
type TokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewTokenManager constructs a TokenManager. secret must be non-empty.
func NewTokenManager(secret, issuer string, ttl time.Duration) (*TokenManager, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, fmt.Errorf("auth: JWT secret must not be empty")
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("auth: JWT TTL must be greater than zero")
	}
	if strings.TrimSpace(issuer) == "" {
		issuer = "futureEnvironsBE"
	}
	return &TokenManager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
	}, nil
}

// TTL returns the configured access-token lifetime.
func (m *TokenManager) TTL() time.Duration {
	return m.ttl
}

// IssueAccessToken signs a JWT for the given user profile.
func (m *TokenManager) IssueAccessToken(user UserProfile) (string, error) {
	now := time.Now().UTC()
	roleCodes := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		if role.RoleCode != "" {
			roleCodes = append(roleCodes, role.RoleCode)
		}
	}

	claims := Claims{
		UserID:         user.UserID,
		EmailID:        user.EmailID,
		OrganizationID: user.OrganizationID,
		Roles:          roleCodes,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.UserID,
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			NotBefore: jwt.NewNumericDate(now),
			ID:        uuid.NewString(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}
	return signed, nil
}

// ParseAccessToken validates signature, expiry, and issuer.
func (m *TokenManager) ParseAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithLeeway(5*time.Second))
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseBearer extracts the raw token from an Authorization header.
func ParseBearer(header string) (string, error) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", ErrMissingToken
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], tokenTypeBearer) || strings.TrimSpace(parts[1]) == "" {
		return "", ErrMissingToken
	}
	return strings.TrimSpace(parts[1]), nil
}
