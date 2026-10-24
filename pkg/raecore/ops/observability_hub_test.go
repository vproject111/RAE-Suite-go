package ops

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestObservabilityHub_CanonicalSchemaAndAlertEngine(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test-obs.jsonl")
	hub := NewObservabilityHub("rae-suite", tmpFile)
	hub.SetPeerURLs(nil)

	rules := CanonicalAlertRules()
	if len(rules) != 6 {
		t.Fatalf("expected 6 canonical alert rules (DS-ALERT-01..06), got %d", len(rules))
	}
	if rules[0].Code != "DS-ALERT-01" {
		t.Fatalf("expected DS-ALERT-01, got %s", rules[0].Code)
	}
	if rules[1].Code != "DS-ALERT-02" {
		t.Fatalf("expected DS-ALERT-02, got %s", rules[1].Code)
	}
	if rules[2].Code != "DS-ALERT-03" {
		t.Fatalf("expected DS-ALERT-03, got %s", rules[2].Code)
	}
	if rules[3].Code != "DS-ALERT-04" {
		t.Fatalf("expected DS-ALERT-04, got %s", rules[3].Code)
	}
	if rules[4].Code != "DS-ALERT-05" {
		t.Fatalf("expected DS-ALERT-05, got %s", rules[4].Code)
	}
	if rules[5].Code != "DS-ALERT-06" {
		t.Fatalf("expected DS-ALERT-06, got %s", rules[5].Code)
	}

	routeTests := []struct {
		in   string
		want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/_go/bundle.js", "/_static_asset"},
		{"/assets/logo.png?v=1", "/_static_asset"},
		{"/static/app.css", "/_static_asset"},
		{"/api/ps_groups/12345/ps_types/678", "/api/ps_groups/{groupID}/ps_types/{typeID}"},
		{"/api/dp_orders/999/items/42", "/api/dp_orders/{orderID}/items/{id}"},
		{"/api/projects/550e8400-e29b-41d4-a716-446655440000", "/api/projects/{projectID}"},
		{"/api/tokens/4bf92f3577b34da6a3ce929d0e0e4736", "/api/tokens/{id}"},
	}
	for _, rt := range routeTests {
		if got := NormalizeRoutePattern(rt.in); got != rt.want {
			t.Fatalf("NormalizeRoutePattern(%q) = %q, want %q", rt.in, got, rt.want)
		}
	}

	norm := NormalizeLogRecord(map[string]any{
		"http_method":    "POST",
		"http_path":      "/api/dp_orders/77",
		"http_status":    502,
		"response_bytes": 512,
		"remote_addr":    "10.0.0.15:54321",
	}, "rae-suite", "production", "5.3.5")
	if norm["method"] != "POST" {
		t.Fatalf("expected method POST, got %v", norm["method"])
	}
	if norm["route"] != "/api/dp_orders/{orderID}" {
		t.Fatalf("expected route /api/dp_orders/{orderID}, got %v", norm["route"])
	}
	if norm["status"] != 502 {
		t.Fatalf("expected status 502, got %v", norm["status"])
	}
	if norm["bytes"] != 512 {
		t.Fatalf("expected bytes 512, got %v", norm["bytes"])
	}
	if norm["client_ip"] != "10.0.0.15" {
		t.Fatalf("expected client_ip 10.0.0.15, got %v", norm["client_ip"])
	}
	if norm["error_code"] != "UPSTREAM_UNAVAILABLE" {
		t.Fatalf("expected error_code UPSTREAM_UNAVAILABLE, got %v", norm["error_code"])
	}
	if norm["error_type"] != "upstream" {
		t.Fatalf("expected error_type upstream, got %v", norm["error_type"])
	}
	if norm["log_category"] != "access" {
		t.Fatalf("expected log_category access, got %v", norm["log_category"])
	}

	norm429 := NormalizeLogRecord(map[string]any{"status": 429, "path": "/api/login"}, "rae-suite", "prod", "5.3.5")
	if norm429["error_code"] != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED, got %v", norm429["error_code"])
	}
	norm400 := NormalizeLogRecord(map[string]any{"status": 400, "path": "/api/login"}, "rae-suite", "prod", "5.3.5")
	if norm400["error_code"] != "VALIDATION_FAILED" {
		t.Fatalf("expected VALIDATION_FAILED, got %v", norm400["error_code"])
	}
	normAudit := NormalizeLogRecord(map[string]any{"action": "DELETE_USER", "msg": "audit delete"}, "rae-suite", "prod", "5.3.5")
	if normAudit["log_category"] != "audit" {
		t.Fatalf("expected audit category, got %v", normAudit["log_category"])
	}
	if normAudit["event"] != "audit_operation" {
		t.Fatalf("expected audit_operation event, got %v", normAudit["event"])
	}

	// 1. Trigger DS-ALERT-01 (High 5xx Error Rate)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:00Z","level":"ERROR","msg":"db timeout","http_method":"POST","http_path":"/api/orders/99","http_status":500,"duration_ms":125.4,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","company_id":"25","domain":"printworks.pl"}`+"\n"), "rae-suite", "test", "5.3.5")
	// 2. Trigger DS-ALERT-02 (CalcEngineLatencyBreach > 150ms on /calculate)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:01Z","level":"WARN","msg":"slow calc","method":"POST","path":"/api/ps_types/12/ps_calculate","status":200,"duration_ms":612.8,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"11f067aa0ba902b8","company_id":"25","domain":"printworks.pl"}`+"\n"), "rae-suite", "test", "5.3.5")
	// 3. Trigger DS-ALERT-03 (ServiceReadinessDown)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:02Z","level":"ERROR","event":"readiness_failed","path":"/readyz","status":503}`+"\n"), "rae-suite", "test", "5.3.5")
	// 4. Trigger DS-ALERT-04 (PodRestartOrOOMKilled)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:03Z","level":"ERROR","event":"http_handler_panic_recovered","error_code":"ERR_INTERNAL_PANIC"}`+"\n"), "rae-suite", "test", "5.3.5")
	// 5. Trigger DS-ALERT-05 (SecurityAuthSpike)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:04Z","level":"WARN","event":"auth_login_failed","path":"/login","status":403}`+"\n"), "rae-suite", "test", "5.3.5")
	// 6. Trigger DS-ALERT-06 (RAEAuditChainBroken)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:05Z","level":"ERROR","event":"audit_chain_broken","log_category":"audit","company_id":"25"}`+"\n"), "rae-suite", "test", "5.3.5")

	alerts := hub.ListAlerts(true)
	if len(alerts) < 6 {
		t.Fatalf("expected all 6 alert rules (DS-ALERT-01..06) to have fired, got %d", len(alerts))
	}

	// Verify filters
	if len(hub.QueryLogs(LogQueryFilter{StatusClass: "5xx"})) < 2 {
		t.Fatalf("expected >= 2 5xx logs")
	}
	if len(hub.QueryLogs(LogQueryFilter{StatusClass: "4xx"})) != 1 {
		t.Fatalf("expected 1 4xx log")
	}
	if len(hub.QueryLogs(LogQueryFilter{StatusClass: "2xx"})) != 1 {
		t.Fatalf("expected 1 2xx log")
	}
	if len(hub.QueryLogs(LogQueryFilter{LogCategory: "audit"})) != 1 {
		t.Fatalf("expected 1 audit log")
	}
	if len(hub.QueryLogs(LogQueryFilter{Domain: "printworks"})) != 2 {
		t.Fatalf("expected 2 printworks domain logs")
	}
	if len(hub.QueryLogs(LogQueryFilter{Query: "slow calc"})) != 1 {
		t.Fatalf("expected 1 log matching query 'slow calc'")
	}

	// Verify Trace Waterfall for trace_id 4bf92f3577b34da6a3ce929d0e0e4736
	report := hub.BuildDashboardReport(context.Background(), LogQueryFilter{
		CompanyID: "25",
		TraceID:   "4bf92f3577b34da6a3ce929d0e0e4736",
		Limit:     50,
	}, false)
	if len(report.Section4Trace) != 2 {
		t.Fatalf("expected 2 spans in trace waterfall, got %d", len(report.Section4Trace))
	}
	if len(report.Section1Uptime) == 0 {
		t.Fatalf("expected Section1Uptime to be populated")
	}
	if len(report.Section2Routes) == 0 {
		t.Fatalf("expected Section2Routes to be populated")
	}
	if len(report.Section3SLA) == 0 {
		t.Fatalf("expected Section3SLA to be populated")
	}
	if report.Section5Health.Goroutines <= 0 {
		t.Fatalf("expected positive goroutines count")
	}
	if len(report.AuditTrail) != 1 {
		t.Fatalf("expected 1 audit trail record, got %d", len(report.AuditTrail))
	}

	// Acknowledge DS-ALERT-01
	if hub.AcknowledgeAlert("DS-ALERT-01") < 1 {
		t.Fatalf("expected AcknowledgeAlert(DS-ALERT-01) to succeed")
	}
	if hub.AcknowledgeAlert("all") < 1 {
		t.Fatalf("expected AcknowledgeAlert(all) to acknowledge remaining alerts")
	}
	if len(hub.ListAlerts(false)) != 0 {
		t.Fatalf("expected 0 unacknowledged alerts after ack all")
	}

	// Reload from persistent JSONL file
	hub2 := NewObservabilityHub("rae-suite", tmpFile)
	entries := hub2.QueryLogs(LogQueryFilter{Limit: 100})
	if len(entries) < 6 {
		t.Fatalf("expected persisted JSONL entries to reload (>=6), got %d", len(entries))
	}
}

