package phoenix

import "testing"

func TestPhoenixImpactParityAndCallbacks(t *testing.T) {
	svc := NewService()
	var failedReason string
	svc.OnRepairFailure = func(reason string) {
		failedReason = reason
	}

	imp := svc.AnalyzeImpact("core.go", []string{"a.go", "b.go"})
	if !imp.SafeToRefactor {
		t.Fatalf("expected SafeToRefactor=true")
	}

	res := svc.ExecuteClosedLoopRepair("EXPECTED", []string{"BAD1", "BAD2", "BAD3"})
	if !res.RolledBack || failedReason == "" {
		t.Fatalf("expected rollback and failure callback, got %+v", res)
	}
}
