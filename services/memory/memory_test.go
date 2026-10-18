package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestMemoryServiceLifecycleAndHTTPRoutes(t *testing.T) {
	store := memorystore.NewUnifiedStore(nil, nil)
	svc := NewService(store, store, nil)
	ctx := context.Background()

	// Store a Working memory and consolidate -> Episodic -> Semantic -> LTM
	wm := &models.MemoryItem{
		ID:         "wm-1",
		Content:    "Observed high cache hit ratio in Go service",
		Layer:      models.LayerWorking,
		InfoClass:  models.InfoInternal,
		Project:    "p1",
		Importance: 0.90,
	}
	if err := store.StoreMemory(ctx, wm); err != nil {
		t.Fatalf("StoreMemory: %v", err)
	}
	c, err := svc.ConsolidateFromWorking(ctx, "p1")
	if err != nil || c != 1 {
		t.Fatalf("expected 1 consolidated memory, got %d (err=%v)", c, err)
	}
	u, err := svc.UpgradeToSemantic(ctx, "p1", 1)
	if err != nil || u != 1 {
		t.Fatalf("expected 1 upgraded memory, got %d (err=%v)", u, err)
	}
	p, err := svc.PromoteToLTM(ctx, "p1", 0.85)
	if err != nil || p != 1 {
		t.Fatalf("expected 1 LTM promoted memory, got %d (err=%v)", p, err)
	}

	// Graph community detection & resonance
	_ = store.AddGraphEdge(ctx, "n1", "n2", "RELATES")
	comms, err := svc.DetectCommunities(ctx)
	if err != nil || len(comms) != 1 {
		t.Fatalf("unexpected communities: %+v", comms)
	}
	res, err := svc.PropagateResonance(ctx, map[string]float64{"n1": 1.0}, 1)
	if err != nil || res["n2"] <= 0 {
		t.Fatalf("unexpected resonance: %+v", res)
	}

	// HTTP routes test
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	reqStats := httptest.NewRequest(http.MethodGet, "/v2/memories/stats?project=p1", nil)
	recStats := httptest.NewRecorder()
	mux.ServeHTTP(recStats, reqStats)
	if recStats.Code != http.StatusOK {
		t.Fatalf("expected 200 on /v2/memories/stats, got %d", recStats.Code)
	}

	// Mesh CMT HTTP receive
	tok, _ := svc.Mesh().GenerateCapabilityToken("node1-lumina", "mesh:write", 5*time.Minute)
	body, _ := json.Marshal(map[string]interface{}{
		"token": tok,
		"memories": []models.MemoryItem{
			{ID: "m-rx-1", Content: "Mesh synced item", Layer: models.LayerSemantic, InfoClass: models.InfoInternal},
		},
	})
	reqMesh := httptest.NewRequest(http.MethodPost, "/v2/mesh/receive", bytes.NewReader(body))
	recMesh := httptest.NewRecorder()
	mux.ServeHTTP(recMesh, reqMesh)
	if recMesh.Code != http.StatusOK {
		t.Fatalf("expected 200 on /v2/mesh/receive, got %d", recMesh.Code)
	}

	// DELETE /v2/memories/wm-1
	reqDel := httptest.NewRequest(http.MethodDelete, "/v2/memories/wm-1", nil)
	recDel := httptest.NewRecorder()
	mux.ServeHTTP(recDel, reqDel)
	if recDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on DELETE /v2/memories/wm-1, got %d", recDel.Code)
	}
}
