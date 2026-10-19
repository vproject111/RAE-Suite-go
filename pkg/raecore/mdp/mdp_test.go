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
