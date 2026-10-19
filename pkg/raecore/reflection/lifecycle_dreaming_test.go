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
