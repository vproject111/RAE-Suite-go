package workers

import (
	"context"
	"testing"
)

func TestGraphAlgorithmsAndRepositories(t *testing.T) {
	repo := &GraphRepository{}
	n := repo.StoreGraphTriples([][3]string{
		{"A", "CALLS", "B"},
		{"B", "CALLS", "C"},
		{"A", "DEPENDS_ON", "C"},
	})
	if n != 3 {
		t.Fatalf("expected 3 triples stored, got %d", n)
	}

	bfs := repo.TraverseGraphBFS("A", 3)
	if len(bfs) != 3 {
		t.Fatalf("expected BFS to visit 3 nodes, got %v", bfs)
	}
	dfs := repo.TraverseGraphDFS("A", 3)
	if len(dfs) != 3 {
		t.Fatalf("expected DFS to visit 3 nodes, got %v", dfs)
	}

	alg := &GraphAlgorithmsService{Damping: 0.85}
	alg.AddEdge("A", "B")
	alg.AddEdge("B", "C")
	alg.AddEdge("A", "C")

	pr := alg.PageRank([]string{"A", "B", "C"})
	if pr["C"] <= pr["A"] {
		t.Fatalf("expected sink node C to have higher PageRank than A, got %v", pr)
	}
	sp := alg.ShortestPath("A", "C")
	if len(sp) != 2 || sp[0] != "A" || sp[1] != "C" {
		t.Fatalf("expected shortest path [A C], got %v", sp)
	}
	allPaths := alg.FindAllPaths("A", "C")
	if len(allPaths) < 2 {
		t.Fatalf("expected at least 2 paths from A to C, got %v", allPaths)
	}

	core := &RAECoreService{}
	core.EnableSzubarMode(true)
	stats := core.GetStatistics()
	if stats["szubar_mode"] != true {
		t.Fatalf("expected szubar_mode=true")
	}
	_ = core.DeleteMemory(context.Background(), "nonexistent")
}
