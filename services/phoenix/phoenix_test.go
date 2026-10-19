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

func TestPhoenixASTImportGraphAndRecipes(t *testing.T) {
	svc := NewService()

	sources := map[string]string{
		"pkg/raecore/math.go":      "package raecore\n",
		"services/memory/svc.go":   "package memory\nimport \"github.com/vproject111/RAE-Suite-go/pkg/raecore\"\n",
		"cmd/rae-memory/main.go":   "package main\nimport \"github.com/vproject111/RAE-Suite-go/services/memory\"\n",
		"unrelated/standalone.py":  "import os\n",
	}

	rep := svc.AnalyzeImpactWithGraph("pkg/raecore/math.go", nil, sources)
	if len(rep.AffectedFiles) != 1 || rep.AffectedFiles[0] != "services/memory/svc.go" {
		t.Fatalf("expected services/memory/svc.go as direct dependent, got %+v", rep.AffectedFiles)
	}
	if len(rep.TransitiveDependents) != 2 {
		t.Fatalf("expected 2 transitive dependents (svc.go -> main.go), got %+v", rep.TransitiveDependents)
	}

	recipes := svc.ListRecipes()
	if len(recipes) < 4 {
		t.Fatalf("expected at least 4 refactoring recipes, got %d", len(recipes))
	}
}
