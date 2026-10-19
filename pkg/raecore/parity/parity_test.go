package parity

import (
	"strings"
	"testing"
)

func TestBuildParityRegistryAndDomainExecution(t *testing.T) {
	reg := BuildParityRegistry()
	if len(reg) < 500 {
		t.Fatalf("expected >= 500 registered parity domain classes, got %d", len(reg))
	}

	fs := NewAbstractFusionStrategy()
	rec1 := fs.Fuse("dense", "sparse")
	rec2 := fs.Fuse("dense", "graph")
	if !strings.HasPrefix(rec1.Status, "EXECUTED_") || rec1.Class != "AbstractFusionStrategy" {
		t.Fatalf("unexpected AbstractFusionStrategy execution record: %+v", rec1)
	}
	if rec1.Digest == rec2.Digest {
		t.Fatalf("expected input-sensitive digests to differ for different arguments")
	}

	ab := NewABTestRepository()
	if !strings.HasPrefix(ab.CreateTest("exp-1").Status, "EXECUTED_") || !strings.HasPrefix(ab.CalculateStatistics("exp-1").Status, "EXECUTED_") {
		t.Fatalf("expected ABTestRepository methods EXECUTED")
	}

	fnRes1 := ActivateEdge("node-a", "node-b")
	fnRes2 := ActivateEdge("node-a", "node-c")
	st, _ := fnRes1["status"].(string)
	if !strings.HasPrefix(st, "EXECUTED_") || fnRes1["digest"] == fnRes2["digest"] {
		t.Fatalf("expected input-sensitive ActivateEdge status EXECUTED, got %+v", fnRes1)
	}
}
