package hive

import (
	"context"
	"testing"
)

func TestHiveParallelAndWatchdog(t *testing.T) {
	svc := NewService()
	svc.RecordHeartbeat("worker-1")
	stale := svc.CheckStaleHeartbeats(120)
	if len(stale) != 0 {
		t.Fatalf("expected 0 stale heartbeats, got %v", stale)
	}
	broken, sim, status := svc.EvaluateEchoLoop([]string{"same error message", "same error message"})
	if !broken || sim < 0.9 || status != "BLOCKED_LOOP" {
		t.Fatalf("expected BLOCKED_LOOP, got broken=%v, sim=%f, status=%s", broken, sim, status)
	}
	res := svc.RunParallelCommands(context.Background(), []string{"echo branch-a", "echo branch-b"})
	if len(res) != 2 || res[0].ExitCode != 0 {
		t.Fatalf("unexpected parallel results: %+v", res)
	}
}
