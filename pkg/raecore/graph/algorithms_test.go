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

func TestGraphTopologyLeidenAndEdgeCases(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	adj := map[string][]string{
		"A": {"B", "C"},
		"B": {"A", "C"},
		"C": {"A", "B", "D"},
		"D": {"E", "F"},
		"E": {"D", "F"},
		"F": {"D", "E"},
	}
	topo := AnalyzeTopology(adj)
	if topo["node_count"] != 6 || topo["edge_count"] == 0 {
		t.Fatalf("unexpected AnalyzeTopology result: %+v", topo)
	}
	comms := DetectLeidenCommunities(adj)
	if len(comms) == 0 {
		t.Fatalf("expected non-empty Leiden communities")
	}

	// Empty / small graph edge cases
	if len(ComputePageRank(nil, 0, 0, 0)) != 0 {
		t.Fatalf("expected empty PageRank on nil graph")
	}
	_ = ComputePageRank(map[string][]string{"A": {}}, 0, 0, 0)
	if len(ComputeBetweennessCentrality(map[string][]string{"A": {"B"}})) != 2 {
		t.Fatalf("expected 2 nodes in small graph betweenness")
	}
	if ComputeModularityQ(nil, nil) != 0.0 {
		t.Fatalf("expected 0 modularity on nil graph")
	}
	if DetectLeidenCommunities(nil) != nil {
		t.Fatalf("expected nil Leiden communities on empty graph")
	}
	_, _ = ExtractSubgraph(adj, []string{"A"}, 0, 0)
}

