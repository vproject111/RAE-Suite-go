package reflection

import (
	"strings"
	"testing"
)

func TestReflectionCoordinatorL1ToL4(t *testing.T) {
	coord := NewCoordinator()
	report := coord.Reflect(ReflectionPayload{
		AnswerDraft:      "Based on the telemetry log, system latency is stable.",
		RetrievedSources: []string{"telemetry_log_2026.json", "metrics_summary.md"},
		ContractStatus:   "ok",
		CoverageRatio:    0.92,
		UncertaintyLevel: 0.12,
		Analysis:         "Grounding verified across telemetry logs",
	})
	if report.Blocked {
		t.Fatalf("expected unblocked reflection report, got blocked")
	}
	if !strings.Contains(report.SynthesizedLesson, "Lesson Learned:") {
		t.Fatalf("expected synthesized lesson, got %q", report.SynthesizedLesson)
	}

	// Test blocking when uncertainty exceeds MaxUncertainty
	blocked := coord.Reflect(ReflectionPayload{
		AnswerDraft:      "Always true except when false in confidential public logs.",
		RetrievedSources: nil,
		ContractStatus:   "failed",
		CoverageRatio:    0.10,
		UncertaintyLevel: 0.90,
	})
	if !blocked.Blocked {
		t.Fatalf("expected blocked reflection report on low coverage / high uncertainty")
	}
}
