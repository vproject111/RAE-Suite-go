package portal

import (
	"bytes"
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
	if strings.Contains(recRoot.Body.String(), "OpenClawEscalation") || !strings.Contains(recRoot.Body.String(), "HermesOpenCodeEscalation") {
		t.Fatalf("expected HermesOpenCodeEscalation instead of OpenClawEscalation in portal HTML")
	}

	reqTel := httptest.NewRequest(http.MethodGet, "/v2/portal/telemetry", nil)
	recTel := httptest.NewRecorder()
	mux.ServeHTTP(recTel, reqTel)
	if recTel.Code != http.StatusOK || !strings.Contains(recTel.Body.String(), "rae-memory") {
		t.Fatalf("expected 200 on /v2/portal/telemetry, got %d", recTel.Code)
	}

	reqS1 := httptest.NewRequest(http.MethodPost, "/v2/portal/global-search", bytes.NewBufferString(`{"query":"AgnosticStorageBroker","project":"alpha"}`))
	recS1 := httptest.NewRecorder()
	mux.ServeHTTP(recS1, reqS1)

	reqS2 := httptest.NewRequest(http.MethodPost, "/v2/portal/global-search", bytes.NewBufferString(`{"query":"MerkleBucketTree","project":"beta"}`))
	recS2 := httptest.NewRecorder()
	mux.ServeHTTP(recS2, reqS2)

	if recS1.Code != http.StatusOK || recS2.Code != http.StatusOK || recS1.Body.String() == recS2.Body.String() {
		t.Fatalf("expected distinct global-search outputs for distinct inputs")
	}
}
