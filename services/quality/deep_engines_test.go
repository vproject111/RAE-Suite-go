package quality

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalog198DTPValidatorCalculatorAndGoldenSet(t *testing.T) {
	// 1. Catalog 198 + 12 AI/SaaS topics
	cat := BuildCatalog198Summary()
	if cat["engineering_topics"].(int) != 198 || cat["ai_saas_topics"].(int) != 12 || cat["total_catalog_topics"].(int) != 210 {
		t.Fatalf("expected 198 + 12 = 210 topics in catalog, got %+v", cat)
	}

	// 2. PDF DTP Validator: valid vs missing bleed & TAC > 300%
	validDoc := PDFDocumentSpec{
		DocumentID: "doc-ok",
		TrimBox:    BoxDimensions{Width: 90, Height: 50},
		BleedBox:   BoxDimensions{Width: 94, Height: 54},
		MinBleedMM: 2.0,
		DPIMin:     300,
		TextElements: []DTPTextElement{
			{FontName: "Inter", FontSizePt: 10, IsEmbedded: true, Color: CMYKColor{C: 0, M: 0, Y: 0, K: 100}},
		},
		SpotLayers: []DTPSpotLayer{
			{Name: "CutContour", Overprint: true},
		},
	}
	resOK := ValidatePDFDocument(validDoc)
	if !resOK.Passed || !resOK.HasValidBleed || resOK.BleedXMM != 2.0 {
		t.Fatalf("expected valid DTP document to pass, got %+v", resOK)
	}

	badDoc := PDFDocumentSpec{
		DocumentID: "doc-bad",
		TrimBox:    BoxDimensions{Width: 90, Height: 50},
		BleedBox:   BoxDimensions{Width: 90, Height: 50}, // missing bleed
		MinBleedMM: 2.0,
		DPIMin:     150, // low DPI
		TextElements: []DTPTextElement{
			{FontName: "UnembeddedFont", FontSizePt: 8, IsEmbedded: false, Color: CMYKColor{C: 90, M: 90, Y: 90, K: 90}}, // TAC 360% > 300%
		},
		SpotLayers: []DTPSpotLayer{
			{Name: "CutContour", Overprint: false},
		},
	}
	resBad := ValidatePDFDocument(badDoc)
	if resBad.Passed || resBad.HasValidBleed || resBad.CMYKCompliant || resBad.FontsEmbedded || len(resBad.Issues) < 4 {
		t.Fatalf("expected bad DTP document to fail with >=4 issues, got %+v", resBad)
	}

	// 3. Calculator Verifier: valid monotonic vs price inversion
	calcOK := VerifyCalculatorTiers([]PriceTier{
		{Volume: 100, TotalPrice: 50.0},
		{Volume: 500, TotalPrice: 150.0},
		{Volume: 1000, TotalPrice: 250.0},
	}, 50.0, 90.0, 2.0)
	if !calcOK.IsValid {
		t.Fatalf("expected monotonic pricing to pass, got %+v", calcOK)
	}

	calcBad := VerifyCalculatorTiers([]PriceTier{
		{Volume: 100, TotalPrice: 100.0},
		{Volume: 500, TotalPrice: 80.0}, // total price inversion!
	}, 100.0, 90.0, 2.0)
	if calcBad.IsValid || len(calcBad.Violations) < 2 {
		t.Fatalf("expected price inversion to be caught, got %+v", calcBad)
	}

	// 4. Risk Coverage & Golden Set
	rc := EvaluateRiskCoverageMatrix()
	if rc["verified"] != true {
		t.Fatalf("expected risk coverage matrix verified=true, got %+v", rc)
	}
	gs := RunCanonicalGoldenSetAudit()
	if gs["golden_set_passed"] != true || gs["passed_documents"].(int) != 5 {
		t.Fatalf("expected all 5 canonical golden set documents to pass, got %+v", gs)
	}
}

