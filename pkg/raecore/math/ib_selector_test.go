package math

import "testing"

func TestInformationBottleneckSelectorPenalizesRedundancy(t *testing.T) {
	sel := NewInformationBottleneckSelector(0.45)
	cands := []IBCandidate{
		{ID: "m1", Content: "PostgreSQL RLS tenant isolation policy enforced on all 12 tables", Relevance: 0.92, Importance: 0.90, TokenCount: 10},
		{ID: "m2", Content: "PostgreSQL RLS tenant isolation policy enforced on all 12 tables", Relevance: 0.91, Importance: 0.89, TokenCount: 10}, // near duplicate of m1
		{ID: "m3", Content: "Qdrant Named Vectors support dense 384d and ollama 768d embeddings", Relevance: 0.85, Importance: 0.85, TokenCount: 10},
	}
	res := sel.SelectContext("database storage architecture", cands, 20)
	if len(res.SelectedIDs) != 2 {
		t.Fatalf("expected 2 items selected within 20 token budget, got %+v", res)
	}
	if res.SelectedIDs[0] != "m1" || res.SelectedIDs[1] != "m3" {
		t.Fatalf("expected IB selector to pick m1 and diverse m3 over redundant m2, got %+v", res.SelectedIDs)
	}
	if res.DiversityScore < 0.75 {
		t.Fatalf("expected high diversity score >= 0.75, got %.4f", res.DiversityScore)
	}
}
