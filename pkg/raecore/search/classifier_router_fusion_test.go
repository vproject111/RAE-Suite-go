package search

import (
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestQueryClassifierRouterFusionAndEvaluatorDrift(t *testing.T) {
	plan := BuildRoutingPlan("How do RAEMesh and Qdrant connect in the knowledge graph?")
	if plan.Classification.Intent != IntentRelationalGraph {
		t.Fatalf("expected RELATIONAL_GRAPH intent, got %s", plan.Classification.Intent)
	}
	if !plan.UseGraphResonance {
		t.Fatalf("expected UseGraphResonance=true for graph intent")
	}

	fused := ConfidenceWeightedFuse(
		map[string][]models.ScoredMemoryItem{
			"vector": {{Memory: models.MemoryItem{ID: "m1"}, Score: 0.92}},
			"sparse": {{Memory: models.MemoryItem{ID: "m1"}, Score: 0.88}, {Memory: models.MemoryItem{ID: "m2"}, Score: 0.40}},
		},
		map[string]float64{"vector": 0.90, "sparse": 0.85},
		0.10,
	)
	if len(fused) == 0 || fused[0].Memory.ID != "m1" {
		t.Fatalf("expected m1 to rank first after ConfidenceWeightedFuse")
	}

	ir := EvaluateSearchIR([]string{"m2", "m1", "m3"}, []string{"m1"}, 5)
	if ir.MRR != 0.50 || ir.HitRate != 1.0 {
		t.Fatalf("expected MRR=0.50 and HitRate=1.0, got MRR=%v HitRate=%v", ir.MRR, ir.HitRate)
	}

	drift := ComputeKLAndJSDDrift([]float64{0.80, 0.10, 0.05, 0.05}, []float64{0.10, 0.10, 0.40, 0.40})
	if !drift.DriftDetected || drift.KLDivergence <= 0.25 {
		t.Fatalf("expected drift detected on shifted distribution, got %+v", drift)
	}

	ab := EvaluateABTestSignificance("fluid_100", []float64{0.95, 0.96, 0.94}, "classic_64", []float64{0.70, 0.72, 0.71})
	if ab["winner"] != "fluid_100" || ab["significant"] != true {
		t.Fatalf("expected significant winner fluid_100, got %+v", ab)
	}
}
