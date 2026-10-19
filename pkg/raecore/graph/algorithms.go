package graph

import (
	"math"
	"sort"
)

// allGraphNodes collects a sorted deterministic list of all nodes appearing as sources or targets.
func allGraphNodes(adjacency map[string][]string) []string {
	set := make(map[string]struct{})
	for u, nbrs := range adjacency {
		if u != "" {
			set[u] = struct{}{}
		}
		for _, v := range nbrs {
			if v != "" {
				set[v] = struct{}{}
			}
		}
	}
	nodes := make([]string, 0, len(set))
	for n := range set {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	return nodes
}

// ComputePageRank computes iterative PageRank centrality over a directed graph adjacency map
// matching apps/memory_api/services/graph_algorithms.py.
func ComputePageRank(adjacency map[string][]string, damping float64, maxIterations int, tol float64) map[string]float64 {
	nodes := allGraphNodes(adjacency)
	n := len(nodes)
	if n == 0 {
		return map[string]float64{}
	}
	if damping <= 0 || damping >= 1.0 {
		damping = 0.85
	}
	if maxIterations <= 0 {
		maxIterations = 50
	}
	if tol <= 0 {
		tol = 1e-6
	}

	pr := make(map[string]float64, n)
	initVal := 1.0 / float64(n)
	for _, u := range nodes {
		pr[u] = initVal
	}

	for iter := 0; iter < maxIterations; iter++ {
		next := make(map[string]float64, n)
		danglingSum := 0.0
		for _, u := range nodes {
			if len(adjacency[u]) == 0 {
				danglingSum += pr[u]
			}
		}
		base := (1.0-damping)/float64(n) + damping*(danglingSum/float64(n))
		for _, u := range nodes {
			next[u] = base
		}
		for _, u := range nodes {
			nbrs := adjacency[u]
			if len(nbrs) == 0 {
				continue
			}
			share := damping * (pr[u] / float64(len(nbrs)))
			for _, v := range nbrs {
				next[v] += share
			}
		}
		diff := 0.0
		for _, u := range nodes {
			diff += math.Abs(next[u] - pr[u])
			pr[u] = next[u]
		}
		if diff < tol {
			break
		}
	}

	for u, val := range pr {
		pr[u] = math.Round(val*100000) / 100000
	}
	return pr
}

// ComputeBetweennessCentrality implements Brandes' algorithm for shortest-path betweenness centrality.
func ComputeBetweennessCentrality(adjacency map[string][]string) map[string]float64 {
	nodes := allGraphNodes(adjacency)
	n := len(nodes)
	cb := make(map[string]float64, n)
	for _, u := range nodes {
		cb[u] = 0.0
	}
	if n <= 2 {
		return cb
	}

	for _, s := range nodes {
		var stack []string
		pred := make(map[string][]string, n)
		sigma := make(map[string]float64, n)
		dist := make(map[string]int, n)
		for _, v := range nodes {
			dist[v] = -1
		}
		sigma[s] = 1.0
		dist[s] = 0
		queue := []string{s}

		for len(queue) > 0 {
			v := queue[0]
			queue = queue[1:]
			stack = append(stack, v)
			for _, w := range adjacency[v] {
				if dist[w] < 0 {
					queue = append(queue, w)
					dist[w] = dist[v] + 1
				}
				if dist[w] == dist[v]+1 {
					sigma[w] += sigma[v]
					pred[w] = append(pred[w], v)
				}
			}
		}

		delta := make(map[string]float64, n)
		for i := len(stack) - 1; i >= 0; i-- {
			w := stack[i]
			for _, v := range pred[w] {
				if sigma[w] > 0 {
					delta[v] += (sigma[v] / sigma[w]) * (1.0 + delta[w])
				}
			}
			if w != s {
				cb[w] += delta[w]
			}
		}
	}

	norm := float64((n - 1) * (n - 2))
	for _, u := range nodes {
		if norm > 0 {
			cb[u] = math.Round((cb[u]/norm)*100000) / 100000
		}
	}
	return cb
}

// ComputeClusteringCoefficients computes local clustering coefficients (triangle closure ratio)
// and the global average clustering coefficient.
func ComputeClusteringCoefficients(adjacency map[string][]string) (map[string]float64, float64) {
	nodes := allGraphNodes(adjacency)
	undirected := make(map[string]map[string]bool, len(nodes))
	for _, u := range nodes {
		undirected[u] = make(map[string]bool)
	}
	for u, nbrs := range adjacency {
		for _, v := range nbrs {
			if u != v && u != "" && v != "" {
				undirected[u][v] = true
				undirected[v][u] = true
			}
		}
	}

	local := make(map[string]float64, len(nodes))
	sum := 0.0
	for _, u := range nodes {
		nbrMap := undirected[u]
		k := len(nbrMap)
		if k < 2 {
			local[u] = 0.0
			continue
		}
		nbrList := make([]string, 0, k)
		for v := range nbrMap {
			nbrList = append(nbrList, v)
		}
		triangles := 0
		for i := 0; i < len(nbrList); i++ {
			for j := i + 1; j < len(nbrList); j++ {
				if undirected[nbrList[i]][nbrList[j]] {
					triangles++
				}
			}
		}
		coeff := float64(2*triangles) / float64(k*(k-1))
		local[u] = math.Round(coeff*10000) / 10000
		sum += coeff
	}
	avg := 0.0
	if len(nodes) > 0 {
		avg = math.Round((sum/float64(len(nodes)))*10000) / 10000
	}
	return local, avg
}

// ComputeModularityQ computes Newman-Girvan Modularity Q for a community partition.
func ComputeModularityQ(adjacency map[string][]string, communities [][]string) float64 {
	nodes := allGraphNodes(adjacency)
	if len(nodes) == 0 || len(communities) == 0 {
		return 0.0
	}
	undirected := make(map[string]map[string]float64, len(nodes))
	degree := make(map[string]float64, len(nodes))
	for _, u := range nodes {
		undirected[u] = make(map[string]float64)
	}
	m2 := 0.0
	for u, nbrs := range adjacency {
		for _, v := range nbrs {
			if u == "" || v == "" || u == v {
				continue
			}
			if undirected[u][v] == 0 {
				undirected[u][v] = 1.0
				undirected[v][u] = 1.0
				degree[u] += 1.0
				degree[v] += 1.0
				m2 += 2.0
			}
		}
	}
	if m2 == 0 {
		return 0.0
	}

	q := 0.0
	for _, comm := range communities {
		for _, u := range comm {
			for _, v := range comm {
				aij := undirected[u][v]
				expected := (degree[u] * degree[v]) / m2
				q += aij - expected
			}
		}
	}
	return math.Round((q/m2)*10000) / 10000
}

// ExtractSubgraph extracts a bounded k-hop ego/seed subgraph with node and edge lists.
func ExtractSubgraph(adjacency map[string][]string, seeds []string, maxHops int, maxNodes int) ([]string, []map[string]string) {
	if maxHops <= 0 {
		maxHops = 2
	}
	if maxNodes <= 0 {
		maxNodes = 50
	}
	visited := make(map[string]int)
	type item struct {
		node string
		hop  int
	}
	queue := make([]item, 0, len(seeds))
	for _, s := range seeds {
		if s != "" {
			visited[s] = 0
			queue = append(queue, item{node: s, hop: 0})
		}
	}

	for len(queue) > 0 && len(visited) < maxNodes {
		curr := queue[0]
		queue = queue[1:]
		if curr.hop >= maxHops {
			continue
		}
		for _, nbr := range adjacency[curr.node] {
			if _, seen := visited[nbr]; !seen && len(visited) < maxNodes {
				visited[nbr] = curr.hop + 1
				queue = append(queue, item{node: nbr, hop: curr.hop + 1})
			}
		}
	}

	subNodes := make([]string, 0, len(visited))
	for n := range visited {
		subNodes = append(subNodes, n)
	}
	sort.Strings(subNodes)

	edges := make([]map[string]string, 0)
	for _, u := range subNodes {
		for _, v := range adjacency[u] {
			if _, ok := visited[v]; ok {
				edges = append(edges, map[string]string{
					"source": u,
					"target": v,
				})
			}
		}
	}
	return subNodes, edges
}

// DetectLeidenCommunities partitions the graph into densely connected communities via label propagation / modularity refinement.
func DetectLeidenCommunities(adjacency map[string][]string) [][]string {
	nodes := allGraphNodes(adjacency)
	if len(nodes) == 0 {
		return nil
	}
	undirected := make(map[string][]string, len(nodes))
	for u, nbrs := range adjacency {
		for _, v := range nbrs {
			if u != "" && v != "" && u != v {
				undirected[u] = append(undirected[u], v)
				undirected[v] = append(undirected[v], u)
			}
		}
	}
	labels := make(map[string]string, len(nodes))
	for _, u := range nodes {
		labels[u] = u
	}
	for iter := 0; iter < 8; iter++ {
		changed := false
		for _, u := range nodes {
			counts := make(map[string]int)
			for _, v := range undirected[u] {
				counts[labels[v]]++
			}
			bestLbl := labels[u]
			bestCnt := counts[bestLbl]
			for lbl, cnt := range counts {
				if cnt > bestCnt || (cnt == bestCnt && lbl < bestLbl) {
					bestCnt = cnt
					bestLbl = lbl
				}
			}
			if bestLbl != labels[u] {
				labels[u] = bestLbl
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	groups := make(map[string][]string)
	for _, u := range nodes {
		lbl := labels[u]
		groups[lbl] = append(groups[lbl], u)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	comms := make([][]string, 0, len(keys))
	for _, k := range keys {
		sort.Strings(groups[k])
		comms = append(comms, groups[k])
	}
	return comms
}

// AnalyzeTopology computes comprehensive graph topology metrics (PageRank, Betweenness, Clustering, Modularity Q).
func AnalyzeTopology(adjacency map[string][]string) map[string]interface{} {
	nodes := allGraphNodes(adjacency)
	pr := ComputePageRank(adjacency, 0.85, 50, 1e-6)
	bc := ComputeBetweennessCentrality(adjacency)
	localCC, avgCC := ComputeClusteringCoefficients(adjacency)
	comms := DetectLeidenCommunities(adjacency)
	modQ := ComputeModularityQ(adjacency, comms)

	edgeCount := 0
	for _, nbrs := range adjacency {
		edgeCount += len(nbrs)
	}
	density := 0.0
	if len(nodes) > 1 {
		density = math.Round((float64(edgeCount)/float64(len(nodes)*(len(nodes)-1)))*10000) / 10000
	}

	return map[string]interface{}{
		"node_count":              len(nodes),
		"edge_count":              edgeCount,
		"density":                 density,
		"pagerank":                pr,
		"betweenness_centrality":  bc,
		"clustering_coefficients": localCC,
		"avg_clustering":          avgCC,
		"communities":             comms,
		"modularity_q":            modQ,
	}
}

