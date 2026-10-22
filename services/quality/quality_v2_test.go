package quality

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRAEQualityV2FullCatalogAndPolicyEngine(t *testing.T) {
	pe := NewPolicyEngine()
	cat := pe.Catalog()
	if len(cat) != 219 {
		t.Fatalf("expected 219 total rules (198 + 12 AI/SaaS + 9 RAE-EDITOR-SECURITY), got %d", len(cat))
	}

	// Verify canonical RAE-SEC-006 override and RAE-EDT-001..009 presence
	r006, ok := pe.GetRule("RAE-SEC-006")
	if !ok || r006.Enforcement != EnforcementBlock || len(r006.EvidenceRequired) != 4 {
		t.Fatalf("expected canonical RAE-SEC-006 rule with 4 evidence requirements, got %+v", r006)
	}
	for i := 1; i <= 9; i++ {
		edtID := "RAE-EDT-00" + string(rune('0'+i))
		if _, exists := pe.GetRule(edtID); !exists {
			t.Fatalf("missing RAE-EDITOR-SECURITY rule %s", edtID)
		}
	}

	// Cosmetic UI change -> LOW risk, only 4 rules
	cosmeticPol := pe.GetPolicyForTask("Zmień kolor przycisku", []string{"styles/button.css"}, nil, "", "")
	if cosmeticPol["is_cosmetic_ui_only"] != true || cosmeticPol["risk_level"] != "LOW" || cosmeticPol["selected_rules_count"].(int) != 4 {
		t.Fatalf("expected LOW risk cosmetic policy with 4 rules, got %+v", cosmeticPol)
	}

	// Order + Editor + DB change -> CRITICAL risk
	orderPol := pe.GetPolicyForTask("Update order checkout and PDF bleed calculator", []string{"api/orders_controller.go", "editor/pdf_export.go"}, []string{"postgresql", "redis"}, "", "etap_4_prod")
	if orderPol["risk_level"] != "CRITICAL" || orderPol["selected_rules_count"].(int) < 25 {
		t.Fatalf("expected CRITICAL risk policy with >=25 selected rules, got %+v", orderPol)
	}

	// Rollout phase filtering: etap_1_p0 (P0 only) vs etap_4_prod
	p0Pol := pe.GetPolicyForTask("Update order database migration and k8s deployment", []string{"migrations/002_orders.py", "k8s/deployment.yaml"}, nil, "", "etap_1_p0")
	p0Rules, _ := p0Pol["rules"].([]QualityRule)
	for _, r := range p0Rules {
		if r.Priority != PriorityP0 {
			t.Fatalf("etap_1_p0 policy must only select P0 rules, got %s (%s)", r.ID, r.Priority)
		}
	}

	// Evidence validation: bare agent statement rejected
	badEv := pe.ValidateRuleEvidence("RAE-SEC-006", EvidenceItem{
		RuleID:         "RAE-SEC-006",
		AgentStatement: "sprawdziłem bezpieczeństwo",
		VerifiedByTool: false,
	})
	if badEv["valid"] != false {
		t.Fatalf("expected bare agent statement to be rejected, got %+v", badEv)
	}

	goodEv := pe.ValidateRuleEvidence("RAE-SEC-006", EvidenceItem{
		RuleID:            "RAE-SEC-006",
		SourceLocation:    "orders.go:42",
		AffectedEndpoint:  "/api/v1/orders",
		AuthorizationPath: "jwt->tenant_id",
		TestResult:        "PASS: NEG-IDOR-001",
		VerifiedByTool:    true,
	})
	if goodEv["valid"] != true {
		t.Fatalf("expected tool-verified complete evidence to pass, got %+v", goodEv)
	}

	// Agent Tampering (RAE-SEC-208)
	tamper := pe.CheckAgentTampering("rogue-agent", []string{"services/quality/security_gateway.go"}, true, []string{"RAE-SEC-001"})
	if tamper["tamper_detected"] != true || tamper["enforcement"] != "BLOCK" {
		t.Fatalf("expected agent tampering to be blocked, got %+v", tamper)
	}
}

