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

func TestSecurityComprehensiveNegativeAndEdgeCases(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}

	// RulesEngine AddRule and ListRules
	re := NewRulesEngine()
	added := re.AddRule(ECARule{
		Name:      "Custom Cost Alert",
		EventType: "on_cost_threshold",
		Action:    "emit_alert",
	})
	if added.RuleID == "" || !added.Enabled {
		t.Fatalf("expected generated RuleID and Enabled=true, got %+v", added)
	}
	rules, hist := re.ListRules()
	if len(rules) < 4 || len(hist) != 0 {
		t.Fatalf("unexpected ListRules: len(rules)=%d len(hist)=%d", len(rules), len(hist))
	}
	// Evaluate with low importance (skip MinImportance) and mismatched class
	_ = re.EvaluateEvent("on_memory_store", "m-low", 0.10, "public")

	// PolicyVersioningService GetActive, ListVersions, Rollback failure
	pvs := NewPolicyVersioningService()
	act := pvs.GetActive("")
	if act.Status != PolicyActive {
		t.Fatalf("expected active default policy, got %+v", act)
	}
	_ = pvs.CreateVersion("", false, 0.9, 100, false) // Draft version
	if len(pvs.ListVersions("")) < 2 {
		t.Fatalf("expected at least 2 versions")
	}
	if _, ok := pvs.Rollback("unknown-policy", 1); ok {
		t.Fatalf("expected Rollback on unknown policy to return false")
	}
	// GetActive on unknown policy fallback
	fallbackPol := pvs.GetActive("non-existent")
	if fallbackPol.PolicyID != "non-existent" {
		t.Fatalf("unexpected fallback policy: %+v", fallbackPol)
	}

	// RetentionService ListDeletionAudits and empty subject match by tenant
	store := &mockMemStore{}
	_ = store.StoreMemory(context.Background(), &models.MemoryItem{
		ID:       "t-mem-1",
		TenantID: "tenant-wipe",
		Content:  "data",
	})
	ret := NewRetentionService()
	_ = ret.ExecuteRightToBeForgotten(context.Background(), store, "tenant-wipe", "")
	if len(ret.ListDeletionAudits()) != 1 {
		t.Fatalf("expected 1 deletion audit entry")
	}

	// BudgetService defaults & auto-init on CheckAndRecordSpend
	bs := NewBudgetService()
	bs.SetBudget("", 0, 0)
	bNew := bs.CheckAndRecordSpend("new-tenant", 0.5, 100)
	if !bNew.Allowed || bNew.Status != BudgetStatusOK {
		t.Fatalf("expected OK status for new-tenant small spend, got %+v", bNew)
	}

	// ContextProvenanceService fallbacks and non-compliant uncited claims
	prov := NewContextProvenanceService()
	autoDec := prov.RecordDecision("", "agent-1", "TEST", "out", []string{"unverified-claim-id"})
	lin := prov.GetDecisionLineage("non-existent-dec-id")
	if lin["verified"] != true {
		t.Fatalf("expected fallback lineage verification, got %+v", lin)
	}
	aud := prov.AuditContextQuality("non-existent-ctx-id")
	if aud.ISO42001Compliant || len(aud.UncitedClaimIDs) == 0 {
		t.Fatalf("expected non-compliant audit for uncited claim, got %+v (dec=%+v)", aud, autoDec)
	}
	_ = prov.CreateDecisionContext("p", "q", []ContextSource{{SourceID: "s1", TrustScore: 0, Weight: 0}})
}

