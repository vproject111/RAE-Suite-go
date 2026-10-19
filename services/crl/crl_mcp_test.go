package crl_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/mcp"
)

func TestRAECRLAndMCPIntegration(t *testing.T) {
	ctx := context.Background()
	guard := security.NewHardFramesGuard("crl-secret")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	crlSvc := crl.NewService(store, store, guard)
	mcpSvc := mcp.NewService(store, crlSvc)

	if len(mcpSvc.ListTools()) != 31 {
		t.Fatalf("expected 31 total MCP tools (13 RAE-Suite + 7 RAE-CRL + 5 Supervisor + 6 Integrations), got %d", len(mcpSvc.ListTools()))
	}

	// 1. Create Quick Note (24h Grace Period) via MCP
	res, err := mcpSvc.CallTool(ctx, "quick_note", map[string]interface{}{
		"content": "Doping PET-G with 2% graphene will increase tensile strength",
		"project": "doktorat-kubus",
		"author":  "kubus",
	})
	if err != nil {
		t.Fatalf("quick_note failed: %v", err)
	}
	art := res["artifact"].(*models.EpistemicArtifact)
	if !art.IsInGracePeriod(art.CreatedAt) {
		t.Fatal("expected trace to be in 24h Grace Period")
	}

	// 2. Refine to Hypothesis
	refRes, err := mcpSvc.CallTool(ctx, "refine_artifact", map[string]interface{}{
		"artifact_id": art.ID,
		"new_type":    "hypothesis",
		"title":       "Graphene Doping Hypothesis",
	})
	if err != nil {
		t.Fatalf("refine_artifact failed: %v", err)
	}
	refined := refRes["artifact"].(*models.EpistemicArtifact)
	if refined.IsInGracePeriod(refined.UpdatedAt) || refined.Type != models.ArtifactHypothesis {
		t.Fatal("expected refined hypothesis to have ended Grace Period")
	}

	// 3. Fork Artifact & Detect Logical Conflict
	_, err = mcpSvc.CallTool(ctx, "fork_artifact", map[string]interface{}{
		"original_id": art.ID,
		"new_author":  "hermes",
		"reason":      "Graphene 2% may decrease ductility at low temperature",
	})
	if err != nil {
		t.Fatalf("fork_artifact failed: %v", err)
	}
	conflicts, err := crlSvc.DetectLogicalConflicts(ctx, "doktorat-kubus")
	if err != nil || len(conflicts) == 0 {
		t.Fatalf("expected conflict candidate on forked hypothesis, got %v", conflicts)
	}

	// 4. Record Informative Failure
	_, err = mcpSvc.CallTool(ctx, "record_failure", map[string]interface{}{
		"experiment_id": art.ID,
		"reason":        "Brittle fracture at 48.2 MPa without thermal annealing",
	})
	if err != nil {
		t.Fatalf("record_failure failed: %v", err)
	}

	// 5. Seal Evidence Package (requires SBOM reference per Section 13)
	if _, err := crlSvc.SealEvidencePackage(ctx, "doktorat-kubus", "", "NOT_AFFECTED", nil); err == nil {
		t.Fatal("expected sealing without SBOM reference to fail")
	}
	pkg, err := crlSvc.SealEvidencePackage(ctx, "doktorat-kubus", "sbom://cyclonedx/doktorat-v1.json", "NOT_AFFECTED", []string{"bielik-11b"})
	if err != nil || !strings.HasPrefix(pkg.VersionCID, "mh:1220") {
		t.Fatalf("SealEvidencePackage failed: %v", err)
	}

	// 6. Test Stdio JSON-RPC
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n")
	var out bytes.Buffer
	if err := mcpSvc.ServeStdio(ctx, in, &out); err != nil {
		t.Fatalf("ServeStdio failed: %v", err)
	}
	if !strings.Contains(out.String(), "quick_note") || !strings.Contains(out.String(), "rae_memory_store") {
		t.Fatalf("unexpected stdio output: %s", out.String())
	}

	// 7. Test HTTP Lab Desk (:8503)
	mux := http.NewServeMux()
	crlSvc.RegisterRoutes(mux)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from RAE-CRL /health, got %d", w.Code)
	}
}
