package cognition

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ToTNode represents a node in the Tree of Thoughts (ToT) / MCTS tree.
type ToTNode struct {
	StateID  string     `json:"state_id"`
	Action   string     `json:"action"`
	Parent   *ToTNode   `json:"-"`
	Children []*ToTNode `json:"children,omitempty"`
	Visits   int        `json:"visits"`
	Value    float64    `json:"value"`
	PRMScore float64    `json:"prm_score"`
}

// SelectUCT selects the best child node using Upper Confidence Bound for Trees (UCT).
func (n *ToTNode) SelectUCT(explorationWeight float64) *ToTNode {
	if len(n.Children) == 0 {
		return n
	}
	if explorationWeight <= 0 {
		explorationWeight = 1.414
	}
	var bestChild *ToTNode
	bestScore := -math.MaxFloat64
	for _, child := range n.Children {
		var uctScore float64
		if child.Visits == 0 {
			uctScore = math.MaxFloat64
		} else {
			exploitation := child.Value / float64(child.Visits)
			exploration := explorationWeight * math.Sqrt(math.Log(float64(maxInt(n.Visits, 1)))/float64(child.Visits))
			uctScore = exploitation + exploration
		}
		if uctScore > bestScore {
			bestScore = uctScore
			bestChild = child
		}
	}
	return bestChild
}

// CognitivePlanStep represents an evaluated step in a cognitive plan.
type CognitivePlanStep struct {
	StepIndex int     `json:"step_index"`
	Action    string  `json:"action"`
	PRMScore  float64 `json:"prm_score"`
	Visits    int     `json:"visits"`
}

// CognitivePlanResult represents the output of CognitivePlanner.CreatePlan.
type CognitivePlanResult struct {
	PlanStatus           string              `json:"plan_status"`
	RiskClass            string              `json:"risk_class"`
	MCTSIterations       int                 `json:"mcts_iterations"`
	EstimatedSuccessProb float64             `json:"estimated_success_prob"`
	Steps                []CognitivePlanStep `json:"steps"`
}

// CognitivePlanner implements Tree-of-Thoughts + MCTS + Process Reward Model (PRM).
type CognitivePlanner struct {
	BaseIterations map[string]int
}

func NewCognitivePlanner() *CognitivePlanner {
	return &CognitivePlanner{
		BaseIterations: map[string]int{
			"R0": 10,
			"R1": 20,
			"R2": 40,
			"R3": 60,
			"R4": 100,
			"R5": 150,
			"R6": 250,
		},
	}
}

// DetermineIterations dynamically computes MCTS rollouts based on risk class and task complexity.
func (p *CognitivePlanner) DetermineIterations(riskClass string, intent string) int {
	rc := strings.ToUpper(strings.TrimSpace(riskClass))
	base, ok := p.BaseIterations[rc]
	if !ok {
		base = 50
	}
	lowerIntent := strings.ToLower(intent)
	complexityMultiplier := 1.0
	for _, kw := range []string{"refactor", "architecture", "migrate", "security"} {
		if strings.Contains(lowerIntent, kw) {
			complexityMultiplier = 1.5
			break
		}
	}
	return int(float64(base) * complexityMultiplier)
}

// EvaluateStepPRM scores a single step using heuristic Process Reward Model (PRM) rules.
func (p *CognitivePlanner) EvaluateStepPRM(stepContent string, context string) float64 {
	_ = context
	score := 0.7
	lower := strings.ToLower(stepContent)
	if strings.Contains(lower, "verify ") || strings.Contains(lower, "test ") {
		score += 0.15
	}
	if strings.Contains(lower, "assert ") || strings.Contains(lower, "validate") {
		score += 0.10
	}
	if strings.Contains(lower, "rm -rf") || strings.Contains(lower, "drop table") {
		score -= 0.30
	}
	if score < 0.01 {
		score = 0.01
	}
	if score > 0.99 {
		score = 0.99
	}
	return score
}