func TestObservabilityHTTPHandlersAndFederation(t *testing.T) {
	// Peer server for federation test
	peerMux := http.NewServeMux()
	peerMux.HandleFunc("/_meta/logs", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[{"time":"2026-10-24T12:10:00Z","level":"INFO","service":"backend-go","msg":"peer federated log","status":200,"path":"/api/calc","route":"/api/calc","duration_ms":11.2,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"22f067aa0ba902b9"}]}`))
	})
	peerSrv := httptest.NewServer(peerMux)
	defer peerSrv.Close()

	GlobalObservabilityHub().SetPeerURLs([]string{peerSrv.URL})
	defer GlobalObservabilityHub().SetPeerURLs(nil)

	if !IsObservabilityMetaPath("/_meta/logs") {
		t.Fatalf("expected /_meta/logs to be recognized as meta path")
	}
	if !IsObservabilityMetaPath("/api/_meta/logs/ui") {
		t.Fatalf("expected /api/_meta/logs/ui to be recognized as meta path")
	}
	if !IsObservabilityMetaPath("/system-logs") {
		t.Fatalf("expected /system-logs to be recognized as meta path")
	}
	if IsObservabilityMetaPath("/api/v1/memories") {
		t.Fatalf("did not expect /api/v1/memories to be meta path")
	}

	rec := NewTestLogRecorder()
	logger := NewCanonicalLogger("rae-suite-test", "test", "5.3.5", rec, slog.LevelDebug)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/test/123", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})
	handler := RequestLoggerMiddleware(logger, mux)

	// 1. Trigger a 500 error through middleware
	req := httptest.NewRequest(http.MethodGet, "/api/test/123", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}

	// 2. Ingest endpoint /_meta/logs/ingest
	reqIngest := httptest.NewRequest(http.MethodPost, "/_meta/logs/ingest", bytes.NewBufferString(`{"time":"2026-10-24T12:11:00Z","level":"INFO","service":"editor-go","msg":"ingested log"}`))
	wIngest := httptest.NewRecorder()
	handler.ServeHTTP(wIngest, reqIngest)
	if wIngest.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/logs/ingest, got %d", wIngest.Code)
	}

	// 3. Query /_meta/logs with federation enabled
	reqLogs := httptest.NewRequest(http.MethodGet, "/_meta/logs?limit=50", nil)
	wLogs := httptest.NewRecorder()
	handler.ServeHTTP(wLogs, reqLogs)
	if wLogs.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/logs, got %d", wLogs.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(wLogs.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode /_meta/logs: %v", err)
	}
	if _, ok := payload["section1_service_availability"]; !ok {
		t.Fatalf("expected section1_service_availability in /_meta/logs response")
	}
	if _, ok := payload["section4_trace_waterfall"]; !ok {
		t.Fatalf("expected section4_trace_waterfall in /_meta/logs response")
	}

	// 4. Query /_meta/alerts, trigger test alert, and ack
	reqTestAlert := httptest.NewRequest(http.MethodPost, "/_meta/alerts/test?service=rae-suite-test", nil)
	wTestAlert := httptest.NewRecorder()
	handler.ServeHTTP(wTestAlert, reqTestAlert)
	if wTestAlert.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/alerts/test, got %d", wTestAlert.Code)
	}

	reqAck := httptest.NewRequest(http.MethodPost, "/_meta/alerts/ack", bytes.NewBufferString(`{"id":"all"}`))
	wAck := httptest.NewRecorder()
	handler.ServeHTTP(wAck, reqAck)
	if wAck.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/alerts/ack, got %d", wAck.Code)
	}

	reqAlerts := httptest.NewRequest(http.MethodGet, "/_meta/alerts?all=1", nil)
	wAlerts := httptest.NewRecorder()
	handler.ServeHTTP(wAlerts, reqAlerts)
	if wAlerts.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/alerts, got %d", wAlerts.Code)
	}

	// 5. Query HTML Presentation Layer /_meta/logs/ui
	reqUI := httptest.NewRequest(http.MethodGet, "/_meta/logs/ui", nil)
	wUI := httptest.NewRecorder()
	handler.ServeHTTP(wUI, reqUI)
	if wUI.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/logs/ui, got %d", wUI.Code)
	}
	uiBody := wUI.Body.String()
	if !strings.Contains(uiBody, "Dreamsoft &amp; RAE Operations Center") {
		t.Fatalf("expected HTML UI to contain title 'Dreamsoft &amp; RAE Operations Center'")
	}
	if !strings.Contains(uiBody, "1. Dostępność Usług Go") {
		t.Fatalf("expected HTML UI to contain Section 1 header")
	}
	if !strings.Contains(uiBody, "2. Aktywne Alerty (DS-ALERT-01..06)") {
		t.Fatalf("expected HTML UI to contain Section 2 header")
	}
	if !strings.Contains(uiBody, "3. Opóźnienia API p50 / p95 / p99") {
		t.Fatalf("expected HTML UI to contain Section 3 header")
	}
	if !strings.Contains(uiBody, "4. Rozproszona Oś Czasu Trace Waterfall") {
		t.Fatalf("expected HTML UI to contain Section 4 header")
	}
	if !strings.Contains(uiBody, "5. Zasoby Procesu Go, Retencja i Ślad Audytowy") {
		t.Fatalf("expected HTML UI to contain Section 5 header")
	}

	// 6. Verify helper conversions (firstString, firstInt, firstFloat, percentileFloat, matchesStatusClass)
	sampleMap := map[string]any{
		"str_val":   "hello",
		"int_val":   42,
		"int64_val": int64(99),
		"flt_val":   3.14,
		"str_num":   "128",
		"str_flt":   "2.718",
	}
	if firstString(sampleMap, "missing", "str_val") != "hello" {
		t.Fatalf("expected firstString hello")
	}
	if firstString(sampleMap, "missing") != "" {
		t.Fatalf("expected empty firstString")
	}
	if firstInt(sampleMap, "int_val") != 42 {
		t.Fatalf("expected firstInt 42")
	}
	if firstInt(sampleMap, "int64_val") != 99 {
		t.Fatalf("expected firstInt 99")
	}
	if firstInt(sampleMap, "flt_val") != 3 {
		t.Fatalf("expected firstInt 3")
	}
	if firstInt(sampleMap, "str_num") != 128 {
		t.Fatalf("expected firstInt 128")
	}
	if firstInt(sampleMap, "missing") != 0 {
		t.Fatalf("expected firstInt 0")
	}
	if firstFloat(sampleMap, "flt_val") != 3.14 {
		t.Fatalf("expected firstFloat 3.14")
	}
	if firstFloat(sampleMap, "int_val") != 42.0 {
		t.Fatalf("expected firstFloat 42.0")
	}
	if firstFloat(sampleMap, "int64_val") != 99.0 {
		t.Fatalf("expected firstFloat 99.0")
	}
	if firstFloat(sampleMap, "str_flt") != 2.718 {
		t.Fatalf("expected firstFloat 2.718")
	}
	if firstFloat(sampleMap, "missing") != 0.0 {
		t.Fatalf("expected firstFloat 0.0")
	}
	if percentileFloat(nil, 0.95) != 0.0 {
		t.Fatalf("expected percentileFloat(nil) == 0")
	}
	if percentileFloat([]float64{10, 20, 30}, 1.0) != 30.0 {
		t.Fatalf("expected percentileFloat(1.0) == 30.0")
	}
	if !matchesStatusClass(200, "200") {
		t.Fatalf("expected matchesStatusClass(200, '200') to be true")
	}
	if matchesStatusClass(200, "500") {
		t.Fatalf("expected matchesStatusClass(200, '500') to be false")
	}
	if !matchesStatusClass(200, "ALL") {
		t.Fatalf("expected matchesStatusClass(200, 'ALL') to be true")
	}
	if ServeObservabilityEndpoint(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/unknown", nil)) {
		t.Fatalf("did not expect /unknown to be handled by ServeObservabilityEndpoint")
	}
}
