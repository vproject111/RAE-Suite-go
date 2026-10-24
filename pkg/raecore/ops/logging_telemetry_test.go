package ops

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/accounting"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestCanonicalJSONLoggerAndGenAITelemetry(t *testing.T) {
	rec := NewTestLogRecorder()
	logger := NewCanonicalLogger("rae-memory", "staging", "2.9.0", rec, slog.LevelInfo)

	ctx := WithTelemetryContext(context.Background(), TelemetryContext{
		TraceID:   "0123456789abcdef0123456789abcdef",
		SpanID:    "0123456789abcdef",
		Sampled:   true,
		RequestID: "req-abc-123",
		TenantID:  "tenant-dreamsoft",
		AgentID:   "agent-ceo",
	})

	LogGenAIInvocation(ctx, logger, "llm_inference_completed", GenAITelemetry{
		System:        "openai",
		RequestModel:  "z-ai/glm-5.3-flash",
		InputTokens:   512,
		OutputTokens:  128,
		CostUSD:       0.00042,
		CascadeTier:   "tier1_generate",
		MemoryLayer:   "working",
		SecurityClass: "RESTRICTED",
	}, slog.String("password", "super-secret-pass-999"), slog.String("user_email", "grzegorz@dreamsoft.pro"), slog.String("pesel_note", "PESEL 90010112345 and Bearer eyJhbGciOiJIUzI1NiJ9"))

	entries, err := rec.Entries()
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 valid JSON log entry, got %d (err=%v)", len(entries), err)
	}
	e := entries[0]

	if e["service"] != "rae-memory" || e["env"] != "staging" || e["version"] != "2.9.0" {
		t.Fatalf("missing canonical service attributes: %+v", e)
	}
	if e["trace_id"] != "0123456789abcdef0123456789abcdef" || e["span_id"] != "0123456789abcdef" || e["request_id"] != "req-abc-123" {
		t.Fatalf("missing W3C trace context attributes: %+v", e)
	}
	if e["tenant_id"] != "tenant-dreamsoft" || e["agent_id"] != "agent-ceo" {
		t.Fatalf("missing tenant/agent attributes: %+v", e)
	}
	if e["gen_ai.system"] != "openai" || e["gen_ai.request.model"] != "z-ai/glm-5.3-flash" || e["rae.cascade.tier"] != "tier1_generate" || e["rae.memory.layer"] != "working" || e["rae.security.info_class"] != "RESTRICTED" {
		t.Fatalf("missing OpenTelemetry GenAI attributes: %+v", e)
	}
	if e["password"] != "[REDACTED]" {
		t.Fatalf("expected password redacted, got %v", e["password"])
	}
	if err := rec.AssertNoSecretsLeaked("super-secret-pass-999", "90010112345", "eyJhbGciOiJIUzI1NiJ9", "grzegorz@dreamsoft.pro"); err != nil {
		t.Fatalf("PII/secret leak detected: %v", err)
	}
	if _, ok := rec.AssertContainsEvent("llm_inference_completed"); !ok {
		t.Fatalf("expected llm_inference_completed event in recorder")
	}
	if _, ok := rec.AssertContainsEvent("non_existent_event"); ok {
		t.Fatalf("did not expect non_existent_event")
	}
	if err := rec.AssertNoSecretsLeaked("[REDACTED]"); err == nil {
		t.Fatalf("expected error when checking for literal substring present in logs")
	}
}