func TestPerfectionUpgradeG1ToG5(t *testing.T) {
	// G1: AIReviewEngine semantic checks
	aiEngine := NewAIReviewEngine()
	res1 := aiEngine.ReviewChangeSemantics("def get_all_orders(user):\n    if user.role == \"printshop_admin\":\n        return db.query(Order).all()\n", "orders_admin.py", "")
	if res1["verdict"] != "BLOCK" {
		t.Fatalf("expected RAE-EDT-003 BLOCK on printshop_admin without tenant_id/audit_log, got %+v", res1)
	}
	res2 := aiEngine.ReviewChangeSemantics("def update_order(req, order):\n    order.status = req.body.status\n    discount_percent = req.body.discount\n", "checkout.py", "")
	if res2["verdict"] != "BLOCK" {
		t.Fatalf("expected RAE-SEC-202 BLOCK on client-controlled order status/discount, got %+v", res2)
	}
	cleanCode := "@router.get(\"/projects/{project_id}\")\ndef get_project(project_id: str, current_user = Depends(get_current_tenant)):\n    check_project_owner(project_id, current_user.tenant_id, current_user.owner_id)\n    audit_log.record(\"project_read\", project_id=project_id, tenant_id=current_user.tenant_id)\n    return db.query(Project).filter_by(id=project_id, tenant_id=current_user.tenant_id).first()\n"
	res3 := aiEngine.ReviewChangeSemantics(cleanCode, "projects.py", "")
	if res3["verdict"] != "PASS" || res3["requirement_aligned"] != true {
		t.Fatalf("expected PASS on clean semantic review, got %+v", res3)
	}

	// G2: All 9 RAE-EDITOR-SECURITY rules active
	gw := NewSecurityGateway()
	edtSamples := map[string]string{
		"RAE-EDT-001": "project = Project.get_by_id_unscoped(project_id)",
		"RAE-EDT-002": `redis.set(f"project:{project_id}", payload)`,
		"RAE-EDT-003": "allow_cross_printshop_admin = True",
		"RAE-EDT-004": "autosave_overwrite_without_version = True",
		"RAE-EDT-005": "order.total_price = req.body.price",
		"RAE-EDT-006": "preserve_exif_gps = True",
		"RAE-EDT-007": "share_token = md5(project_id)",
		"RAE-EDT-008": "render_worker_privileged = True",
		"RAE-EDT-009": "skip_storage_cleanup_on_gdpr_delete = True",
	}
	for ruleID, snippet := range edtSamples {
		findings := gw.ScanCode(snippet, "editor_service.py")
		found := false
		for _, f := range findings {
			if f.RuleID == ruleID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing active detector for %s on snippet: %s", ruleID, snippet)
		}
	}

	// G3: Extended OWASP & Identity detectors
	secSamples := map[string]string{
		"RAE-SEC-003": "verify_csrf = False",
		"RAE-SEC-004": `allow_origins = ["*"], allow_credentials = True`,
		"RAE-SEC-010": "Content-Security-Policy: default-src 'self' 'unsafe-eval'",
		"RAE-SEC-023": "digest = hashlib.md5(password).hexdigest()",
		"RAE-SEC-024": `jwt.decode(token, key, algorithms=["none"])`,
		"RAE-SEC-032": "if webhook_signature == expected_signature: pass",
		"RAE-SEC-034": "verify_webhook_signature = False",
	}
	for ruleID, snippet := range secSamples {
		findings := gw.ScanCode(snippet, "auth_security.py")
		found := false
		for _, f := range findings {
			if f.RuleID == ruleID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing active detector for %s on snippet: %s", ruleID, snippet)
		}
	}

	// G5: TestIntegrityGuard assertion softening in VerifyFix
	cqe := NewCodeQualityEngine(gw)
	softRes := cqe.VerifyFix(
		"tenant_id = req.body.tenantId",
		"tenant_id = session.authenticated_tenant_id",
		"RAE-SEC-006",
		"def test_tenant_isolation():\n    assert response.status_code == 403\n    assert response.json()[\"error\"] == \"forbidden\"\n",
		"def test_tenant_isolation():\n    assert response.status_code == 403\n    assert True\n",
	)
	if softRes["verdict"] != "BLOCK" || softRes["tamper_detected"] != true {
		t.Fatalf("expected assertion softening (assert True) to be blocked, got %+v", softRes)
	}
}

