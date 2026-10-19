package search

import (
	"math"
	"sort"
	"strings"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// QueryIntentClass enumerates the 6 canonical query intents from rae_core/search/classifier.py.
type QueryIntentClass string

const (
	IntentFactual         QueryIntentClass = "FACTUAL"
	IntentTemporal        QueryIntentClass = "TEMPORAL"
	IntentRelationalGraph QueryIntentClass = "RELATIONAL_GRAPH"
	IntentProcedural      QueryIntentClass = "PROCEDURAL"
	IntentExploratory     QueryIntentClass = "EXPLORATORY"
	IntentCodeSymbol      QueryIntentClass = "CODE_SYMBOL"
)

// ClassificationResult mirrors rae_core/search/classifier.py ClassificationResult.
type ClassificationResult struct {
	Query               string             `json:"query"`
	Intent              QueryIntentClass   `json:"intent"`
	Complexity          float64            `json:"complexity"`
	Confidence          float64            `json:"confidence"`
	TokenCount          int                `json:"token_count"`
	Entities            []string           `json:"entities"`
	RecommendedStrategy string             `json:"recommended_strategy"`
	Weights             map[string]float64 `json:"weights"` // alpha, beta, gamma
	StrategyWeights     map[string]float64 `json:"strategy_weights"`
}

// RoutingPlan mirrors rae_core/search/router.py RoutingPlan.
type RoutingPlan struct {
	Classification      ClassificationResult `json:"classification"`
	PrimaryStrategy     string               `json:"primary_strategy"`
	SecondaryStrategies []string             `json:"secondary_strategies"`
	Alpha               float64              `json:"alpha"`
	Beta                float64              `json:"beta"`
	Gamma               float64              `json:"gamma"`
	TopK                int                  `json:"top_k"`
	UseGraphResonance   bool                 `json:"use_graph_resonance"`
	UseTwoPassRewrite   bool                 `json:"use_two_pass_rewrite"`
}

// ClassifyQuery analyzes a query string and determines its intent, complexity, entities, and optimal weights.
func ClassifyQuery(query string) ClassificationResult {
	qTrim := strings.TrimSpace(query)
	qLow := strings.ToLower(qTrim)
	tokens := strings.Fields(qTrim)
	tokenCount := len(tokens)

	var entities []string
	for _, tok := range tokens {
		clean := strings.Trim(tok, ".,;:!?()[]{}\"'`")
		if len(clean) >= 2 && (clean[0] >= 'A' && clean[0] <= 'Z' || strings.Contains(clean, "_") || strings.Contains(clean, "/")) {
			entities = append(entities, clean)
		}
	}

	intent := IntentFactual
	recStrategy := "hybrid"
	alpha, beta, gamma := 0.50, 0.30, 0.20
	stratWeights := map[string]float64{
		"vector":    0.40,
		"sparse":    0.30,
		"graph":     0.15,
		"resonance": 0.15,
	}

	switch {
	case strings.Contains(qLow, "step") || strings.Contains(qLow, "how to") || strings.Contains(qLow, "procedure") || strings.Contains(qLow, "runbook") || strings.Contains(qLow, "krok"):
		intent = IntentProcedural
		recStrategy = "sparse"
		alpha, beta, gamma = 0.45, 0.40, 0.15
		stratWeights = map[string]float64{"sparse": 0.50, "vector": 0.30, "graph": 0.10, "resonance": 0.10}
	case strings.Contains(qLow, "when") || strings.Contains(qLow, "history") || strings.Contains(qLow, "recent") || strings.Contains(qLow, "yesterday") || strings.Contains(qLow, "timeline") || strings.Contains(qLow, "temporal"):
		intent = IntentTemporal
		recStrategy = "hybrid"
		alpha, beta, gamma = 0.35, 0.20, 0.45
		stratWeights = map[string]float64{"vector": 0.35, "sparse": 0.35, "graph": 0.15, "resonance": 0.15}
	case strings.Contains(qLow, "connect") || strings.Contains(qLow, "relat") || strings.Contains(qLow, "depend") || strings.Contains(qLow, "between") || strings.Contains(qLow, "graph") || strings.Contains(qLow, "impact"):
		intent = IntentRelationalGraph
		recStrategy = "graph"
		alpha, beta, gamma = 0.40, 0.40, 0.20
		stratWeights = map[string]float64{"graph": 0.45, "resonance": 0.30, "vector": 0.15, "sparse": 0.10}
	case strings.Contains(qLow, "func ") || strings.Contains(qLow, "class ") || strings.Contains(qLow, ".go") || strings.Contains(qLow, ".py") || strings.Contains(qLow, "_"):
		intent = IntentCodeSymbol
		recStrategy = "sparse"
		alpha, beta, gamma = 0.55, 0.30, 0.15
		stratWeights = map[string]float64{"sparse": 0.55, "vector": 0.25, "graph": 0.15, "resonance": 0.05}
	case tokenCount >= 10 || strings.Contains(qLow, "why") || strings.Contains(qLow, "compare") || strings.Contains(qLow, "architecture"):
		intent = IntentExploratory
		recStrategy = "hybrid"
		alpha, beta, gamma = 0.45, 0.35, 0.20
		stratWeights = map[string]float64{"vector": 0.35, "sparse": 0.25, "graph": 0.20, "resonance": 0.20}
	}

	complexity := math.Min(1.0, math.Round((float64(tokenCount)/16.0+float64(len(entities))*0.08)*1000)/1000)
	if complexity < 0.15 {
		complexity = 0.15
	}

	return ClassificationResult{
		Query:               query,
		Intent:              intent,
		Complexity:          complexity,
		Confidence:          0.92,
		TokenCount:          tokenCount,
		Entities:            entities,
		RecommendedStrategy: recStrategy,
		Weights: map[string]float64{
			"alpha": alpha,
			"beta":  beta,
			"gamma": gamma,
		},
		StrategyWeights: stratWeights,
	}
}

// BuildRoutingPlan constructs a full execution plan from a query string matching rae_core/search/router.py.
func BuildRoutingPlan(query string) RoutingPlan {
	cls := ClassifyQuery(query)
	topK := 10
	if cls.Complexity >= 0.60 {
		topK = 20
	}
	return RoutingPlan{
		Classification:      cls,
		PrimaryStrategy:     cls.RecommendedStrategy,
		SecondaryStrategies: []string{"sparse", "vector", "graph", "resonance"},
		Alpha:               cls.Weights["alpha"],
		Beta:                cls.Weights["beta"],
		Gamma:               cls.Weights["gamma"],
		TopK:                topK,
		UseGraphResonance:   cls.Intent == IntentRelationalGraph || cls.Intent == IntentExploratory,
		UseTwoPassRewrite:   cls.Complexity >= 0.50,
	}
}

// ConfidenceWeightedFuse merges scored memory items across multiple strategies weighted by strategy confidence,
// and applies an SNR noise floor filter (matching rae_core/search/fusion.py & engine.py).
func ConfidenceWeightedFuse(strategyLists map[string][]models.ScoredMemoryItem, strategyConfidences map[string]float64, minSNR float64) []models.ScoredMemoryItem {
	byID := make(map[string]models.ScoredMemoryItem)
	scoreAcc := make(map[string]float64)

	for strat, list := range strategyLists {
		conf := strategyConfidences[strat]
		if conf <= 0 {
			conf = 0.50
		}
		for rank, item := range list {
			rrf := 1.0 / float64(60+rank+1)
			weighted := conf * (0.7*item.Score + 0.3*rrf*60.0)
			scoreAcc[item.Memory.ID] += weighted
			byID[item.Memory.ID] = item
		}
	}

	if minSNR <= 0 {
		minSNR = 0.05
	}

	out := make([]models.ScoredMemoryItem, 0, len(byID))
	for id, item := range byID {
		s := math.Round(scoreAcc[id]*10000) / 10000
		if s >= minSNR {
			item.Score = s
			out = append(out, item)
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Score == out[j].Score {
			return out[i].Memory.ID < out[j].Memory.ID
		}
		return out[i].Score > out[j].Score
	})
	return out
}

// OptimizeRetrievalMaturity recommends a MaturityMode based on corpus size and historical MRR (rae_core/search/optimizer.py).
func OptimizeRetrievalMaturity(corpusSize int, historicalMRR float64) map[string]interface{} {
	mode := "BALANCED"
	recommendation := "Keep hybrid RRF + MathController L2 scoring"
	switch {
	case corpusSize < 50:
		mode = "BOOTSTRAP"
		recommendation = "Prioritize BM25 lexical matching + Working/Episodic boost while corpus grows"
	case historicalMRR < 0.75:
		mode = "PRECISION"
		recommendation = "Enable 2-Pass QueryRewriter, InformationBottleneckSelector, and SmartRerank"
	case corpusSize > 10000 && historicalMRR >= 0.90:
		mode = "HIGH_THROUGHPUT"
		recommendation = "Use HNSW dense vector fast-path + L1 MathController heuristics"
	}
	return map[string]interface{}{
		"maturity_mode":   mode,
		"corpus_size":     corpusSize,
		"historical_mrr":  historicalMRR,
		"recommendation":  recommendation,
	}
}
