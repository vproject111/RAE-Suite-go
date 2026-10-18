package quality_test

import (
	"context"
	"testing"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/services/hive"
	"github.com/vproject111/RAE-Suite-go/services/lab"
	"github.com/vproject111/RAE-Suite-go/services/phoenix"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func TestQualityTribunalDisjointPoolsAndHardGates(t *testing.T) {
	q := quality.NewService()
	rev := q.ReviewerModels()
	jud := q.JudgeModels()
	for _, r := range rev {
		for _, j := range jud {
			if r == j {
				t.Fatalf("Tribunal model pool collision between Reviewer and Judge: %s", r)
			}
		}
	}
	// UI Freeze violation check
	rep := q.AuditCode("body { color: red; }", []string{"styles/main.css"}, "Fix backend bug")
	if rep.Passed || rep.UIFreezeOK {
		t.Fatal("expected unauthorized CSS change to fail UI Freeze gate")
	}
}

func TestHiveParallelExecutionAndWatchdog(t *testing.T) {
	h := hive.NewService()
	res := h.RunParallelCommands(context.Background(), []string{
		"echo branch1",
		"echo branch2",
		"echo branch3",
	})
	if len(res) != 3 || res[0].ExitCode != 0 {
		t.Fatalf("expected 3 parallel commands to succeed, got %v", res)
	}
	blocked, sim, status := h.EvaluateEchoLoop([]string{
		"TypeError: invalid argument in handler",
		"TypeError: invalid argument in handler",
	})
	if !blocked || sim < 0.95 || status != "BLOCKED_LOOP" {
		t.Fatalf("expected echo loop to trigger BLOCKED_LOOP, got blocked=%v sim=%f status=%s", blocked, sim, status)
	}
}

func TestPhoenixThreeAttemptBoundAndBehavioralParity(t *testing.T) {
	p := phoenix.NewService()
	if p.MaxAttempts != 3 {
		t.Fatalf("expected PHOENIX_MAX_ATTEMPTS=3, got %d", p.MaxAttempts)
	}
	// 4 failing candidates -> should stop at 3 and roll back
	failRes := p.ExecuteClosedLoopRepair("expected_output", []string{"bad1", "bad2", "bad3", "expected_output"})
	if !failRes.RolledBack || failRes.AttemptsUsed != 3 {
		t.Fatalf("expected Phoenix to stop after 3 attempts and roll back, got %+v", failRes)
	}
	// Succeed on 2nd attempt
	okRes := p.ExecuteClosedLoopRepair("expected_output", []string{"bad1", "expected_output"})
	if okRes.RolledBack || !okRes.Parity.ParityVerified || okRes.AttemptsUsed != 2 {
		t.Fatalf("expected Phoenix to succeed on attempt 2, got %+v", okRes)
	}
}

func TestLabMABWeightsAndCandidateGuardrails(t *testing.T) {
	l := lab.NewService(raemath.NewMathController(models.MathL3))
	l.RegisterScan(false, "HARDCODED_PATH_DETECTED")
	cands := l.GetCandidates("CANDIDATE")
	if len(cands) != 1 || cands[0].Status != "CANDIDATE" {
		t.Fatalf("expected 1 CANDIDATE guardrail, got %v", cands)
	}
	w := l.GetWeights()
	if w.Alpha <= 0 || w.Beta <= 0 || w.Gamma <= 0 {
		t.Fatalf("invalid MAB weights: %+v", w)
	}
}
