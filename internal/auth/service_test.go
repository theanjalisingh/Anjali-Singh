package auth

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type fakeStore struct {
	rows []LoginRow
	err  error
}

func (f *fakeStore) GetLoginDetails(context.Context, string) ([]LoginRow, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.rows, nil
}

func (f *fakeStore) RecordSuccessfulLogin(context.Context, string) error {
	return nil
}

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func testService(t *testing.T, store Store) *Service {
	t.Helper()
	tokens, err := NewTokenManager("unit-test-secret-unit-test-secret", "futureEnvironsBE", time.Hour)
	if err != nil {
		t.Fatalf("token manager: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewService(store, tokens, logger)
}

func sampleUser(t *testing.T) LoginRow {
	t.Helper()
	return LoginRow{
		UserID:           "11111111-1111-1111-1111-111111111111",
		UserName:         "System Administrator",
		EmailID:          "admin@futureenvirons.com",
		ContactNo:        "000",
		OrganizationID:   "22222222-2222-2222-2222-222222222222",
		OrganizationName: "Future Environs",
		IsActive:         true,
		IsApproved:       true,
		PasswordHash:     mustHash(t, "Admin@123"),
		RoleID:           "33333333-3333-3333-3333-333333333333",
		RoleCode:         "SUPER_ADMIN",
		RoleName:         "Super Admin",
	}
}

func TestLoginSuccessAggregatesRoles(t *testing.T) {
	primary := sampleUser(t)
	second := primary
	second.RoleID = "44444444-4444-4444-4444-444444444444"
	second.RoleCode = "ORG_ADMIN"
	second.RoleName = "Org Admin"

	svc := testService(t, &fakeStore{rows: []LoginRow{primary, second}})
	got, err := svc.Login(context.Background(), "Admin@futureenvirons.com", "Admin@123")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if got.AccessToken == "" {
		t.Fatal("expected access token")
	}
	if got.TokenType != "Bearer" {
		t.Fatalf("token_type = %q", got.TokenType)
	}
	if got.User.UserName != "System Administrator" {
		t.Fatalf("user_name = %q", got.User.UserName)
	}
	if len(got.User.Roles) != 2 {
		t.Fatalf("roles = %#v, want 2", got.User.Roles)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	svc := testService(t, &fakeStore{rows: []LoginRow{sampleUser(t)}})
	_, err := svc.Login(context.Background(), "admin@futureenvirons.com", "wrong")
	if err != ErrInvalidCredentials {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginRejectsUnknownUser(t *testing.T) {
	svc := testService(t, &fakeStore{rows: nil})
	_, err := svc.Login(context.Background(), "nobody@futureenvirons.com", "Admin@123")
	if err != ErrInvalidCredentials {
		t.Fatalf("error = %v, want ErrInvalidCredentials", err)
	}
}

func TestLoginRejectsInactiveAndExpired(t *testing.T) {
	inactive := sampleUser(t)
	inactive.IsActive = false
	svc := testService(t, &fakeStore{rows: []LoginRow{inactive}})
	if _, err := svc.Login(context.Background(), inactive.EmailID, "Admin@123"); err != ErrAccountInactive {
		t.Fatalf("inactive error = %v", err)
	}

	unapproved := sampleUser(t)
	unapproved.IsApproved = false
	svc = testService(t, &fakeStore{rows: []LoginRow{unapproved}})
	if _, err := svc.Login(context.Background(), unapproved.EmailID, "Admin@123"); err != ErrAccountNotApproved {
		t.Fatalf("unapproved error = %v", err)
	}

	expired := sampleUser(t)
	past := time.Now().UTC().Add(-time.Hour)
	expired.ValidTill = &past
	svc = testService(t, &fakeStore{rows: []LoginRow{expired}})
	if _, err := svc.Login(context.Background(), expired.EmailID, "Admin@123"); err != ErrAccountExpired {
		t.Fatalf("expired error = %v", err)
	}
}

func TestLogoutRequiresValidToken(t *testing.T) {
	svc := testService(t, &fakeStore{rows: []LoginRow{sampleUser(t)}})

	if err := svc.Logout(context.Background(), ""); err != ErrMissingToken {
		t.Fatalf("empty header error = %v", err)
	}
	if err := svc.Logout(context.Background(), "Bearer not-a-jwt"); err != ErrInvalidToken {
		t.Fatalf("bad token error = %v", err)
	}

	login, err := svc.Login(context.Background(), "admin@futureenvirons.com", "Admin@123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if err := svc.Logout(context.Background(), "Bearer "+login.AccessToken); err != nil {
		t.Fatalf("logout valid token: %v", err)
	}
}

func TestTokenRoundTrip(t *testing.T) {
	tm, err := NewTokenManager("unit-test-secret-unit-test-secret", "futureEnvironsBE", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	token, err := tm.IssueAccessToken(UserProfile{
		UserID:         "11111111-1111-1111-1111-111111111111",
		EmailID:        "admin@futureenvirons.com",
		OrganizationID: "22222222-2222-2222-2222-222222222222",
		Roles:          []Role{{RoleCode: "SUPER_ADMIN"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := tm.ParseAccessToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.EmailID != "admin@futureenvirons.com" {
		t.Fatalf("email = %q", claims.EmailID)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "SUPER_ADMIN" {
		t.Fatalf("roles = %#v", claims.Roles)
	}
}
