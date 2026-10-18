package lab

import "testing"

func TestLabExperimentAndAnalyzeFailures(t *testing.T) {
	svc := NewService(nil)
	svc.RegisterScan(false, "SEC-CMD-INJECTION")
	exp := svc.RunExperiment("hyper_37", 0.95)
	if exp["status"] != "completed" {
		t.Fatalf("unexpected experiment result: %+v", exp)
	}
	analysis := svc.AnalyzeFailures()
	if analysis["total_failures_analyzed"] != 1 {
		t.Fatalf("unexpected failure analysis: %+v", analysis)
	}
}
