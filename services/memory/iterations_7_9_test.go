package memory_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/memory"
	"github.com/vproject111/RAE-Suite-go/services/orchestrator"
	"github.com/vproject111/RAE-Suite-go/services/portal"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func TestEightLayerMemoryAndOutboxAndProceduralAndSycon(t *testing.T) {
	ctx := context.Background()
	guard := security.NewHardFramesGuard("mem-secret")
	mc := raemath.NewMathController(models.MathL3)
	store := memorystore.NewUnifiedStore(mc, guard)
	memSvc := memory.NewService(store, store, guard)

	mux := http.NewServeMux()
	memSvc.RegisterRoutes(mux)

	// 1. Store memory via POST /v2/memories/
	body := bytes.NewBufferString(`{"content":"Step 1: Calibrate sensor\nStep 2: Verify speed","layer":"procedural","project":"ScreenWatcher"}`)
	req := httptest.NewRequest(http.MethodPost, "/v2/memories/", body)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on memory store, got %d: %s", w.Code, w.Body.String())
	}

	// 2. Process GovernanceOutbox (SKIP LOCKED)
	delivered, err := memSvc.ProcessGovernanceOutboxOnce(ctx, 10)
	if err != nil || delivered != 1 {
		t.Fatalf("expected 1 outbox event delivered, got %d (err=%v)", delivered, err)
	}

	// 3. Test ProceduralChunker
	chunks := memory.ProceduralChunk("Step 1: Init\nStep 2: Run\nKrok 3: Stop", 20)
	if len(chunks) < 2 {
		t.Fatalf("expected procedural chunking on Step/Krok boundaries, got %v", chunks)
	}

	// 4. Test Sycon Knowledge Conflict Resolution (POST /v2/knowledge/resolve)
	resBody := bytes.NewBufferString(`{"project":"RAE-Suite","entity_id":"ent-1","claims":["Old claim","Verified claim"],"trust_scores":[0.3,0.95]}`)
	resReq := httptest.NewRequest(http.MethodPost, "/v2/knowledge/resolve", resBody)
	resW := httptest.NewRecorder()
	mux.ServeHTTP(resW, resReq)
	if resW.Code != http.StatusOK || !strings.Contains(resW.Body.String(), "Verified claim") {
		t.Fatalf("expected Verified claim to win Sycon resolution, got %s", resW.Body.String())
	}
}

func TestOrchestratorStage0CRLAndLayerTriadAndHITLAndPortal(t *testing.T) {
	ctx := context.Background()
	guard := security.NewHardFramesGuard("orch-secret")
	mc := raemath.NewMathController(models.MathL3)
	store := memorystore.NewUnifiedStore(mc, guard)
	crlSvc := crl.NewService(store, store, guard)
	qualSvc := quality.NewService()
	orchSvc := orchestrator.NewService(mc, crlSvc, qualSvc, guard)

	// 1. Execute Stage 0 RAE-CRL + 3-Tier Cascade
	res, err := orchSvc.ExecuteCascade(ctx, "Optimize tensor batching", "doktorat-kubus", "package main\nfunc Solve() int { return 42 }\n")
	if err != nil || !res.AuditPassed || res.ResearchID == "" {
		t.Fatalf("expected Stage 0 RAE-CRL ResearchID and passing 3-tier cascade, got %+v (err=%v)", res, err)
	}
	if len(res.Tier1Generate) != 3 || len(res.Tier2Review) != 3 || len(res.Tier3Judge) != 3 {
		t.Fatalf("expected 9 specialized models across 3 tiers, got %+v", res)
	}

	// 2. Test HITL cryptographic HMAC-SHA256 approval
	hitl := orchSvc.CreateHITLRequest("DEPLOY_PRODUCTION_CLUSTER")
	if hitl.Status != "NEEDS_APPROVAL" {
		t.Fatalf("expected NEEDS_APPROVAL, got %s", hitl.Status)
	}
	approved, err := orchSvc.ApproveHITLRequest(hitl.RequestID, "grzegorz")
	if err != nil || approved.Status != "APPROVED" || approved.ApprovalToken == "" {
		t.Fatalf("expected APPROVED with HMAC token, got %+v (err=%v)", approved, err)
	}

	// 3. Test RAE-Portal 14 views (including RaeCrl) & PWA manifest
	portalSvc := portal.NewService()
	pMux := http.NewServeMux()
	portalSvc.RegisterRoutes(pMux)

	vReq := httptest.NewRequest(http.MethodGet, "/v2/portal/views", nil)
	vW := httptest.NewRecorder()
	pMux.ServeHTTP(vW, vReq)
	var vData map[string][]string
	_ = json.Unmarshal(vW.Body.Bytes(), &vData)
	if len(vData["views"]) != 14 {
		t.Fatalf("expected 14 portal views, got %d", len(vData["views"]))
	}

	mReq := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	mW := httptest.NewRecorder()
	pMux.ServeHTTP(mW, mReq)
	if mW.Code != http.StatusOK || !strings.Contains(mW.Body.String(), "RAE Suite Command Center") {
		t.Fatalf("expected valid PWA manifest.json, got %s", mW.Body.String())
	}
}
