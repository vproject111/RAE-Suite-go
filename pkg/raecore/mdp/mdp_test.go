package mdp

import (
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestMDPEngineStatePolicyAndReward(t *testing.T) {
	items := []models.MemoryItem{
		{ID: "1", Layer: models.LayerEpisodic, Content: "episodic log 1", Importance: 0.7},
		{ID: "2", Layer: models.LayerWorking, Content: "working context item", Importance: 0.8},
		{ID: "3", Layer: models.LayerSemantic, Content: "semantic concept", Importance: 0.9},
	}
	state := ComputeStateFromMemories(items, 1024, 50000, 5.0, 10, 18)
	if state.MemoryState.TotalCount != 3 || state.MemoryState.LayerEntropy <= 0 {
		t.Fatalf("unexpected memory state: %+v", state.MemoryState)
	}

	eng := NewEngine()
	act, qTable, rationale := eng.SelectOptimalAction(state)
	if act == "" || len(qTable) != 11 || rationale == "" {
		t.Fatalf("unexpected policy output: act=%s len(q)=%d rationale=%s", act, len(qTable), rationale)
	}

	out := eng.Step(state, ActionConsolidateMemory, 0.85, 40, 6.0)
	if out.TotalReward <= 0 || out.NextState.Step != 2 {
		t.Fatalf("unexpected transition outcome: %+v", out)
	}

	// Exhausted budget must force ActionNoOp
	exhausted := state
	exhausted.BudgetState.RemainingTokens = 0
	exhausted.BudgetState.IsExhausted = true
	actEx, _, _ := eng.SelectOptimalAction(exhausted)
	if actEx != ActionNoOp {
		t.Fatalf("expected ActionNoOp on exhausted budget, got %s", actEx)
	}
}

func TestMDPFullTransitionsStateMethodsAndMetrics(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	items := []models.MemoryItem{
		{ID: "1", Layer: models.LayerEpisodic, Content: "e1", Importance: 0.7},
		{ID: "2", Layer: models.LayerWorking, Content: "w1", Importance: 0.8},
		{ID: "3", Layer: models.LayerSemantic, Content: "s1", Importance: 0.9},
		{ID: "4", Layer: models.LayerLTM, Content: "l1", Importance: 0.95},
		{ID: "5", Layer: models.LayerReflective, Content: "r1", Importance: 0.85},
		{ID: "6", Layer: models.LayerSensory, Content: "sen1", Importance: 0.5},
	}
	s1 := ComputeStateFromMemories(items, 0, 10000, 2.0, 5, 10)
	s2 := ComputeStateFromMemories(nil, 2048, 10000, 2.0, 1, 0)
	if !s1.IsValid() || !s2.IsValid() {
		t.Fatalf("expected valid states")
	}
	if len(s1.ToDict()) == 0 || len(s1.LogState()) == 0 {
		t.Fatalf("expected non-empty ToDict and LogState")
	}
	diff := s1.Compare(s2)
	if _, ok := diff["entropy_delta"]; !ok {
		t.Fatalf("expected entropy_delta in state comparison")
	}

	eng := NewEngine()
	// Exercise different optimal action rationales
	highPressure := s1
	highPressure.WorkingContext.TokenPressure = 0.95
	if act, _, rat := eng.SelectOptimalAction(highPressure); act != ActionPruneContext || rat == "" {
		t.Fatalf("expected ActionPruneContext on high token pressure, got %s (%s)", act, rat)
	}
	highEpisodic := s1
	highEpisodic.WorkingContext.TokenPressure = 0.1
	highEpisodic.MemoryState.EpisodicCount = 30
	if act, _, _ := eng.SelectOptimalAction(highEpisodic); act != ActionGenerateReflection && act != ActionConsolidateMemory {
		t.Fatalf("expected reflection or consolidation on high episodic count, got %s", act)
	}
	consolidateState := s1
	consolidateState.WorkingContext.TokenPressure = 0.7
	consolidateState.MemoryState.EpisodicCount = 25
	_, _, _ = eng.SelectOptimalAction(consolidateState)

	// Step through all action types with default (0) qualityGain/tokens/latency to cover all branches
	for _, act := range AllActionTypes() {
		out := eng.Step(s1, act, 0, 0, 0)
		if out.NextState.Step != s1.Step+1 {
			t.Fatalf("expected step increment for action %s", act)
		}
	}
	// Test non-compliant safety transition (exhausted budget + non-noop/prune action)
	exState := s1
	exState.BudgetState.IsExhausted = true
	unsafeOut := eng.Step(exState, ActionCallLLM, 0.5, 100, 50)
	if unsafeOut.SafetyCompliant || unsafeOut.SafetyComponent != -1.0 {
		t.Fatalf("expected non-compliant safety penalty -1.0, got %+v", unsafeOut)
	}

	metrics := eng.GetMetrics()
	if metrics.TotalTransitions < len(AllActionTypes()) {
		t.Fatalf("expected metrics to record all transitions, got %+v", metrics)
	}
}

