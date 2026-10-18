package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

// PersistentOutbox implements ports.OutboxQueuePort with WAL disk persistence
// and Dead-Letter Queue (DEAD_LETTER) transition after MaxAttempts.
type PersistentOutbox struct {
	mu          sync.Mutex
	filePath    string
	MaxAttempts int
	events      []ports.OutboxEvent
}

// NewPersistentOutbox creates or loads a persistent transactional outbox.
func NewPersistentOutbox(filePath string, maxAttempts int) *PersistentOutbox {
	if maxAttempts <= 0 {
		maxAttempts = 3
	}
	ob := &PersistentOutbox{
		filePath:    filePath,
		MaxAttempts: maxAttempts,
	}
	if filePath != "" {
		if b, err := os.ReadFile(filePath); err == nil {
			_ = json.Unmarshal(b, &ob.events)
		}
	}
	return ob
}

// EnqueueOutbox appends a PENDING event and flushes to WAL if configured.
func (ob *PersistentOutbox) EnqueueOutbox(_ context.Context, ev *ports.OutboxEvent) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	if ev.ID == "" {
		ev.ID = models.GenerateUUIDv7()
	}
	if ev.Status == "" {
		ev.Status = "PENDING"
	}
	ob.events = append(ob.events, *ev)
	return ob.flushLocked()
}

// ClaimPendingOutbox claims up to limit PENDING events using FOR UPDATE SKIP LOCKED semantics.
func (ob *PersistentOutbox) ClaimPendingOutbox(_ context.Context, limit int) ([]ports.OutboxEvent, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	var claimed []ports.OutboxEvent
	for i := range ob.events {
		if ob.events[i].Status == "PENDING" {
			ob.events[i].Attempts++
			if ob.events[i].Attempts > ob.MaxAttempts {
				ob.events[i].Status = "DEAD_LETTER"
				continue
			}
			ob.events[i].Status = "PROCESSING"
			claimed = append(claimed, ob.events[i])
			if limit > 0 && len(claimed) >= limit {
				break
			}
		}
	}
	_ = ob.flushLocked()
	return claimed, nil
}

// MarkOutboxDelivered marks an event as DELIVERED.
func (ob *PersistentOutbox) MarkOutboxDelivered(_ context.Context, id string) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	for i := range ob.events {
		if ob.events[i].ID == id {
			ob.events[i].Status = "DELIVERED"
			return ob.flushLocked()
		}
	}
	return errors.New("outbox event not found")
}

// RequeueOrDeadLetter returns a failed PROCESSING event to PENDING or promotes it to DEAD_LETTER.
func (ob *PersistentOutbox) RequeueOrDeadLetter(id string) (string, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	for i := range ob.events {
		if ob.events[i].ID == id {
			if ob.events[i].Attempts >= ob.MaxAttempts {
				ob.events[i].Status = "DEAD_LETTER"
			} else {
				ob.events[i].Status = "PENDING"
			}
			_ = ob.flushLocked()
			return ob.events[i].Status, nil
		}
	}
	return "", errors.New("outbox event not found")
}

func (ob *PersistentOutbox) flushLocked() error {
	if ob.filePath == "" {
		return nil
	}
	b, err := json.Marshal(ob.events)
	if err != nil {
		return err
	}
	return os.WriteFile(ob.filePath, b, 0o600)
}
