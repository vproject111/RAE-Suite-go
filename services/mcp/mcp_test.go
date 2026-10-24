package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
)

func TestAllSixteenMCPToolsWiredAndStdio(t *testing.T) {
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(nil, guard)
	crlSvc := crl.NewService(store, store, guard)
	svc := NewService(store, crlSvc)

	tools := svc.ListTools()
	if len(tools) != 31 {
		t.Fatalf("expected 31 MCP tools, got %d", len(tools))
	}

	ctx := context.Background()

	// 1-2. Memory store & search
	if _, err := svc.CallTool(ctx, "rae_memory_store", map[string]interface{}{
		"content": "MCP stored memory",
		"layer":   "episodic",
		"project": "mcp-proj",
	}); err != nil {
		t.Fatalf("rae_memory_store failed: %v", err)
	}
	if _, err := svc.CallTool(ctx, "rae_memory_search", map[string]interface{}{
		"query":   "MCP stored",
		"project": "mcp-proj",
	}); err != nil {
		t.Fatalf("rae_memory_search failed: %v", err)
	}

	// 3. Quality audit
	qRes, err := svc.CallTool(ctx, "rae_quality_audit", map[string]interface{}{
		"code": "package main\nfunc main() {}\n",
	})
	if err != nil || qRes["passed"] != true {
		t.Fatalf("rae_quality_audit failed: %+v, err=%v", qRes, err)
	}

	// 4-5. Phoenix impact & repair
	impRes, err := svc.CallTool(ctx, "rae_phoenix_impact", map[string]interface{}{
		"target_file": "service.go",
		"dependents":  []string{"a.go", "b.go"},
	})
	if err != nil || impRes["safe_to_refactor"] != true {
		t.Fatalf("rae_phoenix_impact failed: %+v, err=%v", impRes, err)
	}
	repRes, err := svc.CallTool(ctx, "rae_phoenix_repair", map[string]interface{}{
		"baseline_out": "OK",
		"candidates":   []string{"BAD", "OK"},
	})
	if err != nil || repRes["status"] != "REPAIRED" {
		t.Fatalf("rae_phoenix_repair failed: %+v, err=%v", repRes, err)
	}

	// 6-7. Sandbox exec & parallel run
	if _, err := svc.CallTool(ctx, "rae_sandbox_exec", map[string]interface{}{
		"command": "echo hello-mcp",
	}); err != nil {
		t.Fatalf("rae_sandbox_exec failed: %v", err)
	}
	runRes, err := svc.CallTool(ctx, "rae_sandbox_run", map[string]interface{}{
		"commands": []string{"echo b1", "echo b2"},
	})
	if err != nil || runRes["branch_count"] != 2 {
		t.Fatalf("rae_sandbox_run failed: %+v, err=%v", runRes, err)
	}

	// 8-9. Cluster dispatch & consolidate traces
	if _, err := svc.CallTool(ctx, "rae_cluster_dispatch", map[string]interface{}{
		"node": "node1-lumina",
		"task": "benchmark",
	}); err != nil {
		t.Fatalf("rae_cluster_dispatch failed: %v", err)
	}
	consRes, err := svc.CallTool(ctx, "rae_consolidate_traces", map[string]interface{}{
		"project": "mcp-proj",
	})
	if err != nil || consRes["consolidated_count"].(int) < 1 {
		t.Fatalf("rae_consolidate_traces failed: %+v, err=%v", consRes, err)
	}

	// 10-16. RAE-CRL 7 tools
	qnRes, err := svc.CallTool(ctx, "quick_note", map[string]interface{}{
		"content": "Initial trace note",
		"project": "mcp-proj",
	})
	if err != nil {
		t.Fatalf("quick_note failed: %v", err)
	}
	// Verify list_my_traces
	trRes, err := svc.CallTool(ctx, "list_my_traces", map[string]interface{}{
		"project": "mcp-proj",
	})
	if err != nil || trRes["count"].(int) != 1 {
		t.Fatalf("list_my_traces failed: %+v, err=%v", trRes, err)
	}

	// Extract trace ID via CRL service
	traces, _ := crlSvc.ListTraces(ctx, "mcp-proj", "")
	traceID := traces[0].ID

	if _, err := svc.CallTool(ctx, "refine_artifact", map[string]interface{}{
		"artifact_id": traceID,
		"new_type":    "experiment",
		"title":       "Exp 1",
		"description": "Test experiment",
	}); err != nil {
		t.Fatalf("refine_artifact failed: %v", err)
	}
	if _, err := svc.CallTool(ctx, "record_failure", map[string]interface{}{
		"experiment_id": traceID,
		"reason":        "Divergent gradient",
	}); err != nil {
		t.Fatalf("record_failure failed: %v", err)
	}
	if _, err := svc.CallTool(ctx, "fork_artifact", map[string]interface{}{
		"original_id": traceID,
		"new_author":  "agent-2",
		"reason":      "Alternative learning rate",
	}); err != nil {
		t.Fatalf("fork_artifact failed: %v", err)
	}
	if _, err := svc.CallTool(ctx, "request_approval", map[string]interface{}{
		"artifact_id": traceID,
		"proposer":    "agent-1",
	}); err != nil {
		t.Fatalf("request_approval failed: %v", err)
	}
	if _, err := svc.CallTool(ctx, "approve_artifact", map[string]interface{}{
		"artifact_id":   traceID,
		"approver_name": "PI-Lead",
	}); err != nil {
		t.Fatalf("approve_artifact failed: %v", err)
	}
	_ = qnRes

	// JSON-RPC Stdio test
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var out strings.Builder
	if err := svc.ServeStdio(ctx, in, &out); err != nil || !strings.Contains(out.String(), "rae_memory_store") {
		t.Fatalf("ServeStdio failed: out=%q, err=%v", out.String(), err)
	}
}

