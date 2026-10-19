package orchestrator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAllCoreEngineHTTPRoutes(t *testing.T) {
	svc := NewService(nil, nil, nil, nil)
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	postJSON := func(path string, body interface{}) (*httptest.ResponseRecorder, map[string]interface{}) {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(raw))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var out map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}

	getJSON := func(path string) (*httptest.ResponseRecorder, map[string]interface{}) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		var out map[string]interface{}
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec, out
	}

	// 1. Audit Ledger record, summary, receipts, verify, project-summary
	rec, out := postJSON("/v2/audit/ledger/record", map[string]interface{}{
		"model_name":        "z-ai/glm-5.3-flash",
		"provider":          "openrouter",
		"user_identity":     "grzegorz@cloud",
		"task_intent":       "Port BatchOptimizationEngine to Go",
		"rationale":         "Ensure 100% parity with Python core",
		"decision":          "APPROVED",
		"prompt_tokens":     800,
		"completion_tokens": 400,
		"cost_usd":          0.00012,
		"latency_ms":        35.4,
		"quality_effect":    "PASSED_ALL_TESTS",
		"project":           "RAE-Suite-go",
	})
	if rec.Code != http.StatusOK || out["status"] != "recorded" {
		t.Fatalf("expected 200 recorded from /v2/audit/ledger/record, got %d: %v", rec.Code, out)
	}

	rec, out = getJSON("/v2/audit/ledger/summary")
	if rec.Code != http.StatusOK || out["total_invocations"].(float64) < 1 {
		t.Fatalf("unexpected /v2/audit/ledger/summary: %v", out)
	}

	rec, out = getJSON("/v2/audit/ledger/receipts")
	if rec.Code != http.StatusOK || out["count"].(float64) < 1 {
		t.Fatalf("unexpected /v2/audit/ledger/receipts: %v", out)
	}

	rec, out = postJSON("/v2/audit/receipt/verify", map[string]interface{}{
		"receipt_id":            "rcpt-verify-1",
		"task_id":               "task-1",
		"step_id":               "step-1",
		"execution_status":      "VERIFIED_SUCCESS",
		"git_diff_hash":         "abcdef1234567890",
		"capability_compliance": true,
		"test_result": map[string]interface{}{
			"command":      "go test ./...",
			"exit_code":    0,
			"passed_count": 32,
			"failed_count": 0,
		},
	})
	if rec.Code != http.StatusOK || out["valid"] != true {
		t.Fatalf("expected valid execution receipt, got %d: %v", rec.Code, out)
	}

	rec, out = getJSON("/v2/audit/project-summary?project=RAE-Suite-go")
	if rec.Code != http.StatusOK || out["project_id"] != "RAE-Suite-go" {
		t.Fatalf("unexpected /v2/audit/project-summary: %v", out)
	}

	// 2. Batch Optimization Engine (Knowledge Scale Effect + Warm Agent Dispatch)
	_, _ = postJSON("/v2/batch/tasks", map[string]interface{}{
		"task_id": "t1", "action_type": "refactor", "module": "modA", "target_file": "shared.go", "token_cost": 500,
	})
	_, _ = postJSON("/v2/batch/tasks", map[string]interface{}{
		"task_id": "t2", "action_type": "test", "module": "modB", "target_file": "shared.go", "token_cost": 600,
	})
	rec, out = postJSON("/v2/batch/optimize", map[string]interface{}{})
	if rec.Code != http.StatusOK || out["batches_count"].(float64) < 1 {
		t.Fatalf("expected coalesced batches via Knowledge Scale Effect, got %v", out)
	}
	batches := out["batches"].([]interface{})
	rec, out = postJSON("/v2/batch/dispatch", map[string]interface{}{
		"batch":    batches[0],
		"agent_id": "agent-1",
	})
	if rec.Code != http.StatusOK || out["tasks_completed"].(float64) != 2 {
		t.Fatalf("unexpected /v2/batch/dispatch output: %v", out)
	}
	rec, out = getJSON("/v2/batch/savings")
	if rec.Code != http.StatusOK || out["tokens_saved"].(float64) <= 0 {
		t.Fatalf("expected positive tokens_saved in /v2/batch/savings, got %v", out)
	}

	// 3. Cognition: Plan, Council, Consensus
	rec, out = postJSON("/v2/cognition/plan", map[string]interface{}{
		"intent":     "Refactor security module",
		"context":    "Strict ISO 42001 mode",
		"risk_class": "R4",
	})
	if rec.Code != http.StatusOK || out["plan_status"] != "OPTIMAL_PLAN_FOUND" {
		t.Fatalf("unexpected /v2/cognition/plan: %v", out)
	}

	rec, out = postJSON("/v2/cognition/council", map[string]interface{}{
		"task_class": "arch_review",
		"risk_tier":  3,
	})
	if rec.Code != http.StatusOK || len(out["quorum"].([]interface{})) != 3 {
		t.Fatalf("expected 3 models in Tier 3 council quorum, got %v", out)
	}

	rec, out = postJSON("/v2/cognition/consensus", map[string]interface{}{
		"votes": []map[string]interface{}{
			{"agent_role": "rae-hive", "decision": "APPROVE", "confidence": 0.9, "evidence_count": 3},
			{"agent_role": "rae-quality", "decision": "APPROVE", "confidence": 0.95, "evidence_count": 5},
		},
	})
	if rec.Code != http.StatusOK || out["status"] != "CONSENSUS_APPROVED" {
		t.Fatalf("unexpected /v2/cognition/consensus: %v", out)
	}

	// 4. Context Economy & Trust
	rec, out = postJSON("/v2/context/depth", map[string]interface{}{
		"initial_scores": []float64{0.92, 0.70, 0.65},
	})
	if rec.Code != http.StatusOK || out["k"].(float64) != 3 {
		t.Fatalf("unexpected /v2/context/depth: %v", out)
	}

	rec, out = postJSON("/v2/context/prune", map[string]interface{}{
		"constitution": "HARD RULES",
		"envelopes": []map[string]interface{}{
			{"context_id": "c1", "memory_layer": "semantic", "trust_score": 0.95, "retrieved_content": "Leaf 1"},
		},
	})
	if rec.Code != http.StatusOK || !strings.Contains(out["pruned_context"].(string), "HARD RULES") {
		t.Fatalf("unexpected /v2/context/prune: %v", out)
	}

	rec, out = postJSON("/v2/context/trust", map[string]interface{}{
		"metadata": map[string]interface{}{
			"information_class":       "internal",
			"layer":                   "semantic",
			"historical_success_rate": 0.9,
			"risk_class":              "R2",
		},
	})
	if rec.Code != http.StatusOK || out["classification"] != "USABLE_PLANNING_INPUT" {
		t.Fatalf("unexpected /v2/context/trust: %v", out)
	}

	// 5. Workflows: Saga, Stream, Speculative, ClaimCheck, Replay, A2A
	rec, out = postJSON("/v2/workflows/saga", map[string]interface{}{
		"steps":        []string{"s1", "s2", "s3"},
		"fail_at_step": "s3",
	})
	if rec.Code != http.StatusOK || out["status"] != "SAGA_ROLLED_BACK" {
		t.Fatalf("unexpected /v2/workflows/saga: %v", out)
	}

	rec, out = postJSON("/v2/workflows/stream", map[string]interface{}{
		"chunks": []string{
			`STEP: {"tool":"search_code","args":{"query":"BatchEngine"}}`,
			`STEP: {"tool":"verify_match","args":{"target":"$prev_output"}}`,
		},
	})
	if rec.Code != http.StatusOK || out["steps_executed"].(float64) != 2 {
		t.Fatalf("unexpected /v2/workflows/stream: %v", out)
	}

	rec, out = postJSON("/v2/workflows/speculative", map[string]interface{}{
		"commands": []string{"git status", "rm -rf /"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected /v2/workflows/speculative status: %d", rec.Code)
	}

	rec, out = postJSON("/v2/workflows/claim-check", map[string]interface{}{
		"payload":  strings.Repeat("LARGE-BLOB-", 2000),
		"retrieve": true,
	})
	if rec.Code != http.StatusOK || out["offloaded"] != true || out["integrity_verified"] != true {
		t.Fatalf("unexpected /v2/workflows/claim-check: %v", out)
	}

	rec, out = postJSON("/v2/workflows/replay", map[string]interface{}{
		"step_data": map[string]interface{}{"step_id": "st-1", "action": "deploy", "side_effect": true},
		"mode":      "SIMULATE",
	})
	if rec.Code != http.StatusOK || out["status"] != "SIMULATED" {
		t.Fatalf("unexpected /v2/workflows/replay: %v", out)
	}

	rec, out = postJSON("/v2/a2a/send", map[string]interface{}{
		"sender_id": "rae-suite",
		"recipient": "rae-hive",
		"nonce":     "test-nonce-999",
		"payload":   map[string]interface{}{"cmd": "ping"},
	})
	if rec.Code != http.StatusOK || out["status"] != "A2A_DELIVERED" {
		t.Fatalf("unexpected /v2/a2a/send: %v", out)
	}

	// 6. Autonomy & Runtime
	rec, out = postJSON("/v2/autonomy/shadow", map[string]interface{}{
		"failure_pattern":     "panic: nil pointer",
		"test_input":          "Observed panic: nil pointer in worker",
		"promote":             true,
		"hours_active":        80.0,
		"false_positive_rate": 0.0001,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected /v2/autonomy/shadow: %v", out)
	}

	rec, out = postJSON("/v2/autonomy/curiosity", map[string]interface{}{
		"hypothesis": "Warm agent pooling saves 15k tokens per task",
		"module":     "economy",
		"run_idle":   true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected /v2/autonomy/curiosity: %v", out)
	}

	rec, out = postJSON("/v2/autonomy/tool-forge", map[string]interface{}{
		"tool_name": "calc_stats",
		"code":      "def calc(x): return x * 2",
	})
	if rec.Code != http.StatusOK || out["status"] != "SYNTHESIZED_VERIFIED" {
		t.Fatalf("unexpected /v2/autonomy/tool-forge: %v", out)
	}

	rec, out = postJSON("/v2/runtime/device-profile", map[string]interface{}{
		"total_ram_mb":  2048,
		"has_local_gpu": false,
	})
	if rec.Code != http.StatusOK || out["profile"] != "THIN_MOBILE" {
		t.Fatalf("unexpected /v2/runtime/device-profile: %v", out)
	}
}
