package sqlstore

import (
	"context"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestPersistentStoreSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	store, err := DriverFactory("persistent", dir)
	if err != nil {
		t.Fatalf("DriverFactory error: %v", err)
	}

	ctx := context.Background()
	mem := &models.MemoryItem{
		ID:         "mem-persist-1",
		Content:    "Persistent Go memory record",
		Layer:      models.LayerSemantic,
		TenantID:   "t1",
		Project:    "rae-go",
		Importance: 0.9,
		Strength:   1.0,
		CreatedAt:  time.Now().UTC(),
	}
	if err := store.StoreMemory(ctx, mem); err != nil {
		t.Fatalf("StoreMemory error: %v", err)
	}

	art := &models.EpistemicArtifact{
		ID:          "art-persist-1",
		Type:        models.ArtifactHypothesis,
		Status:      models.StatusCompleted,
		Title:       "Persistent Hypothesis",
		Description: "Confirmed hypothesis",
		Project:     "rae-go",
		Author:      "agent-1",
		InfoClass:   models.InfoInternal,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := store.SaveArtifact(ctx, art); err != nil {
		t.Fatalf("SaveArtifact error: %v", err)
	}

	reloaded, err := NewPersistentStore(dir)
	if err != nil {
		t.Fatalf("NewPersistentStore reload error: %v", err)
	}
	gotMem, err := reloaded.GetMemory(ctx, "mem-persist-1")
	if err != nil || gotMem.Content != mem.Content {
		t.Fatalf("expected reloaded memory %q, got %v (err=%v)", mem.Content, gotMem, err)
	}
	gotArt, err := reloaded.GetArtifact(ctx, "art-persist-1")
	if err != nil || gotArt.Title != art.Title {
		t.Fatalf("expected reloaded artifact %q, got %v (err=%v)", art.Title, gotArt, err)
	}
}
