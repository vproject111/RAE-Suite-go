package parity

import "testing"

func TestBuildParityRegistryAndDomainExecution(t *testing.T) {
	reg := BuildParityRegistry()
	if len(reg) < 500 {
		t.Fatalf("expected >= 500 registered parity domain classes, got %d", len(reg))
	}

	fs := NewAbstractFusionStrategy()
	rec := fs.Fuse("dense", "sparse")
	if rec.Status != "OK" || rec.Class != "AbstractFusionStrategy" {
		t.Fatalf("unexpected AbstractFusionStrategy execution record: %+v", rec)
	}

	ab := NewABTestRepository()
	if ab.CreateTest("exp-1").Status != "OK" || ab.CalculateStatistics("exp-1").Status != "OK" {
		t.Fatalf("expected ABTestRepository methods OK")
	}

	fnRes := ActivateEdge("node-a", "node-b")
	if fnRes["status"] != "OK" {
		t.Fatalf("expected ActivateEdge status OK, got %+v", fnRes)
	}
}
