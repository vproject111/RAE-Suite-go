package ops

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	}, "rae-suite", "production", "5.3.7")
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

	norm429 := NormalizeLogRecord(map[string]any{"status": 429, "path": "/api/login"}, "rae-suite", "prod", "5.3.7")
	if norm429["error_code"] != "RATE_LIMITED" {
		t.Fatalf("expected RATE_LIMITED, got %v", norm429["error_code"])
	}
	norm400 := NormalizeLogRecord(map[string]any{"status": 400, "path": "/api/login"}, "rae-suite", "prod", "5.3.7")
	if norm400["error_code"] != "VALIDATION_FAILED" {
		t.Fatalf("expected VALIDATION_FAILED, got %v", norm400["error_code"])
	}
	normAudit := NormalizeLogRecord(map[string]any{"action": "DELETE_USER", "msg": "audit delete"}, "rae-suite", "prod", "5.3.7")
	if normAudit["log_category"] != "audit" {
		t.Fatalf("expected audit category, got %v", normAudit["log_category"])
	}
	if normAudit["event"] != "audit_operation" {
		t.Fatalf("expected audit_operation event, got %v", normAudit["event"])
	}

	// 1. Trigger DS-ALERT-01 (High 5xx Error Rate)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:00Z","level":"ERROR","msg":"db timeout","http_method":"POST","http_path":"/api/orders/99","http_status":500,"duration_ms":125.4,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7","company_id":"25","domain":"printworks.pl"}`+"\n"), "rae-suite", "test", "5.3.7")
	// 2. Trigger DS-ALERT-02 (CalcEngineLatencyBreach > 150ms on /calculate)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:01Z","level":"WARN","msg":"slow calc","method":"POST","path":"/api/ps_types/12/ps_calculate","status":200,"duration_ms":612.8,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"11f067aa0ba902b8","company_id":"25","domain":"printworks.pl"}`+"\n"), "rae-suite", "test", "5.3.7")
	// 3. Trigger DS-ALERT-03 (ServiceReadinessDown)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:02Z","level":"ERROR","event":"readiness_failed","path":"/readyz","status":503}`+"\n"), "rae-suite", "test", "5.3.7")
	// 4. Trigger DS-ALERT-04 (PodRestartOrOOMKilled)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:03Z","level":"ERROR","event":"http_handler_panic_recovered","error_code":"ERR_INTERNAL_PANIC"}`+"\n"), "rae-suite", "test", "5.3.7")
	// 5. Trigger DS-ALERT-05 (SecurityAuthSpike)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:04Z","level":"WARN","event":"auth_login_failed","path":"/login","status":403}`+"\n"), "rae-suite", "test", "5.3.7")
	// 6. Trigger DS-ALERT-06 (RAEAuditChainBroken)
	hub.IngestRawBytes([]byte(`{"time":"2026-10-24T12:00:05Z","level":"ERROR","event":"audit_chain_broken","log_category":"audit","company_id":"25"}`+"\n"), "rae-suite", "test", "5.3.7")

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

	// 1. Ingest a synthetic 500 error directly into GlobalObservabilityHub
	GlobalObservabilityHub().IngestRawBytes([]byte(`{"time":"2026-10-24T12:10:30Z","level":"ERROR","msg":"boom","method":"GET","path":"/api/test/123","status":500,"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736","span_id":"00f067aa0ba902b7"}`+"\n"), "rae-suite-test", "test", "5.3.7")

	// 2. Ingest endpoint /_meta/logs/ingest
	reqIngest := httptest.NewRequest(http.MethodPost, "/_meta/logs/ingest", bytes.NewBufferString(`{"time":"2026-10-24T12:11:00Z","level":"INFO","service":"editor-go","msg":"ingested log"}`))
	wIngest := httptest.NewRecorder()
	if !ServeObservabilityEndpoint(wIngest, reqIngest) || wIngest.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/logs/ingest, got %d", wIngest.Code)
	}

	// 3. Query /_meta/logs with federation enabled
	reqLogs := httptest.NewRequest(http.MethodGet, "/_meta/logs?limit=50", nil)
	wLogs := httptest.NewRecorder()
	if !ServeObservabilityEndpoint(wLogs, reqLogs) || wLogs.Code != http.StatusOK {
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
	if !ServeObservabilityEndpoint(wTestAlert, reqTestAlert) || wTestAlert.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/alerts/test, got %d", wTestAlert.Code)
	}

	reqAck := httptest.NewRequest(http.MethodPost, "/_meta/alerts/ack", bytes.NewBufferString(`{"id":"all"}`))
	wAck := httptest.NewRecorder()
	if !ServeObservabilityEndpoint(wAck, reqAck) || wAck.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/alerts/ack, got %d", wAck.Code)
	}

	reqAlerts := httptest.NewRequest(http.MethodGet, "/_meta/alerts?all=1", nil)
	wAlerts := httptest.NewRecorder()
	if !ServeObservabilityEndpoint(wAlerts, reqAlerts) || wAlerts.Code != http.StatusOK {
		t.Fatalf("expected 200 from /_meta/alerts, got %d", wAlerts.Code)
	}

	// 5. Query HTML Presentation Layer /_meta/logs/ui
	reqUI := httptest.NewRequest(http.MethodGet, "/_meta/logs/ui", nil)
	wUI := httptest.NewRecorder()
	if !ServeObservabilityEndpoint(wUI, reqUI) || wUI.Code != http.StatusOK {
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

func makeHS256TestJWT(secret string, claims map[string]any) string {
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payBytes, _ := json.Marshal(claims)
	pay := base64.RawURLEncoding.EncodeToString(payBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(hdr + "." + pay))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hdr + "." + pay + "." + sig
}

func TestObservabilityKeycloakAndManagerJWTAuth(t *testing.T) {
	// 1. External unauthenticated requests MUST be rejected with 401
	protectedPaths := []string{
		"/_meta/logs",
		"/_meta/logs/ingest",
		"/_meta/alerts",
		"/_meta/alerts/ack",
		"/_meta/alerts/test",
	}
	for _, p := range protectedPaths {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.42")
		w := httptest.NewRecorder()
		ServeObservabilityEndpoint(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized on unauthenticated %s, got %d", p, w.Code)
		}
	}

	// 2. Malformed, bad-base64, bad-signature, expired, and empty-claim JWTs must fail
	if ok, reason := verifyObservabilityJWT("not-a-jwt"); ok || reason != "invalid_jwt_format" {
		t.Fatalf("expected invalid_jwt_format, got ok=%v reason=%s", ok, reason)
	}
	if ok, reason := verifyObservabilityJWT("!!!.@@@.###"); ok || reason != "invalid_jwt_base64" {
		t.Fatalf("expected invalid_jwt_base64, got ok=%v reason=%s", ok, reason)
	}
	badJSONTok := base64.RawURLEncoding.EncodeToString([]byte("{bad")) + "." + base64.RawURLEncoding.EncodeToString([]byte("{bad")) + "." + base64.RawURLEncoding.EncodeToString([]byte("sig"))
	if ok, reason := verifyObservabilityJWT(badJSONTok); ok || reason != "invalid_jwt_json" {
		t.Fatalf("expected invalid_jwt_json, got ok=%v reason=%s", ok, reason)
	}

	expiredTok := makeHS256TestJWT("@!&#*@JHAKJHJKSD", map[string]any{
		"exp":  time.Now().Add(-10 * time.Minute).Unix(),
		"user": map[string]any{"ID": 1, "super": 1, "user": "admin@printworks.pl"},
	})
	if ok, reason := verifyObservabilityJWT(expiredTok); ok || reason != "jwt_expired" {
		t.Fatalf("expected jwt_expired, got ok=%v reason=%s", ok, reason)
	}

	wrongSigTok := makeHS256TestJWT("wrong-secret-key", map[string]any{
		"exp":  time.Now().Add(10 * time.Minute).Unix(),
		"user": map[string]any{"ID": 1, "super": 1, "user": "admin@printworks.pl"},
	})
	if ok, reason := verifyObservabilityJWT(wrongSigTok); ok || reason != "invalid_jwt_signature" {
		t.Fatalf("expected invalid_jwt_signature, got ok=%v reason=%s", ok, reason)
	}

	emptyClaimsTok := makeHS256TestJWT("@!&#*@JHAKJHJKSD", map[string]any{
		"exp": time.Now().Add(10 * time.Minute).Unix(),
	})
	if ok, reason := verifyObservabilityJWT(emptyClaimsTok); ok || reason != "insufficient_jwt_claims" {
		t.Fatalf("expected insufficient_jwt_claims, got ok=%v reason=%s", ok, reason)
	}

	// 3. Valid Manager HS256 JWT via Authorization Bearer, Cookie, and Query param
	validManagerTok := makeHS256TestJWT("@!&#*@JHAKJHJKSD", map[string]any{
		"exp":  time.Now().Add(30 * time.Minute).Unix(),
		"user": map[string]any{"ID": 7, "super": 1, "user": "admin@printworks.pl"},
	})
	reqBearer := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqBearer.Header.Set("X-Forwarded-For", "203.0.113.42")
	reqBearer.Header.Set("Authorization", "Bearer "+validManagerTok)
	wBearer := httptest.NewRecorder()
	ServeObservabilityEndpoint(wBearer, reqBearer)
	if wBearer.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid Manager HS256 Bearer token, got %d", wBearer.Code)
	}

	reqCookie := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqCookie.Header.Set("X-Forwarded-For", "203.0.113.42")
	reqCookie.AddCookie(&http.Cookie{Name: "access-token", Value: validManagerTok})
	wCookie := httptest.NewRecorder()
	ServeObservabilityEndpoint(wCookie, reqCookie)
	if wCookie.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid access-token cookie, got %d", wCookie.Code)
	}

	reqQuery := httptest.NewRequest(http.MethodGet, "/_meta/logs?access-token="+validManagerTok, nil)
	reqQuery.Header.Set("X-Forwarded-For", "203.0.113.42")
	wQuery := httptest.NewRecorder()
	ServeObservabilityEndpoint(wQuery, reqQuery)
	if wQuery.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid ?access-token= query parameter, got %d", wQuery.Code)
	}

	// 4. Valid Keycloak OIDC RS256 JWT verified via JWKS endpoint
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nStr := base64.RawURLEncoding.EncodeToString(privKey.PublicKey.N.Bytes())
		eStr := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privKey.PublicKey.E)).Bytes())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"keys":[{"kid":"kc-key-1","kty":"RSA","n":"` + nStr + `","e":"` + eStr + `"}]}`))
	}))
	defer jwksSrv.Close()
	t.Setenv("KEYCLOAK_JWKS_URL", jwksSrv.URL)

	kcHdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT","kid":"kc-key-1"}`))
	kcPayBytes, _ := json.Marshal(map[string]any{
		"exp":                time.Now().Add(30 * time.Minute).Unix(),
		"sub":                "f901-admin-uuid",
		"preferred_username": "grzegorz",
		"realm_access":       map[string]any{"roles": []string{"admin", "superadmin"}},
	})
	kcPay := base64.RawURLEncoding.EncodeToString(kcPayBytes)
	digest := sha256.Sum256([]byte(kcHdr + "." + kcPay))
	rsaSig, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("failed to sign RS256 token: %v", err)
	}
	kcToken := kcHdr + "." + kcPay + "." + base64.RawURLEncoding.EncodeToString(rsaSig)

	reqKC := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqKC.Header.Set("X-Forwarded-For", "203.0.113.42")
	reqKC.Header.Set("Authorization", "Bearer "+kcToken)
	wKC := httptest.NewRecorder()
	ServeObservabilityEndpoint(wKC, reqKC)
	if wKC.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid Keycloak RS256 token, got %d", wKC.Code)
	}

	// 5. Internal Federation from private IP without X-Forwarded-For is allowed; with X-Forwarded-For is blocked
	reqFedOK := httptest.NewRequest(http.MethodGet, "/_meta/logs?federated=0", nil)
	reqFedOK.RemoteAddr = "10.42.1.15:54321"
	reqFedOK.Header.Set("X-Internal-Federation", "1")
	if ok, reason := AuthorizeObservabilityRequest(reqFedOK); !ok || reason != "internal_federation" {
		t.Fatalf("expected internal_federation to be allowed, got ok=%v reason=%s", ok, reason)
	}
	reqFedSpoof := httptest.NewRequest(http.MethodGet, "/_meta/logs?federated=0", nil)
	reqFedSpoof.RemoteAddr = "10.42.1.15:54321"
	reqFedSpoof.Header.Set("X-Internal-Federation", "1")
	reqFedSpoof.Header.Set("X-Forwarded-For", "198.51.100.77")
	if ok, _ := AuthorizeObservabilityRequest(reqFedSpoof); ok {
		t.Fatalf("expected spoofed X-Internal-Federation with X-Forwarded-For to be rejected")
	}

	// 6. Static OBS_ADMIN_TOKEN / RAE_API_KEY
	t.Setenv("OBS_ADMIN_TOKEN", "obs-secret-token-2026")
	reqAPIKey := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqAPIKey.Header.Set("X-Forwarded-For", "203.0.113.42")
	reqAPIKey.Header.Set("X-API-Key", "obs-secret-token-2026")
	if ok, reason := AuthorizeObservabilityRequest(reqAPIKey); !ok || reason != "api_key:OBS_ADMIN_TOKEN" {
		t.Fatalf("expected OBS_ADMIN_TOKEN to be authorized, got ok=%v reason=%s", ok, reason)
	}
	RegisterObservabilityRSAPublicKey("manual-kid", &privKey.PublicKey)
	if lookupKeycloakRSAPublicKey("manual-kid") == nil {
		t.Fatalf("expected manual-kid RSA public key to be cached")
	}
	if !isPrivateOrLoopbackHost("127.0.0.1:8000") {
		t.Fatalf("expected 127.0.0.1:8000 to be loopback")
	}
	if !isPrivateOrLoopbackHost("192.168.1.10:8080") {
		t.Fatalf("expected 192.168.1.10:8080 to be private")
	}
	if !isPrivateOrLoopbackHost("172.18.0.5:9000") {
		t.Fatalf("expected 172.18.0.5:9000 to be private")
	}
	if !isPrivateOrLoopbackHost("::1") {
		t.Fatalf("expected ::1 to be loopback")
	}
	if isPrivateOrLoopbackHost("8.8.8.8:53") {
		t.Fatalf("did not expect 8.8.8.8:53 to be private/loopback")
	}
	if isPrivateOrLoopbackHost("invalid-ip") {
		t.Fatalf("did not expect invalid-ip to be private/loopback")
	}

	// Verify additional meta paths and token extraction branches
	if !IsObservabilityMetaPath("/_meta/logs/ingest") {
		t.Fatalf("expected /_meta/logs/ingest to be meta path")
	}
	if !IsObservabilityMetaPath("/api/_meta/logs/ingest") {
		t.Fatalf("expected /api/_meta/logs/ingest to be meta path")
	}
	if !IsObservabilityMetaPath("/_meta/alerts") {
		t.Fatalf("expected /_meta/alerts to be meta path")
	}
	if !IsObservabilityMetaPath("/api/_meta/alerts") {
		t.Fatalf("expected /api/_meta/alerts to be meta path")
	}
	if !IsObservabilityMetaPath("/_meta/alerts/ack") {
		t.Fatalf("expected /_meta/alerts/ack to be meta path")
	}
	if !IsObservabilityMetaPath("/api/_meta/alerts/ack") {
		t.Fatalf("expected /api/_meta/alerts/ack to be meta path")
	}
	if !IsObservabilityMetaPath("/_meta/alerts/test") {
		t.Fatalf("expected /_meta/alerts/test to be meta path")
	}
	if !IsObservabilityMetaPath("/api/_meta/alerts/test") {
		t.Fatalf("expected /api/_meta/alerts/test to be meta path")
	}
	if !IsObservabilityMetaPath("/v2/portal/logs") {
		t.Fatalf("expected /v2/portal/logs to be meta path")
	}
	if !IsObservabilityMetaPath("/v2/portal/alerts") {
		t.Fatalf("expected /v2/portal/alerts to be meta path")
	}
	if !IsObservabilityMetaPath("/_meta/logs/ui") {
		t.Fatalf("expected /_meta/logs/ui to be meta path")
	}
	if !IsObservabilityMetaPath("/api/_meta/logs") {
		t.Fatalf("expected /api/_meta/logs to be meta path")
	}
	if IsObservabilityMetaPath("/api/calculatePublic") {
		t.Fatalf("did not expect /api/calculatePublic to be meta path")
	}
	if IsObservabilityMetaPath("/healthz") {
		t.Fatalf("did not expect /healthz to be meta path")
	}
	if IsObservabilityMetaPath("/readyz") {
		t.Fatalf("did not expect /readyz to be meta path")
	}
	reqHdrTok := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqHdrTok.Header.Set("Access-Token", "hdr-tok-1")
	if extractObservabilityToken(reqHdrTok) != "hdr-tok-1" {
		t.Fatalf("expected Access-Token header extraction")
	}
	reqXTok := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqXTok.Header.Set("X-Access-Token", "hdr-tok-2")
	if extractObservabilityToken(reqXTok) != "hdr-tok-2" {
		t.Fatalf("expected X-Access-Token header extraction")
	}
	reqMgrCookie := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqMgrCookie.AddCookie(&http.Cookie{Name: "manager-access-token", Value: "mgr-cookie-tok"})
	if extractObservabilityToken(reqMgrCookie) != "mgr-cookie-tok" {
		t.Fatalf("expected manager-access-token cookie extraction")
	}
	reqKCCookie := httptest.NewRequest(http.MethodGet, "/_meta/logs", nil)
	reqKCCookie.AddCookie(&http.Cookie{Name: "KEYCLOAK_IDENTITY", Value: "kc-cookie-tok"})
	if extractObservabilityToken(reqKCCookie) != "kc-cookie-tok" {
		t.Fatalf("expected KEYCLOAK_IDENTITY cookie extraction")
	}
	topClaimsTok := makeHS256TestJWT("@!&#*@INNYSECRET", map[string]any{
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
		"super":  1,
		"userID": 42,
	})
	if ok, _ := verifyObservabilityJWT(topClaimsTok); !ok {
		t.Fatalf("expected top-level super=1 HS256 token with secondary secret to verify")
	}

	// 7. BFF SignLegacyToken style claims (mail, sessionID, aud=dreamsoft-bff, userID=0) must NOT fail with insufficient_jwt_claims
	bffLegacyTok := makeHS256TestJWT("@!&#*@INNYSECRET", map[string]any{
		"userID":       0,
		"sessionID":    "bff_1234567890abcdef",
		"userEditorID": "0",
		"firstname":    "Grzegorz",
		"mail":         "grzegorz@printworks.pl",
		"aud":          "dreamsoft-bff",
		"exp":          time.Now().Add(30 * time.Minute).Unix(),
	})
	if ok, reason := verifyObservabilityJWT(bffLegacyTok); !ok {
		t.Fatalf("expected BFF SignLegacyToken claims to verify, got ok=%v reason=%s", ok, reason)
	}

	// 8. Multiple cookies: first cookie invalid/expired, second cookie valid -> must succeed
	reqMultiCookie := httptest.NewRequest(http.MethodGet, "/api/_meta/logs", nil)
	reqMultiCookie.Header.Set("X-Forwarded-For", "203.0.113.42")
	reqMultiCookie.AddCookie(&http.Cookie{Name: "access-token", Value: expiredTok})
	reqMultiCookie.AddCookie(&http.Cookie{Name: "manager-access-token", Value: bffLegacyTok})
	wMultiCookie := httptest.NewRecorder()
	ServeObservabilityEndpoint(wMultiCookie, reqMultiCookie)
	if wMultiCookie.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when second cookie is valid, got %d", wMultiCookie.Code)
	}

	// 9. Direct browser navigation (Accept: text/html) to /api/_meta/logs must render HTML Operations Center with Keycloak login banner
	reqBrowserHTML := httptest.NewRequest(http.MethodGet, "/api/_meta/logs", nil)
	reqBrowserHTML.Header.Set("X-Forwarded-For", "203.0.113.42")
	reqBrowserHTML.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	wBrowserHTML := httptest.NewRecorder()
	ServeObservabilityEndpoint(wBrowserHTML, reqBrowserHTML)
	if wBrowserHTML.Code != http.StatusOK {
		t.Fatalf("expected 200 OK HTML when browser navigates to /api/_meta/logs, got %d", wBrowserHTML.Code)
	}
	if !strings.Contains(wBrowserHTML.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected text/html Content-Type on browser navigation to /api/_meta/logs, got %s", wBrowserHTML.Header().Get("Content-Type"))
	}
	if !strings.Contains(wBrowserHTML.Body.String(), "Zaloguj w Manager / Keycloak") {
		t.Fatalf("expected HTML response to contain Keycloak/Manager login button")
	}
	if !strings.Contains(wBrowserHTML.Body.String(), "handleCrossSubdomainSSOBridge") {
		t.Fatalf("expected HTML response to contain cross-subdomain SSO bridge")
	}
	if shouldServeHTMLUI(httptest.NewRequest(http.MethodGet, "/api/_meta/logs?format=json", nil)) {
		t.Fatalf("did not expect format=json to serve HTML UI")
	}
	if shouldServeHTMLUI(httptest.NewRequest(http.MethodGet, "/api/_meta/logs?format=ndjson", nil)) {
		t.Fatalf("did not expect format=ndjson to serve HTML UI")
	}

	// 10. Verify human-readable Polish error descriptions, 404 RESOURCE_NOT_FOUND, status_label, 4xx_err filter, and NDJSON stdout stream
	norm404Icon := NormalizeLogRecord(map[string]any{"method": "GET", "path": "/api/uploaded-icons/954", "status": 404}, "backend-go", "prod", "2.0.59")
	if norm404Icon["error_code"] != "RESOURCE_NOT_FOUND" || norm404Icon["error_type"] != "not_found" {
		t.Fatalf("expected 404 to be classified as RESOURCE_NOT_FOUND / not_found, got %v / %v", norm404Icon["error_code"], norm404Icon["error_type"])
	}
	if !strings.Contains(firstString(norm404Icon, "error_description"), "ps_uploaded_file_blobs") {
		t.Fatalf("expected uploaded-icons 404 description to explain ps_uploaded_file_blobs, got %v", norm404Icon["error_description"])
	}
	norm404Theme := NormalizeLogRecord(map[string]any{"method": "GET", "path": "/api/layout/theme/active", "status": 404}, "backend-go", "prod", "2.0.59")
	if !strings.Contains(firstString(norm404Theme, "error_description"), "motywu") {
		t.Fatalf("expected theme 404 description, got %v", norm404Theme["error_description"])
	}
	norm405 := NormalizeLogRecord(map[string]any{"method": "HEAD", "path": "/api/uploaded-icons/891/pudelka.png", "status": 405}, "backend-go", "prod", "2.0.59")
	if norm405["error_code"] != "METHOD_NOT_ALLOWED" {
		t.Fatalf("expected 405 METHOD_NOT_ALLOWED, got %v", norm405["error_code"])
	}
	norm409 := NormalizeLogRecord(map[string]any{"method": "POST", "path": "/api/users", "status": 409, "error_detail": "duplicate email"}, "backend-go", "prod", "2.0.59")
	if norm409["error_code"] != "CONFLICT" || !strings.Contains(firstString(norm409, "error_description"), "duplicate email") {
		t.Fatalf("expected 409 CONFLICT with error_detail, got %+v", norm409)
	}
	if !matchesStatusClass(401, "4xx_err") || matchesStatusClass(404, "4xx_err") || !matchesStatusClass(404, "404") {
		t.Fatalf("expected 4xx_err to match 401 and exclude 404")
	}

	reqNDJSON := httptest.NewRequest(http.MethodGet, "/api/_meta/logs?format=ndjson&limit=5", nil)
	reqNDJSON.Header.Set("Authorization", "Bearer "+bffLegacyTok)
	wNDJSON := httptest.NewRecorder()
	ServeObservabilityEndpoint(wNDJSON, reqNDJSON)
	if wNDJSON.Code != http.StatusOK || !strings.Contains(wNDJSON.Header().Get("Content-Type"), "application/x-ndjson") {
		t.Fatalf("expected 200 application/x-ndjson, got %d %s", wNDJSON.Code, wNDJSON.Header().Get("Content-Type"))
	}

	t.Setenv("LOG_PERSIST_PATH", "")
	if resolveDefaultPersistPath("backend-go") != "" {
		t.Fatalf("expected empty persistPath by default (stdout-only mode)")
	}
	if ExtractErrorDetailFromBody([]byte(`{"error":"missing icon file","reason":"empty_upload"}`)) != "missing icon file (empty_upload)" {
		t.Fatalf("unexpected ExtractErrorDetailFromBody JSON result")
	}
	if ExtractErrorDetailFromBody([]byte("404 page not found\n")) != "404 page not found" {
		t.Fatalf("unexpected ExtractErrorDetailFromBody plain text result")
	}
	if ExtractErrorDetailFromBody([]byte("<!DOCTYPE html><html></html>")) != "" {
		t.Fatalf("expected HTML body to be ignored by ExtractErrorDetailFromBody")
	}
	stLbl, eCode, eCodeLbl, eType, eDesc := BuildHTTPDiagnosticFields("POST", "/api/dp_categories/uploadIcon", 400, []byte(`{"error":"missing icon file"}`))
	if !strings.Contains(stLbl, "400") || eCode != "VALIDATION_FAILED" || eCodeLbl == "" || eType != "validation" || !strings.Contains(eDesc, "missing icon file") {
		t.Fatalf("unexpected BuildHTTPDiagnosticFields: %s | %s | %s | %s | %s", stLbl, eCode, eCodeLbl, eType, eDesc)
	}
	if describeHTTPStatusLabel(200) != "200 OK (Sukces)" {
		t.Fatalf("unexpected 200 status label")
	}
	if describeHTTPStatusLabel(201) != "201 Created (Utworzono)" {
		t.Fatalf("unexpected 201 status label")
	}
	if describeHTTPStatusLabel(204) != "204 No Content (Brak treści)" {
		t.Fatalf("unexpected 204 status label")
	}
	if describeHTTPStatusLabel(401) == "" {
		t.Fatalf("expected non-empty 401 status label")
	}
	if describeHTTPStatusLabel(403) == "" {
		t.Fatalf("expected non-empty 403 status label")
	}
	if describeHTTPStatusLabel(404) == "" {
		t.Fatalf("expected non-empty 404 status label")
	}
	if describeHTTPStatusLabel(405) == "" {
		t.Fatalf("expected non-empty 405 status label")
	}
	if describeHTTPStatusLabel(409) == "" {
		t.Fatalf("expected non-empty 409 status label")
	}
	if describeHTTPStatusLabel(422) == "" {
		t.Fatalf("expected non-empty 422 status label")
	}
	if describeHTTPStatusLabel(429) == "" {
		t.Fatalf("expected non-empty 429 status label")
	}
	if describeHTTPStatusLabel(500) == "" {
		t.Fatalf("expected non-empty 500 status label")
	}
	if describeHTTPStatusLabel(502) == "" {
		t.Fatalf("expected non-empty 502 status label")
	}
	if describeHTTPStatusLabel(503) == "" {
		t.Fatalf("expected non-empty 503 status label")
	}
	if describeHTTPStatusLabel(504) == "" {
		t.Fatalf("expected non-empty 504 status label")
	}
	if describeErrorCodePolish("RESOURCE_NOT_FOUND") == "" {
		t.Fatalf("expected RESOURCE_NOT_FOUND polish label")
	}
	if describeErrorCodePolish("METHOD_NOT_ALLOWED") == "" {
		t.Fatalf("expected METHOD_NOT_ALLOWED polish label")
	}
	if describeErrorCodePolish("AUTH_FORBIDDEN") == "" {
		t.Fatalf("expected AUTH_FORBIDDEN polish label")
	}
	if describeErrorCodePolish("CONFLICT") == "" {
		t.Fatalf("expected CONFLICT polish label")
	}
	if describeErrorCodePolish("RATE_LIMITED") == "" {
		t.Fatalf("expected RATE_LIMITED polish label")
	}
	if describeErrorCodePolish("UPSTREAM_UNAVAILABLE") == "" {
		t.Fatalf("expected UPSTREAM_UNAVAILABLE polish label")
	}
	if describeErrorCodePolish("INTERNAL_ERROR") == "" {
		t.Fatalf("expected INTERNAL_ERROR polish label")
	}

	// Verify ClassifyHTTPStatusLevelAndEvent (Section 4.1)
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 200); lvl != slog.LevelInfo || evt != "http_request" {
		t.Fatalf("unexpected 200 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 400); lvl != slog.LevelInfo || evt != "http_bad_request" {
		t.Fatalf("unexpected 400 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 404); lvl != slog.LevelInfo || evt != "http_not_found" {
		t.Fatalf("unexpected 404 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 422); lvl != slog.LevelInfo || evt != "http_validation_failed" {
		t.Fatalf("unexpected 422 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/bff/auth/check", 401); lvl != slog.LevelInfo || evt != "http_unauthenticated" {
		t.Fatalf("unexpected 401 auth check classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/login", 401); lvl != slog.LevelWarn || evt != "auth_login_failed" {
		t.Fatalf("unexpected 401 login classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 401); lvl != slog.LevelWarn || evt != "auth_token_invalid" {
		t.Fatalf("unexpected 401 protected classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/admin/users", 403); lvl != slog.LevelWarn || evt != "auth_forbidden_access" {
		t.Fatalf("unexpected 403 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 429); lvl != slog.LevelWarn || evt != "http_rate_limited" {
		t.Fatalf("unexpected 429 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 409); lvl != slog.LevelWarn || evt != "http_client_error" {
		t.Fatalf("unexpected 409 classification: %v, %s", lvl, evt)
	}
	if lvl, evt := ClassifyHTTPStatusLevelAndEvent("/api/orders", 500); lvl != slog.LevelError || evt != "http_request_failed" {
		t.Fatalf("unexpected 500 classification: %v, %s", lvl, evt)
	}

	// Verify SanitizeURLPath (Section 5.2)
	uClean, _ := url.Parse("https://printworks.pl/api/login?token=secret123&code=abc&page=2&password=xyz")
	sanitized := SanitizeURLPath(uClean)
	if strings.Contains(sanitized, "secret123") || strings.Contains(sanitized, "abc") || strings.Contains(sanitized, "xyz") || !strings.Contains(sanitized, "page=2") {
		t.Fatalf("SanitizeURLPath failed to redact sensitive query params: %s", sanitized)
	}
	if SanitizeURLPath(nil) != "" {
		t.Fatalf("expected empty string for nil url")
	}

	// Verify TrustedProxyExtractor (Section 5.5)
	reqTrusted := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	reqTrusted.RemoteAddr = "10.42.0.15:54321"
	reqTrusted.Header.Set("X-Forwarded-For", "203.0.113.77, 10.42.0.1")
	if gotIP := TrustedProxyExtractor(reqTrusted); gotIP != "203.0.113.77" {
		t.Fatalf("expected XFF IP 203.0.113.77 from trusted proxy, got %s", gotIP)
	}
	reqUntrusted := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	reqUntrusted.RemoteAddr = "198.51.100.44:12345"
	reqUntrusted.Header.Set("X-Forwarded-For", "1.2.3.4")
	if gotIP := TrustedProxyExtractor(reqUntrusted); gotIP != "198.51.100.44" {
		t.Fatalf("expected direct RemoteAddr IP 198.51.100.44 from untrusted peer, got %s", gotIP)
	}
	if TrustedProxyExtractor(nil) != "" {
		t.Fatalf("expected empty IP for nil request")
	}

	// Verify ExtractActorFromRequest, ComputeAuditHMAC, and LogAuditEvent
	reqAudit := httptest.NewRequest(http.MethodPost, "/api/admin/orders/55/price", nil)
	reqAudit.RemoteAddr = "127.0.0.1:8080"
	reqAudit.Header.Set("X-User-ID", "admin-42")
	reqAudit.Header.Set("X-Company-ID", "25")
	reqAudit.Header.Set("X-Domain", "printworks.pl")
	if actor := ExtractActorFromRequest(reqAudit); actor != "admin-42" {
		t.Fatalf("expected actor admin-42, got %s", actor)
	}
	auditHash := ComputeAuditHMAC("ORDER_PRICE_CHANGE", "order", "55", "admin-42", "SUCCESS", map[string]any{"new_price": 199.99})
	if len(auditHash) != 64 {
		t.Fatalf("expected 64-char SHA-256 HMAC hex, got %s", auditHash)
	}
	auditLogger := slog.New(slog.NewJSONHandler(GlobalObservabilityHub().WrapWriter(io.Discard, "backend-go", "2.0.60", "test"), nil))
	returnedHash := LogAuditEvent(context.Background(), auditLogger, reqAudit, "ORDER_PRICE_CHANGE", "order", "55", "SUCCESS", map[string]any{"new_price": 199.99})
	if returnedHash != auditHash {
		t.Fatalf("expected LogAuditEvent hash %s to equal ComputeAuditHMAC %s", returnedHash, auditHash)
	}
}