func TestRequestLoggerAndRecoveryMiddlewares(t *testing.T) {
	rec := NewTestLogRecorder()
	logger := NewCanonicalLogger("", "", "", rec, slog.LevelInfo)
	_ = NewCanonicalLogger("", "", "", nil, slog.LevelDebug)
	LogGenAIInvocation(context.Background(), nil, "noop", GenAITelemetry{})

	ctxNil := WithTelemetryContext(nil, TelemetryContext{})
	tcNil := ExtractTelemetryContext(ctxNil)
	if len(tcNil.TraceID) != 32 || len(tcNil.SpanID) != 16 || tcNil.RequestID == "" {
		t.Fatalf("expected auto-generated telemetry IDs, got %+v", tcNil)
	}
	if emptyTC := ExtractTelemetryContext(nil); emptyTC.TraceID != "" {
		t.Fatalf("expected empty TelemetryContext on nil ctx")
	}

	childLogger := logger.With("component", "http").WithGroup("req")
	_ = childLogger.Enabled(context.Background(), slog.LevelInfo)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	mux.HandleFunc("/v1/ok", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/v1/warn", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})
	mux.HandleFunc("/v1/err", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/v1/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("simulated database pointer panic")
	})

	handler := RequestLoggerMiddleware(logger, RecoveryMiddleware(logger, mux))

	// 1. Health endpoint -> suppressed at INFO level
	reqH := httptest.NewRequest(http.MethodGet, "/health", nil)
	wH := httptest.NewRecorder()
	handler.ServeHTTP(wH, reqH)
	if entries, _ := rec.Entries(); len(entries) != 0 {
		t.Fatalf("expected /health suppressed at INFO level, got %d entries", len(entries))
	}

	// 2. Normal request with incoming W3C traceparent
	reqOK := httptest.NewRequest(http.MethodPost, "/v1/ok", nil)
	reqOK.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	reqOK.Header.Set("X-Tenant-ID", "t-1")
	reqOK.Header.Set("X-Agent-ID", "a-1")
	wOK := httptest.NewRecorder()
	handler.ServeHTTP(wOK, reqOK)
	if !strings.HasPrefix(wOK.Header().Get("traceparent"), "00-4bf92f3577b34da6a3ce929d0e0e4736-") {
		t.Fatalf("expected propagated trace_id in response traceparent, got %s", wOK.Header().Get("traceparent"))
	}

	// 3. 400 and 503 requests
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/warn", nil))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/err", nil))

	// 4. Panic request -> caught by RecoveryMiddleware -> HTTP 500 + stack_trace logged
	wPanic := httptest.NewRecorder()
	handler.ServeHTTP(wPanic, httptest.NewRequest(http.MethodGet, "/v1/panic", nil))
	if wPanic.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500 on panic, got %d", wPanic.Code)
	}
	panicEntry, ok := rec.AssertContainsEvent("http_handler_panic_recovered")
	if !ok || panicEntry["error_code"] != "ERR_INTERNAL_PANIC" || panicEntry["stack_trace"] == "" {
		t.Fatalf("expected panic entry with stack_trace, got %+v", panicEntry)
	}
}

func TestParseW3CTraceparentEdgeCases(t *testing.T) {
	cases := []struct {
		header  string
		valid   bool
		sampled bool
	}{
		{"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", true, true},
		{"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-00", true, false},
		{"01-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", false, false},
		{"00-short-00f067aa0ba902b7-01", false, false},
		{"00-00000000000000000000000000000000-00f067aa0ba902b7-01", false, false},
		{"00-4bf92f3577b34da6a3ce929d0e0e4736-0000000000000000-01", false, false},
		{"00-zzzz2f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", false, false},
		{"invalid", false, false},
	}
	for _, tc := range cases {
		_, _, sampled, ok := ParseW3CTraceparent(tc.header)
		if ok != tc.valid || sampled != tc.sampled {
			t.Fatalf("ParseW3CTraceparent(%q) = (ok=%v, sampled=%v), want (ok=%v, sampled=%v)", tc.header, ok, sampled, tc.valid, tc.sampled)
		}
	}
}

func FuzzInjectW3CTraceContext(f *testing.F) {
	f.Add("trace-1234567890abcdef1234567890abcdef", "1234567890abcdef", true)
	f.Add("", "", false)
	f.Add("NON-HEX-CHARS-!@#$%^&*()", "SPAN-!@#$", true)
	f.Add(strings.Repeat("f", 64), strings.Repeat("e", 32), false)

	f.Fuzz(func(t *testing.T, traceID, spanID string, sampled bool) {
		tp := InjectW3CTraceContext(traceID, spanID, sampled)
		parsedTrace, parsedSpan, parsedSampled, ok := ParseW3CTraceparent(tp)
		if !ok {
			t.Fatalf("InjectW3CTraceContext produced invalid W3C traceparent %q for inputs (%q, %q, %v)", tp, traceID, spanID, sampled)
		}
		if len(parsedTrace) != 32 || len(parsedSpan) != 16 || parsedSampled != sampled {
			t.Fatalf("unexpected parsed values from %q", tp)
		}
	})
}

