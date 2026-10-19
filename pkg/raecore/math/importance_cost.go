package math

import (
	"math"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// SixFactorImportanceBreakdown holds the 6 sub-factors and composite score from ImportanceScoringService
// (apps/memory_api/services/importance_scoring.py).
type SixFactorImportanceBreakdown struct {
	Novelty         float64 `json:"novelty"`
	Relevance       float64 `json:"relevance"`
	RiskIntensity   float64 `json:"risk_intensity"`
	Frequency       float64 `json:"frequency"`
	Recency         float64 `json:"recency"`
	GraphCentrality float64 `json:"graph_centrality"`
	CompositeScore  float64 `json:"composite_score"`
}

// ComputeSixFactorImportance computes the multi-factor importance score I in [0.15, 1.0] for a memory item.
func ComputeSixFactorImportance(item models.MemoryItem, existing []models.MemoryItem, graphDegree int) SixFactorImportanceBreakdown {
	// 1. Novelty: 1 - max similarity against up to 25 recent memories
	maxSim := 0.0
	limit := len(existing)
	if limit > 25 {
		limit = 25
	}
	for i := 0; i < limit; i++ {
		if existing[i].ID == item.ID {
			continue
		}
		sim := CalculateSemanticSimilarity(item.Content, existing[i].Content)
		if sim > maxSim {
			maxSim = sim
		}
	}
	novelty := clamp(1.0-maxSim*0.85, 0.15, 1.0)

	// 2. Relevance: structural density, tags, layer weight
	lower := strings.ToLower(item.Content)
	rel := 0.50
	for _, kw := range []string{"architecture", "decision", "contract", "invariant", "fix", "benchmark", "protocol", "schema", "migration"} {
		if strings.Contains(lower, kw) {
			rel += 0.10
		}
	}
	rel += float64(len(item.Tags)) * 0.05
	if item.Layer == models.LayerReflective || item.Layer == models.LayerSemantic {
		rel += 0.15
	}
	relevance := clamp(rel, 0.20, 1.0)

	// 3. RiskIntensity: security, incident, CVE, critical, production signals
	risk := 0.35
	for _, kw := range []string{"security", "cve", "critical", "error", "failure", "regression", "restricted", "pii", "incident", "panic", "leak"} {
		if strings.Contains(lower, kw) {
			risk += 0.15
		}
	}
	if item.InfoClass == models.InfoRestricted || item.InfoClass == models.InfoConfidential {
		risk += 0.25
	}
	riskIntensity := clamp(risk, 0.20, 1.0)

	// 4. Frequency: log-scaled usage count
	freq := clamp(0.40+math.Log1p(float64(item.UsageCount))*0.20, 0.30, 1.0)

	// 5. Recency: exponential decay from CreatedAt / LastAccessedAt
	refTime := item.LastAccessedAt
	if refTime.IsZero() {
		refTime = item.CreatedAt
	}
	ageHours := 0.0
	if !refTime.IsZero() {
		ageHours = math.Max(0, time.Since(refTime).Hours())
	}
	recency := clamp(math.Exp(-0.008*ageHours), 0.20, 1.0)

	// 6. GraphCentrality: degree centrality in knowledge graph
	centrality := clamp(0.30+float64(graphDegree)*0.10, 0.20, 1.0)

	composite := 0.25*novelty +
		0.20*relevance +
		0.20*riskIntensity +
		0.15*freq +
		0.10*recency +
		0.10*centrality
	composite = clamp(math.Round(composite*10000)/10000, 0.15, 1.0)

	return SixFactorImportanceBreakdown{
		Novelty:         math.Round(novelty*10000) / 10000,
		Relevance:       math.Round(relevance*10000) / 10000,
		RiskIntensity:   math.Round(riskIntensity*10000) / 10000,
		Frequency:       math.Round(freq*10000) / 10000,
		Recency:         math.Round(recency*10000) / 10000,
		GraphCentrality: math.Round(centrality*10000) / 10000,
		CompositeScore:  composite,
	}
}

// ModelPricingEntry defines per-1M token input/output USD rates for a model in CostController
// (apps/memory_api/services/cost_controller.py).
type ModelPricingEntry struct {
	ModelID          string  `json:"model_id"`
	Tier             string  `json:"tier"`
	InputPer1MUSD    float64 `json:"input_per_1m_usd"`
	OutputPer1MUSD   float64 `json:"output_per_1m_usd"`
	CacheHitDiscount float64 `json:"cache_hit_discount"`
}

// CanonicalModelPricingTable returns the 9-model RAE Triad + Tier 0 Local GPU pricing table.
func CanonicalModelPricingTable() map[string]ModelPricingEntry {
	return map[string]ModelPricingEntry{
		"deepseek-r1:8b":              {ModelID: "deepseek-r1:8b", Tier: "tier0_local_gpu", InputPer1MUSD: 0.0, OutputPer1MUSD: 0.0, CacheHitDiscount: 1.0},
		"qwen3.5:9b":                  {ModelID: "qwen3.5:9b", Tier: "tier0_local_gpu", InputPer1MUSD: 0.0, OutputPer1MUSD: 0.0, CacheHitDiscount: 1.0},
		"z-ai/glm-5.3-flash":          {ModelID: "z-ai/glm-5.3-flash", Tier: "tier1_generate", InputPer1MUSD: 0.05, OutputPer1MUSD: 0.10, CacheHitDiscount: 0.50},
		"deepseek/deepseek-v4-flash":  {ModelID: "deepseek/deepseek-v4-flash", Tier: "tier1_generate", InputPer1MUSD: 0.07, OutputPer1MUSD: 0.14, CacheHitDiscount: 0.50},
		"xiaomi/mimo-v2.5":            {ModelID: "xiaomi/mimo-v2.5", Tier: "tier1_generate", InputPer1MUSD: 0.08, OutputPer1MUSD: 0.16, CacheHitDiscount: 0.50},
		"deepseek/deepseek-v4-pro-0813": {ModelID: "deepseek/deepseek-v4-pro-0813", Tier: "tier2_review_repair", InputPer1MUSD: 0.66, OutputPer1MUSD: 1.32, CacheHitDiscount: 0.50},
		"z-ai/glm-5.3":                  {ModelID: "z-ai/glm-5.3", Tier: "tier2_review_repair", InputPer1MUSD: 0.85, OutputPer1MUSD: 1.70, CacheHitDiscount: 0.50},
		"moonshotai/kimi-k2.7-code":     {ModelID: "moonshotai/kimi-k2.7-code", Tier: "tier2_review_repair", InputPer1MUSD: 1.17, OutputPer1MUSD: 2.34, CacheHitDiscount: 0.50},
		"openai/gpt-5.6-sol":          {ModelID: "openai/gpt-5.6-sol", Tier: "tier3_judge", InputPer1MUSD: 2.00, OutputPer1MUSD: 6.00, CacheHitDiscount: 0.50},
		"moonshotai/kimi-k3":          {ModelID: "moonshotai/kimi-k3", Tier: "tier3_judge", InputPer1MUSD: 2.20, OutputPer1MUSD: 5.50, CacheHitDiscount: 0.50},
		"anthropic/claude-sonnet-5":   {ModelID: "anthropic/claude-sonnet-5", Tier: "tier3_judge", InputPer1MUSD: 2.55, OutputPer1MUSD: 7.50, CacheHitDiscount: 0.50},
	}
}

// CalculateModelCost computes the exact USD cost and cache savings for an LLM invocation.
func CalculateModelCost(modelID string, promptTokens, completionTokens, cachedTokens int) (costUSD float64, savedUSD float64, tier string) {
	table := CanonicalModelPricingTable()
	entry, ok := table[modelID]
	if !ok {
		if strings.Contains(modelID, "local") || strings.Contains(modelID, "ollama") {
			return 0.0, 0.0, "tier0_local_gpu"
		}
		entry = ModelPricingEntry{ModelID: modelID, Tier: "tier1_generate", InputPer1MUSD: 0.10, OutputPer1MUSD: 0.20, CacheHitDiscount: 0.50}
	}
	uncachedInput := promptTokens - cachedTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	inputCost := (float64(uncachedInput) / 1_000_000.0) * entry.InputPer1MUSD
	cachedCost := (float64(cachedTokens) / 1_000_000.0) * entry.InputPer1MUSD * (1.0 - entry.CacheHitDiscount)
	outputCost := (float64(completionTokens) / 1_000_000.0) * entry.OutputPer1MUSD

	costUSD = math.Round((inputCost+cachedCost+outputCost)*1_000_000) / 1_000_000
	savedUSD = math.Round(((float64(cachedTokens)/1_000_000.0)*entry.InputPer1MUSD*entry.CacheHitDiscount)*1_000_000) / 1_000_000
	return costUSD, savedUSD, entry.Tier
}