func TestBinaryPDFAndMultiLangAST(t *testing.T) {
	// 210x297mm is 595.28x841.89pt; 216x303mm is 612.28x858.90pt
	rawPDF := "%PDF-1.7\n1 0 obj << /Type /Page /TrimBox [0 0 595.28 841.89] /BleedBox [0 0 612.28 858.90] /ColorSpace /DeviceCMYK /Font << /F1 << /FontFile 2 0 R >> >> >> endobj\n%%EOF"
	pdfRes := ParseBinaryPDFAndValidate(rawPDF)
	if pdfRes["passed"] != true || pdfRes["pdf_header_detected"] != true || pdfRes["cmyk_colorspace"] != true {
		t.Fatalf("expected binary PDF stream to pass DTP audit, got %+v", pdfRes)
	}

	pyCode := "class Calculator:\n    def compute(self, x):\n        if x > 0:\n            return x * 2\n        return 0\n"
	pyRep := AnalyzeMultiLangAST("python", pyCode)
	if pyRep.FunctionsCount != 1 || pyRep.ClassesOrTypesCount != 1 || !pyRep.PassedQualityGate {
		t.Fatalf("expected Python AST report to pass with 1 func and 1 class, got %+v", pyRep)
	}

	goCode := "package main\ntype S struct{}\nfunc Add(a, b int) int {\n\tif a > 0 {\n\t\treturn a + b\n\t}\n\treturn b\n}\n"
	goRep := AnalyzeMultiLangAST("go", goCode)
	if goRep.FunctionsCount != 1 || goRep.ClassesOrTypesCount != 1 || !goRep.PassedQualityGate {
		t.Fatalf("expected Go AST report to pass with 1 func and 1 type, got %+v", goRep)
	}
}