func TestOpsRemainingBranchesCoverage(t *testing.T) {
	// ImprovementPlane edge cases
	imp := NewImprovementPlane(0)
	h := imp.CreateHypothesis("h-empty", "stmt", "mot", "")
	if h.TargetMetric != "latency" {
		t.Fatalf("expected default targetMetric=latency")
	}
	if rb := imp.EvaluateCanaryHealth("exp-ok", 0.01); rb.RollbackTriggered {
		t.Fatalf("expected healthy canary")
	}
	imp.RecordRun(ExperimentRun{RunID: "r-fail", ExperimentID: "exp-fail", Mode: "shadow", Result: "fail"})
	dec := imp.EvaluatePromotion(ImprovementProposal{ProposalID: "p-fail", ExperimentID: "exp-fail"})
	if dec.Approved || !strings.Contains(dec.Reason, "shadow_passed") {
		t.Fatalf("expected shadow_passed failure reason, got %+v", dec)
	}

	// ClusterManager routing branches
	cm := NewClusterManager()
	if n := cm.RouteTask("benchmark_heavy"); n.NodeID != "node2_julka" {
		t.Fatalf("expected node2_julka, got %s", n.NodeID)
	}
	if n := cm.RouteTask("default_local"); n.NodeID != "node1_lumina" {
		t.Fatalf("expected node1_lumina, got %s", n.NodeID)
	}

	// CircuitBreaker recovery and HalfOpen
	cb := NewTransportCircuitBreaker(0, 0)
	if !cb.AllowExecution() {
		t.Fatalf("expected closed circuit to allow execution")
	}
	cb2 := NewTransportCircuitBreaker(1, 0.001)
	cb2.RecordFailure()
	cb2.LastStateChange = cb2.LastStateChange.Add(-10_000_000_000)
	if !cb2.AllowExecution() || cb2.State != CircuitHalfOpen {
		t.Fatalf("expected HalfOpen after timeout")
	}
	if cb2.RecordSuccess() != CircuitClosed {
		t.Fatalf("expected Closed after RecordSuccess in HalfOpen")
	}

	// RateLimiter defaults
	rl := NewDistributedRateLimiter(0, 0)
	if st := rl.CheckRateLimit("c1", 0); !st.Allowed {
		t.Fatalf("expected allowed")
	}

	// RollbackSLAManager unknown action & verified
	sla := NewRollbackSLAManager()
	if ok, _ := sla.VerifyPlan(RollbackPlan{PlanID: "p-ok", ActionType: "custom", SLAThresholdSeconds: 10.0, VerifiedInSandbox: true}, "R6"); !ok {
		t.Fatalf("expected verified plan")
	}

	// DualEscalationEngine branches
	esc := NewDualEscalationEngine(0)
	if r := esc.EvaluateEscalation(EscalationRequest{TaskID: "t-r6", RiskClass: "R6"}); r.Target != "HUMAN_OPERATOR" {
		t.Fatalf("expected HUMAN_OPERATOR")
	}
	if r := esc.EvaluateEscalation(EscalationRequest{TaskID: "t-h", FailedRepairAttempts: 5}); r.Target != "HERMES" {
		t.Fatalf("expected HERMES")
	}

	// InfraReconciler healthy & helpers
	ir := NewInfraReconciler()
	if m := ir.CheckAndRecoverService("svc", true); m["status"] != "HEALTHY" {
		t.Fatalf("expected HEALTHY")
	}
	if !ir.VerifyTCPPort("localhost", 8080) || ir.VerifyTCPPort("", -1) {
		t.Fatalf("unexpected VerifyTCPPort result")
	}
	if !ir.VerifyAlembicCompatibility("rev1", "rev1") {
		t.Fatalf("unexpected VerifyAlembicCompatibility")
	}
	if rb := ir.AutoRollbackDatabase("postgres"); rb["rolled_back"] != true {
		t.Fatalf("unexpected AutoRollbackDatabase")
	}

	// FederatedPromptRegistry layers
	pr := NewFederatedPromptRegistry()
	if err := pr.RegisterBase("   "); err == nil {
		t.Fatalf("expected error on empty base")
	}
	if err := pr.RegisterLayer("team", "core", "s1", ""); err == nil {
		t.Fatalf("expected error on empty layer")
	}
	_ = pr.RegisterLayer("team", "core", "s1", "team-rule")
	_ = pr.RegisterLayer("feature", "search", "s2", "feature-rule")
	comp, _ := pr.CompilePrompt("org1", "core", "search")
	if !strings.Contains(comp, "team-rule") || !strings.Contains(comp, "feature-rule") {
		t.Fatalf("expected team and feature layers in compiled prompt")
	}

	// EmbeddingDriftDetector edge cases
	dd := NewEmbeddingDriftDetector(0)
	if psi, alert := dd.CalculatePSI(nil, nil, 0); psi != 0 || alert {
		t.Fatalf("expected 0 on empty slices")
	}
	if psi, alert := dd.CalculatePSI([]float64{1, 1}, []float64{1, 1}, 0); psi != 0 || alert {
		t.Fatalf("expected 0 on constant slices")
	}

	// InjectW3CTraceContext short IDs & sampled=false
	tpShort := InjectW3CTraceContext("abc", "def", false)
	if !strings.HasSuffix(tpShort, "-00") {
		t.Fatalf("expected -00 suffix, got %s", tpShort)
	}

	// KeycloakAuthValidator error branches
	kc := NewKeycloakAuthValidator("sec", "https://issuer.example.com/")
	if _, err := kc.VerifyJWT("bad-token", ""); err == nil {
		t.Fatalf("expected error on malformed token")
	}
	tok := kc.CreateJWT("u", "t", []string{"r"}, []string{"scope1"})
	if _, err := kc.VerifyJWT(tok+"tampered", ""); err == nil {
		t.Fatalf("expected signature error")
	}
	if _, err := kc.VerifyJWT(tok, "missing_scope"); err == nil {
		t.Fatalf("expected missing capability error")
	}
	kcOther := NewKeycloakAuthValidator("sec", "https://other.example.com")
	if _, err := kcOther.VerifyJWT(tok, ""); err == nil {
		t.Fatalf("expected issuer mismatch error")
	}

	// AutonomyEvaluationHarness helpers
	har := NewAutonomyEvaluationHarness()
	if !har.VerifyCostBounds(1.0, 2.0) {
		t.Fatalf("expected within bounds")
	}
	if har.CalculateRetentionDays("R0") != 90 || har.CalculateRetentionDays("UNKNOWN") != 365 {
		t.Fatalf("unexpected retention days")
	}
	if !har.VerifySimulationMode("shadow") || har.VerifySimulationMode("prod") {
		t.Fatalf("unexpected VerifySimulationMode")
	}
	if res := har.RunFullEvaluationSuite("normal prompt"); res["blocked"] != false {
		t.Fatalf("expected normal prompt allowed")
	}

	// GitOps, PhaseAnalyzer, GlobalSearch, HermesOpenCode, RegisteredPackageTypes
	gd := NewGitOpsDaemon()
	gRes := gd.ExecuteGitOpsWorkflow("", "summary", "rationale", "")
	if gRes.BranchName != "feature/task-001" {
		t.Fatalf("unexpected default branch: %s", gRes.BranchName)
	}
	for _, desc := range []string{"research topic", "arch design", "unit test", "release tag", "code feature"} {
		_ = InferSDLCPhase(desc)
	}
	entries := ConvertLedgerReceiptsToPhaseEntries([]accounting.AuditCostReceipt{
		{TaskIntent: "research", ModelName: "m1", TotalTokens: 10, CostUSD: 0.01, LatencyMs: 5, ASTViolationsCount: 1},
	})
	_ = AnalyzeProjectPhases("", nil)
	_ = AnalyzeProjectPhases("proj", append(entries, PhaseEntry{Phase: "", Model: "m2", Success: false}))

	_ = ExecuteLiveGlobalSearch("unmatched_xyz_query_999", "proj",
		[]models.ScoredMemoryItem{{Memory: models.MemoryItem{ID: "m1", Content: "alpha beta"}, Score: 0}},
		nil, nil, nil,
	)
	_ = ExecuteLiveGlobalSearch("alpha missingword", "proj",
		[]models.ScoredMemoryItem{{Memory: models.MemoryItem{ID: "m1", Content: "alpha beta"}, Score: 0}},
		nil, nil, nil,
	)

	hb := NewHermesOpenCodeBridge()
	_ = hb.Dispatch(HermesOpenCodeRequest{FailedRepairAttempts: 4, DryRun: false, Prompt: "fix"})
	_ = hb.Dispatch(HermesOpenCodeRequest{FailedRepairAttempts: 0, DryRun: false, Prompt: "fix"})

	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
}
