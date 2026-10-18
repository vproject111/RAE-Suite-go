package quality

import (
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