func TestSecurityGateway24RulesAndSARIF(t *testing.T) {
	sg := NewSecurityGateway()

	// PHP scanner compatibility -> CodeQL skipped with fallback warning
	phpMat := sg.GetCompatibleScanners([]string{"php"})
	notes, _ := phpMat["compatibility_notes"].([]string)
	if len(notes) == 0 {
		t.Fatalf("expected CodeQL PHP compatibility warning, got %+v", phpMat)
	}

	vulnerableCode := `
package handlers
func BadHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	q := fmt.Sprintf("SELECT * FROM orders WHERE tenant_id = '%s'", tenantID)
	apiKey := "sk-1234567890abcdefghijklmnop"
	order.TotalPrice = req.Price
	rdb.Set(ctx, fmt.Sprintf("project:%d", projectID), data, 0)
	_ = q
	_ = apiKey
}
`
	res := sg.ScanCodeDetailed(vulnerableCode, "handlers/bad.go", []string{"go"}, nil)
	if res["verdict"] != "BLOCK" || res["findings_count"].(int) < 5 {
		t.Fatalf("expected BLOCK verdict and >=5 findings (SQLi, IDOR, Secret, Price, Tenant Cache), got %+v", res)
	}

	// Verify baseline fingerprint separation
	findings, _ := res["findings"].([]SecurityFinding)
	var fps []string
	for _, f := range findings {
		if fp, ok := f.Evidence["fingerprint"].(string); ok {
			fps = append(fps, fp)
		}
	}
	resWithBaseline := sg.ScanCodeDetailed(vulnerableCode, "handlers/bad.go", []string{"go"}, fps)
	if resWithBaseline["verdict"] != "PASS" || resWithBaseline["findings_count"].(int) != 0 || resWithBaseline["baseline_debt_count"].(int) != len(findings) {
		t.Fatalf("expected all findings moved to baseline_debt_findings, got %+v", resWithBaseline)
	}
}

