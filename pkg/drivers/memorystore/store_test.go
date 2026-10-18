package memorystore

import (
	"context"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

func TestUnifiedStoreFullLifecycle(t *testing.T) {
	store := NewUnifiedStore(nil, nil)
	ctx := context.Background()

	m1 := &models.MemoryItem{
		ID:           "m1",
		Content:      "Go concurrency patterns with goroutines and channels",
		Layer:        models.LayerEpisodic,
		TenantID:     "default",
		Project:      "p1",
		Importance:   0.8,
		Strength:     1.0,
		NamedVectors: map[string][]float32{"dense": {1, 0, 0}},
		CreatedAt:    time.Now().UTC(),
	}
	m2 := &models.MemoryItem{
		ID:           "m2",
		Content:      "Epistemic DAG cycle prevention in RAE-CRL",
		Layer:        models.LayerSemantic,
		TenantID:     "default",
		Project:      "p1",
		Importance:   0.9,
		Strength:     1.0,
		NamedVectors: map[string][]float32{"dense": {0, 1, 0}},
		CreatedAt:    time.Now().UTC(),
	}
	if err := store.StoreMemory(ctx, m1); err != nil {
		t.Fatalf("StoreMemory m1: %v", err)
	}
	if err := store.StoreMemory(ctx, m2); err != nil {
		t.Fatalf("StoreMemory m2: %v", err)
	}

	// Sparse BM25 search
	results, err := store.QueryMemories(ctx, ports.QueryFilter{
		Query:    "goroutines channels",
		Strategy: "sparse",
		TenantID: "default",
		TopK:     5,
	})
	if err != nil || len(results) == 0 || results[0].Memory.ID != "m1" {
		t.Fatalf("expected m1 top in sparse search, got %+v (err=%v)", results, err)
	}

	// Named Vector search
	vecRes, err := store.QueryMemories(ctx, ports.QueryFilter{
		Strategy:    "multi_vector",
		VectorSpace: "dense",
		QueryVector: []float32{0.9, 0.1, 0},
		TopK:        2,
	})
	if err != nil || len(vecRes) != 2 || vecRes[0].Memory.ID != "m1" {
		t.Fatalf("expected m1 top in multi_vector search, got %+v (err=%v)", vecRes, err)
	}

	// Graph traversal & adjacency
	_ = store.AddGraphEdge(ctx, "m1", "m2", "RELATES_TO")
	adj, err := store.GetGraphAdjacency(ctx)
	if err != nil || len(adj["m1"]) != 1 || adj["m1"][0] != "m2" {
		t.Fatalf("unexpected graph adjacency: %+v", adj)
	}

	// Memory stats and delete
	stats, err := store.GetMemoryStats(ctx, "p1")
	if err != nil || stats.TotalCount != 2 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if err := store.DeleteMemory(ctx, "m1"); err != nil {
		t.Fatalf("DeleteMemory failed: %v", err)
	}

	// Multi-hop DAG cycle prevention
	if err := store.LinkArtifacts(ctx, "a1", "a2", "refines"); err != nil {
		t.Fatalf("LinkArtifacts a1->a2: %v", err)
	}
	if err := store.LinkArtifacts(ctx, "a2", "a3", "supports"); err != nil {
		t.Fatalf("LinkArtifacts a2->a3: %v", err)
	}
	if err := store.LinkArtifacts(ctx, "a3", "a1", "derived_from"); err == nil {
		t.Fatalf("expected multi-hop DAG cycle error, got nil")
	}
}
