package orchestrator

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func TestAutonomyKernelStateMachineAndHITL(t *testing.T) {
	store := memorystore.NewUnifiedStore(nil, nil)
	crlSvc := crl.NewService(store, store, nil)
	qualSvc := quality.NewService()
	svc := NewService(nil, crlSvc, qualSvc, nil)

	res, err := svc.ExecuteCascade(context.Background(), "Optimize HNSW vector index", "rae-go", "package main\nfunc main() {}\n")
	if err != nil || res.ResearchID == "" || len(res.StateHistory) != 6 || res.StateHistory[5] != StateCompleted {
		t.Fatalf("unexpected ExecuteCascade result: %+v, err=%v", res, err)
	}

	hitl := svc.CreateHITLRequest("DEPLOY_PRODUCTION")
	if hitl.Status != string(StateNeedsApproval) {
		t.Fatalf("expected NEEDS_APPROVAL")
	}
	approved, err := svc.ApproveHITLRequest(hitl.RequestID, "admin")
	if err != nil || approved.Status != string(StateApproved) || approved.ApprovalToken == "" {
		t.Fatalf("ApproveHITLRequest failed: %+v, err=%v", approved, err)
	}
}

func TestAll24OrchestratorRoutes(t *testing.T) {
	store := memorystore.NewUnifiedStore(nil, nil)
	crlSvc := crl.NewService(store, store, nil)
	qualSvc := quality.NewService()
	svc := NewService(nil, crlSvc, qualSvc, nil)

	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{"GET", "/health", ""},
		{"POST", "/v2/factory/reconcile", `{}`},
		{"POST", "/v2/orchestrator/reconcile", `{}`},
		{"POST", "/v2/orchestrator/cascade", `{"task":"test","project":"rae-go","code_draft":"package main"}`},
		{"POST", "/v2/kernel/execute", `{"task":"test","project":"rae-go","code_draft":"package main"}`},
		{"POST", "/v2/triad/run", `{"task":"test triad"}`},
		{"POST", "/v2/departments/register", `{"name":"RAE-Custom","port":8099}`},
		{"POST", "/v2/fabric/route", `{}`},
		{"POST", "/v2/audit/evaluate", `{"code":"package main"}`},
		{"POST", "/v2/quality/rpc", `{"method":"audit","params":{"code":"package main"}}`},
		{"GET", "/v2/quality/policy", ""},
		{"POST", "/v2/quality/analyze-diff", `{"diff":"package main"}`},
		{"POST", "/v2/quality/security-scan", `{"code":"package main"}`},
		{"POST", "/v2/quality/generate-tests", `{"target_function":"Run"}`},
		{"POST", "/v2/quality/verify-fix", `{"code":"package main"}`},
		{"POST", "/v2/quality/check-architecture", `{}`},
		{"POST", "/v2/quality/release-gate", `{"code":"package main"}`},
		{"POST", "/v2/quality/report", `{"code":"package main"}`},
		{"GET", "/v2/portal/overview", ""},
		{"GET", "/v2/portal/hitl/requests", ""},
		{"POST", "/v2/portal/hitl/request", `{"operation":"DEPLOY"}`},
		{"GET", "/v2/portal/hitl/iso-report", ""},
		{"POST", "/v2/knowledge/resolve", `{"query":"HNSW","project":"rae-go"}`},
	}

	for _, ep := range endpoints {
		req, _ := http.NewRequest(ep.method, srv.URL+ep.path, bytes.NewBufferString(ep.body))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s error: %v", ep.method, ep.path, err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s %s status=%d", ep.method, ep.path, resp.StatusCode)
		}
	}
}
