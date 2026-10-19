package ops

import "testing"

func TestGitOpsPhaseAnalyzerAndGlobalSearch(t *testing.T) {
	gitops := NewGitOpsDaemon()
	res := gitops.ExecuteGitOpsWorkflow("f3-upgrade", "Add GitOps daemon", "Parity with Python core/gitops_daemon.py", "claude-sonnet-5")
	if res.BranchName != "feature/f3-upgrade" || len(res.CommitSignature) != 64 {
		t.Fatalf("expected valid GitOps branch and SHA-256 commit signature, got %+v", res)
	}

	phases := AnalyzeProjectPhases("RAE-Suite-go", nil)
	if phases.TotalTasks != 3 || len(phases.Phases) != 3 {
		t.Fatalf("expected 3 SDLC phases analyzed, got %+v", phases)
	}

	gs := ExecuteGlobalSearch("AgnosticStorageBroker", "RAE-Suite-go")
	if gs.TotalHits != 4 || len(gs.Departments) != 4 {
		t.Fatalf("expected 4 federated global search hits across 4 departments, got %+v", gs)
	}
}
