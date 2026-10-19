package economy

import (
	"strings"
	"testing"
)

func TestBatchOptimizationEngineAndContextEconomy(t *testing.T) {
	eng := NewBatchOptimizationEngine(100)

	// Add 3 tasks: 2 in module "memory", and 1 in "memory_sub" touching the same target_file "store.go"
	if err := eng.AddTask(BatchTask{TaskID: "t1", Module: "memory", TargetFile: "store.go", ActionType: "refactor"}); err != nil {
		t.Fatalf("AddTask t1 failed: %v", err)
	}
	if err := eng.AddTask(BatchTask{TaskID: "t2", Module: "memory", TargetFile: "index.go", ActionType: "test"}); err != nil {
		t.Fatalf("AddTask t2 failed: %v", err)
	}
	if err := eng.AddTask(BatchTask{TaskID: "t3", Module: "memory_sub", TargetFile: "store.go", ActionType: "lint"}); err != nil {
		t.Fatalf("AddTask t3 failed: %v", err)
	}

	batches := eng.OptimizeAndGroup()
	if len(batches) < 1 {
		t.Fatalf("expected batches to be created")
	}
	// First batch ("memory") should have pulled t3 via Knowledge Scale Effect (touches store.go)
	if len(batches[0].Tasks) != 3 {
		t.Fatalf("expected Knowledge Scale Effect to coalesce 3 tasks touching store.go, got %d", len(batches[0].Tasks))
	}

	tokensSaved, durSaved := eng.DispatchBatch(batches[0])
	if tokensSaved != 30000 || durSaved != 3.0 {
		t.Fatalf("unexpected DispatchBatch: tokensSaved=%d, durSaved=%f", tokensSaved, durSaved)
	}

	rep := eng.GetSavingsReport()
	if rep["switches_prevented"] != 2 || rep["tokens_saved"] != 30000 {
		t.Fatalf("unexpected GetSavingsReport: %+v", rep)
	}

	// Test ContextBroker
	broker := NewContextBroker(100, 0.4)
	k, rerank, err := broker.DetermineRetrievalDepth([]float64{0.92, 0.70, 0.50})
	if err != nil || k != 3 || rerank {
		t.Fatalf("DetermineRetrievalDepth high gap failed: k=%d, rerank=%v, err=%v", k, rerank, err)
	}
	kCluster, rerankCluster, _ := broker.DetermineRetrievalDepth([]float64{0.81, 0.79, 0.78})
	if kCluster != 30 || !rerankCluster {
		t.Fatalf("DetermineRetrievalDepth clustered failed: k=%d, rerank=%v", kCluster, rerankCluster)
	}

	pruned := broker.PruneContext([]ContextEnvelopeItem{
		{ContextID: "c1", MemoryLayer: "semantic", TrustScore: 0.90, SourceHash: "1234567890", RetrievedContent: "High trust fact"},
		{ContextID: "c2", MemoryLayer: "working", TrustScore: 0.20, SourceHash: "abcdef1234", RetrievedContent: "Low trust noise"},
	}, "ISO 42001 active")
	if !strings.Contains(pruned, "High trust fact") || strings.Contains(pruned, "Low trust noise") {
		t.Fatalf("unexpected PruneContext output: %s", pruned)
	}

	// Test ContextTrustEvaluator
	eval := NewContextTrustEvaluator()
	if score, cls := eval.EvaluateTrust(map[string]interface{}{"information_class": "restricted", "layer": "semantic"}); score != 0.0 || cls != "REJECTED" {
		t.Fatalf("expected restricted outside working to be REJECTED, got %f %s", score, cls)
	}
	if score, cls := eval.EvaluateTrust(map[string]interface{}{"information_class": "internal", "layer": "semantic", "historical_success_rate": 0.9}); cls != "USABLE_PLANNING_INPUT" || score < 0.7 {
		t.Fatalf("expected semantic high success to be USABLE_PLANNING_INPUT, got %f %s", score, cls)
	}

	// Test HardenedSemanticCache Singleflight
	cache := NewHardenedSemanticCache(60.0, 0.15)
	calls := 0
	v1, _ := cache.GetOrCompute("k1", func() interface{} {
		calls++
		return "computed-value"
	}, false)
	v2, _ := cache.GetOrCompute("k1", func() interface{} {
		calls++
		return "computed-value-2"
	}, false)
	if v1 != "computed-value" || v2 != "computed-value" || calls != 1 {
		t.Fatalf("Singleflight cache failed: v1=%v, v2=%v, calls=%d", v1, v2, calls)
	}
}
