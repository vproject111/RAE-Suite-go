package agnostic

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestAgnosticFullCoverageWithMockRedisAndQdrant(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes")
	}

	ctx := context.Background()

	// Mock Redis RESP TCP server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen error: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				buf := make([]byte, 256)
				_, _ = conn.Read(buf)
				_, _ = conn.Write([]byte("+OK\r\n"))
			}(c)
		}
	}()

	rc := NewRedisRESPClient(ln.Addr().String())
	reply, err := rc.Exec("PING")
	if err != nil || reply != "+OK" {
		t.Fatalf("RedisRESPClient.Exec failed: %q, %v", reply, err)
	}

	// Mock Qdrant HTTP REST server
	qSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer qSrv.Close()

	qa := NewQdrantRESTAdapter(qSrv.URL, "")
	if qa.EnsureVectorSpace("", 10) || qa.EnsureVectorSpace("dense", 0) {
		t.Fatal("expected false for invalid EnsureVectorSpace args")
	}
	if qa.EnsureVectorSpace("dense", 384) {
		t.Fatal("expected false when dimension unchanged")
	}
	if !qa.EnsureVectorSpace("custom", 512) {
		t.Fatal("expected true when new vector space added")
	}
	ok := qa.UpsertPoint(ctx, &models.MemoryItem{
		ID:           "q-1",
		Content:      "qdrant item",
		Embedding:    []float32{1, 2, 3, 4},
		NamedVectors: map[string][]float32{"custom": {1, 2}},
	})
	if !ok {
		t.Fatal("expected UpsertPoint against mock Qdrant server to succeed")
	}

	// Broker env initialization & Hybrid mode
	_ = os.Setenv("RAE_STORAGE_BACKEND", "hybrid")
	_ = os.Setenv("QDRANT_URL", qSrv.URL)
	_ = os.Setenv("REDIS_ADDR", ln.Addr().String())
	broker := NewAgnosticStorageBroker(nil)
	_ = os.Unsetenv("RAE_STORAGE_BACKEND")
	_ = os.Unsetenv("QDRANT_URL")
	_ = os.Unsetenv("REDIS_ADDR")

	if broker.ActiveBackend() != BackendHybrid {
		t.Fatalf("expected BackendHybrid, got %s", broker.ActiveBackend())
	}

	m := &models.MemoryItem{
		ID:        "hyb-1",
		Content:   "hybrid memory",
		Layer:     models.LayerWorking,
		Project:   "p-hyb",
		Embedding: []float32{0.5, 0.5},
	}
	if err := broker.StoreMemory(ctx, m); err != nil {
		t.Fatalf("StoreMemory in hybrid mode failed: %v", err)
	}
	if _, err := broker.GetMemory(ctx, "hyb-1"); err != nil {
		t.Fatalf("GetMemory failed: %v", err)
	}
	_, _ = broker.GetMemoryStats(ctx, "p-hyb")
	_, _ = broker.DeleteExpiredSensory(ctx)
	_ = broker.AddGraphEdge(ctx, "hyb-1", "hyb-2", "LINKS")
	_, _ = broker.GetGraphNeighbors(ctx, "hyb-1")
	_, _ = broker.GetGraphAdjacency(ctx)
	_ = broker.DeleteMemory(ctx, "hyb-1")

	// Error and edge branches in SwitchBackend / Migrate
	if _, err := broker.SwitchBackend("invalid_backend", true); err == nil {
		t.Fatal("expected error for unsupported backend")
	}
	// Migrate from Embedded to Hybrid (same underlying store pointer)
	if _, err := broker.Migrate(ctx, BackendEmbedded, BackendHybrid); err != nil {
		t.Fatalf("Migrate same store failed: %v", err)
	}
	if _, err := broker.Migrate(ctx, "invalid", BackendPostgres); err == nil {
		t.Fatal("expected error for invalid source backend")
	}
}

