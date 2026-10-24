package memorystore

import (
	"context"
	"os"
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

func TestUnifiedStoreExtendedBranches(t *testing.T) {
	ctx := context.Background()
	_ = os.Setenv("RAE_LIVE_DB", "1")
	liveStore := NewUnifiedStore(nil, nil)
	_ = os.Unsetenv("RAE_LIVE_DB")
	_ = liveStore.StoreMemory(ctx, &models.MemoryItem{
		ID:        "live-vec-mem",
		Content:   "live vector memory",
		Layer:     models.LayerWorking,
		Project:   "p-ext",
		Embedding: []float32{1, 0, 0},
	})
	_ = liveStore.DeleteMemory(ctx, "live-vec-mem")

	store := NewUnifiedStore(nil, nil)

	// Guard error: RESTRICTED in non-working layer
	if err := store.StoreMemory(ctx, &models.MemoryItem{
		Content:   "secret",
		Layer:     models.LayerSemantic,
		InfoClass: models.InfoRestricted,
	}); err == nil {
		t.Fatal("expected guard error for RESTRICTED memory in semantic layer")
	}

	// Sensory layer auto-expiration & Pre-computed Embedding with liveEnabled
	sens := &models.MemoryItem{
		Content:   "sensory signal",
		Layer:     models.LayerSensory,
		Project:   "p-ext",
		AgentID:   "ag-1",
		Embedding: []float32{1, 0, 0},
	}
	if err := store.StoreMemory(ctx, sens); err != nil {
		t.Fatalf("StoreMemory sensory failed: %v", err)
	}
	if sens.ExpiresAt == nil {
		t.Fatal("expected sensory memory ExpiresAt to be set")
	}

	// Expired sensory memory
	past := time.Now().UTC().Add(-time.Hour)
	expSens := &models.MemoryItem{
		ID:        "exp-sens",
		Content:   "old sensory",
		Layer:     models.LayerSensory,
		Project:   "p-ext",
		ExpiresAt: &past,
	}
	_ = store.StoreMemory(ctx, expSens)

	// Not found errors
	if _, err := store.GetMemory(ctx, "nonexistent"); err == nil {
		t.Fatal("expected error for missing memory")
	}
	if err := store.DeleteMemory(ctx, "nonexistent"); err == nil {
		t.Fatal("expected error deleting missing memory")
	}

	// Query strategies: anchor, graph, hybrid with QueryVector, filters
	_ = store.AddGraphEdge(ctx, sens.ID, "target-node", "LINKS")
	neighbors, _ := store.GetGraphNeighbors(ctx, sens.ID)
	if len(neighbors) != 1 {
		t.Fatalf("expected 1 neighbor, got %v", neighbors)
	}

	anchorRes, _ := store.QueryMemories(ctx, ports.QueryFilter{
		Query:    "sensory",
		Strategy: "anchor",
		Project:  "p-ext",
		AgentID:  "ag-1",
		Layers:   []models.MemoryLayer{models.LayerSensory},
		TopK:     1,
	})
	if len(anchorRes) != 1 {
		t.Fatalf("expected 1 anchor result, got %d", len(anchorRes))
	}

	graphRes, _ := store.QueryMemories(ctx, ports.QueryFilter{
		Query:    "sensory signal",
		Strategy: "graph",
		Project:  "p-ext",
	})
	if len(graphRes) != 1 {
		t.Fatalf("expected 1 graph result, got %d", len(graphRes))
	}

	hybridVecRes, _ := store.QueryMemories(ctx, ports.QueryFilter{
		Query:       "sensory",
		Strategy:    "hybrid",
		Project:     "p-ext",
		QueryVector: []float32{1, 0, 0},
	})
	if len(hybridVecRes) != 1 {
		t.Fatalf("expected 1 hybrid vector result, got %d", len(hybridVecRes))
	}

	// DeleteExpiredSensory
	if removed, _ := store.DeleteExpiredSensory(ctx); removed < 1 {
		t.Fatalf("expected at least 1 expired sensory memory removed, got %d", removed)
	}

	// Artifacts, Edges, Journal
	a1 := &models.EpistemicArtifact{Title: "Art 1", Project: "p-ext", Type: models.ArtifactHypothesis}
	a2 := &models.EpistemicArtifact{Title: "Art 2", Project: "p-ext", Type: models.ArtifactType("EVIDENCE"), CreatedAt: time.Now().UTC().Add(time.Second)}
	_ = store.SaveArtifact(ctx, a1)
	_ = store.SaveArtifact(ctx, a2)
	if _, err := store.GetArtifact(ctx, a1.ID); err != nil {
		t.Fatalf("GetArtifact failed: %v", err)
	}
	if _, err := store.GetArtifact(ctx, "missing-art"); err == nil {
		t.Fatal("expected error for missing artifact")
	}
	arts, _ := store.ListArtifacts(ctx, "p-ext", "")
	if len(arts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(arts))
	}
	artsFiltered, _ := store.ListArtifacts(ctx, "p-ext", models.ArtifactHypothesis)
	if len(artsFiltered) != 1 {
		t.Fatalf("expected 1 filtered artifact, got %d", len(artsFiltered))
	}
	_ = store.LinkArtifacts(ctx, a1.ID, a2.ID, "supports")
	edges, _ := store.ListEdges(ctx, "p-ext")
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(edges))
	}

	_ = store.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{ArtifactID: a1.ID, Project: "p-ext", Action: "CREATE", Actor: "u1"})
	_ = store.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{ArtifactID: a2.ID, Project: "p-ext", Action: "LINK", Actor: "u1"})
	jEvents, _ := store.ListJournalEvents(ctx, "p-ext")
	if len(jEvents) != 2 || jEvents[1].PrevEventCID == "" {
		t.Fatalf("expected 2 chained journal events, got %+v", jEvents)
	}

	// Outbox
	ev1 := &ports.OutboxEvent{EventType: "SYNC"}
	ev2 := &ports.OutboxEvent{EventType: "AUDIT"}
	_ = store.EnqueueOutbox(ctx, ev1)
	_ = store.EnqueueOutbox(ctx, ev2)
	claimed, _ := store.ClaimPendingOutbox(ctx, 1)
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed outbox event, got %d", len(claimed))
	}
	if err := store.MarkOutboxDelivered(ctx, claimed[0].ID); err != nil {
		t.Fatalf("MarkOutboxDelivered failed: %v", err)
	}
	if err := store.MarkOutboxDelivered(ctx, "missing-outbox"); err == nil {
		t.Fatal("expected error for missing outbox event")
	}
}