// CreatePlan builds a multi-step plan using MCTS rollouts and PRM step scoring.
func (p *CognitivePlanner) CreatePlan(intent string, context string, riskClass string) CognitivePlanResult {
	if riskClass == "" {
		riskClass = "R3"
	}
	iterations := p.DetermineIterations(riskClass, intent)
	root := &ToTNode{
		StateID:  "root",
		Action:   fmt.Sprintf("Understand intent: %s", intent),
		PRMScore: 1.0,
	}

	candidateSequence := []string{
		fmt.Sprintf("Analyze context and dependencies for: %s", intent),
		fmt.Sprintf("Formulate execution strategy and validate preconditions for: %s", intent),
		fmt.Sprintf("Execute implementation changes and verify constraints for: %s", intent),
		fmt.Sprintf("Run test suite and assert regression safety for: %s", intent),
	}

	for i := 0; i < iterations; i++ {
		node := root
		depth := 0
		for len(node.Children) > 0 && depth < len(candidateSequence) {
			node = node.SelectUCT(1.414)
			depth++
		}
		if depth < len(candidateSequence) {
			action := candidateSequence[depth]
			prm := p.EvaluateStepPRM(action, context)
			child := &ToTNode{
				StateID:  fmt.Sprintf("step_%d_%d", depth+1, i),
				Action:   action,
				Parent:   node,
				PRMScore: prm,
			}
			node.Children = append(node.Children, child)
			node = child
		}
		reward := node.PRMScore
		curr := node
		for curr != nil {
			curr.Visits++
			curr.Value += reward
			curr = curr.Parent
		}
	}

	var bestPlan []CognitivePlanStep
	curr := root
	stepIdx := 1
	for len(curr.Children) > 0 {
		bestChild := curr.Children[0]
		for _, ch := range curr.Children[1:] {
			if ch.Visits > bestChild.Visits {
				bestChild = ch
			}
		}
		bestPlan = append(bestPlan, CognitivePlanStep{
			StepIndex: stepIdx,
			Action:    bestChild.Action,
			PRMScore:  round4(bestChild.PRMScore),
			Visits:    bestChild.Visits,
		})
		curr = bestChild
		stepIdx++
	}

	avgValue := 0.0
	if root.Visits > 0 {
		avgValue = round4(root.Value / float64(root.Visits))
	}
	return CognitivePlanResult{
		PlanStatus:           "OPTIMAL_PLAN_FOUND",
		RiskClass:            strings.ToUpper(riskClass),
		MCTSIterations:       iterations,
		EstimatedSuccessProb: avgValue,
		Steps:                bestPlan,
	}
}

// ModelSpec defines a model candidate in the Dynamic Council Catalog.
type ModelSpec struct {
	ModelID              string   `json:"model_id"`
	ProviderFamily       string   `json:"provider_family"`
	QualityScore         float64  `json:"quality_score"`
	CostPer1M            float64  `json:"cost_per_1m"`
	AvgLatencyMs         float64  `json:"avg_latency_ms"`
	SupportedTaskClasses []string `json:"supported_task_classes"`
}

// DynamicCouncilSelector selects a Pareto-optimal, provider-diverse quorum of models.
type DynamicCouncilSelector struct {
	Catalog []ModelSpec
}

func NewDynamicCouncilSelector(catalog []ModelSpec) *DynamicCouncilSelector {
	if len(catalog) == 0 {
		catalog = DefaultModelCatalog()
	}
	return &DynamicCouncilSelector{Catalog: catalog}
}

func DefaultModelCatalog() []ModelSpec {
	return []ModelSpec{
		{
			ModelID:              "z-ai/glm-5.3-flash",
			ProviderFamily:       "zhipu",
			QualityScore:         0.84,
			CostPer1M:            0.08,
			AvgLatencyMs:         650,
			SupportedTaskClasses: []string{"code_edit", "arch_review", "security_audit", "fast_check"},
		},
		{
			ModelID:              "deepseek/deepseek-v4-pro-0813",
			ProviderFamily:       "deepseek",
			QualityScore:         0.91,
			CostPer1M:            0.66,
			AvgLatencyMs:         1100,
			SupportedTaskClasses: []string{"code_edit", "arch_review", "security_audit"},
		},
		{
			ModelID:              "moonshotai/kimi-k2.7-code",
			ProviderFamily:       "moonshot",
			QualityScore:         0.89,
			CostPer1M:            0.85,
			AvgLatencyMs:         1250,
			SupportedTaskClasses: []string{"code_edit", "arch_review", "security_audit"},
		},
		{
			ModelID:              "openai/gpt-5.6-sol",
			ProviderFamily:       "openai",
			QualityScore:         0.96,
			CostPer1M:            2.20,
			AvgLatencyMs:         1600,
			SupportedTaskClasses: []string{"code_edit", "arch_review", "security_audit"},
		},
		{
			ModelID:              "anthropic/claude-sonnet-5",
			ProviderFamily:       "anthropic",
			QualityScore:         0.97,
			CostPer1M:            2.55,
			AvgLatencyMs:         1550,
			SupportedTaskClasses: []string{"code_edit", "arch_review", "security_audit"},
		},
	}
}

type scoredModel struct {
	score float64
	spec  ModelSpec
}

