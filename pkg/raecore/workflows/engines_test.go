package workflows

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestWorkflowsA2AScrubberAndAutonomyEngines(t *testing.T) {
	// 1. Saga Coordinator rollback test
	saga := NewSagaCoordinator()
	var compensated []string
	saga.AddStep("step_db_write", func() (any, error) { return "ok", nil }, func() error {
		compensated = append(compensated, "step_db_write")
		return nil
	})
	saga.AddStep("step_mesh_publish", func() (any, error) { return "ok", nil }, func() error {
		compensated = append(compensated, "step_mesh_publish")
		return nil
	})
	saga.AddStep("step_failing_deploy", func() (any, error) { return nil, fmt.Errorf("deploy timeout") }, func() error {
		compensated = append(compensated, "step_failing_deploy")
		return nil
	})
	sagaRes := saga.ExecuteSaga()
	if sagaRes.Status != "SAGA_ROLLED_BACK" {
		t.Fatalf("expected SAGA_ROLLED_BACK, got %s", sagaRes.Status)
	}
	if len(sagaRes.RolledBackSteps) != 2 || sagaRes.RolledBackSteps[0] != "step_mesh_publish" || sagaRes.RolledBackSteps[1] != "step_db_write" {
		t.Fatalf("expected reverse rollback order [step_mesh_publish, step_db_write], got %v", sagaRes.RolledBackSteps)
	}

	// 2. Streaming Function Composer
	composer := NewStreamingFunctionComposer(map[string]ToolHandler{
		"grep_symbol": func(args map[string]interface{}) (interface{}, error) {
			return "pkg/raecore/economy/engine.go:42", nil
		},
		"inspect_line": func(args map[string]interface{}) (interface{}, error) {
			return fmt.Sprintf("inspected(%v)", args["location"]), nil
		},
	})
	streamRes := composer.PipeSteps([]string{
		`Thought... STEP: {"tool":"grep_symbol","args":{"q":"BatchOptimizationEngine"}}`,
		`Next... STEP: {"tool":"inspect_line","args":{"location":"$prev_output"}}`,
	})
	if len(streamRes) != 2 || streamRes[1].Output != "inspected(pkg/raecore/economy/engine.go:42)" {
		t.Fatalf("unexpected streaming composer result: %+v", streamRes)
	}

	// 3. Speculative Tool Executor
	specExec := NewSpeculativeToolExecutor(nil)
	specRes := specExec.ExecuteSpeculatively([]string{
		"git status",
		"rm -rf /tmp/foo",
		"ls -la",
	})
	if specRes["git status"].Status != "SUCCESS" {
		t.Fatalf("expected git status SUCCESS, got %+v", specRes["git status"])
	}
	if specRes["rm -rf /tmp/foo"].Status != "BLOCKED_UNSAFE" {
		t.Fatalf("expected rm -rf BLOCKED_UNSAFE, got %+v", specRes["rm -rf /tmp/foo"])
	}

	// 4. ClaimCheck Manager
	claimDir := t.TempDir()
	claimMgr := NewClaimCheckManager(claimDir)
	largePayload := strings.Repeat("RAE-CLAIM-CHECK-PAYLOAD-", 1000) // ~24 KB > 16 KB
	pointer := claimMgr.OffloadIfNeeded(largePayload)
	if !strings.HasPrefix(pointer, "<CLAIM_CHECK:") {
		t.Fatalf("expected claim check pointer, got %s", pointer[:32])
	}
	restored, err := claimMgr.Retrieve(pointer)
	if err != nil || restored != largePayload {
		t.Fatalf("failed to restore claim check payload: err=%v", err)
	}

	// 5. Safe Replay Engine
	replay := NewSafeReplayEngine()
	simOut, err := replay.ReplayStep(map[string]interface{}{
		"step_id":     "step-101",
		"action":      "git push origin main",
		"side_effect": true,
	}, ReplaySimulate, "")
	if err != nil || simOut["status"] != "SIMULATED" || simOut["side_effect_blocked"] != true {
		t.Fatalf("unexpected simulate replay output: %+v (err=%v)", simOut, err)
	}
	if _, err := replay.ReplayStep(map[string]interface{}{"step_id": "step-101"}, ReplayExecuteAuthorized, ""); err == nil {
		t.Fatalf("expected error when EXECUTE_AUTHORIZED has empty authorizationToken")
	}

	// 6. A2A Protocol Adapter
	a2a := NewA2AProtocolAdapter()
	a2a.RegisterAgent("agent-ceo", "top-secret-hmac-key")
	env, err := a2a.SignMessage("agent-ceo", "agent-hive", "HS256", "nonce-001", time.Now().Unix(), map[string]interface{}{
		"task": "verify-contracts",
	})
	if err != nil {
		t.Fatalf("SignMessage failed: %v", err)
	}
	recv, err := a2a.VerifyAndReceive(env)
	if err != nil || recv["status"] != "A2A_DELIVERED" {
		t.Fatalf("VerifyAndReceive failed: %v", err)
	}
	// Replay with same nonce must fail
	if _, err := a2a.VerifyAndReceive(env); err == nil {
		t.Fatalf("expected anti-replay error on duplicate nonce")
	}

	// 7. Ingestion PII Scrubber
	scrubber := NewIngestionPIIScrubber()
	scrubbed := scrubber.ScrubText("Contact admin@example.com with api_key=sk_live_1234567890abcdef and JWT eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")
	if scrubbed.RedactionsCount < 3 || strings.Contains(scrubbed.ScrubbedText, "admin@example.com") || strings.Contains(scrubbed.ScrubbedText, "sk_live_1234567890abcdef") {
		t.Fatalf("expected PII redactions, got %+v", scrubbed)
	}

	// 8. Shadow Evaluator
	shadow := NewShadowEvaluator()
	rule := shadow.MineFailure("SQL injection DROP TABLE")
	matched := shadow.EvaluateShadowGuardrails("Attempted sql injection drop table users")
	if len(matched) != 1 || matched[0] != rule.RuleID {
		t.Fatalf("expected shadow guardrail match, got %v", matched)
	}
	if _, err := shadow.PromoteGuardrail(rule.RuleID, 24.0, 0.0001); err == nil {
		t.Fatalf("expected error promoting guardrail before 72h")
	}
	promotedRule, err := shadow.PromoteGuardrail(rule.RuleID, 75.0, 0.0002)
	if err != nil || promotedRule.Status != "HARD_ENFORCED" {
		t.Fatalf("expected HARD_ENFORCED after 75h, got %+v (err=%v)", promotedRule, err)
	}

	// 9. Curiosity Engine, Tool Forge & Thin Device Runtime
	curiosity := NewCuriosityEngine()
	curiosity.EnqueueIdleVerification("cur-1", "Batching reduces context switches by 60%", "economy")
	promotedItems := curiosity.RunIdleCycle()
	if len(promotedItems) != 1 || promotedItems[0].Status != "VERIFIED_SEMANTIC_RULE" {
		t.Fatalf("unexpected curiosity idle cycle result: %+v", promotedItems)
	}

	forge := NewToolForge()
	if res := forge.SynthesizeTool("bad_tool", "import os; os.system('rm -rf /')"); res.Status != "REJECTED_UNSAFE" {
		t.Fatalf("expected REJECTED_UNSAFE for os.system, got %+v", res)
	}
	if res := forge.SynthesizeTool("safe_tool", "def add(a, b): return a + b"); res.Status != "SYNTHESIZED_VERIFIED" {
		t.Fatalf("expected SYNTHESIZED_VERIFIED for safe_tool, got %+v", res)
	}

	runtime := NewThinDeviceRuntime()
	if prof := runtime.DetectHardwareProfile(2048, false); prof.Profile != "THIN_MOBILE" {
		t.Fatalf("expected THIN_MOBILE for 2048MB, got %+v", prof)
	}
	if prof := runtime.DetectHardwareProfile(65536, true); prof.Profile != "MESH_NODE" {
		t.Fatalf("expected MESH_NODE for 64GB+GPU, got %+v", prof)
	}
}

