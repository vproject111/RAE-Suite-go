package graph

import "testing"

func TestGraphAlgorithmsPageRankBetweennessClusteringModularity(t *testing.T) {
	adj := map[string][]string{
		"A": {"B", "C"},
		"B": {"C"},
		"C": {"A", "D"},
		"D": {"E"},
		"E": {"D"},
	}

	pr := ComputePageRank(adj, 0.85, 50, 1e-6)
	if len(pr) != 5 || pr["C"] <= 0 {
		t.Fatalf("unexpected pagerank: %+v", pr)
	}

	bc := ComputeBetweennessCentrality(adj)
	if bc["C"] <= 0 || bc["D"] <= 0 {
		t.Fatalf("expected bridge nodes C and D to have positive betweenness: %+v", bc)
	}

	localCC, avgCC := ComputeClusteringCoefficients(adj)
	if localCC["A"] <= 0 || avgCC <= 0 {
		t.Fatalf("expected triangle A-B-C to yield positive clustering: local=%+v avg=%f", localCC, avgCC)
	}

	comms := [][]string{{"A", "B", "C"}, {"D", "E"}}
	modQ := ComputeModularityQ(adj, comms)
	if modQ <= 0 {
		t.Fatalf("expected positive Modularity Q for two distinct clusters, got %f", modQ)
	}

	subNodes, subEdges := ExtractSubgraph(adj, []string{"A"}, 1, 10)
	if len(subNodes) != 3 || len(subEdges) == 0 {
		t.Fatalf("unexpected 1-hop subgraph from A: nodes=%v edges=%v", subNodes, subEdges)
	}
}