func TestMCPInterceptorAllToolsStdioAndHTTPRoutes(t *testing.T) {
	ctx := context.Background()
	guard := security.NewHardFramesGuard("mcp-secret")
	store := memorystore.NewUnifiedStore(nil, guard)
	crlSvc := crl.NewService(store, store, guard)
	svc := NewService(store, crlSvc)

	// 1. MaskEmail & ScrubPII
	maskedShort := MaskEmail("ab@domain.com and grzegorz@dreamsoft.pro")
	if !strings.Contains(maskedShort, "**@domain.com") || !strings.Contains(maskedShort, "gr***@dreamsoft.pro") {
		t.Fatalf("unexpected MaskEmail output: %s", maskedShort)
	}
	scrubbed := ScrubPII("contact john@example.com with token sk-1234567890abcdefghij")
	if !strings.Contains(scrubbed, "[REDACTED_SECRET]") || !strings.Contains(scrubbed, "jo***@example.com") {
		t.Fatalf("unexpected ScrubPII output: %s", scrubbed)
	}

	// 2. PassiveMemoryInterceptor
	ic := svc.Interceptor()
	san := ic.SanitizeArguments(map[string]interface{}{
		"api_key":  "secret-123",
		"password": "pw",
		"email":    "alice@example.com",
		"count":    42,
	})
	if san["api_key"] != "[REDACTED]" || san["password"] != "[REDACTED]" || san["count"] != 42 {
		t.Fatalf("unexpected SanitizeArguments: %+v", san)
	}
	sumDefault := ic.SummarizeResult(map[string]interface{}{"msg": "short"}, 0)
	if !strings.Contains(sumDefault, "short") {
		t.Fatalf("unexpected SummarizeResult default: %s", sumDefault)
	}
	sumTrunc := ic.SummarizeResult(map[string]interface{}{"msg": strings.Repeat("x", 100)}, 20)
	if !strings.HasSuffix(sumTrunc, "...[truncated]") {
		t.Fatalf("expected truncated summary, got: %s", sumTrunc)
	}
	if _, err := ic.CaptureTrace(ctx, "rae_memory_store", map[string]interface{}{"token": "abc"}, map[string]interface{}{"ok": true}); err != nil {
		t.Fatalf("CaptureTrace failed: %v", err)
	}
	nilIc := NewPassiveMemoryInterceptor(nil)
	if c, err := nilIc.ConsolidateTraces(ctx, "p"); err != nil || c != 0 {
		t.Fatalf("expected 0 from nil memory ConsolidateTraces, got %d, %v", c, err)
	}

	// 3. sliceStringArg edge cases & remaining tools
	_, _ = svc.CallTool(ctx, "save_memory", map[string]interface{}{
		"content": "user email bob@example.com",
		"tags":    []interface{}{"tag1", "tag2", 123},
		"project": "mcp-proj",
	})
	_, _ = svc.CallTool(ctx, "save_memory", map[string]interface{}{
		"content": "restricted note",
		"layer":   "working",
		"info_class": "restricted",
		"tags":    "invalid-slice-type",
		"project": "mcp-proj",
	})
	_, _ = svc.CallTool(ctx, "rae_consolidate_traces", map[string]interface{}{"project": "mcp-proj"})

	// Exercise remaining tools in CallTool
	toolCalls := []struct {
		name string
		args map[string]interface{}
	}{
		{"rae_quality_rpc", map[string]interface{}{"operation": "quality.get_policy", "params": map[string]interface{}{"task_type": "code_edit"}}},
		{"rae_quality_rpc", map[string]interface{}{"operation": "quality_get_policy"}},
		{"rae_quality_rpc", map[string]interface{}{"operation": "audit", "code": "package main\nfunc main() {}\n"}},
		{"rae_sandbox_run", map[string]interface{}{"command": "echo single-branch"}},
		{"rae_kernel_execute", map[string]interface{}{"task": "verify kernel", "tier": "tier1_generate"}},
		{"rae_knowledge_resolve", map[string]interface{}{"symbol": "bob", "project": "mcp-proj"}},
		{"rae_lab_insights", map[string]interface{}{"arm": "hyper_37"}},
		{"request_approval", map[string]interface{}{"artifact_id": "nonexistent-art-id", "proposer": "tester"}},
		{"check_approval_status", map[string]interface{}{"artifact_id": "nonexistent-art-id"}},
		{"check_approval_status", map[string]interface{}{"artifact_id": "unknown-id"}},
		{"get_cloud_status", nil},
		{"run_diagnostic", nil},
		{"get_service_logs", map[string]interface{}{"service": "rae-mcp"}},
		{"get_circuit_breakers", nil},
		{"get_related_context", map[string]interface{}{"query": "bob", "project": "mcp-proj"}},
		{"list_policies", nil},
		{"get_risk_coverage", nil}, // delegated to qualitySvc.CallQualityMCPTool
	}
	for _, tc := range toolCalls {
		if _, err := svc.CallTool(ctx, tc.name, tc.args); err != nil {
			t.Fatalf("CallTool(%s) failed: %v", tc.name, err)
		}
	}

	// Error paths in CallTool
	if _, err := svc.CallTool(ctx, "rae_sandbox_exec", map[string]interface{}{"command": ""}); err == nil {
		t.Fatal("expected error on empty command in rae_sandbox_exec")
	}
	_, _ = svc.CallTool(ctx, "rae_sandbox_exec", map[string]interface{}{"command": "false"})
	if _, err := svc.CallTool(ctx, "unknown_tool_xyz", nil); err == nil {
		t.Fatal("expected error for unknown tool")
	}

	// Ring-buffer log truncation (>200 calls)
	for i := 0; i < 205; i++ {
		_, _ = svc.CallTool(ctx, "get_cloud_status", nil)
	}

	// 4. Prompts & Resources via ServeStdio
	rpcInputs := []string{
		`{invalid-json}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":2,"method":"prompts/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"prompts/get","params":{"name":"rae-context-injection","arguments":{"project":"p1"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"rae://memory/stats"}}`,
		`{"jsonrpc":"2.0","id":6,"method":"resources/read","params":{"uri":"rae://governance/policies"}}`,
		`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_cloud_status","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}`,
	}
	var stdioOut strings.Builder
	if err := svc.ServeStdio(ctx, strings.NewReader(strings.Join(rpcInputs, "\n")+"\n"), &stdioOut); err != nil {
		t.Fatalf("ServeStdio failed: %v", err)
	}
	if !strings.Contains(stdioOut.String(), "rae-context-injection") || !strings.Contains(stdioOut.String(), "-32000") {
		t.Fatalf("unexpected ServeStdio output: %s", stdioOut.String())
	}

	// 5. HTTP Routes (:8005)
	// Test HTTP routes in a separate helper function below
	testMCPHTTPRoutes(t, svc)
}

