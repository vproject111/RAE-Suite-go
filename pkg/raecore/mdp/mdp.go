package mdp

import (
	"math"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// ActionType enumerates the 11 canonical RAE Markov Decision Process (MDP) cognitive actions
// from apps/memory_api/core/actions.py.
type ActionType string

const (
	ActionRetrieveEpisodic   ActionType = "retrieve_episodic"
	ActionRetrieveWorking    ActionType = "retrieve_working"
	ActionRetrieveSemantic   ActionType = "retrieve_semantic"
	ActionRetrieveLTM        ActionType = "retrieve_ltm"
	ActionCallLLM            ActionType = "call_llm"
	ActionPruneContext       ActionType = "prune_context"
	ActionConsolidateMemory  ActionType = "consolidate_memory"
	ActionGenerateReflection ActionType = "generate_reflection"
	ActionUpdateGraph        ActionType = "update_graph"
	ActionCrystallizePattern ActionType = "crystallize_pattern"
	ActionNoOp               ActionType = "no_op"
)

// AllActionTypes returns all 11 canonical MDP actions.
func AllActionTypes() []ActionType {
	return []ActionType{
		ActionRetrieveEpisodic,
		ActionRetrieveWorking,
		ActionRetrieveSemantic,
		ActionRetrieveLTM,
		ActionCallLLM,
		ActionPruneContext,
		ActionConsolidateMemory,
		ActionGenerateReflection,
		ActionUpdateGraph,
		ActionCrystallizePattern,
		ActionNoOp,
	}
}

// WorkingContextState captures the working context window utilization at step t.
type WorkingContextState struct {
	TokenCount    int     `json:"token_count"`
	MaxTokens     int     `json:"max_tokens"`
	ItemsCount    int     `json:"items_count"`
	AvgImportance float64 `json:"avg_importance"`
	AvgCentrality float64 `json:"avg_centrality"`
	TokenPressure float64 `json:"token_pressure"`
}

// MemoryDistributionState captures memory counts and Shannon entropy across layers.
type MemoryDistributionState struct {
	EpisodicCount   int     `json:"episodic_count"`
	WorkingCount    int     `json:"working_count"`
	SemanticCount   int     `json:"semantic_count"`
	LTMCount        int     `json:"ltm_count"`
	ReflectiveCount int     `json:"reflective_count"`
	TotalCount      int     `json:"total_count"`
	LayerEntropy    float64 `json:"layer_entropy"`
}

// BudgetState captures remaining token, USD, and latency budgets.
type BudgetState struct {
	RemainingTokens  int     `json:"remaining_tokens"`
	RemainingCostUSD float64 `json:"remaining_cost_usd"`
	RemainingCalls   int     `json:"remaining_calls"`
	LatencyBudgetMs  float64 `json:"latency_budget_ms"`
	IsExhausted      bool    `json:"is_exhausted"`
}

// GraphTopologicalState captures macroscopic knowledge graph topology metrics.
type GraphTopologicalState struct {
	NodeCount           int     `json:"node_count"`
	EdgeCount           int     `json:"edge_count"`
	Density             float64 `json:"density"`
	ConnectedComponents int     `json:"connected_components"`
	AvgClustering       float64 `json:"avg_clustering"`
}

// RAEState represents state s_t in the RAE Markov Decision Process (apps/memory_api/core/state.py).
type RAEState struct {
	Step           int                     `json:"step"`
	WorkingContext WorkingContextState     `json:"working_context"`
	MemoryState    MemoryDistributionState `json:"memory_state"`
	BudgetState    BudgetState             `json:"budget_state"`
	GraphState     GraphTopologicalState   `json:"graph_state"`
	Timestamp      time.Time               `json:"timestamp"`
}

// ComputeStateFromMemories constructs a live RAEState s_t from current memory items, graph stats, and budget.
func ComputeStateFromMemories(items []models.MemoryItem, maxContextTokens int, remainingTokens int, remainingUSD float64, nodeCount, edgeCount int) RAEState {
	if maxContextTokens <= 0 {
		maxContextTokens = 8192
	}
	var ep, wk, sem, ltm, ref int
	var wkTokens int
	var impSum float64
	for _, it := range items {
		impSum += it.Importance
		switch it.Layer {
		case models.LayerEpisodic:
			ep++
		case models.LayerWorking:
			wk++
			wkTokens += len(it.Content)/4 + 12
		case models.LayerSemantic:
			sem++
		case models.LayerLTM:
			ltm++
		case models.LayerReflective:
			ref++
		default:
			ep++
		}
	}
	total := len(items)
	avgImp := 0.5
	if total > 0 {
		avgImp = math.Round((impSum/float64(total))*10000) / 10000
	}
	entropy := 0.0
	if total > 0 {
		counts := []int{ep, wk, sem, ltm, ref}
		for _, c := range counts {
			if c > 0 {
				p := float64(c) / float64(total)
				entropy -= p * math.Log2(p)
			}
		}
		entropy = math.Round(entropy*10000) / 10000
	}
	density := 0.0
	if nodeCount > 1 {
		density = math.Round((float64(edgeCount)/float64(nodeCount*(nodeCount-1)))*10000) / 10000
	}
	pressure := math.Min(1.0, math.Round((float64(wkTokens)/float64(maxContextTokens))*10000)/10000)

	return RAEState{
		Step: 1,
		WorkingContext: WorkingContextState{
			TokenCount:    wkTokens,
			MaxTokens:     maxContextTokens,
			ItemsCount:    wk,
			AvgImportance: avgImp,
			AvgCentrality: math.Min(1.0, density*2.0+0.15),
			TokenPressure: pressure,
		},
		MemoryState: MemoryDistributionState{
			EpisodicCount:   ep,
			WorkingCount:    wk,
			SemanticCount:   sem,
			LTMCount:        ltm,
			ReflectiveCount: ref,
			TotalCount:      total,
			LayerEntropy:    entropy,
		},
		BudgetState: BudgetState{
			RemainingTokens:  remainingTokens,
			RemainingCostUSD: remainingUSD,
			RemainingCalls:   100,
			LatencyBudgetMs:  1500.0,
			IsExhausted:      remainingTokens <= 0 || remainingUSD <= 0,
		},
		GraphState: GraphTopologicalState{
			NodeCount:           nodeCount,
			EdgeCount:           edgeCount,
			Density:             density,
			ConnectedComponents: maxInt(1, nodeCount/maxInt(1, edgeCount+1)),
			AvgClustering:       math.Min(1.0, density*1.5),
		},
		Timestamp: time.Now().UTC(),
	}
}

// RewardHyperparameters defines the Lagrangian multipliers (lambda, mu, nu) for R(s_t, a_t, s_{t+1}).
type RewardHyperparameters struct {
	LambdaTokenWeight   float64 `json:"lambda_token_weight"`
	MuLatencyWeight     float64 `json:"mu_latency_weight"`
	NuSafetyBonusWeight float64 `json:"nu_safety_bonus_weight"`
}

// DefaultRewardHyperparameters returns the canonical RAE MDP reward parameters.
func DefaultRewardHyperparameters() RewardHyperparameters {
	return RewardHyperparameters{
		LambdaTokenWeight:   0.001,
		MuLatencyWeight:     0.0005,
		NuSafetyBonusWeight: 0.10,
	}
}

// TransitionOutcome captures the inputs and decomposed reward components of a state transition.
type TransitionOutcome struct {
	Action           ActionType `json:"action"`
	QualityGain      float64    `json:"quality_gain"`
	TokensConsumed   int        `json:"tokens_consumed"`
	LatencyMs        float64    `json:"latency_ms"`
	SafetyCompliant  bool       `json:"safety_compliant"`
	TokenPenalty     float64    `json:"token_penalty"`
	LatencyPenalty   float64    `json:"latency_penalty"`
	SafetyComponent  float64    `json:"safety_component"`
	TotalReward      float64    `json:"total_reward"`
	InformationRatio float64    `json:"information_ratio"`
	NextState        RAEState   `json:"next_state"`
}

// ComputeReward calculates R(s_t, a_t, s_{t+1}) = QualityGain - lambda*TokenCost - mu*LatencyCost + nu*SafetyBonus.
func ComputeReward(params RewardHyperparameters, qualityGain float64, tokensConsumed int, latencyMs float64, safetyCompliant bool) (total float64, tokPen, latPen, safComp float64) {
	tokPen = math.Round(params.LambdaTokenWeight*float64(tokensConsumed)*10000) / 10000
	latPen = math.Round(params.MuLatencyWeight*latencyMs*10000) / 10000
	if safetyCompliant {
		safComp = params.NuSafetyBonusWeight
	} else {
		safComp = -1.0 // Hard constitutional penalty for budget/safety violation
	}
	total = math.Round((qualityGain-tokPen-latPen+safComp)*10000) / 10000
	return total, tokPen, latPen, safComp
}

// MDPMetrics tracks cumulative MDP performance across agent episodes.
type MDPMetrics struct {
	TotalTransitions          int                `json:"total_transitions"`
	CumulativeReward          float64            `json:"cumulative_reward"`
	AvgReward                 float64            `json:"avg_reward"`
	AvgEfficiencyScore        float64            `json:"avg_efficiency_score"`
	SafetyViolationsPrevented int                `json:"safety_violations_prevented"`
	ActionCounts              map[ActionType]int `json:"action_counts"`
}

// Engine executes MDP policy selection, transitions, and reward accounting.
type Engine struct {
	mu      sync.RWMutex
	params  RewardHyperparameters
	metrics MDPMetrics
}

// NewEngine creates a new RAE MDP Engine.
func NewEngine() *Engine {
	return &Engine{
		params: DefaultRewardHyperparameters(),
		metrics: MDPMetrics{
			ActionCounts: make(map[ActionType]int),
		},
	}
}

// SelectOptimalAction evaluates Q(s_t, a) across all 11 actions and returns the optimal action and Q-table.
func (e *Engine) SelectOptimalAction(state RAEState) (ActionType, map[ActionType]float64, string) {
	qValues := make(map[ActionType]float64, 11)

	if state.BudgetState.IsExhausted || state.BudgetState.RemainingTokens <= 0 {
		e.mu.Lock()
		e.metrics.SafetyViolationsPrevented++
		e.mu.Unlock()
		for _, a := range AllActionTypes() {
			qValues[a] = -1.0
		}
		qValues[ActionPruneContext] = 0.85
		qValues[ActionNoOp] = 0.95
		return ActionNoOp, qValues, "budget_exhausted_hard_guardrail"
	}

	pressure := state.WorkingContext.TokenPressure
	qValues[ActionPruneContext] = round4(pressure*1.25 - 0.15)
	qValues[ActionConsolidateMemory] = round4(float64(state.MemoryState.EpisodicCount)*0.04 + pressure*0.5)
	qValues[ActionGenerateReflection] = round4(float64(state.MemoryState.EpisodicCount)*0.05 + (1.0-pressure)*0.35)
	qValues[ActionRetrieveWorking] = round4((1.0 - pressure) * 0.65)
	qValues[ActionRetrieveSemantic] = round4((1.0-pressure)*0.75 + 0.15)
	qValues[ActionRetrieveEpisodic] = round4((1.0 - pressure) * 0.60)
	qValues[ActionRetrieveLTM] = round4((1.0 - pressure) * 0.55)
	qValues[ActionUpdateGraph] = round4((1.0 - state.GraphState.Density) * 0.55)
	qValues[ActionCrystallizePattern] = round4(float64(state.MemoryState.ReflectiveCount) * 0.08)
	qValues[ActionCallLLM] = round4((1.0-pressure)*0.70 - e.params.LambdaTokenWeight*350.0)
	qValues[ActionNoOp] = 0.10

	bestAction := ActionRetrieveSemantic
	bestQ := -999.0
	rationale := "semantic_retrieval_optimal_information_gain"

	for _, a := range AllActionTypes() {
		q := qValues[a]
		if q > bestQ {
			bestQ = q
			bestAction = a
		}
	}

	switch bestAction {
	case ActionPruneContext:
		rationale = "high_working_context_token_pressure_requires_pruning"
	case ActionConsolidateMemory:
		rationale = "episodic_buffer_ready_for_consolidation"
	case ActionGenerateReflection:
		rationale = "sufficient_episodic_density_for_reflection_synthesis"
	}

	return bestAction, qValues, rationale
}

// Step executes action a_t on state s_t, produces s_{t+1}, computes R(s_t, a_t, s_{t+1}), and updates MDPMetrics.
func (e *Engine) Step(state RAEState, action ActionType, qualityGain float64, tokensConsumed int, latencyMs float64) TransitionOutcome {
	next := state
	next.Step = state.Step + 1
	next.Timestamp = time.Now().UTC()

	if tokensConsumed <= 0 {
		switch action {
		case ActionPruneContext:
			tokensConsumed = 10
		case ActionNoOp:
			tokensConsumed = 0
		case ActionCallLLM, ActionGenerateReflection:
			tokensConsumed = 320
		default:
			tokensConsumed = 45
		}
	}
	if latencyMs <= 0 {
		switch action {
		case ActionCallLLM, ActionGenerateReflection:
			latencyMs = 180.0
		case ActionNoOp:
			latencyMs = 0.5
		default:
			latencyMs = 8.0
		}
	}
	if qualityGain == 0 {
		switch action {
		case ActionPruneContext:
			qualityGain = 0.45
		case ActionConsolidateMemory, ActionGenerateReflection, ActionCrystallizePattern:
			qualityGain = 0.82
		case ActionRetrieveSemantic, ActionRetrieveWorking, ActionRetrieveEpisodic, ActionRetrieveLTM:
			qualityGain = 0.74
		case ActionCallLLM:
			qualityGain = 0.88
		case ActionUpdateGraph:
			qualityGain = 0.65
		default:
			qualityGain = 0.05
		}
	}

	switch action {
	case ActionPruneContext:
		next.WorkingContext.TokenCount = maxInt(0, next.WorkingContext.TokenCount/2)
		next.WorkingContext.TokenPressure = round4(float64(next.WorkingContext.TokenCount) / float64(maxInt(1, next.WorkingContext.MaxTokens)))
	case ActionConsolidateMemory:
		moved := minInt(5, next.MemoryState.EpisodicCount)
		next.MemoryState.EpisodicCount -= moved
		next.MemoryState.SemanticCount += moved
	case ActionGenerateReflection:
		next.MemoryState.ReflectiveCount++
		next.MemoryState.TotalCount++
	case ActionUpdateGraph:
		next.GraphState.EdgeCount += 2
	}

	next.BudgetState.RemainingTokens = maxInt(0, state.BudgetState.RemainingTokens-tokensConsumed)
	next.BudgetState.RemainingCostUSD = math.Max(0, round4(state.BudgetState.RemainingCostUSD-float64(tokensConsumed)*0.000001))
	next.BudgetState.IsExhausted = next.BudgetState.RemainingTokens <= 0 || next.BudgetState.RemainingCostUSD <= 0

	safetyOK := !state.BudgetState.IsExhausted || action == ActionNoOp || action == ActionPruneContext
	totalReward, tokPen, latPen, safComp := ComputeReward(e.params, qualityGain, tokensConsumed, latencyMs, safetyOK)

	infoRatio := qualityGain
	if tokensConsumed > 0 {
		infoRatio = round4((qualityGain * 100.0) / float64(tokensConsumed))
	}

	e.mu.Lock()
	e.metrics.TotalTransitions++
	e.metrics.CumulativeReward = round4(e.metrics.CumulativeReward + totalReward)
	e.metrics.AvgReward = round4(e.metrics.CumulativeReward / float64(e.metrics.TotalTransitions))
	e.metrics.AvgEfficiencyScore = round4((e.metrics.AvgEfficiencyScore*float64(e.metrics.TotalTransitions-1) + infoRatio) / float64(e.metrics.TotalTransitions))
	e.metrics.ActionCounts[action]++
	e.mu.Unlock()

	return TransitionOutcome{
		Action:           action,
		QualityGain:      qualityGain,
		TokensConsumed:   tokensConsumed,
		LatencyMs:        latencyMs,
		SafetyCompliant:  safetyOK,
		TokenPenalty:     tokPen,
		LatencyPenalty:   latPen,
		SafetyComponent:  safComp,
		TotalReward:      totalReward,
		InformationRatio: infoRatio,
		NextState:        next,
	}
}

// GetMetrics returns a snapshot of the cumulative MDP metrics.
func (e *Engine) GetMetrics() MDPMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()
	counts := make(map[ActionType]int, len(e.metrics.ActionCounts))
	for k, v := range e.metrics.ActionCounts {
		counts[k] = v
	}
	out := e.metrics
	out.ActionCounts = counts
	return out
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