func TestWorkflowsAdditionalBranchesAndFailureModes(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	// Saga success & compensation error
	sagaOK := NewSagaCoordinator()
	sagaOK.AddStep("s1", func() (any, error) { return "ok", nil }, nil)
	if res := sagaOK.ExecuteSaga(); res.Status != "SAGA_COMPLETED" {
		t.Fatalf("expected SAGA_COMPLETED, got %+v", res)
	}
	sagaCompFail := NewSagaCoordinator()
	sagaCompFail.AddStep("s1", func() (any, error) { return "ok", nil }, func() error { return fmt.Errorf("comp failed") })
	sagaCompFail.AddStep("s2", func() (any, error) { return nil, fmt.Errorf("step2 failed") }, nil)
	if res := sagaCompFail.ExecuteSaga(); res.Status != "SAGA_COMPENSATION_FAILED" {
		t.Fatalf("expected SAGA_COMPENSATION_FAILED, got %+v", res)
	}

	// StreamingFunctionComposer nil map, unknown tool, tool error
	compNil := NewStreamingFunctionComposer(nil)
	resUnk := compNil.PipeSteps([]string{`STEP: {"tool":"missing","args":{}}`})
	if len(resUnk) != 1 || resUnk[0].Status != "FAILED" {
		t.Fatalf("expected FAILED for unknown tool, got %+v", resUnk)
	}
	compErr := NewStreamingFunctionComposer(map[string]ToolHandler{
		"fail_tool": func(args map[string]interface{}) (interface{}, error) {
			return nil, fmt.Errorf("tool boom")
		},
	})
	resErr := compErr.PipeSteps([]string{`STEP: {"tool":"fail_tool","args":{"k":"v"}}`})
	if len(resErr) != 1 || resErr[0].Status != "ERROR" {
		t.Fatalf("expected ERROR for failing tool, got %+v", resErr)
	}

	// SpeculativeToolExecutor with custom failing runner, >3 commands, empty command, non-whitelisted command
	failSpec := NewSpeculativeToolExecutor(func(cmd string) (string, error) {
		return "", fmt.Errorf("runner error")
	})
	if failSpec.IsCommandSafe("") || failSpec.IsCommandSafe("python3 script.py") {
		t.Fatalf("expected empty or python3 command to be unsafe")
	}
	specMap := failSpec.ExecuteSpeculatively([]string{"ls", "cat file", "wc -l", "head -n 5"})
	if len(specMap) != 3 || specMap["ls"].Status != "FAILED" {
		t.Fatalf("expected capped 3 commands with FAILED runner status, got %+v", specMap)
	}

	// ClaimCheckManager small payload pass-through, default dir, and malformed/missing pointers
	defClaim := NewClaimCheckManager("")
	if small := defClaim.OffloadIfNeeded("tiny"); small != "tiny" {
		t.Fatalf("expected small payload to pass through unchanged")
	}
	if pass, err := defClaim.Retrieve("plain text"); err != nil || pass != "plain text" {
		t.Fatalf("expected plain text retrieve to pass through")
	}
	if _, err := defClaim.Retrieve("<CLAIM_CHECK:malformed>"); err == nil {
		t.Fatalf("expected error for malformed claim check pointer")
	}
	if _, err := defClaim.Retrieve("<CLAIM_CHECK:deadbeef:100>"); err == nil {
		t.Fatalf("expected error for missing claim check blob")
	}

	// SafeReplayEngine AUDIT_READ_ONLY, EXECUTE_AUTHORIZED (side effect vs idempotent), and unsupported mode
	replay := NewSafeReplayEngine()
	if aud, err := replay.ReplayStep(map[string]interface{}{"action": "read"}, ReplayAuditReadOnly, ""); err != nil || aud["status"] != "AUDITED" {
		t.Fatalf("expected AUDITED, got %+v (%v)", aud, err)
	}
	if blk, err := replay.ReplayStep(map[string]interface{}{"step_id": "s1", "side_effect": true}, ReplayExecuteAuthorized, "tok"); err != nil || blk["status"] != "BLOCKED_SIDE_EFFECT" {
		t.Fatalf("expected BLOCKED_SIDE_EFFECT, got %+v (%v)", blk, err)
	}
	if reex, err := replay.ReplayStep(map[string]interface{}{"step_id": "s2", "side_effect": false}, ReplayExecuteAuthorized, "tok"); err != nil || reex["status"] != "RE_EXECUTED" {
		t.Fatalf("expected RE_EXECUTED, got %+v (%v)", reex, err)
	}
	if _, err := replay.ReplayStep(map[string]interface{}{}, ReplayMode("INVALID"), "tok"); err == nil {
		t.Fatalf("expected error on unsupported replay mode")
	}

	// A2AProtocolAdapter negative branches: unknown sender, default alg/timestamp/nonce, bad alg, clock skew, bad sig
	a2a := NewA2AProtocolAdapter()
	if _, err := a2a.SignMessage("unknown", "rec", "", "", 0, nil); err == nil {
		t.Fatalf("expected error signing for unknown sender")
	}
	a2a.RegisterAgent("a1", "sec1")
	envDef, err := a2a.SignMessage("a1", "a2", "", "", 0, map[string]interface{}{"x": 1})
	if err != nil || envDef.Algorithm != "HS256" || envDef.Nonce == "" {
		t.Fatalf("unexpected default SignMessage: %+v (%v)", envDef, err)
	}
	badAlg := envDef
	badAlg.Algorithm = "NONE"
	if _, err := a2a.VerifyAndReceive(badAlg); err == nil {
		t.Fatalf("expected error on disallowed algorithm")
	}
	skewEnv := envDef
	skewEnv.Timestamp = time.Now().Unix() - 1000
	if _, err := a2a.VerifyAndReceive(skewEnv); err == nil {
		t.Fatalf("expected error on clock skew")
	}
	unregEnv := envDef
	unregEnv.SenderID = "ghost"
	if _, err := a2a.VerifyAndReceive(unregEnv); err == nil {
		t.Fatalf("expected error on unregistered sender")
	}
	badSigEnv := envDef
	badSigEnv.Signature = "0000"
	if _, err := a2a.VerifyAndReceive(badSigEnv); err == nil {
		t.Fatalf("expected error on signature mismatch")
	}

	// ShadowEvaluator unknown guardrail & high false positive rate, plus ThinDeviceRuntime laptop profile
	shadow := NewShadowEvaluator()
	if _, err := shadow.PromoteGuardrail("missing", 100, 0); err == nil {
		t.Fatalf("expected error for unknown guardrail")
	}
	g := shadow.MineFailure("panic")
	if _, err := shadow.PromoteGuardrail(g.RuleID, 80.0, 0.05); err == nil {
		t.Fatalf("expected error when falsePositiveRate >= 0.001")
	}
	rt := NewThinDeviceRuntime()
	if prof := rt.DetectHardwareProfile(8192, false); prof.Profile != "THIN_LAPTOP" {
		t.Fatalf("expected THIN_LAPTOP for 8GB RAM, got %+v", prof)
	}
}

