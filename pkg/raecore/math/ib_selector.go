package math

import (
	"math"
	"strings"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// IBCandidate represents a candidate memory item evaluated by the Information Bottleneck selector.
type IBCandidate struct {
	ID         string             `json:"id"`
	Content    string             `json:"content"`
	Layer      models.MemoryLayer `json:"layer"`
	Relevance  float64            `json:"relevance"`
	Importance float64            `json:"importance"`
	TokenCount int                `json:"token_count"`
}

// IBSelectionResult captures the Information Bottleneck optimization result J_IB(Z) = I(Z; Y) - beta * I(Z; X).
type IBSelectionResult struct {
	SelectedIDs     []string      `json:"selected_ids"`
	SelectedItems   []IBCandidate `json:"selected_items"`
	TotalTokens     int           `json:"total_tokens"`
	MaxTokens       int           `json:"max_tokens"`
	AdaptiveBeta    float64       `json:"adaptive_beta"`
	MutualInfoIZY   float64       `json:"I_Z_Y"`
	CompressionIZX  float64       `json:"I_Z_X"`
	IBObjectiveScore float64      `json:"ib_objective_score"`
	DiversityScore  float64       `json:"diversity_score"`
}

// InformationBottleneckSelector implements Tishby's Information Bottleneck context selection
// matching Python's apps/memory_api/core/information_bottleneck.py.
type InformationBottleneckSelector struct {
	BaseBeta float64
}

// NewInformationBottleneckSelector creates a new InformationBottleneckSelector.
func NewInformationBottleneckSelector(baseBeta float64) *InformationBottleneckSelector {
	if baseBeta <= 0 {
		baseBeta = 0.35
	}
	return &InformationBottleneckSelector{BaseBeta: baseBeta}
}

// AdaptiveBeta dynamically adjusts beta based on token budget pressure and candidate pool size.
func (s *InformationBottleneckSelector) AdaptiveBeta(totalCandidateTokens, maxTokens int) float64 {
	if maxTokens <= 0 {
		maxTokens = 2000
	}
	if totalCandidateTokens <= maxTokens {
		return math.Round(s.BaseBeta*0.5*10000) / 10000
	}
	ratio := float64(totalCandidateTokens) / float64(maxTokens)
	beta := s.BaseBeta * math.Min(ratio, 3.0)
	return math.Round(beta*10000) / 10000
}

func lexicalJaccardSimilarity(a, b string) float64 {
	wa := strings.Fields(strings.ToLower(a))
	wb := strings.Fields(strings.ToLower(b))
	if len(wa) == 0 || len(wb) == 0 {
		return 0
	}
	setA := make(map[string]bool, len(wa))
	for _, w := range wa {
		setA[w] = true
	}
	inter := 0
	setB := make(map[string]bool, len(wb))
	for _, w := range wb {
		if !setB[w] {
			setB[w] = true
			if setA[w] {
				inter++
			}
		}
	}
	union := len(setA) + len(setB) - inter
	if union <= 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// EstimateIZY computes mutual information I(Z_i; Y) between candidate Z_i and query Y.
func (s *InformationBottleneckSelector) EstimateIZY(c IBCandidate, query string) float64 {
	rel := c.Relevance
	if rel <= 0 && query != "" {
		rel = lexicalJaccardSimilarity(c.Content, query)
	}
	imp := c.Importance
	if imp <= 0 {
		imp = 0.5
	}
	return 0.70*rel + 0.30*imp
}

// EstimateIZX computes compression & redundancy cost I(Z_i; X | Z_selected).
func (s *InformationBottleneckSelector) EstimateIZX(c IBCandidate, selected []IBCandidate, maxTokens int) float64 {
	toks := c.TokenCount
	if toks <= 0 {
		toks = len(strings.Fields(c.Content)) + 1
	}
	if maxTokens <= 0 {
		maxTokens = 2000
	}
	tokenCost := float64(toks) / float64(maxTokens)
	maxRedundancy := 0.0
	for _, sel := range selected {
		sim := lexicalJaccardSimilarity(c.Content, sel.Content)
		if sim > maxRedundancy {
			maxRedundancy = sim
		}
	}
	return 0.45*tokenCost + 0.55*maxRedundancy
}

// SelectContext selects the subset Z of candidates that maximizes J_IB(Z) = I(Z; Y) - beta * I(Z; X) within maxTokens.
func (s *InformationBottleneckSelector) SelectContext(query string, candidates []IBCandidate, maxTokens int) IBSelectionResult {
	if maxTokens <= 0 {
		maxTokens = 1000
	}
	totalCandTokens := 0
	for i := range candidates {
		if candidates[i].TokenCount <= 0 {
			candidates[i].TokenCount = len(strings.Fields(candidates[i].Content)) + 1
		}
		totalCandTokens += candidates[i].TokenCount
	}
	beta := s.AdaptiveBeta(totalCandTokens, maxTokens)

	used := make([]bool, len(candidates))
	var selected []IBCandidate
	var selectedIDs []string
	usedTokens := 0
	sumIZY := 0.0
	sumIZX := 0.0

	for step := 0; step < len(candidates); step++ {
		bestIdx := -1
		bestGain := -1e9
		bestIZY := 0.0
		bestIZX := 0.0

		for i, cand := range candidates {
			if used[i] || usedTokens+cand.TokenCount > maxTokens {
				continue
			}
			izy := s.EstimateIZY(cand, query)
			izx := s.EstimateIZX(cand, selected, maxTokens)
			gain := izy - beta*izx
			if gain > bestGain {
				bestGain = gain
				bestIdx = i
				bestIZY = izy
				bestIZX = izx
			}
		}
		if bestIdx == -1 || bestGain <= -0.25 {
			break
		}
		used[bestIdx] = true
		chosen := candidates[bestIdx]
		selected = append(selected, chosen)
		selectedIDs = append(selectedIDs, chosen.ID)
		usedTokens += chosen.TokenCount
		sumIZY += bestIZY
		sumIZX += bestIZX
	}

	diversity := 1.0
	if len(selected) > 1 {
		pairSimSum := 0.0
		pairs := 0
		for i := 0; i < len(selected); i++ {
			for j := i + 1; j < len(selected); j++ {
				pairSimSum += lexicalJaccardSimilarity(selected[i].Content, selected[j].Content)
				pairs++
			}
		}
		if pairs > 0 {
			diversity = math.Max(0, 1.0-(pairSimSum/float64(pairs)))
		}
	}

	obj := sumIZY - beta*sumIZX
	return IBSelectionResult{
		SelectedIDs:      selectedIDs,
		SelectedItems:    selected,
		TotalTokens:      usedTokens,
		MaxTokens:        maxTokens,
		AdaptiveBeta:     beta,
		MutualInfoIZY:    math.Round(sumIZY*10000) / 10000,
		CompressionIZX:   math.Round(sumIZX*10000) / 10000,
		IBObjectiveScore: math.Round(obj*10000) / 10000,
		DiversityScore:   math.Round(diversity*10000) / 10000,
	}
}
