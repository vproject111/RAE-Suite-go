package models

import (
	"testing"
	"time"
)

func TestMemoryLayerAndArtifactModels(t *testing.T) {
	if len(AllMemoryLayers()) != 8 {
		t.Fatalf("expected 8 memory layers, got %d", len(AllMemoryLayers()))
	}
	if NormalizeLayer("stm") != LayerSensory || NormalizeLayer("wm") != LayerWorking || NormalizeLayer("ltm") != LayerLTM || NormalizeLayer("em") != LayerEpisodic {
		t.Fatalf("unexpected layer normalization")
	}

	future := time.Now().UTC().Add(12 * time.Hour)
	art := &EpistemicArtifact{
		ID:             GenerateUUIDv7(),
		TenantID:       GenerateULID(),
		Type:           ArtifactDecision,
		Status:         StatusDraft,
		Visibility:     VisibilityPrivate,
		InfoClass:      InfoInternal,
		GracePeriodEnd: &future,
	}
	if !art.IsInGracePeriod(time.Now().UTC()) {
		t.Fatalf("expected artifact to be in grace period")
	}
}
