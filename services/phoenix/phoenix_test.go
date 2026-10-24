package phoenix

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPhoenixImpactParityAndCallbacks(t *testing.T) {
	svc := NewService()
	var failedReason string
	svc.OnRepairFailure = func(reason string) {
		failedReason = reason
	}

	imp := svc.AnalyzeImpact("core.go", []string{"a.go", "b.go"})
	if !imp.SafeToRefactor {
		t.Fatalf("expected SafeToRefactor=true")
	}

	res := svc.ExecuteClosedLoopRepair("EXPECTED", []string{"BAD1", "BAD2", "BAD3"})
	if !res.RolledBack || failedReason == "" {
		t.Fatalf("expected rollback and failure callback, got %+v", res)
	}
}

func TestPhoenixASTImportGraphAndRecipes(t *testing.T) {
	svc := NewService()

	sources := map[string]string{
		"pkg/raecore/math.go":     "package raecore\n",
		"services/memory/svc.go":  "package memory\nimport \"github.com/vproject111/RAE-Suite-go/pkg/raecore\"\n",
		"cmd/rae-memory/main.go":  "package main\nimport \"github.com/vproject111/RAE-Suite-go/services/memory\"\n",
		"unrelated/standalone.py": "import os\nfrom sys import path\n",
	}

	rep := svc.AnalyzeImpactWithGraph("pkg/raecore/math.go", nil, sources)
	if len(rep.AffectedFiles) != 1 || rep.AffectedFiles[0] != "services/memory/svc.go" {
		t.Fatalf("expected services/memory/svc.go as direct dependent, got %+v", rep.AffectedFiles)
	}
	if len(rep.TransitiveDependents) != 2 {
		t.Fatalf("expected 2 transitive dependents (svc.go -> main.go), got %+v", rep.TransitiveDependents)
	}

	// Test high risk (> 1.0 cap) with >= 9 dependents
	manyDeps := []string{"d1.go", "d2.go", "d3.go", "d4.go", "d5.go", "d6.go", "d7.go", "d8.go", "d9.go", "d10.go"}
	repHigh := svc.AnalyzeImpact("root.go", manyDeps)
	if repHigh.RiskScore != 1.0 || repHigh.SafeToRefactor {
		t.Fatalf("expected capped RiskScore=1.0 and SafeToRefactor=false, got %+v", repHigh)
	}

	recipes := svc.ListRecipes()
	if len(recipes) < 4 {
		t.Fatalf("expected at least 4 refactoring recipes, got %d", len(recipes))
	}
}

