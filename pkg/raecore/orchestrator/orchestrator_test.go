package orchestrator

import (
	"testing"
)

func TestOrchestratorFullCoverage(t *testing.T) {
	types := RegisteredPackageTypes()
	if len(types) == 0 {
		t.Fatal("expected RegisteredPackageTypes to return non-empty slice")
	}

	mr := NewModelRouter()
	if mr.DefaultTier != "tier1" {
		t.Fatalf("unexpected DefaultTier: %s", mr.DefaultTier)
	}
	if mr.ChoosePlanner("high") != "openai/gpt-5.6-sol" {
		t.Fatal("unexpected high planner")
	}
	if mr.ChoosePlanner("critical") != "openai/gpt-5.6-sol" {
		t.Fatal("unexpected critical planner")
	}
	if mr.ChoosePlanner("low") != "z-ai/glm-5.3" {
		t.Fatal("unexpected low planner")
	}
	if mr.ChoosePlanReviewer("medium") != "deepseek/deepseek-v4-pro-0813" {
		t.Fatal("unexpected plan reviewer")
	}
	if mr.ChooseImplementer("low") != "z-ai/glm-5.3-flash" {
		t.Fatal("unexpected low implementer")
	}
	if mr.ChooseImplementer("high") != "deepseek/deepseek-v4-flash" {
		t.Fatal("unexpected high implementer")
	}
	if mr.ChooseCodeReviewer("high") != "anthropic/claude-sonnet-5" {
		t.Fatal("unexpected high code reviewer")
	}
	if mr.ChooseCodeReviewer("low") != "moonshotai/kimi-k2.7-code" {
		t.Fatal("unexpected low code reviewer")
	}
	if mr.RouteTask("low") != "z-ai/glm-5.3-flash" {
		t.Fatal("unexpected RouteTask")
	}
	if len(mr.GetTribunalQuorumModels()) != 3 {
		t.Fatal("expected 3 tribunal quorum models")
	}

	v2 := NewModelRouterV2()
	if mr.CalculateEstimatedCost(1000000, "tier1") != 0.12 {
		t.Fatalf("unexpected tier1 cost: %f", mr.CalculateEstimatedCost(1000000, "tier1"))
	}
	if v2.EstimateTaskCost(1000000, "tier2") != 0.85 {
		t.Fatalf("unexpected tier2 cost: %f", v2.EstimateTaskCost(1000000, "tier2"))
	}
	if v2.EstimateTaskCost(1000000, "tier3") != 2.25 {
		t.Fatalf("unexpected tier3 cost: %f", v2.EstimateTaskCost(1000000, "tier3"))
	}
	if v2.EstimateTaskCost(1000000, "tier0") != 0.0 {
		t.Fatalf("unexpected tier0 cost: %f", v2.EstimateTaskCost(1000000, "tier0"))
	}

	arNil := NewAdaptiveModelRouter(nil)
	if arNil.ChoosePlanner("low") == "" || arNil.ChooseImplementer("low") == "" {
		t.Fatal("unexpected empty adaptive router choice")
	}
	if arNil.AdaptStrategy(0.10) != "maintain_tier1_generate" {
		t.Fatal("expected maintain_tier1_generate")
	}
	if arNil.AdaptStrategy(0.50) != "escalate_to_tier2_review" {
		t.Fatal("expected escalate_to_tier2_review")
	}

	pt := NewPerformanceTracker()
	pt.RecordExecution("z-ai/glm-5.3-flash", "coding", "SUCCESS", 120.5, 0.001)
	pt.RecordExecution("z-ai/glm-5.3-flash", "coding", "FAILURE", 200.0, 0.002)
	pt.RecordExecution("openai/gpt-5.6-sol", "review", "SUCCESS", 350.0, 0.015)

	if len(pt.GetAllRecords()) != 3 {
		t.Fatalf("expected 3 records, got %d", len(pt.GetAllRecords()))
	}
	if len(pt.GetRecordsByOutcome("SUCCESS")) != 2 {
		t.Fatalf("expected 2 SUCCESS records")
	}
	if len(pt.GetRecordsByArea("coding")) != 2 {
		t.Fatalf("expected 2 coding records")
	}
	if len(pt.GetRecordsByModel("z-ai/glm-5.3-flash")) != 2 {
		t.Fatalf("expected 2 glm-5.3-flash records")
	}

	ar := NewAdaptiveModelRouter(pt)
	summary := ar.GetLearningSummary()
	if summary["total_records"] != 3 {
		t.Fatalf("expected 3 learning records, got %+v", summary)
	}

	paNil := NewPerformanceAnalytics(nil)
	emptyPerf := paNil.AnalyzeModelPerformance("unknown-model")
	if emptyPerf["success_rate"] != 1.0 {
		t.Fatalf("expected default success_rate 1.0, got %+v", emptyPerf)
	}

	pa := NewPerformanceAnalytics(pt)
	perf := pa.AnalyzeModelPerformance("z-ai/glm-5.3-flash")
	if perf["invocations"] != 2 || perf["success_rate"] != 0.5 {
		t.Fatalf("unexpected model performance: %+v", perf)
	}
	pattern := pa.AnalyzeTaskPattern("coding")
	if pattern["count"] != 2 {
		t.Fatalf("unexpected task pattern: %+v", pattern)
	}
	if len(pa.RankModelsForTask("coding")) != 3 {
		t.Fatal("expected 3 ranked models")
	}
	if len(pa.GetProviderComparison()) != 4 {
		t.Fatal("expected 4 provider comparisons")
	}
	if len(pa.IdentifyOptimizationOpportunities()) == 0 {
		t.Fatal("expected optimization opportunities")
	}

	qg := NewQualityGate()
	resPass := qg.Validate("valid code", true)
	if resPass["passed"] != true {
		t.Fatal("expected quality gate to pass")
	}
	resFail := qg.Validate("   ", true)
	if resFail["passed"] != false {
		t.Fatal("expected quality gate to fail on empty output")
	}

	ot := NewOrchestratorTelemetry()
	ot.TraceTask("t1", "desc")
	ot.TraceStep("t1", "step1")
	ot.TraceLLMCall("glm-5.3", 512)
	ot.RecordTaskComplete("t1", "DONE")
	if len(ot.traces) != 4 {
		t.Fatalf("expected 4 telemetry traces, got %d", len(ot.traces))
	}

	rl := NewRunLogger()
	rl.StartRun("run-1")
	rl.LogTaskStart("t1")
	rl.LogRoutingDecision("glm-5.3")
	rl.LogLLMCall("glm-5.3", 512)
	rl.LogQualityGate(true)
	if len(rl.events) != 5 {
		t.Fatalf("expected 5 run logger events, got %d", len(rl.events))
	}

	orch := NewOrchestrator()
	out := orch.RunTask("t-main", "build module", "high")
	if out["status"] != "COMPLETED" {
		t.Fatalf("unexpected RunTask output: %+v", out)
	}
}
