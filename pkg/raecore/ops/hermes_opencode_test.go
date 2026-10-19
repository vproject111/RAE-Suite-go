package ops

import "testing"

func TestHermesOpenCodeBridgeReplacesOpenClaw(t *testing.T) {
	bridge := NewHermesOpenCodeBridge()

	res1 := bridge.Dispatch(HermesOpenCodeRequest{
		TaskID:        "task-oc-1",
		TargetEngine:  "OPENCODE",
		Prompt:        "Refactor database port",
		ModifiedFiles: []string{"pkg/drivers/agnostic/broker.go"},
		DryRun:        true,
	})
	if !res1.OpenClawDisabled || res1.DispatchedEngine != "OPENCODE" || !res1.WorktreeIsolated || res1.EscalationHash == "" {
		t.Fatalf("unexpected OpenCode dispatch result: %+v", res1)
	}

	res2 := bridge.Dispatch(HermesOpenCodeRequest{
		TaskID:               "task-hermes-1",
		FailedRepairAttempts: 3,
		ErrorTrace:           "3 attempts exhausted in RAE-Phoenix",
		DryRun:               true,
	})
	if res2.DispatchedEngine != "HERMES" {
		t.Fatalf("expected auto-routing to HERMES after 3 failed repair attempts, got %s", res2.DispatchedEngine)
	}
}
