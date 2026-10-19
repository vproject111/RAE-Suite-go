package agnostic

import (
	"context"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

func TestAgnosticStorageBrokerSwitchMigrateAndExportSQL(t *testing.T) {
	ctx := context.Background()
	broker := NewAgnosticStorageBroker(nil)

	item := &models.MemoryItem{
		Content:    "Database agnostic memory item",
		Layer:      models.LayerSemantic,
		Project:    "agno-test",
		Importance: 0.9,
		NamedVectors: map[string][]float32{
			"dense":  {0.1, 0.2, 0.3},
			"bielik": {0.4, 0.5, 0.6, 0.7},
		},
	}
	if err := broker.StoreMemory(ctx, item); err != nil {
		t.Fatalf("StoreMemory failed: %v", err)
	}
	_ = broker.AddGraphEdge(ctx, item.ID, "node-target", "RELATES_TO")

	// Switch from embedded to postgres with live auto-migration
	sw, err := broker.SwitchBackend(BackendPostgres, true)
	if err != nil {
		t.Fatalf("SwitchBackend failed: %v", err)
	}
	if sw["migrated_records"].(int) != 1 || broker.ActiveBackend() != BackendPostgres {
		t.Fatalf("expected 1 migrated record to postgres, got %+v", sw)
	}

	res, err := broker.QueryMemories(ctx, ports.QueryFilter{Query: "agnostic", TopK: 5})
	if err != nil || len(res) != 1 {
		t.Fatalf("expected 1 memory in postgres backend, got %d (%v)", len(res), err)
	}

	sqlDump, cnt, err := broker.ExportSQL(ctx)
	if err != nil || cnt != 1 || !strings.Contains(sqlDump, "INSERT INTO memories") {
		t.Fatalf("unexpected ExportSQL output: cnt=%d err=%v sql=%s", cnt, err, sqlDump)
	}

	respWire := EncodeRESP("SET", "rae:key", "val")
	if !strings.HasPrefix(respWire, "*3\r\n$3\r\nSET\r\n") {
		t.Fatalf("invalid RESP encoding: %q", respWire)
	}

	backends := broker.ListBackends(ctx)
	if len(backends) != 5 {
		t.Fatalf("expected 5 agnostic backends, got %d", len(backends))
	}
}
