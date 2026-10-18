package orchestrator

import (
	"context"
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