// SelectCouncilQuorum selects models based on Pareto score and Risk Tier diversity rules.
func (s *DynamicCouncilSelector) SelectCouncilQuorum(taskClass string, riskTier int, maxCostPer1M float64, maxLatencyMs float64) ([]ModelSpec, error) {
	if maxCostPer1M <= 0 {
		maxCostPer1M = 50.0
	}
	if maxLatencyMs <= 0 {
		maxLatencyMs = 10000.0
	}

	var candidates []ModelSpec
	for _, m := range s.Catalog {
		if m.CostPer1M <= maxCostPer1M && m.AvgLatencyMs <= maxLatencyMs && hasTaskClass(m.SupportedTaskClasses, taskClass) {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no models in catalog satisfy constraints for task_class=%s", taskClass)
	}

	scored := make([]scoredModel, 0, len(candidates))
	for _, m := range candidates {
		normCost := math.Min(m.CostPer1M/10.0, 1.0)
		normLatency := math.Min(m.AvgLatencyMs/5000.0, 1.0)
		paretoScore := (0.60 * m.QualityScore) - (0.25 * normCost) - (0.15 * normLatency)
		scored = append(scored, scoredModel{score: paretoScore, spec: m})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if riskTier <= 1 {
		return []ModelSpec{scored[0].spec}, nil
	}

	if riskTier == 2 {
		selected := []ModelSpec{scored[0].spec}
		for _, item := range scored[1:] {
			if item.spec.ProviderFamily != selected[0].ProviderFamily {
				selected = append(selected, item.spec)
				break
			}
		}
		return selected, nil
	}

	// Tier 3+: require 3 models from 3 distinct provider families
	var selected []ModelSpec
	seenFamilies := make(map[string]bool)
	for _, item := range scored {
		if !seenFamilies[item.spec.ProviderFamily] {
			selected = append(selected, item.spec)
			seenFamilies[item.spec.ProviderFamily] = true
			if len(selected) == 3 {
				break
			}
		}
	}
	if len(selected) < 3 {
		return nil, fmt.Errorf("insufficient provider diversity for Tier 3 council (found %d families, need 3)", len(selected))
	}
	return selected, nil
}

// AgentVote represents a single agent's vote in SwarmConsensusEngine.
type AgentVote struct {
	AgentRole     string  `json:"agent_role"`
	Decision      string  `json:"decision"` // APPROVE, REJECT, VETO
	Confidence    float64 `json:"confidence"`
	Reasoning     string  `json:"reasoning"`
	EvidenceCount int     `json:"evidence_count"`
}

// ConsensusResult represents the outcome of SwarmConsensusEngine.EvaluateConsensus.
type ConsensusResult struct {
	Status        string  `json:"status"` // CONSENSUS_APPROVED, CONSENSUS_REJECTED, CONSENSUS_VETOED, NO_VOTES
	ApprovalRatio float64 `json:"approval_ratio,omitempty"`
	Threshold     float64 `json:"threshold,omitempty"`
	TotalVoters   int     `json:"total_voters,omitempty"`
	VetoBy        string  `json:"veto_by,omitempty"`
	Reason        string  `json:"reason,omitempty"`
}

// SwarmConsensusEngine resolves multi-agent disagreements using weighted voting and VETO power.
type SwarmConsensusEngine struct {
	RoleWeights map[string]float64
}

func NewSwarmConsensusEngine() *SwarmConsensusEngine {
	return &SwarmConsensusEngine{
		RoleWeights: map[string]float64{
			"rae-hive":       1.0,
			"rae-phoenix":    1.0,
			"rae-quality":    2.0,
			"human_override": 100.0,
		},
	}
}

// EvaluateConsensus computes weighted multi-agent consensus with VETO enforcement.
func (e *SwarmConsensusEngine) EvaluateConsensus(votes []AgentVote) ConsensusResult {
	if len(votes) == 0 {
		return ConsensusResult{
			Status: "NO_VOTES",
			Reason: "Empty vote list",
		}
	}

	// Check for VETO from rae-quality
	for _, v := range votes {
		if v.AgentRole == "rae-quality" && strings.EqualFold(v.Decision, "VETO") {
			return ConsensusResult{
				Status: "CONSENSUS_VETOED",
				VetoBy: v.AgentRole,
				Reason: v.Reasoning,
			}
		}
	}

	approveScore := 0.0
	rejectScore := 0.0
	totalWeight := 0.0

	for _, v := range votes {
		weight, ok := e.RoleWeights[v.AgentRole]
		if !ok {
			weight = 1.0
		}
		evidenceBonus := 1.0 + (0.1 * float64(minInt(v.EvidenceCount, 5)))
		voteScore := weight * v.Confidence * evidenceBonus
		if strings.EqualFold(v.Decision, "APPROVE") {
			approveScore += voteScore
		} else {
			rejectScore += voteScore
		}
		totalWeight += voteScore
	}

	approvalRatio := 0.0
	if totalWeight > 0 {
		approvalRatio = approveScore / totalWeight
	}
	status := "CONSENSUS_REJECTED"
	if approvalRatio >= 0.65 {
		status = "CONSENSUS_APPROVED"
	}

	return ConsensusResult{
		Status:        status,
		ApprovalRatio: round4(approvalRatio),
		Threshold:     0.65,
		TotalVoters:   len(votes),
	}
}

func hasTaskClass(classes []string, target string) bool {
	for _, c := range classes {
		if c == target {
			return true
		}
	}
	return false
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

func round4(val float64) float64 {
	return math.Round(val*10000) / 10000
}