func TestPhoenixRoutesBehaviorPostMortemAndMeta(t *testing.T) {
	t.Setenv("PHOENIX_MAX_ATTEMPTS", "4")
	svc := NewService()
	if svc.MaxAttempts != 4 {
		t.Fatalf("expected MaxAttempts=4 from env, got %d", svc.MaxAttempts)
	}

	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}

	var completedOK bool
	svc.OnRepairCompleted = func(repaired bool, _ string) {
		completedOK = repaired
	}
	svc.MaxAttempts = 0 // tests <= 0 fallback to 3
	repOK := svc.ExecuteClosedLoopRepair("OK_OUT", []string{"BAD", "OK_OUT", "EXTRA"})
	if repOK.Status != "REPAIRED" || !completedOK {
		t.Fatalf("expected REPAIRED, got %+v", repOK)
	}
	// Exceeding limit slice
	svc.MaxAttempts = 2
	repFail := svc.ExecuteClosedLoopRepair("TARGET", []string{"B1", "B2", "B3", "B4"})
	if !repFail.RolledBack || repFail.AttemptsUsed != 2 {
		t.Fatalf("expected 2 attempts used and rollback, got %+v", repFail)
	}

	// CompareBehavior6D all branches (passing and all 6 violation types)
	base := BehaviorSnapshot{
		HTTPStatus:   200,
		CLIExitCode:  0,
		DOMSelectors: []string{"#main", ".btn"},
		JSONPayload:  map[string]interface{}{"id": 1, "name": "ok"},
	}
	candGood := BehaviorSnapshot{
		HTTPStatus:   200,
		CLIExitCode:  0,
		CLIStdout:    "build-success-123",
		DOMSelectors: []string{"#main", ".btn"},
		LogLines:     []string{"info: ready"},
		JSONPayload:  map[string]interface{}{"id": 1, "name": "ok"},
	}
	chkPass := CompareBehavior6D(base, candGood, `success-\d+`)
	if !chkPass.Passed {
		t.Fatalf("expected 6D check to pass, got %+v", chkPass)
	}

	candBad := BehaviorSnapshot{
		HTTPStatus:   500,
		CLIExitCode:  1,
		CLIStdout:    "failed",
		DOMSelectors: []string{"#main"},
		LogLines:     []string{"panic: runtime error", "Traceback (most recent call last)"},
		JSONPayload:  map[string]interface{}{"id": 1},
	}
	chkFail := CompareBehavior6D(base, candBad, `success-\d+`)
	if chkFail.Passed || len(chkFail.Violations) < 5 || chkFail.RiskScore != 1.0 {
		t.Fatalf("expected full 6D failure with capped risk=1.0, got %+v", chkFail)
	}

	// AnalyzePostMortem categories
	pm1 := AnalyzePostMortem("", 3, []string{"nil pointer dereference"}, 0)
	if pm1.RootCauseCategory != "NIL_DEREFERENCE" || !pm1.RecommendedHermesRoute {
		t.Fatalf("unexpected pm1: %+v", pm1)
	}
	pm2 := AnalyzePostMortem("s2", 2, []string{"vector dimension mismatch"}, 0.05)
	if pm2.RootCauseCategory != "VECTOR_DIMENSION_MISMATCH" {
		t.Fatalf("unexpected pm2: %+v", pm2)
	}
	pm3 := AnalyzePostMortem("s3", 1, []string{"schema timeout"}, 0.01)
	if pm3.RootCauseCategory != "CONTRACT_DRIFT" {
		t.Fatalf("unexpected pm3: %+v", pm3)
	}

	// ApplyRecipe all switch cases
	for _, rID := range []string{
		"feniks-01-extract-port",
		"feniks-02-angularjs-component",
		"directive_to_component",
		"routing_to_app_router",
		"scope_to_hooks",
		"template_to_jsx",
		"feniks-03-zero-hardcode-path",
		"feniks-04-controller-split",
		"custom-recipe",
	} {
		ar := svc.ApplyRecipe(rID, "$scope.x = '/home/user/data';")
		if ar.Status != "APPLIED_AND_VERIFIED" {
			t.Fatalf("recipe %s failed: %+v", rID, ar)
		}
	}

	// Meta-reflection functions edge cases
	smDefault := BuildSystemModel("", nil)
	if smDefault.ProjectName != "RAE-Suite" || len(smDefault.Components) == 0 {
		t.Fatalf("unexpected default BuildSystemModel: %+v", smDefault)
	}
	smLayers := BuildSystemModel("CustomProj", map[string]string{
		"api_handler.go":   `import "a"`,
		"store_adapter.go": `import "b"`,
		"bg_worker.go":     `import "c"`,
		"domain_logic.go":  `import "d"`,
	})
	if len(smLayers.Components) != 4 {
		t.Fatalf("expected 4 components")
	}

	ltDefault := AnalyzeLongitudinalTrends(nil, nil)
	if ltDefault["trend_direction"] != "IMPROVING" {
		t.Fatalf("expected IMPROVING default trend, got %+v", ltDefault)
	}
	ltDegrading := AnalyzeLongitudinalTrends([]float64{0.95, 0.80, 0.70}, []float64{0.2, 0.4})
	if ltDegrading["trend_direction"] != "DEGRADING" {
		t.Fatalf("expected DEGRADING trend, got %+v", ltDegrading)
	}
	ltStable := AnalyzeLongitudinalTrends([]float64{0.90, 0.905}, []float64{0.2, 0.2})
	if ltStable["trend_direction"] != "STABLE" {
		t.Fatalf("expected STABLE trend, got %+v", ltStable)
	}

	capsDefault := DetectCapabilities(nil)
	if capsDefault["capability_count"].(int) < 5 {
		t.Fatalf("unexpected default capabilities: %+v", capsDefault)
	}

	// HTTP routes
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	routes := []struct {
		method string
		path   string
		body   string
		expect string
	}{
		{http.MethodGet, "/health", "", "RAE-Phoenix"},
		{http.MethodGet, "/mcp/sse", "", "/mcp/messages"},
		{http.MethodPost, "/mcp/messages", `{"id":1}`, "RAE-Phoenix"},
		{http.MethodPost, "/v2/phoenix/impact", `{"target_file":"a.go","dependents":["b.go"]}`, "safe_to_refactor"},
		{http.MethodGet, "/v2/phoenix/recipes", "", "feniks-01-extract-port"},
		{http.MethodPost, "/v2/phoenix/recipes/apply", `{"code":"/home/tester/app"}`, "APPLIED_AND_VERIFIED"},
		{http.MethodPost, "/v2/phoenix/repair", `{"baseline_out":"ok","candidates":["ok"]}`, "REPAIRED"},
		{http.MethodPost, "/v2/phoenix/create", `{"spec":"type S struct{}"}`, "synthesized"},
		{http.MethodPost, "/sessions/analyze", `{"session_id":"s1","events":[{"msg":"panic error"}],"error_traces":["nil ptr"]}`, "anomalies_detected"},
		{http.MethodPost, "/sessions/analyze", `{"session_id":"s2","events":[{"msg":"all good"}]}`, "CLEAN_EXECUTION"},
		{http.MethodGet, "/report/rep-42", "", "rep-42"},
		{http.MethodGet, "/patterns/errors?filter=nil", "", "nil_pointer_deref"},
		{http.MethodGet, "/metrics", "", "repairs_succeeded"},
		{http.MethodPost, "/v2/phoenix/behavior-check", `{"baseline":{"http_status":200},"candidate":{"http_status":200}}`, "passed"},
		{http.MethodPost, "/v2/phoenix/post-mortem", `{"session_id":"pm1","attempts":3,"error_traces":["nil"]}`, "NIL_DEREFERENCE"},
		{http.MethodGet, "/metrics/prometheus", "", "rae_phoenix_max_attempts"},
		{http.MethodPost, "/v1/meta/system-model", `{}`, "hexagonal_modular_clean"},
		{http.MethodPost, "/v1/meta/longitudinal", `{}`, "trend_direction"},
		{http.MethodPost, "/v1/meta/capabilities", `{"code_snippets":["vector sql"]}`, "detected_capabilities"},
	}

	for _, tc := range routes {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s returned %d", tc.method, tc.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.expect) {
			t.Fatalf("%s %s body %q missing %q", tc.method, tc.path, rec.Body.String(), tc.expect)
		}
	}
}

