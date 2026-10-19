package cognition

import (
	"testing"
)

func TestCognitivePlannerCouncilAndSwarmConsensus(t *testing.T) {
	planner := NewCognitivePlanner()

	// R0 simple intent -> 10 iterations
	if it := planner.DetermineIterations("R0", "Check health endpoint"); it != 10 {
		t.Fatalf("expected 10 iterations for R0, got %d", it)
	}
	// R6 architecture/security intent -> 250 * 1.5 = 375 iterations
	if it := planner.DetermineIterations("R6", "Refactor security architecture"); it != 375 {
		t.Fatalf("expected 375 iterations for R6 refactor, got %d", it)
	}

	plan := planner.CreatePlan("Refactor authentication module", "Existing JWT context", "R4")
	if plan.PlanStatus != "OPTIMAL_PLAN_FOUND" || plan.MCTSIterations != 150 || len(plan.Steps) != 4 {
		t.Fatalf("unexpected plan result: %+v", plan)
	}
	if plan.EstimatedSuccessProb <= 0.70 {
		t.Fatalf("expected high estimated_success_prob, got %f", plan.EstimatedSuccessProb)
	}

	// Dynamic Council Selector
	selector := NewDynamicCouncilSelector(nil)
	q1, err := selector.SelectCouncilQuorum("code_edit", 1, 50.0, 10000.0)
	if err != nil || len(q1) != 1 {
		t.Fatalf("expected 1 model for Tier 1, got %v (err=%v)", q1, err)
	}
	q3, err := selector.SelectCouncilQuorum("arch_review", 3, 50.0, 10000.0)
	if err != nil || len(q3) != 3 {
		t.Fatalf("expected 3 models for Tier 3, got %v (err=%v)", q3, err)
	}
	families := map[string]bool{}
	for _, m := range q3 {
		families[m.ProviderFamily] = true
	}
	if len(families) != 3 {
		t.Fatalf("expected 3 distinct provider families in Tier 3 quorum, got %d", len(families))
	}

	// Swarm Consensus Engine
	consensus := NewSwarmConsensusEngine()
	approved := consensus.EvaluateConsensus([]AgentVote{
		{AgentRole: "rae-hive", Decision: "APPROVE", Confidence: 0.9, Reasoning: "Implementation clean", EvidenceCount: 3},
		{AgentRole: "rae-quality", Decision: "APPROVE", Confidence: 0.95, Reasoning: "AST and tests pass", EvidenceCount: 5},
		{AgentRole: "rae-phoenix", Decision: "REJECT", Confidence: 0.6, Reasoning: "Minor concern", EvidenceCount: 1},
	})
	if approved.Status != "CONSENSUS_APPROVED" || approved.ApprovalRatio < 0.65 {
		t.Fatalf("expected CONSENSUS_APPROVED, got %+v", approved)
	}

	vetoed := consensus.EvaluateConsensus([]AgentVote{
		{AgentRole: "rae-hive", Decision: "APPROVE", Confidence: 0.99, Reasoning: "Looks good", EvidenceCount: 5},
		{AgentRole: "rae-quality", Decision: "VETO", Confidence: 1.0, Reasoning: "Hard contract DTO violation", EvidenceCount: 4},
	})
	if vetoed.Status != "CONSENSUS_VETOED" || vetoed.VetoBy != "rae-quality" {
		t.Fatalf("expected CONSENSUS_VETOED by rae-quality, got %+v", vetoed)
	}
}
