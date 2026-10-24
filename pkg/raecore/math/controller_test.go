package math

import (
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestMathControllerScoringAndBandit(t *testing.T) {
	mc := NewMathController(models.MathL3)
	now := time.Now().UTC()
	rec := models.MemoryItem{
		ID:             "m1",
		Content:        "math test",
		Layer:          models.LayerSemantic,
		Importance:     0.9,
		Strength:       1.0,
		UsageCount:     4,
		CreatedAt:      now.Add(-2 * time.Hour),
		LastAccessedAt: now.Add(-1 * time.Hour),
	}
	score, breakdown := mc.ScoreMemory(rec, 0.85, now)
	if score <= 0 || score > 1.0 || breakdown["similarity"] <= 0 {
		t.Fatalf("unexpected ScoreMemory: score=%f, breakdown=%+v", score, breakdown)
	}

	weights := mc.UpdateReward("hyper_37", 0.95)
	if weights.ActiveArm != "hyper_37" {
		t.Fatalf("expected active arm hyper_37, got %s", weights.ActiveArm)
	}

	decayed := ComputeDecayedStrength(1.0, 2, 24.0, 0.05)
	if decayed <= 0 || decayed >= 1.0 {
		t.Fatalf("expected decayed strength in (0,1), got %f", decayed)
	}

	q8 := QuantizeInt8([]float32{-1.0, 0.0, 1.0})
	if len(q8) != 3 || q8[0] != -127 || q8[2] != 127 {
		t.Fatalf("unexpected QuantizeInt8: %+v", q8)
	}
	qb := QuantizeBinary([]float32{1.0, -1.0, 1.0, -1.0})
	if len(qb) != 1 {
		t.Fatalf("unexpected QuantizeBinary length: %d", len(qb))
	}
}

func TestMathControllerFullSuiteAndRuntimeEngines(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	mc := NewMathController(models.MathL1)
	w0 := mc.GetWeights()
	if w0.ActiveArm != "fluid_100" {
		t.Fatalf("expected default active arm fluid_100, got %s", w0.ActiveArm)
	}
	for i := 0; i < 6; i++ {
		arm := mc.SelectArm()
		if arm == "" {
			t.Fatalf("expected non-empty arm from SelectArm")
		}
	}

	if sim := CosineSimilarity(nil, []float32{1, 2}); sim != 0.0 {
		t.Fatalf("expected 0 similarity for mismatched slices, got %f", sim)
	}
	if sim := CosineSimilarity([]float32{0, 0}, []float32{1, 1}); sim != 0.0 {
		t.Fatalf("expected 0 similarity for zero vector, got %f", sim)
	}
	if sim := CosineSimilarity([]float32{1, 0}, []float32{1, 0}); sim < 0.99 {
		t.Fatalf("expected ~1.0 cosine similarity, got %f", sim)
	}

	if s := CalculateSemanticSimilarity("", "abc"); s != 0.0 {
		t.Fatalf("expected 0.0 for empty string, got %f", s)
	}
	if s := CalculateSemanticSimilarity("hello world", "HELLO   WORLD"); s != 1.0 {
		t.Fatalf("expected 1.0 for identical normalized strings, got %f", s)
	}
	if s := CalculateSemanticSimilarity("alpha beta gamma", "beta gamma delta"); s <= 0.0 || s >= 1.0 {
		t.Fatalf("expected partial jaccard similarity, got %f", s)
	}
	if c := clamp(-5, 0, 1); c != 0 {
		t.Fatalf("expected clamp min 0, got %f", c)
	}
	if c := clamp(5, 0, 1); c != 1 {
		t.Fatalf("expected clamp max 1, got %f", c)
	}

	mab := NewMultiArmedBandit()
	if r := mab.DecayExploration(0); r <= 0 {
		t.Fatalf("unexpected exploration rate: %f", r)
	}
	if r := mab.DecayExploration(0.8); r <= 0 {
		t.Fatalf("unexpected exploration rate: %f", r)
	}
	_ = mab.SelectArm()
	_ = mab.Update("classic_ib", 0.9)
	if mab.GetBestArm() != "classic_ib" {
		t.Fatalf("expected best arm classic_ib, got %s", mab.GetBestArm())
	}
	if !mab.CheckDegradation(0.10) || mab.CheckDegradation(0.80) {
		t.Fatalf("unexpected CheckDegradation behavior")
	}

	rt := NewRAERuntime()
	proc := rt.Process("test runtime input")
	if proc["input"] != "test runtime input" {
		t.Fatalf("unexpected runtime process output: %+v", proc)
	}

	eng := rt.Engine
	mem := eng.StoreMemory("stored content", models.LayerSemantic)
	if mem.Content != "stored content" {
		t.Fatalf("unexpected stored memory: %+v", mem)
	}
	ev := eng.SearchEvidence("stored content", []models.MemoryItem{mem, {ID: "low", Content: "other", Importance: 0.2}})
	if len(ev) != 2 {
		t.Fatalf("expected 2 evidence results, got %d", len(ev))
	}
	if txt := eng.GenerateText("hello"); txt == "" {
		t.Fatalf("expected non-empty GenerateText")
	}
	if st := eng.GetStatus(); st["status"] != "ACTIVE" {
		t.Fatalf("unexpected engine status: %+v", st)
	}
	if d := eng.Reflector.EvaluateAnswerDraft("draft text", "draft text"); d != 1.0 {
		t.Fatalf("expected 1.0 draft evaluation, got %f", d)
	}
	cands := eng.Reflector.IdentifyReflectionCandidates([]models.MemoryItem{mem, {ID: "low", Importance: 0.1}})
	if len(cands) != 1 {
		t.Fatalf("expected 1 reflection candidate, got %d", len(cands))
	}

	hse := NewHybridSearchEngine()
	if len(hse.GetAvailableStrategies()) != 5 {
		t.Fatalf("expected 5 hybrid strategies")
	}
	res := hse.Search("stored", []models.MemoryItem{mem})
	reranked := hse.Rerank("stored", res)
	if len(reranked) != 1 {
		t.Fatalf("expected 1 reranked item")
	}

	// IB selector & CostController edge cases
	ib := NewInformationBottleneckSelector(0)
	if beta := ib.AdaptiveBeta(5000, 0); beta <= 0 {
		t.Fatalf("expected positive adaptive beta, got %f", beta)
	}
	_ = ib.SelectContext("query", []IBCandidate{
		{ID: "c1", Content: "alpha beta", Relevance: 0, Importance: 0, TokenCount: 0},
	}, 0)
	if lexicalJaccardSimilarity("", "a") != 0 {
		t.Fatalf("expected 0 jaccard on empty input")
	}

	if cost, _, tier := CalculateModelCost("ollama/local-model", 1000, 1000, 0); cost != 0 || tier != "tier0_local_gpu" {
		t.Fatalf("expected 0 cost for local model, got %f %s", cost, tier)
	}
	if cost, _, tier := CalculateModelCost("unknown/cloud-model", 1000, 1000, 2000); cost <= 0 || tier != "tier1_generate" {
		t.Fatalf("expected fallback tier1_generate cost, got %f %s", cost, tier)
	}
}