func TestPhoenixDeepRecipesEnginesAndBackends(t *testing.T) {
	eng := NewRefactorEngine()
	if eng.ControllerRecipe.Description() == "" || eng.DirectiveRecipe.Description() == "" ||
		eng.RoutingRecipe.Description() == "" || eng.ScopeRecipe.Description() == "" ||
		eng.TemplateRecipe.Description() == "" {
		t.Fatalf("expected non-empty recipe descriptions")
	}

	// ControllerToComponentRecipe
	ctrlOut := eng.Execute("$scope.items = []; $scope.total = 0; $http.get('/api');")
	if !strings.Contains(ctrlOut, "setItems") {
		t.Fatalf("unexpected controller recipe output: %s", ctrlOut)
	}
	wf := eng.RunWorkflow("$scope.name = 'test';")
	if wf["output"] == "" {
		t.Fatalf("unexpected workflow output")
	}

	// DirectiveToComponentRecipe (with and without transclude)
	dir1 := eng.DirectiveRecipe.Execute("transclude: true, link: function() {}")
	dir2 := eng.DirectiveRecipe.Execute("restrict: 'E'")
	_ = eng.DirectiveRecipe.Analyze("transclude: true, link: function() {}")
	if !strings.Contains(dir1, "children") || strings.Contains(dir2, "{children}") {
		t.Fatalf("unexpected directive outputs")
	}

	// RoutingToAppRouterRecipe (with routes, root '/', and fallback)
	rt1 := eng.RoutingRecipe.Execute(".when('/').when('/users/:id').state('admin.settings')")
	rt2 := eng.RoutingRecipe.Execute("// no routes")
	if !strings.Contains(rt1, "users/[id]") || !strings.Contains(rt2, "dashboard") {
		t.Fatalf("unexpected routing outputs: %s / %s", rt1, rt2)
	}

	// ScopeToHooksRecipe
	scOut := eng.ScopeRecipe.Execute("$scope.$watch('count', fn); $scope.$broadcast('ev'); $scope.$apply()")
	if !strings.Contains(scOut, "useEffect") {
		t.Fatalf("unexpected scope output: %s", scOut)
	}

	// TemplateToJsxRecipe
	_ = eng.TemplateRecipe.Analyze(`<div class="box" ng-if="ok" ng-repeat="x in list" ng-click="run()" ng-model="val"></div>`)
	tplOut := eng.TemplateRecipe.Execute(`<div class="box" ng-if="ok" ng-repeat="x in list" ng-click="run()" ng-model="val"></div>`)
	if !strings.Contains(tplOut, "className=") {
		t.Fatalf("unexpected template output: %s", tplOut)
	}

	// BehaviorGuardIntegration & RefactorEngine registration (including zero-value RefactorEngine)
	zeroEng := &RefactorEngine{}
	zeroEng.RegisterRecipe("zero_custom")
	_ = eng.ValidateBehavior("OrderList")
	_ = eng.BehaviorGuard.CreateTestPlan("OrderList", nil)
	_ = eng.BehaviorGuard.GenerateBehaviorScenarios("OrderList")
	_ = eng.BehaviorGuard.GenerateTestScript("OrderList")
	_ = eng.BehaviorGuard.GenerateIntegrationGuide("OrderList")
	if eng.RegisterRecipe("") || !eng.RegisterRecipe("custom_r1") {
		t.Fatalf("unexpected RegisterRecipe behavior")
	}
	if len(eng.ListRecipes()) < 6 {
		t.Fatalf("expected custom recipe in ListRecipes")
	}

	// PhoenixRefactorer
	pr := NewPhoenixRefactorer()
	_ = pr.ProcessRepairRequest("feniks-01-extract-port", "package a")
	_ = pr.ProcessCreateRequest("Btn", `<button class="btn" ng-click="go()">Go</button>`)

	// AnalysisPipeline, ReflectionRuleSet, CapabilityDetector, ReportGenerator, RAEFormatter
	ap := NewAnalysisPipeline()
	pipeRes := ap.Run("TestProj", map[string]string{
		"a_service.go": `import "x" "y" "z" "w" "v"`,
	})
	if pipeRes["status"] != "COMPLETED" {
		t.Fatalf("unexpected pipeline result: %+v", pipeRes)
	}
	highCouplingModel := SystemModelReport{
		ProjectName:   "HighC",
		CouplingIndex: 0.80,
		Components: []SystemModelNode{
			{ComponentID: "hot.go", Layer: "api", HotspotRisk: 0.85},
		},
	}
	_ = ap.RuleSet.Evaluate(highCouplingModel)
	_ = ap.Detector.DetectFromSystemModel(highCouplingModel)
	_ = ap.Detector.EnrichSystemModel(&highCouplingModel, []string{"rwmutex sha256"})
	_ = ap.Reporter.GenerateModuleAnalysis(highCouplingModel)
	_ = ap.Reporter.GenerateCapabilityReport([]string{"cipher vector"})
	_ = ap.Reporter.GenerateRecommendations(highCouplingModel)

	fmttr := &RAEFormatter{}
	_ = fmttr.FormatMetaReflectionsBatch([]map[string]interface{}{{"k": "v"}})
	_ = fmttr.FormatSystemCapabilities(map[string]interface{}{"detected_capabilities": []string{"a"}})
	_ = fmttr.FormatSelfModelSummary(highCouplingModel)

	// ScenarioLibrary
	sl := NewScenarioLibrary()
	_ = sl.PublishScenario("", "Auto ID Scenario", map[string]interface{}{"k": 1})
	_ = sl.ImportScenario(map[string]interface{}{"id": "imp-1", "name": "Imported"})
	if len(sl.SearchScenarios("imported")) != 1 {
		t.Fatalf("expected 1 search match in ScenarioLibrary")
	}
	if !strings.Contains(sl.ExportLibraryToYAML(), "imp-1") {
		t.Fatalf("expected imp-1 in YAML export")
	}

	// UIRunner with httptest server (both novel and already-existing expected_dom)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<div id="hero" class="banner primary"><span class="title">Hi</span></div>`))
	}))
	defer ts.Close()
	uir := &UIRunner{}
	snap := uir.ExecuteScenario(map[string]interface{}{
		"url":          ts.URL,
		"expected_dom": "#custom-expected",
	})
	_ = uir.ExecuteScenario(map[string]interface{}{
		"html":         `<div id="hero"></div>`,
		"expected_dom": "#hero",
	})
	if snap.HTTPStatus != 200 || len(snap.DOMSelectors) < 5 {
		t.Fatalf("unexpected UIRunner snapshot: %+v", snap)
	}

	// BehaviorStore (including zero-value struct to cover ensureInit branches)
	bs := &BehaviorStore{}
	if len(bs.ListScenarios()) != 1 {
		t.Fatalf("expected default scenario when empty")
	}
	_ = NewBehaviorStore()
	if bs.SaveScenario("", nil) || !bs.SaveScenario("s1", map[string]interface{}{"a": 1}) {
		t.Fatalf("unexpected BehaviorStore.SaveScenario")
	}
	_ = bs.LoadScenario("s1")
	_ = bs.LoadScenario("missing")
	_ = bs.ListScenarios()
	_ = bs.DeleteScenario("s1")
	bs.SaveSnapshot("snap1", snap)
	_, _ = bs.GetSnapshot("snap1")
	_ = bs.SaveSnapshotsBatch([]BehaviorSnapshot{snap})
	_ = bs.LoadSnapshots()
	_ = bs.SaveContract("")
	_ = bs.SaveContract("c1")
	_ = bs.LoadContract("c1")
	_ = bs.LoadContract("missing")
	_ = bs.SaveCheckResult("chk1", true)
	_ = bs.LoadCheckResults("chk1")
	_ = bs.LoadCheckResults("missing")

	// PostgresBackend
	pg := &PostgresBackend{}
	_ = pg.ListScenarios()
	_ = pg.SaveScenario("", nil)
	_ = pg.SaveScenario("pg-s1", map[string]interface{}{"x": 1})
	_ = pg.LoadScenario("pg-s1")
	_ = pg.LoadScenario("missing")
	_ = pg.ListScenarios()
	_ = pg.DeleteScenario("pg-s1")
	_ = pg.SaveSnapshot("pg-s1", snap)
	_ = pg.LoadSnapshots("pg-s1")
	_ = pg.SaveContract("pg-c1")
	_ = pg.LoadContract("pg-c1")
	_ = pg.LoadContract("missing")
	_ = pg.SaveCheckResult("pg-chk", false)
	_ = pg.LoadCheckResults("pg-chk")
	_ = pg.LoadCheckResults("missing")

	// QdrantBackend
	qb := &QdrantBackend{}
	_ = qb.ListScenarios()
	_ = qb.SearchSimilarScenarios("query")
	_ = qb.SearchSimilarContracts("query")
	_ = qb.LoadScenario("missing")
	_ = qb.LoadContract("missing")
	_ = qb.SaveScenario("", nil)
	_ = qb.SaveScenario("q-s1", textToDeterministicVector("alpha", 0))
	_ = qb.SaveScenario("q-s2", textToDeterministicVector("beta", 8))
	_ = qb. SaveScenario("q-s3", textToDeterministicVector("alpha", 8)) // equal score tie-breaker
	_ = qb.LoadScenario("q-s1")
	_ = qb.ListScenarios()
	_ = qb.SearchSimilarScenarios("alpha")
	_ = qb.DeleteScenario("q-s1")
	_ = qb.SaveSnapshot("q-snap", []float64{1, 2})
	_ = qb.LoadSnapshots("q-snap")
	_ = qb.SaveContract("q-c1", textToDeterministicVector("contract", 8))
	_ = qb.SaveContract("q-c2", textToDeterministicVector("other", 8))
	_ = qb.LoadContract("q-c1")
	_ = qb.SearchSimilarContracts("contract")
	_ = cosineSimFloat64(nil, nil)
	_ = cosineSimFloat64([]float64{1, 2, 3}, []float64{1})
	_ = cosineSimFloat64([]float64{0, 0}, []float64{1, 1})

	// EnhancedRAEClient & FeniksMemoryRouter
	rc := &EnhancedRAEClient{BaseURL: "http://localhost"}
	_ = rc.QueryReflections("proj1")
	rc.reflections["proj1"] = []string{"ref1"}
	_ = rc.QueryReflections("proj1")
	_ = rc.GetCrossProjectPatterns()
	_ = rc.StoreRefactorOutcome("r1", true)
	_ = rc.GetHistoricalRefactorings("proj1")
	_ = rc.BatchEnrichReflections([]string{"a", "b"})

	fmr := &FeniksMemoryRouter{}
	if fmr.RouteStorage("HIGH", 0.1) != "reflective" || fmr.RouteStorage("LOW", 0.2) != "semantic" {
		t.Fatalf("unexpected FeniksMemoryRouter routing")
	}
	if fmr.Store("   ") || !fmr.Store("stored memory item") {
		t.Fatalf("unexpected FeniksMemoryRouter.Store")
	}
	_ = fmr.Retrieve("stored")
	_ = fmr.Retrieve("nonexistent-fallback")
	_ = fmr.EnrichWithGlobalContext("ctx")
}
