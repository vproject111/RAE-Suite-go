package search_test

import (
	"testing"
	"time"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/search"
)

func TestLogicGatewayBM25AndResonanceAndCommunitiesAndRerank(t *testing.T) {
	if len(search.AllStrategies()) != 9 {
		t.Fatalf("expected 9 search strategies, got %d", len(search.AllStrategies()))
	}
	s := search.ScoreBM25("tensor batching", "tensor batching optimization in GPU kernel", 8.0, 20, 2)
	if s <= 0.3 {
		t.Fatalf("expected positive BM25 score, got %f", s)
	}

	adj := map[string][]string{
		"n1": {"n2"},
		"n2": {"n3"},
		"n4": {"n5"},
	}
	energy := search.PropagateSemanticResonance(map[string]float64{"n1": 1.0}, adj, 2, 0.6)
	if energy["n2"] <= 0 || energy["n3"] <= 0 {
		t.Fatalf("expected resonance to propagate to n2 and n3, got %v", energy)
	}

	comms := search.DetectCommunities([]string{"n1", "n2", "n3", "n4", "n5"}, adj)
	if len(comms) != 2 {
		t.Fatalf("expected 2 connected communities, got %v", comms)
	}

	lg := search.NewLogicGateway(raemath.NewMathController(models.MathL3))
	now := time.Now().UTC()
	reranked := lg.SmartRerank("tensor optimization", []models.ScoredMemoryItem{
		{
			Memory: models.MemoryItem{
				ID:             "m1",
				Content:        "tensor optimization guide",
				Importance:     0.9,
				LastAccessedAt: now,
				Envelope:       &models.ContextEnvelope{TrustLevel: models.TrustHigh},
			},
			Score: 0.90,
		},
	}, now)
	sufficient, topScore := search.EvaluateSufficiency(reranked, 0.75)
	if !sufficient || topScore < 0.75 {
		t.Fatalf("expected reranked item to pass SufficiencyGate, got sufficient=%v score=%f", sufficient, topScore)
	}
}
