package docs_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"futureEnvironsBE/internal/docs"

	"github.com/gin-gonic/gin"
)

func TestScalarDocs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	docs.Register(r)

	t.Run("scalar ui", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("content-type = %q", ct)
		}
		if body := rec.Body.String(); !strings.Contains(body, "standalone.js") {
			t.Fatal("expected Scalar bootstrap script in HTML")
		}
	})

	t.Run("openapi spec", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/docs/openapi.yaml", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/yaml" {
			t.Fatalf("content-type = %q", ct)
		}
		if body := rec.Body.String(); !strings.Contains(body, "openapi: 3.0.3") {
			t.Fatal("expected embedded OpenAPI document")
		}
	})
}
