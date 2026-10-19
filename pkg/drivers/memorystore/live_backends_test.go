package memorystore

import (
	"context"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestLiveBackendsHydrationAndWriteThrough(t *testing.T) {
	store := NewUnifiedStore(nil, nil)
	hydrated := store.EnableLiveBackends()
	t.Logf("EnableLiveBackends hydrated %d memories from live PostgreSQL", hydrated)

	item := &models.MemoryItem{
		ID:         models.GenerateUUIDv7(),
		Content:    "RAE-Suite-go v5.3.0 live backend write-through verification",
		Layer:      models.LayerSemantic,
		InfoClass:  models.InfoInternal,
		Project:    "v53-test",
		Importance: 0.92,
	}
	if err := store.StoreMemory(context.Background(), item); err != nil {
		t.Fatalf("StoreMemory failed: %v", err)
	}
	got, err := store.GetMemory(context.Background(), item.ID)
	if err != nil || got.Content != item.Content {
		t.Fatalf("GetMemory mismatch: %+v, err=%v", got, err)
	}
	_ = store.DeleteMemory(context.Background(), item.ID)
}