func TestArchitectureInfrastructureCodeQualityAndAutoRepair(t *testing.T) {
	ae := NewArchitectureEngine()
	archBad := `
func SaveOrderFlow(ctx context.Context, items []int) {
	SaveOrder(ctx)
	SaveOrderItems(ctx)
	SaveProject(ctx)
	for _, id := range items {
		db.Query(ctx, "SELECT * FROM items WHERE id = $1", id)
	}
	rdb.Set(ctx, "k", "v", 0)
}
`
	archRes := ae.CheckArchitecture(
		archBad,
		"migrations/001_migration.go",
		[]string{"postgresql", "redis"},
		map[string]interface{}{"paths": map[string]interface{}{"/v1/orders": map[string]interface{}{"get": map[string]interface{}{}}}},
		map[string]interface{}{"paths": map[string]interface{}{}},
	)
	if archRes["verdict"] != "BLOCK" || archRes["findings_count"].(int) < 6 {
		t.Fatalf("expected BLOCK and >=6 architecture findings (Down migration, N+1, Tx, Race, TTL, OpenAPI), got %+v", archRes)
	}

	// InfrastructureEngine
	ie := NewInfrastructureEngine()
	dfFindings := ie.AuditDockerfile("FROM golang:1.22\nCMD [\"/app\"]\n", "Dockerfile")
	if len(dfFindings) != 3 {
		t.Fatalf("expected 3 Dockerfile findings (Multi-Stage, Non-Root, Healthcheck), got %d", len(dfFindings))
	}
	k8sFindings := ie.AuditK8sManifest("kind: Deployment\nspec:\n  containers:\n  - image: app:1.0\n", "deployment.yaml")
	if len(k8sFindings) != 2 {
		t.Fatalf("expected 2 K8s findings (Resources, Probes), got %d", len(k8sFindings))
	}

	// CodeQualityEngine & TestIntegrityGuard
	cqe := NewCodeQualityEngine(nil)
	gen := cqe.GenerateTests("/api/v1/orders", "orders_api", nil)
	if gen["total_generated"].(int) != 10 {
		t.Fatalf("expected 10 canonical negative scenarios, got %+v", gen)
	}

	tamperedFix := cqe.VerifyFix(
		"q = fmt.Sprintf(\"SELECT * FROM t WHERE id=%s\", id)",
		"q = \"SELECT * FROM t WHERE id=$1\"",
		"RAE-SEC-001",
		"def test_sqli():\n    assert status == 403\n    assert leaked == 0\n",
		"def test_sqli():\n    assert status == 403\n",
	)
	if tamperedFix["verdict"] != "BLOCK" || tamperedFix["tamper_detected"] != true {
		t.Fatalf("expected test assertion deletion to be blocked, got %+v", tamperedFix)
	}

	cleanFix := cqe.VerifyFix(
		"q := fmt.Sprintf(\"SELECT * FROM t WHERE id=%s\", id)",
		"q := \"SELECT * FROM t WHERE id=$1\"",
		"RAE-SEC-001",
		"def test_sqli():\n    assert status == 403\n",
		"def test_sqli():\n    assert status == 403\n",
	)
	if cleanFix["verdict"] != "PASS" || cleanFix["fix_verified"] != true {
		t.Fatalf("expected clean fix with unmodified test to pass, got %+v", cleanFix)
	}

	// AutoRepairOrchestrator: max 3 attempts -> escalate on 4th
	aro := NewAutoRepairOrchestrator()
	sampleFinding := []SecurityFinding{{RuleID: "RAE-SEC-001", Remediation: "Use bound params"}}
	for i := 1; i <= 3; i++ {
		tTask := aro.EvaluateOrScheduleRepair("CHG-99", "agent-1", sampleFinding)
		if tTask.Verdict != VerdictRepair || tTask.EscalatedToHuman {
			t.Fatalf("attempt %d should be REPAIR without escalation, got %+v", i, tTask)
		}
	}
	t4 := aro.EvaluateOrScheduleRepair("CHG-99", "agent-1", sampleFinding)
	if t4.Verdict != VerdictBlock || !t4.EscalatedToHuman {
		t.Fatalf("4th attempt must BLOCK and escalate to human, got %+v", t4)
	}
}

