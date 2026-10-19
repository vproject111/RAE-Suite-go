package mcp

import (
	"context"
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
