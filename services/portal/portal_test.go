package portal

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPortalDashboardPWAAndTelemetry(t *testing.T) {
	svc := NewService()
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	mux.ServeHTTP(recRoot, reqRoot)
	if recRoot.Code != http.StatusOK || !strings.Contains(recRoot.Body.String(), "RaeCrl") {
		t.Fatalf("expected 200 and RaeCrl in portal HTML, got code=%d", recRoot.Code)
	}

	reqTel := httptest.NewRequest(http.MethodGet, "/v2/portal/telemetry", nil)
	recTel := httptest.NewRecorder()
	mux.ServeHTTP(recTel, reqTel)
	if recTel.Code != http.StatusOK || !strings.Contains(recTel.Body.String(), "rae-memory") {
		t.Fatalf("expected 200 on /v2/portal/telemetry, got %d", recTel.Code)
	}
}
