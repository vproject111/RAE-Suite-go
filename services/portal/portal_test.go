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

func TestPortalAllRoutesAndDeepApps(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}

	svc := NewService()
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// 404 path on "/" handler
	req404 := httptest.NewRequest(http.MethodGet, "/unknown-route", nil)
	rec404 := httptest.NewRecorder()
	mux.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown path, got %d", rec404.Code)
	}

	routes := []struct {
		method string
		path   string
		body   string
		expect string
	}{
		{http.MethodGet, "/health", "", "healthy"},
		{http.MethodGet, "/manifest.json", "", "RAE Suite Command Center"},
		{http.MethodGet, "/sw.js", "", "rae-portal-v4.8"},
		{http.MethodGet, "/v2/portal/views", "", "MissionControl"},
		{http.MethodGet, "/v2/portal/overview", "", "departments"},
		{http.MethodPost, "/logout", "", "logged_out"},
		{http.MethodGet, "/callback?code=oauth123", "", "oauth123"},
		{http.MethodGet, "/ws/dashboard", "", "portal_live_stream"},
		{http.MethodGet, "/v2/portal/global-search", "", "federated_ok"},
		{http.MethodGet, "/v2/portal/global-search?query=hybrid&project=proj1", "", "proj1"},
	}

	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s returned %d", tc.method, tc.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.expect) {
			t.Fatalf("%s %s missing %q in %q", tc.method, tc.path, tc.expect, rec.Body.String())
		}
	}

	// Test PDKnowledgeHubApp
	kh := NewPDKnowledgeHubApp()
	if ch := kh.ChunkText("   ", 0); ch != nil {
		t.Fatalf("expected nil chunks for whitespace text")
	}
	longWords := strings.Repeat("knowledge ", 30)
	chunks := kh.ChunkText(longWords, 40)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	up := kh.HandleFileUpload("doc-1", []byte("  ISO 42001 compliance manual for RAE knowledge hub  "))
	if up["status"] != "INGESTED" {
		t.Fatalf("unexpected upload status: %+v", up)
	}
	hits := kh.SearchISAPAndWeb("42001")
	if len(hits) != 1 {
		t.Fatalf("expected 1 search hit, got %d", len(hits))
	}
	if !kh.DeleteDocumentChunks("doc-1") || kh.DeleteDocumentChunks("doc-1") {
		t.Fatalf("unexpected DeleteDocumentChunks behavior")
	}

	// Test CentralAuditorApp
	aud := NewCentralAuditorApp("")
	sig := aud.SignEvent("payload-1")
	if !aud.VerifyEventSignature("payload-1", sig) || aud.VerifyEventSignature("payload-1", "bad") {
		t.Fatalf("signature verification failed")
	}
	secrets := aud.DetectSecrets("sk-12345 ghp_abcdef password=secret")
	if len(secrets) != 3 {
		t.Fatalf("expected 3 secret patterns, got %v", secrets)
	}
	auditRes := aud.TriggerSentinelAudit("clean code")
	if auditRes["passed"] != true {
		t.Fatalf("expected clean code to pass sentinel audit")
	}
	if !strings.Contains(aud.Render(), "central-auditor") {
		t.Fatalf("unexpected CentralAuditorApp render")
	}

	// Test PhoenixRepairApp
	phx := NewPhoenixRepairApp()
	if !strings.Contains(phx.TriggerSimulatedIncident("INC-99"), "INC-99") {
		t.Fatalf("unexpected TriggerSimulatedIncident")
	}
	if phx.TickIncidentTimer() != 1 {
		t.Fatalf("expected timer tick 1")
	}
	sim := phx.SimulateRepair("func a() {}")
	if sim["status"] != "SIMULATED_OK" {
		t.Fatalf("unexpected SimulateRepair: %+v", sim)
	}
	appRes := phx.ApplyRepair("func a() {}")
	if appRes["applied"] != true {
		t.Fatalf("unexpected ApplyRepair: %+v", appRes)
	}
	if !strings.Contains(phx.Render(), "phoenix-repair") {
		t.Fatalf("unexpected PhoenixRepairApp render")
	}

	// Test MissionControlApp & RAESuitePortal
	portalApp := NewRAESuitePortal()
	sys := portalApp.DetectSystem()
	if sys["runtime"] != "go" {
		t.Fatalf("unexpected DetectSystem: %+v", sys)
	}
	insp := portalApp.OpenInspector("mem-123")
	if insp["inspector_open"] != true {
		t.Fatalf("unexpected OpenInspector: %+v", insp)
	}
	if !portalApp.StartWebSocketClient() {
		t.Fatalf("expected StartWebSocketClient=true")
	}
	for _, page := range []string{"mission_control", "auditor", "phoenix", "nicegui"} {
		portalApp.SetPage(page)
		if portalApp.Render() == "" {
			t.Fatalf("expected non-empty render for page %s", page)
		}
	}
	portalApp.NiceGUIDash = nil
	portalApp.SetPage("nicegui")
	if !strings.Contains(portalApp.Render(), "mission-control") {
		t.Fatalf("expected fallback to MissionCtrl when NiceGUIDash is nil")
	}
}