func testMCPHTTPRoutes(t *testing.T, svc *Service) {
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	for _, p := range []string{"/health", "/v2/mcp/tools", "/v2/mcp/prompts", "/v2/mcp/resources", "/mcp/sse"} {
		req := httptest.NewRequest(http.MethodGet, p, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d", p, w.Code)
		}
	}

	// POST /v2/mcp/call valid, bad JSON, and tool error
	reqCall := httptest.NewRequest(http.MethodPost, "/v2/mcp/call", strings.NewReader(`{"name":"get_cloud_status","arguments":{}}`))
	wCall := httptest.NewRecorder()
	mux.ServeHTTP(wCall, reqCall)
	if wCall.Code != http.StatusOK {
		t.Fatalf("POST /v2/mcp/call status=%d", wCall.Code)
	}

	reqBadJSON := httptest.NewRequest(http.MethodPost, "/v2/mcp/call", strings.NewReader(`{bad`))
	wBadJSON := httptest.NewRecorder()
	mux.ServeHTTP(wBadJSON, reqBadJSON)
	if wBadJSON.Code != http.StatusBadRequest {
		t.Fatalf("POST /v2/mcp/call bad JSON status=%d", wBadJSON.Code)
	}

	reqErrTool := httptest.NewRequest(http.MethodPost, "/v2/mcp/call", strings.NewReader(`{"name":"unknown_tool"}`))
	wErrTool := httptest.NewRecorder()
	mux.ServeHTTP(wErrTool, reqErrTool)
	if wErrTool.Code != http.StatusBadRequest {
		t.Fatalf("POST /v2/mcp/call unknown tool status=%d", wErrTool.Code)
	}

	// POST /mcp/messages
	reqMsg := httptest.NewRequest(http.MethodPost, "/mcp/messages", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n"))
	wMsg := httptest.NewRecorder()
	mux.ServeHTTP(wMsg, reqMsg)
	if wMsg.Code != http.StatusOK || !strings.Contains(wMsg.Body.String(), "rae_memory_store") {
		t.Fatalf("POST /mcp/messages status=%d, body=%s", wMsg.Code, wMsg.Body.String())
	}
}