func TestSection8PRAssessmentAndGovernanceEngines(t *testing.T) {
	svc := NewService()
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// 1. Exact Section 8 Pull Request Assessment Dual Gate simulation from poziom-bezpieczeństwa-kodu.md
	sec8Body, _ := json.Marshal(map[string]interface{}{
		"code":                                 "def get_order(req):\n    tid = req.body.tenantId\n    return tid\n",
		"tests_passed":                         246,
		"tests_total":                          250,
		"concurrency_review":                   true,
		"quality_score_override_for_simulation": 87,
	})
	reqSec8 := httptest.NewRequest(http.MethodPost, "/v2/quality/release-gate", bytes.NewReader(sec8Body))
	recSec8 := httptest.NewRecorder()
	mux.ServeHTTP(recSec8, reqSec8)
	var sec8Res map[string]interface{}
	_ = json.Unmarshal(recSec8.Body.Bytes(), &sec8Res)
	if sec8Res["verdict"] != "BLOCK" || sec8Res["merge_allowed"] != false || int(sec8Res["quality_score"].(float64)) != 87 || int(sec8Res["tests_failed"].(float64)) != 4 {
		t.Fatalf("unexpected Section 8 PR assessment response: %+v", sec8Res)
	}

	// 2. DefectTracker CFR, Escape Rate, Weighted Score & MTTR
	dt := NewDefectTracker()
	dt.RegisterDeployment(true)
	dt.RegisterDeployment(true)
	dt.RegisterDeployment(false)
	m0 := dt.CalculateMetrics()
	if m0.ChangeFailureRate != 33.33 {
		t.Fatalf("expected 33.33%% CFR, got %+v", m0)
	}
	dt.RegisterDefect("CI Syntax Error", "P3", "CI", "editor", 0)
	dt.RegisterDefect("Prod Crash", "P0", "PRODUCTION", "backend", 2.0)
	m1 := dt.CalculateMetrics()
	if m1.TotalDefects != 2 || m1.ProductionDefects != 1 || m1.DefectEscapeRate != 50.0 || m1.SeverityWeightedScore != 101 || m1.MTTRHours != 2.0 {
		t.Fatalf("unexpected DefectMetrics: %+v", m1)
	}

	// 3. RiskCoverageEngine 8 paths, live verification, P0 blocker & P1 warning
	rce := NewRiskCoverageEngine()
	rce.RunAutomatedLiveVerification()
	gateOK := rce.EvaluateDeploymentGateMatrix()
	if gateOK["verdict"] != "LEVEL_1_APPROVED" || gateOK["can_deploy"] != true {
		t.Fatalf("expected LEVEL_1_APPROVED after live verification, got %+v", gateOK)
	}
	rce.UpdatePathScore("RC-01", 0.40, map[string]interface{}{"error": "Negative margin"})
	gateBlocked := rce.EvaluateDeploymentGateMatrix()
	if gateBlocked["verdict"] != "LEVEL_3_BLOCKED" || gateBlocked["can_deploy"] != false {
		t.Fatalf("expected LEVEL_3_BLOCKED on P0 failure, got %+v", gateBlocked)
	}

	// 4. SonarQubeClient strict Rating A evaluation (OK vs Violations)
	sc := NewSonarQubeClient("http://mock:9000", "tok")
	strictOK := sc.EvaluateStrictMeasuresMap("editornext", map[string]string{
		"vulnerabilities": "0", "bugs": "0", "code_smells": "0", "security_hotspots": "0", "coverage": "95.5", "duplicated_lines_density": "0.0",
	})
	if strictOK["is_strict_a"] != true {
		t.Fatalf("expected strict Rating A pass, got %+v", strictOK)
	}
	strictBad := sc.EvaluateStrictMeasuresMap("editornext", map[string]string{
		"vulnerabilities": "1", "bugs": "2", "code_smells": "5", "security_hotspots": "0", "coverage": "84.2", "duplicated_lines_density": "1.5",
	})
	if strictBad["is_strict_a"] != false || len(strictBad["violations"].([]string)) < 4 {
		t.Fatalf("expected strict Rating A violations, got %+v", strictBad)
	}

	// 5. MCP tools/list and tools/call for all 20 RAE-Quality tools
	if len(svc.ListQualityMCPTools()) != 20 {
		t.Fatalf("expected 20 RAE-Quality MCP tools (12 domain + 8 quality.*), got %d", len(svc.ListQualityMCPTools()))
	}
	mcpRep, err := svc.CallQualityMCPTool("quality.report", nil)
	if err != nil || mcpRep["acceptance_criteria_score"] != "9/9" {
		t.Fatalf("CallQualityMCPTool quality.report failed: %+v, err=%v", mcpRep, err)
	}

	// 6. GET /v2/quality/report -> 9/9 Acceptance Criteria
	reqRep := httptest.NewRequest(http.MethodGet, "/v2/quality/report", nil)
	recRep := httptest.NewRecorder()
	mux.ServeHTTP(recRep, reqRep)
	var repRes map[string]interface{}
	_ = json.Unmarshal(recRep.Body.Bytes(), &repRes)
	if repRes["acceptance_criteria_score"] != "9/9" || repRes["all_acceptance_criteria_met"] != true {
		t.Fatalf("expected 9/9 acceptance criteria met, got %+v", repRes)
	}
}
