package graph

import (
	"testing"
	"time"
)

func TestTemporalGraphReconstructCompareAndRestore(t *testing.T) {
	svc := NewTemporalGraphService()
	t1 := time.Now().UTC().Add(-10 * time.Minute)
	t2 := time.Now().UTC()

	svc.RecordChange(GraphChange{
		Project:    "p1",
		ChangeType: ChangeEdgeAdded,
		SourceNode: "RAEMemory",
		TargetNode: "Postgres",
		Timestamp:  t1,
	})
	snap1 := svc.CreateSnapshot("p1", "initial")

	svc.RecordChange(GraphChange{
		Project:    "p1",
		ChangeType: ChangeEdgeAdded,
		SourceNode: "RAEMemory",
		TargetNode: "Qdrant",
		Timestamp:  t2,
	})

	comp := svc.CompareGraphs("p1", t1, t2)
	if len(comp.AddedNodes) != 1 || comp.AddedNodes[0] != "Qdrant" || comp.AddedEdges != 1 {
		t.Fatalf("expected Qdrant added between t1 and t2, got %+v", comp)
	}

	patterns := svc.FindEmergingPatterns("p1")
	if len(patterns) == 0 || patterns[0].Node != "RAEMemory" {
		t.Fatalf("expected RAEMemory as top emerging pattern, got %+v", patterns)
	}

	restored, ok := svc.RestoreSnapshot(snap1.SnapshotID)
	if !ok || len(restored.Nodes) != 2 {
		t.Fatalf("expected snapshot restore to succeed with 2 nodes, got %+v", restored)
	}
}

func TestEntityResolutionMergesSynonyms(t *testing.T) {
	adj := map[string][]string{
		"PostgreSQL":  {"Qdrant"},
		"postgres_db": {"Redis"},
		"Postgres":    {"RAEMesh"},
	}
	res := RunClusteringAndMerging(adj)
	if len(res.Decisions) == 0 {
		t.Fatalf("expected synonym merge decisions for Postgres variants, got %+v", res)
	}
	if res.ResolvedNodeCount >= res.OriginalNodeCount {
		t.Fatalf("expected node count reduction after entity resolution, got %d -> %d", res.OriginalNodeCount, res.ResolvedNodeCount)
	}
}

func TestTemporalGraphRemoveNodeRemoveEdgeAndRestoreNotFound(t *testing.T) {
	svc := NewTemporalGraphService()
	t1 := time.Now().UTC().Add(-5 * time.Minute)
	svc.RecordChange(GraphChange{Project: "p1", ChangeType: ChangeNodeAdded, SourceNode: "N1", Timestamp: t1})
	svc.RecordChange(GraphChange{Project: "p1", ChangeType: ChangeNodeAdded, SourceNode: "N2", Timestamp: t1})
	svc.RecordChange(GraphChange{Project: "p1", ChangeType: ChangeEdgeAdded, SourceNode: "N1", TargetNode: "N2", Timestamp: t1})
	svc.RecordChange(GraphChange{Project: "other", ChangeType: ChangeNodeAdded, SourceNode: "OtherNode", Timestamp: t1})

	t2 := time.Now().UTC()
	svc.RecordChange(GraphChange{Project: "p1", ChangeType: ChangeEdgeRemoved, SourceNode: "N1", TargetNode: "N2", Timestamp: t2})
	svc.RecordChange(GraphChange{Project: "p1", ChangeType: ChangeNodeRemoved, SourceNode: "N2", Timestamp: t2})

	comp := svc.CompareGraphs("p1", t1, t2)
	if len(comp.RemovedNodes) != 1 || comp.RemovedNodes[0] != "N2" {
		t.Fatalf("expected N2 in RemovedNodes, got %+v", comp)
	}
	if CalculateDensity(1, 0) != 0.0 {
		t.Fatalf("expected 0 density for single node")
	}
	if _, ok := svc.RestoreSnapshot("non-existent-snap"); ok {
		t.Fatalf("expected false when restoring non-existent snapshot")
	}
}

