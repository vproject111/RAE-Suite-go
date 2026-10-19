package security

import "testing"

func TestContextProvenanceLineageAndAudit(t *testing.T) {
	svc := NewContextProvenanceService()
	dc := svc.CreateDecisionContext("rae-go", "Which storage backend is active?", []ContextSource{
		{SourceID: "mem-101", SourceType: "semantic", TrustScore: 0.95, Weight: 1.0},
		{SourceID: "mem-102", SourceType: "episodic", TrustScore: 0.85, Weight: 0.8},
	})
	dec := svc.RecordDecision(dc.ContextID, "agent-ceo", "ARCHITECTURE_SELECTION", "Selected postgres via AgnosticStorageBroker", []string{"mem-101"})

	lineage := svc.GetDecisionLineage(dec.DecisionID)
	if lineage["verified"] != true {
		t.Fatalf("expected decision lineage to be verified, got %+v", lineage)
	}

	audit := svc.AuditContextQuality(dc.ContextID)
	if !audit.ISO42001Compliant || audit.CitedSources != 1 || len(audit.UnusedSourceIDs) != 1 {
		t.Fatalf("expected ISO 42001 compliant audit with 1 cited and 1 unused source, got %+v", audit)
	}
}
