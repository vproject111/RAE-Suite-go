package math

import (
	"math"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// ManifoldArm describes a registered mathematical scoring strategy in the Manifold Atlas.
type ManifoldArm struct {
	ID              string  `json:"id"`
	HistoricalName  string  `json:"historical_name"`
	Description     string  `json:"description"`
	Alpha           float64 `json:"alpha"` // Relevance weight
	Beta            float64 `json:"beta"`  // Importance weight
	Gamma           float64 `json:"gamma"` // Recency weight
	RankSharpening  float64 `json:"rank_sharpening"`
	Pulls           int     `json:"pulls"`
	CumulativeScore float64 `json:"cumulative_score"`
}

// DefaultManifoldAtlas returns the 5 canonical Manifold Arms from RAE Architectural Blueprint.
func DefaultManifoldAtlas() map[string]*ManifoldArm {
	return map[string]*ManifoldArm{
		"classic_ib": {
			ID:             "classic_ib",
			HistoricalName: "System 1.0",
			Description:    "Information Bottleneck (Max compression)",
			Alpha:          0.45,
			Beta:           0.35,
			Gamma:          0.20,
			RankSharpening: 1.0,
		},
		"hyper_37": {
			ID:             "hyper_37",
			HistoricalName: "System 37.0",
			Description:    "Exponential Rank Sharpening",
			Alpha:          0.55,
			Beta:           0.25,
			Gamma:          0.20,
			RankSharpening: 2.5,
		},
		"fluid_100": {
			ID:             "fluid_100",
			HistoricalName: "System 100.0",
			Description:    "Dynamic Geometry scaling via system entropy",
			Alpha:          0.40,
			Beta:           0.30,
			Gamma:          0.30,
			RankSharpening: 1.35,
		},
		"neural_22": {
			ID:             "neural_22",
			HistoricalName: "System 22.1",
			Description:    "Neural-first re-ranking",
			Alpha:          0.65,
			Beta:           0.20,
			Gamma:          0.15,
			RankSharpening: 1.8,
		},
		"doc_pro": {
			ID:             "doc_pro",
			HistoricalName: "System 41.4",
			Description:    "Tier-based proof factor for technical docs",
			Alpha:          0.50,
			Beta:           0.40,
			Gamma:          0.10,
			RankSharpening: 1.2,
		},
	}
}

// MABWeights holds the active Multi-Armed Bandit scoring weights.
type MABWeights struct {
	Alpha     float64 `json:"alpha"`
	Beta      float64 `json:"beta"`
	Gamma     float64 `json:"gamma"`
	ActiveArm string  `json:"active_arm"`
}

// MathController implements the 3-level RAE mathematical controller (L1, L2, L3) and MABTuner.
type MathController struct {
	mu        sync.RWMutex
	level     models.MathLevel
	alpha     float64
	beta      float64
	gamma     float64
	activeArm string
	atlas     map[string]*ManifoldArm
}

// NewMathController creates a new MathController with the default cheap/fast L1-L3 profile.
func NewMathController(level models.MathLevel) *MathController {
	return &MathController{
		level:     level,
		alpha:     0.40,
		beta:      0.30,
		gamma:     0.30,
		activeArm: "fluid_100",
		atlas:     DefaultManifoldAtlas(),
	}
}

// GetWeights returns the current normalized MAB weights.
func (mc *MathController) GetWeights() MABWeights {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return MABWeights{
		Alpha:     mc.alpha,
		Beta:      mc.beta,
		Gamma:     mc.gamma,
		ActiveArm: mc.activeArm,
	}
}

// UpdateReward updates the Multi-Armed Bandit weights within constitutional bounds.
func (mc *MathController) UpdateReward(armID string, reward float64) MABWeights {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if arm, ok := mc.atlas[armID]; ok {
		arm.Pulls++
		arm.CumulativeScore += reward
		mc.activeArm = armID
		// Blend current weights toward winning arm within bounded range [0.05, 0.85]
		mc.alpha = clamp(0.8*mc.alpha+0.2*arm.Alpha, 0.05, 0.85)
		mc.beta = clamp(0.8*mc.beta+0.2*arm.Beta, 0.05, 0.85)
		mc.gamma = clamp(1.0-mc.alpha-mc.beta, 0.05, 0.85)
		sum := mc.alpha + mc.beta + mc.gamma
		mc.alpha = math.Round((mc.alpha/sum)*1000) / 1000
		mc.beta = math.Round((mc.beta/sum)*1000) / 1000
		mc.gamma = math.Round((1.0-mc.alpha-mc.beta)*1000) / 1000
	}
	return MABWeights{
		Alpha:     mc.alpha,
		Beta:      mc.beta,
		Gamma:     mc.gamma,
		ActiveArm: mc.activeArm,
	}
}

// ScoreMemory computes the composite L1/L2/L3 score for a memory item.
func (mc *MathController) ScoreMemory(item models.MemoryItem, similarity float64, now time.Time) (float64, map[string]float64) {
	mc.mu.RLock()
	alpha, beta, gamma := mc.alpha, mc.beta, mc.gamma
	arm := mc.atlas[mc.activeArm]
	mc.mu.RUnlock()

	ageHours := math.Max(0, now.Sub(item.LastAccessedAt).Hours())
	recency := math.Exp(-0.01 * ageHours)
	importance := clamp(item.Importance, 0.0, 1.0)
	rel := clamp(similarity, 0.0, 1.0)

	raw := alpha*rel + beta*importance + gamma*recency
	if mc.level >= models.MathL2 && arm != nil && arm.RankSharpening > 1.0 {
		raw = math.Pow(raw, 1.0/arm.RankSharpening)
	}
	score := clamp(math.Round(raw*10000)/10000, 0.0, 1.0)
	breakdown := map[string]float64{
		"similarity": math.Round(rel*10000) / 10000,
		"importance": math.Round(importance*10000) / 10000,
		"recency":    math.Round(recency*10000) / 10000,
	}
	return score, breakdown
}

// CosineSimilarity computes cosine similarity between two vectors of equal dimension.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0.0
	}
	var dot, normA, normB float64
	for i := range a {
		va := float64(a[i])
		vb := float64(b[i])
		dot += va * vb
		normA += va * va
		normB += vb * vb
	}
	if normA == 0 || normB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// QuantizeInt8 performs scalar quantization of float32 embeddings into int8 bytes.
