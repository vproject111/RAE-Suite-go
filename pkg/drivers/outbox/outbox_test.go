package outbox_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/outbox"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

func TestPersistentOutboxAndDeadLetterQueue(t *testing.T) {
	ctx := context.Background()
	wal := filepath.Join(t.TempDir(), "outbox.wal.json")
	ob := outbox.NewPersistentOutbox(wal, 2)

	ev := &ports.OutboxEvent{
		ID:        "ev-1",
		EventType: "MESH_DELTA",
		Payload:   map[string]interface{}{"id": "m1"},
	}
	if err := ob.EnqueueOutbox(ctx, ev); err != nil {
		t.Fatalf("EnqueueOutbox failed: %v", err)
	}

	// Attempt 1 -> fail & requeue
	claimed, _ := ob.ClaimPendingOutbox(ctx, 5)
	if len(claimed) != 1 {
		t.Fatalf("expected 1 claimed event, got %d", len(claimed))
	}
	st, _ := ob.RequeueOrDeadLetter("ev-1")
	if st != "PENDING" {
		t.Fatalf("expected PENDING after 1st failure, got %s", st)
	}

	// Attempt 2 -> fail & dead-letter
	_, _ = ob.ClaimPendingOutbox(ctx, 5)
	st2, _ := ob.RequeueOrDeadLetter("ev-1")
	if st2 != "DEAD_LETTER" {
		t.Fatalf("expected DEAD_LETTER after reaching MaxAttempts=2, got %s", st2)
	}

	// Reload from disk WAL
	ob2 := outbox.NewPersistentOutbox(wal, 2)
	claimedAfterReload, _ := ob2.ClaimPendingOutbox(ctx, 5)
	if len(claimedAfterReload) != 0 {
		t.Fatalf("expected 0 pending events after DEAD_LETTER reload, got %d", len(claimedAfterReload))
	}
}
