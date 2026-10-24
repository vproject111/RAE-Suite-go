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

func TestPersistentOutboxFullCoverage(t *testing.T) {
	if len(outbox.RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes")
	}

	ctx := context.Background()
	ob := outbox.NewPersistentOutbox("", 0)
	if ob.MaxAttempts != 3 {
		t.Fatalf("expected default MaxAttempts 3, got %d", ob.MaxAttempts)
	}

	ev1 := &ports.OutboxEvent{EventType: "EV1"}
	ev2 := &ports.OutboxEvent{EventType: "EV2"}
	ev3 := &ports.OutboxEvent{ID: "ev-exceeded", EventType: "EV3", Attempts: 3}
	_ = ob.EnqueueOutbox(ctx, ev1)
	_ = ob.EnqueueOutbox(ctx, ev2)
	_ = ob.EnqueueOutbox(ctx, ev3)

	claimed, err := ob.ClaimPendingOutbox(ctx, 1)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimPendingOutbox limit=1 failed: %v, %v", claimed, err)
	}
	// Next claim processes ev2 and transitions ev3 (Attempts > MaxAttempts) to DEAD_LETTER
	claimed2, _ := ob.ClaimPendingOutbox(ctx, 10)
	if len(claimed2) != 1 {
		t.Fatalf("expected 1 claimed event, got %d", len(claimed2))
	}

	if err := ob.MarkOutboxDelivered(ctx, claimed[0].ID); err != nil {
		t.Fatalf("MarkOutboxDelivered failed: %v", err)
	}
	if err := ob.MarkOutboxDelivered(ctx, "nonexistent"); err == nil {
		t.Fatal("expected error marking nonexistent outbox event delivered")
	}
	if _, err := ob.RequeueOrDeadLetter("nonexistent"); err == nil {
		t.Fatal("expected error requeuing nonexistent outbox event")
	}
}

