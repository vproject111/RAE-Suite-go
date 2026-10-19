package security

import (
	"context"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

type mockMemStore struct {
	items []models.ScoredMemoryItem
}

func (m *mockMemStore) StoreMemory(_ context.Context, item *models.MemoryItem) error {
	m.items = append(m.items, models.ScoredMemoryItem{Memory: *item})
	return nil
}
func (m *mockMemStore) GetMemory(_ context.Context, _ string) (*models.MemoryItem, error) {
	return nil, nil
}
func (m *mockMemStore) DeleteMemory(_ context.Context, id string) error {
	out := make([]models.ScoredMemoryItem, 0)
	for _, it := range m.items {
		if it.Memory.ID != id {
			out = append(out, it)
		}
	}
	m.items = out
	return nil
}
func (m *mockMemStore) QueryMemories(_ context.Context, _ ports.QueryFilter) ([]models.ScoredMemoryItem, error) {
	return m.items, nil
}
func (m *mockMemStore) GetMemoryStats(_ context.Context, _ string) (ports.MemoryStats, error) {
	return ports.MemoryStats{}, nil
}
func (m *mockMemStore) DeleteExpiredSensory(_ context.Context) (int, error) { return 0, nil }
func (m *mockMemStore) AddGraphEdge(_ context.Context, _, _, _ string) error { return nil }
func (m *mockMemStore) GetGraphNeighbors(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockMemStore) GetGraphAdjacency(_ context.Context) (map[string][]string, error) {
	return nil, nil
}

func TestGovernanceEnginesRulesPolicyRetentionAndBudget(t *testing.T) {
	re := NewRulesEngine()
	fired := re.EvaluateEvent("on_memory_store", "m-1", 0.92, string(models.InfoRestricted))
	if len(fired) < 2 {
		t.Fatalf("expected both high-importance and restricted rules to fire, got %+v", fired)
	}

	pvs := NewPolicyVersioningService()
	v2 := pvs.CreateVersion("iso-42001-default", false, 0.70, 50.0, true)
	if v2.Version != 2 || v2.Status != PolicyActive {
		t.Fatalf("expected active v2 policy, got %+v", v2)
	}
	rb, ok := pvs.Rollback("iso-42001-default", 1)
	if !ok || rb.Version != 1 || rb.Status != PolicyActive {
		t.Fatalf("expected rollback to v1 to succeed, got %+v", rb)
	}

	store := &mockMemStore{}
	_ = store.StoreMemory(context.Background(), &models.MemoryItem{
		ID:       "pii-mem-1",
		TenantID: "t1",
		Content:  "User jan.kowalski@example.com personal record",
		Layer:    models.LayerEpisodic,
	})
	ret := NewRetentionService()
	audit := ret.ExecuteRightToBeForgotten(context.Background(), store, "t1", "jan.kowalski@example.com")
	if audit.DeletedMemoryCount != 1 || len(audit.ProofHashSHA256) != 64 {
		t.Fatalf("expected 1 deleted memory and SHA-256 proof, got %+v", audit)
	}

	bs := NewBudgetService()
	bs.SetBudget("t1", 10.0, 1000)
	b85 := bs.CheckAndRecordSpend("t1", 8.5, 500)
	if b85.Status != BudgetStatusWarn80 || !b85.Allowed {
		t.Fatalf("expected WARN_80_PERCENT, got %+v", b85)
	}
	b96 := bs.CheckAndRecordSpend("t1", 1.1, 100)
	if b96.Status != BudgetStatusCrit95 || !b96.Allowed {
		t.Fatalf("expected CRITICAL_95_PERCENT, got %+v", b96)
	}
	b105 := bs.CheckAndRecordSpend("t1", 1.0, 100)
	if b105.Status != BudgetStatusHardBlock || b105.Allowed {
		t.Fatalf("expected HARD_BLOCK_100_PERCENT, got %+v", b105)
	}
}
