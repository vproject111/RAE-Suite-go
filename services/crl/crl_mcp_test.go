package crl_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestCRLServiceFullCoverage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), nil)
	svc := crl.NewService(store, store, nil)

	// StartSyncEngine with default and short interval
	svc.StartSyncEngine(ctx, 0)
	artExp, err := svc.QuickNote(ctx, "Short grace note that will expire soon for sync engine sweep test exceeding fifty characters easily", "proj-sync", "")
	if err != nil {
		t.Fatalf("QuickNote failed: %v", err)
	}
	past := artExp.CreatedAt.Add(-2 * time.Hour)
	artExp.GracePeriodEnd = &past
	_ = store.SaveArtifact(ctx, artExp)
	svc.StartSyncEngine(ctx, 5*time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	// RefineArtifact with experiment type, description, and not-found error
	if _, err := svc.RefineArtifact(ctx, "missing-id", models.ArtifactExperiment, "T", "D"); err == nil {
		t.Fatal("expected error for missing artifact")
	}
	refinedExp, err := svc.RefineArtifact(ctx, artExp.ID, models.ArtifactExperiment, "Exp Title", "Exp Description")
	if err != nil || refinedExp.Type != models.ArtifactExperiment {
		t.Fatalf("RefineArtifact failed: %v", err)
	}

	// RecordFailure error path and nil MetadataBlob path
	if _, err := svc.RecordFailure(ctx, "missing-id", "reason"); err == nil {
		t.Fatal("expected error for missing experiment")
	}
	artExp.MetadataBlob = nil
	_ = store.SaveArtifact(ctx, artExp)
	if _, err := svc.RecordFailure(ctx, artExp.ID, "thermal runaway"); err != nil {
		t.Fatalf("RecordFailure failed: %v", err)
	}

	// ForkArtifact error path
	if _, err := svc.ForkArtifact(ctx, "missing-id", "author", "reason"); err == nil {
		t.Fatal("expected error for missing original artifact")
	}

	// RequestApproval and ApproveArtifact
	if _, err := svc.RequestApproval(ctx, "missing-id", ""); err == nil {
		t.Fatal("expected error on missing artifact")
	}
	if _, err := svc.RequestApproval(ctx, artExp.ID, ""); err != nil {
		t.Fatalf("RequestApproval failed: %v", err)
	}
	if _, err := svc.ApproveArtifact(ctx, artExp.ID, ""); err == nil {
		t.Fatal("expected error when approver is empty")
	}
	if _, err := svc.ApproveArtifact(ctx, "missing-id", "PI"); err == nil {
		t.Fatal("expected error when artifact missing")
	}
	if _, err := svc.ApproveArtifact(ctx, artExp.ID, "Prof. Nowak"); err != nil {
		t.Fatalf("ApproveArtifact failed: %v", err)
	}

	// DetectLogicalConflicts with semantic contradiction ("increase"/"decrease" and "always"/"never")
	n1, _ := svc.QuickNote(ctx, "H1", "proj-conflict", "a")
	n2, _ := svc.QuickNote(ctx, "H2", "proj-conflict", "b")
	_, _ = svc.RefineArtifact(ctx, n1.ID, models.ArtifactHypothesis, "H1", "This will always increase yield")
	_, _ = svc.RefineArtifact(ctx, n2.ID, models.ArtifactHypothesis, "H2", "This will never decrease yield")
	conflicts, err := svc.DetectLogicalConflicts(ctx, "proj-conflict")
	if err != nil || len(conflicts) == 0 {
		t.Fatalf("expected semantic contradiction conflict, got %v, err=%v", conflicts, err)
	}

	// SealEvidencePackage with in-grace artifact and refined artifact, empty vexStatus
	_, _ = svc.QuickNote(ctx, "In grace trace", "proj-conflict", "a")
	pkg, err := svc.SealEvidencePackage(ctx, "proj-conflict", "sbom://v1", "", []string{"m1"})
	if err != nil || pkg.VEXStatus != "NOT_AFFECTED" {
		t.Fatalf("SealEvidencePackage failed: %+v, err=%v", pkg, err)
	}
	if err := svc.VerifyPackageOffline(pkg); err != nil {
		t.Fatalf("VerifyPackageOffline failed: %v", err)
	}

	// ExportPublication
	pub := svc.ExportPublication(ctx, "proj-conflict", "", "both")
	if pub["artifacts_count"].(int) < 2 {
		t.Fatalf("unexpected ExportPublication: %+v", pub)
	}

	// Full HTTP Routes coverage
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	callHTTP := func(method, path, body string) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	if w := callHTTP(http.MethodGet, "/", ""); w.Code != http.StatusOK {
		t.Fatalf("GET / status=%d", w.Code)
	}
	if w := callHTTP(http.MethodGet, "/not-found-page", ""); w.Code != http.StatusNotFound {
		t.Fatalf("GET /not-found-page status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/traces", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("POST /v2/crl/traces bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/traces", `{"content":"http trace","project":"http-p","author":"tester"}`); w.Code != http.StatusOK {
		t.Fatalf("POST /v2/crl/traces status=%d", w.Code)
	}
	if w := callHTTP(http.MethodGet, "/v2/crl/traces?project=http-p", ""); w.Code != http.StatusOK {
		t.Fatalf("GET /v2/crl/traces status=%d", w.Code)
	}

	traces, _ := svc.ListTraces(ctx, "http-p", "")
	tid := traces[0].ID

	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/refine", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("refine bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/refine", `{"artifact_id":"missing","new_type":"hypothesis"}`); w.Code != http.StatusNotFound {
		t.Fatalf("refine missing status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/refine", `{"artifact_id":"`+tid+`","new_type":"hypothesis","title":"T","description":"D"}`); w.Code != http.StatusOK {
		t.Fatalf("refine ok status=%d", w.Code)
	}

	if w := callHTTP(http.MethodPost, "/v2/crl/experiments/failure", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("failure bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/experiments/failure", `{"experiment_id":"missing","reason":"err"}`); w.Code != http.StatusNotFound {
		t.Fatalf("failure missing status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/experiments/failure", `{"experiment_id":"`+tid+`","reason":"err"}`); w.Code != http.StatusOK {
		t.Fatalf("failure ok status=%d", w.Code)
	}

	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/fork", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("fork bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/fork", `{"original_id":"missing","new_author":"a","reason":"r"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("fork missing status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/fork", `{"original_id":"`+tid+`","new_author":"a","reason":"r"}`); w.Code != http.StatusOK {
		t.Fatalf("fork ok status=%d", w.Code)
	}

	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/request-approval", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("request-approval bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/request-approval", `{"artifact_id":"missing"}`); w.Code != http.StatusNotFound {
		t.Fatalf("request-approval missing status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/request-approval", `{"artifact_id":"`+tid+`","proposer":"p"}`); w.Code != http.StatusOK {
		t.Fatalf("request-approval ok status=%d", w.Code)
	}

	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/approve", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("approve bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/approve", `{"artifact_id":"`+tid+`","approver_name":""}`); w.Code != http.StatusBadRequest {
		t.Fatalf("approve empty approver status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/artifacts/approve", `{"artifact_id":"`+tid+`","approver_name":"PI"}`); w.Code != http.StatusOK {
		t.Fatalf("approve ok status=%d", w.Code)
	}

	if w := callHTTP(http.MethodGet, "/v2/crl/conflicts?project=http-p", ""); w.Code != http.StatusOK {
		t.Fatalf("conflicts status=%d", w.Code)
	}

	if w := callHTTP(http.MethodPost, "/v2/crl/seal", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("seal bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/seal", `{"project":"http-p","sbom_reference":""}`); w.Code != http.StatusBadRequest {
		t.Fatalf("seal missing sbom status=%d", w.Code)
	}
	wSeal := callHTTP(http.MethodPost, "/v2/crl/seal", `{"project":"http-p","sbom_reference":"sbom://1"}`)
	if wSeal.Code != http.StatusOK {
		t.Fatalf("seal ok status=%d", wSeal.Code)
	}

	if w := callHTTP(http.MethodPost, "/v2/crl/verify", "{bad"); w.Code != http.StatusBadRequest {
		t.Fatalf("verify bad JSON status=%d", w.Code)
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/verify", wSeal.Body.String()); w.Code != http.StatusOK {
		t.Fatalf("verify valid package status=%d: %s", w.Code, w.Body.String())
	}
	if w := callHTTP(http.MethodPost, "/v2/crl/verify", `{"package_id":"p1","signature_hmac":"bad"}`); w.Code != http.StatusBadRequest {
		t.Fatalf("verify invalid package status=%d", w.Code)
	}

	if w := callHTTP(http.MethodGet, "/v2/crl/export?project=http-p", ""); w.Code != http.StatusOK {
		t.Fatalf("export status=%d", w.Code)
	}
}
