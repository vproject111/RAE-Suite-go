package search

import (
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestEvidenceConflictDetectorAndQueryRewriter(t *testing.T) {
	items := []models.ScoredMemoryItem{
		{Memory: models.MemoryItem{ID: "m1", Content: "OpenClaw escalation engine is enabled on port 8009"}},
		{Memory: models.MemoryItem{ID: "m2", Content: "OpenClaw escalation engine is disabled and replaced by Hermes"}},
	}
	conflicts := DetectEvidenceConflicts(items)
	if len(conflicts) == 0 || conflicts[0].ConflictType != "POLARITY_CONTRADICTION" {
		t.Fatalf("expected polarity contradiction to be detected, got %+v", conflicts)
	}

	plan := CreateRewritePlan("db mesh", 0.55)
	if !plan.SecondPassRun || plan.BoostFactor != 1.4 || len(plan.ExpandedTerms) < 4 {
		t.Fatalf("expected 2-pass rewrite plan with boost=1.4, got %+v", plan)
	}
}