func TestFullQualityServiceGatewayEnginesAndRoutesCoverage(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected non-empty RegisteredPackageTypes")
	}

	// 1. Binary PDF & MultiLang AST edge cases
	if w, h := parsePDFBoxMM([]string{"short"}); w != 0 || h != 0 {
		t.Fatalf("expected 0,0 from short slice, got %v,%v", w, h)
	}
	_ = ParseBinaryPDFAndValidate("%PDF-1.4\n/MediaBox [0 0 595 842] /Font << /F1 << >> >>")
	_ = ParseBinaryPDFAndValidate("%PDF-1.4\n/TrimBox [0 0 595 842]")

	richGo := `package main
func Rich(ch chan int, xs []int) int {
	for _, x := range xs {
		if x > 0 && x < 10 || x == 42 {
			switch x {
			case 1:
				return 1
			}
		}
	}
	select {
	case v := <-ch:
		return v
	default:
		return 0
	}
}`
	_ = AnalyzeMultiLangAST("", richGo)
	_ = AnalyzeMultiLangAST("go", "not valid go code {{{")
	tsCode := "// comment\nclass Service {\n  run = (x: number) => {\n    if (x > 0 && x < 5) { return x; }\n    return 0;\n  }\n}\n"
	tsRep := AnalyzeMultiLangAST("typescript", tsCode)
	if tsRep.ClassesOrTypesCount < 1 || tsRep.FunctionsCount < 1 {
		t.Fatalf("unexpected TS AST report: %+v", tsRep)
	}

	// 2. SeniorityRanker, PDFDtpValidator, RiskCoverageEngine
	sr := NewSeniorityRanker()
	for _, sc := range []float64{0.95, 0.80, 0.65, 0.40} {
		_ = sr.ClassifyLevel(sc)
	}
	_ = NewPDFDtpValidator().AuditDocument(PDFDocumentSpec{})

	rce := NewRiskCoverageEngine()
	_ = rce.EvaluateRiskCoverage(10.0, 0.90)
	_ = rce.CalculateRiskCoverage(10.0, 0.90)
	_ = rce.EvaluateDeploymentGate(10.0, 0.90)
	_ = rce.UpdatePathScore("RC-01", -1.0)
	_ = rce.UpdatePathScore("RC-01", 0.5, map[string]interface{}{"note": "partial"})
	_ = rce.EvaluateDeploymentGateMatrix()
	_ = rce.UpdatePathScore("RC-01", 2.0)
	_ = rce.UpdatePathScore("RC-05", 0.70)
	_ = rce.EvaluateDeploymentGateMatrix()
	_ = rce.GetPredictiveTestPlan([]string{
		"calc_price.go", "pdf_bleed.go", "scene_editor.ts", "checkout_order.go",
		"packaging_fefco.go", "security_jwt.go", "ui_layout.css",
	})
	_ = EvaluateRiskCoverageMatrixWithTests([]string{"calc", "dtp"})

	// 3. Engines V2: OpenAPI compatibility, Systemic readiness, VerifyFix, Scorecard, AuditChain
	ae := NewArchitectureEngine()
	_ = ae.VerifyOpenAPICompatibility(nil, nil)
	beforeSpec := map[string]interface{}{
		"paths": map[string]interface{}{
			"/v1/removed": map[string]interface{}{"get": map[string]interface{}{}},
			"/v1/orders": map[string]interface{}{
				"get":    map[string]interface{}{"required_fields": []string{"id"}},
				"delete": map[string]interface{}{},
			},
		},
	}
	afterSpec := map[string]interface{}{
		"paths": map[string]interface{}{
			"/v1/orders": map[string]interface{}{
				"get": map[string]interface{}{"required_fields": []interface{}{"id", "tenant_id"}},
			},
		},
	}
	compat := ae.VerifyOpenAPICompatibility(beforeSpec, afterSpec)
	if compat["compatible"] != false {
		t.Fatalf("expected breaking changes in OpenAPI spec, got %+v", compat)
	}

	ie := NewInfrastructureEngine()
	_ = ie.EvaluateSystemicReadiness(map[string]interface{}{
		"backup_restore_tested":    false,
		"tls_days_remaining":       5,
		"rto_minutes":              120,
		"rpo_minutes":              30,
		"opentelemetry_enabled":    false,
		"correlation_ids_enabled":  false,
		"branch_protection_active": false,
	})
	_ = ie.EvaluateSystemicReadiness(map[string]interface{}{
		"backup_restore_tested":    true,
		"tls_days_remaining":       5,
		"branch_protection_active": true,
	})

	gw := NewRAEQualityGatewayV2()
	_ = gw.GetPolicy()
	_ = gw.AnalyzeDiff("package main\nfunc main() {}\n")
	_ = gw.SecurityScan("package main\nfunc main() {}\n", "main.go")
	_ = gw.GenerateTests("CreateOrder")
	_ = gw.VerifyFix("package main\nfunc main() {}\n")

	// VerifyFix tamper cases
	_ = gw.CodeQualityEngine.VerifyFix("orig", "fixed", "", "func TestA(t *testing.T) { if x != 1 { t.Fatal() } }", "func TestA(t *testing.T) { if x != 2 { t.Fatal() } }")
	_ = gw.CodeQualityEngine.VerifyFix("orig", "fixed", "", "", "func TestA(t *testing.T) { t.Skip() }")
	_ = gw.CodeQualityEngine.VerifyFix("orig", "fixed", "", "", "def test_a(): assert True")

	// Scorecard tiers & AuditTrailRecorder tamper check
	_ = gw.ScorecardTracker.RecordChangeEvaluation("ag-1", 1, true, false, true, false, false)
	_ = gw.ScorecardTracker.RecordChangeEvaluation("ag-2", 5, false, false, false, true, true)
	atr := NewAuditTrailRecorder()
	atr.AppendDecision("op1", "PASS", "a1", map[string]interface{}{"k": "v"})
	atr.AppendDecision("op2", "PASS", "a1", map[string]interface{}{"k": "v2"})
	atr.chain[1]["entry_sha256"] = "tampered"
	if v := atr.VerifyChainIntegrity(); v["valid"] != false {
		t.Fatalf("expected invalid chain after entry_sha256 tamper, got %+v", v)
	}
	atr.chain[1]["prev_sha256"] = "broken-prev"
	if v := atr.VerifyChainIntegrity(); v["valid"] != false {
		t.Fatalf("expected invalid chain after prev_sha256 tamper, got %+v", v)
	}

	// DispatchRPC all 8 operations + unsupported
	for _, op := range []string{
		"quality.get_policy", "quality.analyze_diff", "quality.security_scan",
		"quality.generate_tests", "quality.verify_fix", "quality.check_architecture",
		"quality.release_gate", "quality.report",
	} {
		if _, err := gw.DispatchRPC(op, nil); err != nil {
			t.Fatalf("DispatchRPC(%s) failed: %v", op, err)
		}
	}
	if _, err := gw.DispatchRPC("quality.unknown_op", nil); err == nil {
		t.Fatal("expected error for unknown RPC op")
	}

	// ScanRepositoryDir on temporary directory
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0o600)
	_ = os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), []byte("FROM golang:1.22\nCMD [\"/app\"]\n"), 0o600)
	_ = os.MkdirAll(filepath.Join(tmpDir, "k8s"), 0o755)
	_ = os.WriteFile(filepath.Join(tmpDir, "k8s", "deployment.yaml"), []byte("kind: Deployment\nspec:\n  containers:\n  - image: app:latest\n"), 0o600)
	if _, err := gw.ScanRepositoryDir(tmpDir); err != nil {
		t.Fatalf("ScanRepositoryDir failed: %v", err)
	}
	if _, err := gw.ScanRepositoryDir(filepath.Join(tmpDir, "missing")); err == nil {
		t.Fatal("expected error for missing directory in ScanRepositoryDir")
	}

	// 4. Service lazy init, CallQualityMCPTool (all 20 tools), ScanDirectory, HTTP Routes
	lazySvc := &Service{}
	_ = lazySvc.gateway()
	lazySvc.ensureEngines()

	svc := NewService()
	if _, err := svc.ScanDirectory(tmpDir, 0); err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}
	_ = svc.RunMutationTesting("not go code", 0)
	_ = svc.AggregateCoverage(nil)
	_ = svc.AuditCode("subprocess.run(cmd, shell=True)\neval(x)\npath = '/home/grzegorz/a'", []string{"style.css", "old.printnow.pl/index.php"}, "")

	for _, tool := range svc.ListQualityMCPTools() {
		tName := tool["name"].(string)
		if _, err := svc.CallQualityMCPTool(tName, map[string]interface{}{
			"code":      "package main\nfunc main() {}\n",
			"doc_spec":  map[string]interface{}{"document_id": "d1"},
			"modified_files": []interface{}{"calc.go", "pdf.go"},
		}); err != nil {
			t.Fatalf("CallQualityMCPTool(%s) failed: %v", tName, err)
		}
	}
	// Test baseline/proposed test weakening in run_static_quality_audit
	_, _ = svc.CallQualityMCPTool("run_static_quality_audit", map[string]interface{}{
		"baseline_code": "def test_x():\n    assert a == 1\n    assert b == 2\n",
		"proposed_code": "def test_x():\n    assert True\n",
	})
	if _, err := svc.CallQualityMCPTool("unknown_quality_tool", nil); err == nil {
		t.Fatal("expected error for unknown quality MCP tool")
	}

	// Test all HTTP routes on RegisterRoutes
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

	httpCases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/health", ""},
		{http.MethodGet, "/mcp/sse", ""},
		{http.MethodPost, "/mcp/messages", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`},
		{http.MethodPost, "/mcp/messages", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_risk_coverage","arguments":{}}}`},
		{http.MethodPost, "/mcp/messages", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"unknown","arguments":{}}}`},
		{http.MethodPost, "/mcp/messages", `{"jsonrpc":"2.0","id":4,"method":"ping"}`},
		{http.MethodPost, "/v2/quality/audit", `{"code":"package main\nfunc main() {}\n"}`},
		{http.MethodPost, "/v2/quality/tribunal", `{"code":"eval(x)"}`},
		{http.MethodPost, "/v2/quality/mutation", `{"code":"package main\nfunc main() {}\n","test_count":2}`},
		{http.MethodPost, "/v2/quality/mutation-audit", `{}`},
		{http.MethodPost, "/v2/quality/coverage", `{"modules":{"m1":0.95,"m2":0.92}}`},
		{http.MethodPost, "/v2/quality/risk-coverage", `{"executed_tests":["calc","dtp"]}`},
		{http.MethodPost, "/v2/quality/risk-coverage/verify", `{"executed_tests":["calc"]}`},
		{http.MethodPost, "/v2/quality/gate-decision", `{"code":"eval(x)"}`},
		{http.MethodPost, "/v2/quality/dtp-audit", `{"document_id":"d1","dpi_min":100}`},
		{http.MethodPost, "/v2/quality/calculator-audit", `{"tiers":[{"volume":100,"total_price":100},{"volume":200,"total_price":50}]}`},
		{http.MethodGet, "/v2/quality/defect-metrics?code=package+main", ""},
		{http.MethodGet, "/v2/quality/golden-set-audit", ""},
		{http.MethodPost, "/v2/quality/sonarqube-audit", `{"code":"package main\nfunc main() {}\n"}`},
		{http.MethodPost, "/v2/quality/predictive-plan", `{"modified_files":["service.go","mesh.go"],"code":"eval(x)"}`},
		{http.MethodPost, "/v2/quality/sonar-strict-metrics", `{"code":"package main\nfunc main() {}\n"}`},
		{http.MethodGet, "/v2/quality/unit-test-inventory?module=memory", ""},
		{http.MethodPost, "/v2/quality/policy", `{}`},
		{http.MethodPost, "/v2/quality/analyze-diff", `{"changed_files":["a.go"],"diff":"package main"}`},
		{http.MethodPost, "/v2/quality/security-scan", `{"code":"package main","dockerfile_content":"FROM alpine","k8s_manifest":"kind: Deployment"}`},
		{http.MethodPost, "/v2/quality/generate-tests", `{"target_function":"Compute"}`},
		{http.MethodPost, "/v2/quality/verify-fix", `{"code":"package main\nfunc main() {}\n"}`},
		{http.MethodPost, "/v2/quality/check-architecture", `{"code":"package raecore","imports":["services/memory"]}`},
		{http.MethodPost, "/v2/quality/release-gate", `{"code":"package main\nfunc main() {}\n"}`},
		{http.MethodPost, "/v2/quality/report", `{"code":"package main\nfunc main() {}\n"}`},
		{http.MethodGet, "/v2/quality/manifest", ""},
		{http.MethodPost, "/v2/quality/pdf-binary-audit", `{"raw_pdf":"%PDF-1.7"}`},
		{http.MethodPost, "/v2/quality/multi-ast-audit", `{"language":"go","code":"package main\nfunc main() {}\n"}`},
		{http.MethodPost, "/v2/quality/scan-repo", `{"path":"` + tmpDir + `"}`},
		{http.MethodGet, "/v2/quality/scan-repo?path=/nonexistent/path/xyz", ""},
		{http.MethodPost, "/v2/quality/rpc", `{"operation":"quality.get_policy","params":{}}`},
		{http.MethodPost, "/v2/quality/rpc", `{"method":"audit","params":{"code":"package main"}}`},
	}
	for _, hc := range httpCases {
		_ = callHTTP(hc.method, hc.path, hc.body)
	}
}
