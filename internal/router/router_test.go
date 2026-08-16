package router_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"futureEnvironsBE/internal/config"
	"futureEnvironsBE/internal/router"
)

func TestHealthEndpoints(t *testing.T) {
	cfg := &config.Config{
		AppName: "futureEnvironsBE",
		AppEnv:  "test",
		Port:    "8080",
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	engine := router.New(cfg, logger)

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
	engine := router.New(cfg, logger)

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