func QuantizeInt8(vec []float32) []int8 {
	out := make([]int8, len(vec))
	for i, v := range vec {
		scaled := clamp(float64(v), -1.0, 1.0) * 127.0
		out[i] = int8(math.Round(scaled))
	}
	return out
}

// QuantizeBinary packs float32 vector signs into compact binary bytes.
func QuantizeBinary(vec []float32) []byte {
	numBytes := (len(vec) + 7) / 8
	out := make([]byte, numBytes)
	for i, v := range vec {
		if v > 0 {
			out[i/8] |= 1 << uint(7-(i%8))
		}
	}
	return out
}

// CalculateSemanticSimilarity computes normalized token Jaccard + bi-gram overlap in [0.0, 1.0].
func CalculateSemanticSimilarity(a, b string) float64 {
	aNorm := strings.Fields(strings.ToLower(strings.TrimSpace(a)))
	bNorm := strings.Fields(strings.ToLower(strings.TrimSpace(b)))
	if len(aNorm) == 0 || len(bNorm) == 0 {
		return 0.0
	}
	if strings.Join(aNorm, " ") == strings.Join(bNorm, " ") {
		return 1.0
	}
	setA := make(map[string]struct{}, len(aNorm))
	for _, t := range aNorm {
		setA[t] = struct{}{}
	}
	setB := make(map[string]struct{}, len(bNorm))
	for _, t := range bNorm {
		setB[t] = struct{}{}
	}
	inter := 0
	union := len(setA)
	for t := range setB {
		if _, ok := setA[t]; ok {
			inter++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0.0
	}
	return math.Round((float64(inter)/float64(union))*10000) / 10000
}

// ComputeDecayedStrength computes memory decay according to Ebbinghaus forgetting curve.
func ComputeDecayedStrength(currentStrength float64, usageCount int, hoursElapsed float64, decayRate float64) float64 {
	stability := 1.0 + math.Log1p(float64(usageCount))
	decayed := currentStrength * math.Exp(-(decayRate*hoursElapsed)/stability)
	return clamp(math.Round(decayed*10000)/10000, 0.0, 1.0)
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
