package search

import (
	"testing"
	"time"

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

func TestSearchFullIntentsMaturityConflictsAndDriftBranches(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	// ClassifyQuery all intent branches
	queries := []struct {
		q      string
		intent QueryIntentClass
	}{
		{"How to step through deployment runbook", IntentProcedural},
		{"When did recent temporal timeline change yesterday", IntentTemporal},
		{"func BuildRoutingPlan in classifier_router.go", IntentCodeSymbol},
		{"Why compare the architecture across ten or more tokens in the system", IntentExploratory},
		{"What is RAE", IntentFactual},
	}
	for _, tc := range queries {
		plan := BuildRoutingPlan(tc.q)
		if plan.Classification.Intent != tc.intent {
			t.Fatalf("query %q: expected intent %s, got %s", tc.q, tc.intent, plan.Classification.Intent)
		}
	}

	// OptimizeRetrievalMaturity all branches
	if m := OptimizeRetrievalMaturity(10, 0.9)["maturity_mode"]; m != "BOOTSTRAP" {
		t.Fatalf("expected BOOTSTRAP, got %v", m)
	}
	if m := OptimizeRetrievalMaturity(500, 0.60)["maturity_mode"]; m != "PRECISION" {
		t.Fatalf("expected PRECISION, got %v", m)
	}
	if m := OptimizeRetrievalMaturity(20000, 0.95)["maturity_mode"]; m != "HIGH_THROUGHPUT" {
		t.Fatalf("expected HIGH_THROUGHPUT, got %v", m)
	}
	if m := OptimizeRetrievalMaturity(500, 0.85)["maturity_mode"]; m != "BALANCED" {
		t.Fatalf("expected BALANCED, got %v", m)
	}

	// Numeric divergence conflict & default rewrite expansion
	conflicts := DetectEvidenceConflicts([]models.ScoredMemoryItem{
		{Memory: models.MemoryItem{ID: "n1", Content: "Qdrant vector dimension is 384 floats"}},
		{Memory: models.MemoryItem{ID: "n2", Content: "Qdrant vector dimension is 768 floats"}},
	})
	if len(conflicts) == 0 || conflicts[0].ConflictType != "NUMERIC_DIVERGENCE" {
		t.Fatalf("expected NUMERIC_DIVERGENCE conflict, got %+v", conflicts)
	}
	rw := CreateRewritePlan("unmapped keyword", 0.90)
	if rw.SecondPassRun || len(rw.ExpandedTerms) != 3 {
		t.Fatalf("expected SecondPassRun=false and 3 default expanded terms, got %+v", rw)
	}

	// EvaluateSearchIR edge cases
	_ = EvaluateSearchIR(nil, nil, 0)
	_ = EvaluateSearchIR([]string{"m1"}, nil, 5)

	// Drift detector branches (default empty, LOW, MEDIUM, HIGH)
	_ = ComputeKLAndJSDDrift(nil, nil)
	_ = ComputeKLAndJSDDrift([]float64{0.4, 0.3, 0.2, 0.1}, []float64{0.25, 0.25, 0.25, 0.25})
	_ = ComputeKLAndJSDDrift([]float64{0.6, 0.2, 0.1, 0.1}, []float64{0.2, 0.3, 0.3, 0.2})

	// EvaluateABTestSignificance defaults and armB winner
	_ = EvaluateABTestSignificance("", nil, "", nil)
	abB := EvaluateABTestSignificance("armA", []float64{0.5, 0.5}, "armB", []float64{0.9, 0.9})
	if abB["winner"] != "armB" {
		t.Fatalf("expected armB to win, got %+v", abB)
	}

	// LogicGateway defaults
	lg := NewLogicGateway(nil)
	_ = lg.SmartRerank("q", nil, time.Now().UTC())
	if ScoreBM25("", "doc", 0, 0, 0) != 0 {
		t.Fatalf("expected 0 BM25 on empty query")
	}
	_ = ScoreBM25("term", "term in doc", 0, 0, 0)
	_ = PropagateSemanticResonance(map[string]float64{"a": 0.9}, map[string][]string{"a": {"b"}}, 0, 0)
	if ok, sc := EvaluateSufficiency(nil, 0); ok || sc != 0 {
		t.Fatalf("expected false sufficiency on empty slice")
	}
}

