package security

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestWrapSecureHandlerAuthAndHeaders(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	mux.HandleFunc("/v2/memories/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	handler := WrapSecureHandler(mux)

	_ = os.Setenv("RAE_REQUIRE_AUTH", "true")
	_ = os.Setenv("RAE_API_KEY", "test-cluster-token-2026")
	defer func() {
		_ = os.Unsetenv("RAE_REQUIRE_AUTH")
		_ = os.Unsetenv("RAE_API_KEY")
	}()

	// 1. Health endpoint remains accessible without auth and includes security headers
	reqHealth := httptest.NewRequest(http.MethodGet, "/health", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected /health 200, got %d", recHealth.Code)
	}
	if recHealth.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options header")
	}

	// 2. Protected endpoint without token returns 401
	reqUnauth := httptest.NewRequest(http.MethodPost, "/v2/memories/", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", recUnauth.Code)
	}

	// 3. Protected endpoint with X-API-Key succeeds
	reqKey := httptest.NewRequest(http.MethodPost, "/v2/memories/", nil)
	reqKey.Header.Set("X-API-Key", "test-cluster-token-2026")
	recKey := httptest.NewRecorder()
	handler.ServeHTTP(recKey, reqKey)
	if recKey.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid X-API-Key, got %d", recKey.Code)
	}

	// 4. Protected endpoint with Authorization: Bearer succeeds
	reqBearer := httptest.NewRequest(http.MethodPost, "/v2/memories/", nil)
	reqBearer.Header.Set("Authorization", "Bearer test-cluster-token-2026")
	recBearer := httptest.NewRecorder()
	handler.ServeHTTP(recBearer, reqBearer)
	if recBearer.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid Bearer token, got %d", recBearer.Code)
	}
}
