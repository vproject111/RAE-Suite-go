package reflection

import (
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestConsolidationFSMDreamingAndCommunitySummarizer(t *testing.T) {
	now := time.Now().UTC()
	fsm := NewConsolidationFSM()
	items := []models.MemoryItem{
		{ID: "m1", Content: "PostgreSQL RRF hybrid search optimization", Layer: models.LayerEpisodic, Importance: 0.90, Strength: 0.95, UsageCount: 4, CreatedAt: now.Add(-2 * time.Hour)},
		{ID: "m2", Content: "SQLite FTS5 hybrid search deterministic ranking", Layer: models.LayerSemantic, Importance: 0.85, Strength: 0.90, UsageCount: 3, CreatedAt: now.Add(-4 * time.Hour)},
		{ID: "m3", Content: "ephemeral debug scratch", Layer: models.LayerWorking, Importance: 0.10, Strength: 0.10, UsageCount: 0, CreatedAt: now.Add(-240 * time.Hour)},
	}

	records, counts := fsm.EvaluateBatch(items, now)
	if len(records) != 3 || counts[StageConsolidating] == 0 {
		t.Fatalf("unexpected FSM evaluation: records=%+v counts=%+v", records, counts)
	}

	insights := RunDreamingCycle(items, 0.5)
	if len(insights) == 0 {
		t.Fatalf("expected at least 1 DreamInsight from dreaming cycle")
	}

	comms := SummarizeCommunities([][]string{{"m1", "m2"}}, items)
	if len(comms) != 1 || comms[0].SummaryText == "" {
		t.Fatalf("unexpected community summary: %+v", comms)
	}
}

func TestReflectionFullLifecycleSessionAndCoordinatorBranches(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	now := time.Now().UTC()
	fsm := NewConsolidationFSM()

	// Exercise all 7 FSM stages: Archived, Decaying, Active, Consolidated (LTM & Semantic)
	batch := []models.MemoryItem{
		{ID: "archived", Layer: models.LayerEpisodic, Importance: 0.35, Strength: 0.20, UsageCount: 1, CreatedAt: now.Add(-100 * time.Hour)},
		{ID: "decaying", Layer: models.LayerEpisodic, Importance: 0.50, Strength: 0.35, UsageCount: 0, CreatedAt: now.Add(-50 * time.Hour)},
		{ID: "active", Layer: models.LayerEpisodic, Importance: 0.50, Strength: 0.90, UsageCount: 1, CreatedAt: now},
		{ID: "high_ltm", Layer: models.LayerEpisodic, Importance: 0.92, Strength: 0, UsageCount: 5, CreatedAt: now},
		{ID: "high_sem", Layer: models.LayerEpisodic, Importance: 0.80, Strength: 0.90, UsageCount: 4, CreatedAt: now},
	}
	_, _ = fsm.EvaluateBatch(batch, now)
	// Evaluate second time so StageConsolidating transitions to StageConsolidated
	_, counts := fsm.EvaluateBatch(batch, now)
	if counts[StageConsolidated] < 2 {
		t.Fatalf("expected at least 2 StageConsolidated items on second transition, got %+v", counts)
	}

	// Dreaming cycle with single candidate (fallback hypothesis insight) and long text truncation
	longContent := "a b c " + string(make([]byte, 80))
	single := RunDreamingCycle([]models.MemoryItem{
		{ID: "s1", Content: longContent, Importance: 0.9},
	}, 0)
	if len(single) != 1 || single[0].InsightType != "hypothesis" {
		t.Fatalf("expected fallback hypothesis insight for single item, got %+v", single)
	}

	// SummarizeCommunities with tags and unknown nodeID
	comms := SummarizeCommunities([][]string{{"m1", "unknown-node"}}, []models.MemoryItem{
		{ID: "m1", Content: "tagged memory content", Tags: []string{"tag1", "tag2"}},
	})
	if len(comms) != 1 || len(comms[0].KeyThemes) < 2 {
		t.Fatalf("unexpected community summary with tags: %+v", comms)
	}

	// SummarizeSession matching session and fallback when session not found
	sum1 := SummarizeSession("sess-1", "proj-1", []models.MemoryItem{
		{ID: "1", SessionID: "sess-1", Content: "event 1", Tags: []string{"auth"}},
		{ID: "2", SessionID: "sess-1", Content: "event 2", Tags: []string{"auth", "db"}},
	})
	if sum1["events_processed"] != 2 {
		t.Fatalf("expected 2 events processed, got %+v", sum1)
	}
	sumFallback := SummarizeSession("non-existent", "proj-1", []models.MemoryItem{
		{ID: "1", SessionID: "sess-1", Content: "event 1"},
	})
	if sumFallback["events_processed"] != 1 {
		t.Fatalf("expected fallback to all memories when session not matched, got %+v", sumFallback)
	}

	// Coordinator extra branches: log missed source, >20 sources (decrease_top_k), short draft (increase_top_k)
	coord := NewCoordinator()
	manySources := make([]string, 22)
	for i := range manySources {
		manySources[i] = "source_doc"
	}
	repMany := coord.Reflect(ReflectionPayload{
		AnswerDraft:      "Check the log for details on the service.",
		RetrievedSources: manySources,
	})
	if repMany.L2Structural["optimization_suggestion"] != "decrease_top_k" {
		t.Fatalf("expected decrease_top_k for >20 sources, got %+v", repMany.L2Structural)
	}
	repShort := coord.Reflect(ReflectionPayload{
		AnswerDraft:      "Short.",
		RetrievedSources: []string{"one"},
	})
	if repShort.L2Structural["optimization_suggestion"] != "increase_top_k" {
		t.Fatalf("expected increase_top_k for short draft & <2 sources, got %+v", repShort.L2Structural)
	}
}

