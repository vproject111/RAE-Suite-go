package lab

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestLabExperimentAndAnalyzeFailures(t *testing.T) {
	svc := NewService(nil)
	svc.RegisterScan(false, "SEC-CMD-INJECTION")
	exp := svc.RunExperiment("hyper_37", 0.95)
	if exp["status"] != "completed" {
		t.Fatalf("unexpected experiment result: %+v", exp)
	}
	analysis := svc.AnalyzeFailures()
	if analysis["total_failures_analyzed"] != 1 {
		t.Fatalf("unexpected failure analysis: %+v", analysis)
	}
}

func TestLabMiningRolloutAndAllRoutes(t *testing.T) {
	mc := raemath.NewMathController(models.MathL3)
	svc := NewService(mc)

	// MineFailurePatterns with empty traces & no candidates (uses default sample traces)
	defClusters := svc.MineFailurePatterns(nil)
	if len(defClusters) == 0 {
		t.Fatalf("expected default clusters when traces and candidates are empty")
	}

	// Register passed & failed scans
	svc.RegisterScan(true, "")
	svc.RegisterScan(false, "COMPLEXITY_HIGH")
	svc.RegisterScan(false, "HARDCODED_PATH")

	// MineFailurePatterns with empty traces (now pulls from candidates)
	candClusters := svc.MineFailurePatterns(nil)
	if len(candClusters) < 2 {
		t.Fatalf("expected candidate-derived clusters, got %+v", candClusters)
	}

	// MineFailurePatterns with all trace signature categories
	allSigClusters := svc.MineFailurePatterns([]string{
		"AST complexity spike",
		"hardcoded path /home/user",
		"api secret key leaked",
		"context deadline timeout exceeded",
		"some other invariant drift",
	})
	if len(allSigClusters) != 5 {
		t.Fatalf("expected 5 distinct signature clusters, got %d", len(allSigClusters))
	}

	// RunExperiment with empty armID
	expDefault := svc.RunExperiment("", 0.85)
	if expDefault["arm_tested"] != "hyper_37" {
		t.Fatalf("expected default arm_tested=hyper_37, got %v", expDefault["arm_tested"])
	}

	// GetCandidates with filter
	candsActive := svc.GetCandidates("ACTIVE")
	if len(candsActive) != 0 {
		t.Fatalf("expected 0 ACTIVE candidates")
	}
	candsCand := svc.GetCandidates("CANDIDATE")
	if len(candsCand) != 2 {
		t.Fatalf("expected 2 CANDIDATE guardrails, got %d", len(candsCand))
	}

	// EvaluateSafeRollout across all stages and rollback conditions
	rRollbackErr := EvaluateSafeRollout("", 0, 0.05, 0, 1.0)
	if rRollbackErr.Action != "AUTO_ROLLBACK" || rRollbackErr.RegressionPass {
		t.Fatalf("expected AUTO_ROLLBACK on error rate, got %+v", rRollbackErr)
	}
	rRollbackLat := EvaluateSafeRollout("feat-x", 25, 0.001, 0.02, 30.0)
	if rRollbackLat.Action != "AUTO_ROLLBACK" {
		t.Fatalf("expected AUTO_ROLLBACK on latency delta, got %+v", rRollbackLat)
	}
	rStage1 := EvaluateSafeRollout("feat-1", 1, 0.001, 0.02, 2.0)
	if rStage1.NextStagePct != 5 || rStage1.Action != "PROMOTE_STAGE" {
		t.Fatalf("expected NextStagePct=5, got %+v", rStage1)
	}
	rStage5 := EvaluateSafeRollout("feat-5", 5, 0.001, 0.02, 2.0)
	if rStage5.NextStagePct != 25 {
		t.Fatalf("expected NextStagePct=25, got %+v", rStage5)
	}
	rStage25 := EvaluateSafeRollout("feat-25", 25, 0.001, 0.02, 2.0)
	if rStage25.NextStagePct != 50 {
		t.Fatalf("expected NextStagePct=50, got %+v", rStage25)
	}
	rStage50 := EvaluateSafeRollout("feat-50", 50, 0.001, 0.02, 2.0)
	if rStage50.NextStagePct != 100 {
		t.Fatalf("expected NextStagePct=100, got %+v", rStage50)
	}

	// Test all HTTP routes
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	routes := []struct {
		method string
		path   string
		body   string
		expect string
	}{
		{http.MethodGet, "/health", "", "fully_operational"},
		{http.MethodGet, "/mcp/sse", "", "/mcp/messages"},
		{http.MethodPost, "/mcp/messages", `{"id":10}`, "RAE-Lab"},
		{http.MethodGet, "/api/mab/weights", "", "alpha"},
		{http.MethodGet, "/api/guardrails/candidates?status=CANDIDATE", "", "candidates"},
		{http.MethodPost, "/api/scan", `{"passed":false,"failure_reason":"PATH_LEAK"}`, "recorded"},
		{http.MethodPost, "/lab/experiment/run", `{"arm_id":"","reward":0.9}`, "completed"},
		{http.MethodGet, "/lab/analyze/failures", "", "promote_high_frequency_candidates"},
		{http.MethodGet, "/v2/lab/insights", "", "candidates_count"},
		{http.MethodPost, "/v1/mining/failures", `{"traces":["secret key exposed","timeout error"]}`, "cluster_count"},
		{http.MethodPost, "/v1/rollout/evaluate", `{"feature_id":"f1","current_stage_pct":5,"observed_error_rate":0.001,"max_allowed_error_rate":0.02,"latency_delta_pct":1.0}`, "PROMOTE_STAGE"},
	}

	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s returned %d", tc.method, tc.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.expect) {
			t.Fatalf("%s %s body %q missing %q", tc.method, tc.path, rec.Body.String(), tc.expect)
		}
	}
}
