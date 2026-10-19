package math

import (
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestSixFactorImportanceAndCostController(t *testing.T) {
	existing := []models.MemoryItem{
		{ID: "e1", Content: "routine health check ping ok"},
	}
	candidate := models.MemoryItem{
		ID:         "c1",
		Content:    "Critical security architecture decision: enforce AES-256-GCM on RESTRICTED memory migration",
		Layer:      models.LayerReflective,
		InfoClass:  models.InfoRestricted,
		Tags:       []string{"security", "architecture", "iso27000"},
		UsageCount: 5,
		CreatedAt:  time.Now().UTC(),
	}
	bd := ComputeSixFactorImportance(candidate, existing, 4)
	if bd.CompositeScore < 0.70 || bd.Novelty <= 0 || bd.RiskIntensity <= 0.5 {
		t.Fatalf("expected high composite 6-factor importance score, got %+v", bd)
	}

	cost, saved, tier := CalculateModelCost("z-ai/glm-5.3-flash", 100000, 50000, 40000)
	if cost <= 0 || saved <= 0 || tier != "tier1_generate" {
		t.Fatalf("unexpected model cost calculation: cost=%f saved=%f tier=%s", cost, saved, tier)
	}
}
