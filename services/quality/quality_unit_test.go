package quality

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeGoASTScanDirMutationAndCoverage(t *testing.T) {
	svc := NewService()
	code := `package sample
func Add(a, b int) int {
	if a > 0 && b > 0 {
		return a + b
	}
	return 0
}
`
	comp, funcs, err := svc.AnalyzeGoAST(code)
	if err != nil || comp < 3 || funcs != 1 {
		t.Fatalf("AnalyzeGoAST unexpected: comp=%d, funcs=%d, err=%v", comp, funcs, err)
	}

	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "sample.go"), []byte(code), 0o644)
	scans, err := svc.ScanDirectory(dir, 2)
	if err != nil || len(scans) != 1 || !scans["sample.go"].Passed {
		t.Fatalf("ScanDirectory failed: %+v, err=%v", scans, err)
	}

	mut := svc.RunMutationTesting(code, 3)
	if mut["passed"] != true {
		t.Fatalf("RunMutationTesting failed: %+v", mut)
	}

	cov := svc.AggregateCoverage(map[string]float64{"core": 0.95, "crl": 0.92})
	if cov["passed"] != true {
		t.Fatalf("AggregateCoverage failed: %+v", cov)
	}
}

func TestAll29QualityRoutes(t *testing.T) {
	svc := NewService()
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
		{"GET", "/mcp/sse", ""},
		{"POST", "/mcp/messages", `{"id":1}`},
		{"POST", "/v2/quality/audit", `{"code":"package main"}`},
		{"POST", "/v2/quality/tribunal", `{"code":"package main"}`},
		{"POST", "/v2/quality/mutation", `{"code":"package main","test_count":4}`},
		{"POST", "/v2/quality/mutation-audit", `{"code":"package main","test_count":4}`},
		{"POST", "/v2/quality/coverage", `{"modules":{"core":0.95}}`},
		{"POST", "/v2/quality/risk-coverage", `{}`},
		{"POST", "/v2/quality/risk-coverage/verify", `{}`},
		{"POST", "/v2/quality/gate-decision", `{"code":"package main"}`},
		{"POST", "/v2/quality/dtp-audit", `{}`},
		{"POST", "/v2/quality/calculator-audit", `{}`},
		{"POST", "/v2/quality/defect-metrics", `{}`},
		{"POST", "/v2/quality/golden-set-audit", `{}`},
		{"POST", "/v2/quality/sonarqube-audit", `{"code":"package main"}`},
		{"POST", "/v2/quality/predictive-plan", `{}`},
		{"POST", "/v2/quality/sonar-strict-metrics", `{"code":"package main"}`},
		{"POST", "/v2/quality/unit-test-inventory", `{}`},
		{"GET", "/v2/quality/policy", ""},
		{"POST", "/v2/quality/analyze-diff", `{"diff":"package main"}`},
		{"POST", "/v2/quality/security-scan", `{"code":"package main"}`},
		{"POST", "/v2/quality/generate-tests", `{"target_function":"ComputeScore"}`},
		{"POST", "/v2/quality/verify-fix", `{"code":"package main"}`},
		{"POST", "/v2/quality/check-architecture", `{}`},
		{"POST", "/v2/quality/release-gate", `{"code":"package main"}`},
		{"POST", "/v2/quality/report", `{"code":"package main"}`},
		{"GET", "/v2/quality/manifest", ""},
		{"POST", "/v2/quality/rpc", `{"method":"audit","params":{"code":"package main"}}`},
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
