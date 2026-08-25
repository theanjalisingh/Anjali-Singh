package router_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"futureEnvironsBE/internal/auth"
	"futureEnvironsBE/internal/config"
	"futureEnvironsBE/internal/router"

	"golang.org/x/crypto/bcrypt"
)

func TestHealthEndpoints(t *testing.T) {
	cfg := &config.Config{
		AppName: "futureEnvironsBE",
		AppEnv:  "test",
		Port:    "8080",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := router.New(cfg, logger, nil)

	paths := []string{"/health", "/api/v1/health"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			engine.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}

			if got := rec.Header().Get("X-Request-ID"); got == "" {
				t.Fatal("expected X-Request-ID header to be set")
			}

			var body map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}

			success, _ := body["success"].(bool)
			if !success {
				t.Fatalf("success = %#v, want true; body=%s", body["success"], rec.Body.String())
			}

			data, ok := body["data"].(map[string]interface{})
			if !ok {
				t.Fatalf("data missing or wrong type: %#v", body["data"])
			}
			if data["status"] != "ok" {
				t.Fatalf("status = %#v, want ok", data["status"])
			}
		})
	}
}

func TestUnknownRouteReturnsEnvelope(t *testing.T) {
	cfg := &config.Config{
		AppName: "futureEnvironsBE",
		AppEnv:  "test",
		Port:    "8080",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := router.New(cfg, logger, nil)

	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["success"] != false {
		t.Fatalf("success = %#v, want false", body["success"])
	}
}

type fakeLoginStore struct {
	rows []auth.LoginRow
}

func (f *fakeLoginStore) GetLoginDetails(context.Context, string) ([]auth.LoginRow, error) {
	return f.rows, nil
}

func (f *fakeLoginStore) RecordSuccessfulLogin(context.Context, string) error {
	return nil
}

func TestLoginAndLogoutHTTP(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Admin@123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	tokens, err := auth.NewTokenManager("unit-test-secret-unit-test-secret", "futureEnvironsBE", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	store := &fakeLoginStore{rows: []auth.LoginRow{{
		UserID:           "11111111-1111-1111-1111-111111111111",
		UserName:         "System Administrator",
		EmailID:          "admin@futureenvirons.com",
		OrganizationID:   "22222222-2222-2222-2222-222222222222",
		OrganizationName: "Future Environs",
		IsActive:         true,
		IsApproved:       true,
		PasswordHash:     string(hash),
		RoleID:           "33333333-3333-3333-3333-333333333333",
		RoleCode:         "SUPER_ADMIN",
		RoleName:         "Super Admin",
	}}}
	handler := auth.NewHandler(auth.NewService(store, tokens, logger))
	cfg := &config.Config{AppName: "futureEnvironsBE", AppEnv: "test", Port: "8080"}
	engine := router.New(cfg, logger, handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email_id":"admin@futureenvirons.com","password":"Admin@123"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d body=%s", rec.Code, rec.Body.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Success || envelope.Data.AccessToken == "" {
		t.Fatalf("login body = %s", rec.Body.String())
	}

	bad := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email_id":"admin@futureenvirons.com","password":"nope"}`))
	bad.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	engine.ServeHTTP(badRec, bad)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("bad password status = %d", badRec.Code)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+envelope.Data.AccessToken)
	logoutRec := httptest.NewRecorder()
	engine.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d body=%s", logoutRec.Code, logoutRec.Body.String())
	}
}
