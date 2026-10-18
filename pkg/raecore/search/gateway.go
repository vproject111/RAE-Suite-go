package search

import (
	"math"
	"sort"
	"strings"
	"time"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// AllStrategies returns all 9 canonical RAE search strategies.
func AllStrategies() []string {
	return []string{
		"anchor",
		"fulltext",
		"sparse",
		"vector",
		"multi_vector",
		"graph",
		"graph_lite",
		"visual",
		"hybrid",
	}
}

// LogicGateway implements the 3 RAE Math Engines:
// 1. Engine 1: Heuristic Fast Path (BM25, Regex/FullText, Anchor Matching < 1ms)
// 2. Engine 2: Probabilistic Resonance (Semantic Resonance & Graph Induction)
// 3. Engine 3: Neural / Multi-Vector Deep Path (Named Vectors, SmartReranker, SufficiencyGate)
type LogicGateway struct {
	mathCtrl *raemath.MathController
}

// NewLogicGateway creates a new LogicGateway.
func NewLogicGateway(mc *raemath.MathController) *LogicGateway {
	if mc == nil {
		mc = raemath.NewMathController(models.MathL3)
	}
	return &LogicGateway{mathCtrl: mc}
}

// ScoreBM25 computes Okapi BM25 relevance (k1=1.5, b=0.75) normalized to [0.0, 1.0].
func ScoreBM25(query, doc string, avgDocLen float64, docCount, df int) float64 {
	qTerms := strings.Fields(strings.ToLower(query))
	dTerms := strings.Fields(strings.ToLower(doc))
	if len(qTerms) == 0 || len(dTerms) == 0 {
		return 0.0
	}
	if avgDocLen <= 0 {
		avgDocLen = float64(len(dTerms))
	}
	if docCount <= 0 {
		docCount = 10
	}
	if df <= 0 {
		df = 1
	}
	tfMap := make(map[string]int, len(dTerms))
	for _, t := range dTerms {
		tfMap[t]++
	}
	k1 := 1.5
	b := 0.75
	raw := 0.0
	for _, qt := range qTerms {
		tf := float64(tfMap[qt])
		if tf == 0 {
			continue
		}
		idf := math.Log(1.0 + (float64(docCount-df)+0.5)/(float64(df)+0.5))
		num := tf * (k1 + 1.0)
		den := tf + k1*(1.0-b+b*(float64(len(dTerms))/avgDocLen))
		raw += idf * (num / den)
	}
	// Normalize via saturating logistic curve into [0.0, 1.0]
	norm := 1.0 - math.Exp(-0.35*raw)
	return math.Round(norm*10000) / 10000
}

// PropagateSemanticResonance propagates activation energy across Knowledge Graph edges (Engine 2).
func PropagateSemanticResonance(seeds map[string]float64, adj map[string][]string, hops int, decay float64) map[string]float64 {
	if hops <= 0 {
		hops = 2
	}
	if decay <= 0 || decay >= 1.0 {
		decay = 0.65
	}
	energy := make(map[string]float64, len(seeds))
	frontier := make(map[string]float64, len(seeds))
	for id, e := range seeds {
		energy[id] = e
		frontier[id] = e
	}
	for h := 0; h < hops; h++ {
		nextFrontier := make(map[string]float64)
		for node, e := range frontier {
			for _, nb := range adj[node] {
				delta := e * decay
				if delta > nextFrontier[nb] {
					nextFrontier[nb] = delta
				}
				if energy[nb]+delta > 1.0 {
					energy[nb] = 1.0
				} else {
					energy[nb] = math.Round((energy[nb]+delta)*10000) / 10000
				}
			}
		}
		frontier = nextFrontier
	}
	return energy
}

// DetectCommunities finds connected components in the Knowledge Graph (CommunityDetector).
func DetectCommunities(nodes []string, adj map[string][]string) [][]string {
	undirected := make(map[string]map[string]bool)
	for _, n := range nodes {
		if undirected[n] == nil {
			undirected[n] = make(map[string]bool)
		}
	}
	for u, nbrs := range adj {
		if undirected[u] == nil {
			undirected[u] = make(map[string]bool)
		}
		for _, v := range nbrs {
			if undirected[v] == nil {
				undirected[v] = make(map[string]bool)
			}
			undirected[u][v] = true
			undirected[v][u] = true
		}
	}

	visited := make(map[string]bool)
	var communities [][]string
	var allNodes []string
	for n := range undirected {
		allNodes = append(allNodes, n)
	}
	sort.Strings(allNodes)

	for _, start := range allNodes {
		if visited[start] {
			continue
		}
		var comp []string
		queue := []string{start}
		visited[start] = true
		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			comp = append(comp, curr)
			for nb := range undirected[curr] {
				if !visited[nb] {
					visited[nb] = true
					queue = append(queue, nb)
				}
			}
		}
		sort.Strings(comp)
		communities = append(communities, comp)
	}
	return communities
}

// SmartRerank applies Engine 3 hybrid reranking (combining fast path score, semantic overlap, and trust boost).
func (lg *LogicGateway) SmartRerank(query string, candidates []models.ScoredMemoryItem, now time.Time) []models.ScoredMemoryItem {
	out := make([]models.ScoredMemoryItem, len(candidates))
	copy(out, candidates)
	for i := range out {
		overlap := raemath.CalculateSemanticSimilarity(query, out[i].Memory.Content)
		trustBoost := 0.0
		if out[i].Memory.Envelope != nil && out[i].Memory.Envelope.TrustLevel == models.TrustHigh {
			trustBoost = 0.08
		}
		combinedSim := math.Min(1.0, 0.7*out[i].Score+0.3*overlap+trustBoost)
		score, breakdown := lg.mathCtrl.ScoreMemory(out[i].Memory, combinedSim, now)
		breakdown["smart_score"] = score
		out[i].Score = score
		out[i].ScoreBreakdown = breakdown
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Score > out[j].Score
	})
	return out
}

// EvaluateSufficiency checks if top retrieved results satisfy the SufficiencyGate confidence threshold (0.85).
func EvaluateSufficiency(results []models.ScoredMemoryItem, threshold float64) (bool, float64) {
	if threshold <= 0 {
		threshold = 0.85
	}
	if len(results) == 0 {
		return false, 0.0
	}
	top := results[0].Score
	return top >= threshold, top
}
