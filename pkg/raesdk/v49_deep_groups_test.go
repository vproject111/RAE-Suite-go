package raesdk_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/llm"
	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/drivers/sqlstore"
	"github.com/vproject111/RAE-Suite-go/pkg/drivers/vectorstore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/orchestrator"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/workers"
	"github.com/vproject111/RAE-Suite-go/pkg/raesdk"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/mcp"
	"github.com/vproject111/RAE-Suite-go/services/phoenix"
	"github.com/vproject111/RAE-Suite-go/services/portal"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func TestGroup1AllThirtyOneMCPToolsAndInterceptor(t *testing.T) {
	ctx := context.Background()
	guard := security.NewHardFramesGuard("secret")
	store := memorystore.NewUnifiedStore(nil, guard)
	crlSvc := crl.NewService(store, store, guard)
	mcpSvc := mcp.NewService(store, crlSvc)

	if len(mcpSvc.ListTools()) != 31 {
		t.Fatalf("expected 31 MCP tools, got %d", len(mcpSvc.ListTools()))
	}

	// Test PII scrubbing & save_memory
	saveRes, err := mcpSvc.CallTool(ctx, "save_memory", map[string]interface{}{
		"content": "Contact admin@dreamsoft.pro with key sk-1234567890abcdef123456",
		"project": "v49-test",
	})
	if err != nil {
		t.Fatalf("save_memory failed: %v", err)
	}
	cleaned, _ := saveRes["content"].(string)
	if strings.Contains(cleaned, "admin@dreamsoft.pro") || strings.Contains(cleaned, "sk-123456") {
		t.Fatalf("expected PII and secret to be scrubbed, got: %s", cleaned)
	}

	// Test new Suite & Supervisor & Integrations tools
	for _, toolName := range []string{
		"rae_kernel_execute",
		"rae_knowledge_resolve",
		"rae_lab_insights",
		"rae_quality_rpc",
		"get_cloud_status",
		"run_diagnostic",
		"get_service_logs",
		"create_rae_memory",
		"search_rae_memory",
		"check_approval_status",
		"get_circuit_breakers",
		"get_related_context",
		"list_policies",
		"search_memory",
	} {
		if _, err := mcpSvc.CallTool(ctx, toolName, map[string]interface{}{"project": "v49-test", "query": "dreamsoft"}); err != nil {
			t.Fatalf("MCP tool %s failed: %v", toolName, err)
		}
	}

	// Test PassiveMemoryInterceptor
	trace, err := mcpSvc.Interceptor().CaptureTrace(ctx, "rae_kernel_execute", map[string]interface{}{"api_key": "secret123"}, map[string]interface{}{"ok": true})
	if err != nil || strings.Contains(trace.Content, "secret123") {
		t.Fatalf("PassiveMemoryInterceptor failed to sanitize secret: %+v", trace)
	}
}

func TestGroup2DeepPhoenixRecipesAndAnalysisPipeline(t *testing.T) {
	eng := phoenix.NewRefactorEngine()
	ctrlOut := eng.ControllerRecipe.Execute("$scope.total = 42;\n$scope.items = [];")
	if !strings.Contains(ctrlOut, "useState") {
		t.Fatalf("expected ControllerToComponentRecipe to generate useState hooks, got: %s", ctrlOut)
	}

	jsxOut := eng.TemplateRecipe.Execute(`<div class="box" ng-if="visible" ng-click="save()"></div>`)
	if !strings.Contains(jsxOut, "className=") || !strings.Contains(jsxOut, "onClick=") {
		t.Fatalf("expected TemplateToJsxRecipe to convert class and ng-click, got: %s", jsxOut)
	}

	routeOut := eng.RoutingRecipe.Execute(`$routeProvider.when('/orders/:id', {})`)
	if !strings.Contains(routeOut, "app/orders/[id]/page.tsx") {
		t.Fatalf("expected RoutingToAppRouterRecipe to map :id to [id], got: %s", routeOut)
	}

	pipe := phoenix.NewAnalysisPipeline()
	rep := pipe.Run("RAE-Suite-go", map[string]string{
		"services/memory/service.go": `import "sync" "net/http"`,
	})
	if rep["status"] != "COMPLETED" {
		t.Fatalf("expected AnalysisPipeline COMPLETED, got: %+v", rep)
	}

	scenLib := phoenix.NewScenarioLibrary()
	scenLib.PublishScenario("sc-1", "Order Checkout Parity", map[string]interface{}{"expected_dom": "#checkout"})
	if len(scenLib.SearchScenarios("Checkout")) != 1 {
		t.Fatal("expected ScenarioLibrary search to find published scenario")
	}
}

func TestGroup3MemoryWorkersAndDeepPipelines(t *testing.T) {
	ctx := context.Background()
	store := memorystore.NewUnifiedStore(nil, nil)
	sdkClient := raesdk.NewMemoryClient("", store)
	_, _ = sdkClient.Store(ctx, "GraphRAG uses community detection for global summaries", "episodic", "proj-w")
	_, _ = sdkClient.Store(ctx, "Information Bottleneck compresses context windows", "episodic", "proj-w")

	maint := workers.NewMemoryMaintenanceWorker(store, 0.95)
	cycleRes := maint.RunCycle(ctx, "proj-w")
	if cycleRes["status"] != "COMPLETED" {
		t.Fatalf("expected maintenance worker cycle COMPLETED, got: %+v", cycleRes)
	}

	cw := workers.NewCommunityWorker()
	assign := cw.RunCommunityDetection([]string{"A", "B", "C"}, [][2]string{{"A", "B"}})
	if assign["A"] != assign["B"] || assign["A"] == assign["C"] {
		t.Fatalf("unexpected community assignment: %+v", assign)
	}

	cons := workers.NewMemoryConsolidationService(store)
	cnt, err := cons.ConsolidateEpisodicToSemantic(ctx, "proj-w")
	if err != nil || cnt < 1 {
		t.Fatalf("expected episodic-to-semantic consolidation >= 1, got %d (%v)", cnt, err)
	}

	searchPipe := workers.NewSemanticSearchPipeline(store)
	res1, err := searchPipe.Search(ctx, "why GraphRAG summaries", "proj-w")
	if err != nil || res1["intent"] != "reflective_causal" {
		t.Fatalf("unexpected SemanticSearchPipeline result: %+v, err=%v", res1, err)
	}
	// Second call hits HybridSearchCache
	_, _ = searchPipe.Search(ctx, "why GraphRAG summaries", "proj-w")
	if searchPipe.Cache.GetHitRate() <= 0 {
		t.Fatal("expected HybridSearchCache hit rate > 0")
	}
}

func TestGroup4AdaptersBanditOrchestratorAndLLMRouter(t *testing.T) {
	ctx := context.Background()
	qvs := vectorstore.NewQdrantVectorStore("", "test_col")
	qvs.EnsureVectorConfig(map[string]int{"dense": 4, "ollama": 4})
	_ = qvs.StoreVector("dense", "m1", []float32{1, 0, 0, 0})
	hits, err := qvs.SearchSimilar("dense", []float32{1, 0, 0, 0}, 5)
	if err != nil || len(hits) != 1 {
		t.Fatalf("QdrantVectorStore SearchSimilar failed: %+v, %v", hits, err)
	}

	sgs := sqlstore.NewSQLiteGraphStore()
	sgs.CreateNode("n1", map[string]interface{}{"label": "Memory"})
	sgs.CreateEdge("n1", "n2", "LINKS_TO")
	if len(sgs.GetNeighbors("n1")) != 1 {
		t.Fatal("expected 1 neighbor in SQLiteGraphStore")
	}

	mab := raemath.NewMultiArmedBandit()
	if mab.DecayExploration(0.9) >= 0.25 {
		t.Fatal("expected exploration rate to decay")
	}
	rt := raemath.NewRAERuntime()
	if rt.Process("test runtime")["selected"] == "" {
		t.Fatal("expected RAERuntime to select arm")
	}

	orch := orchestrator.NewOrchestrator()
	taskRes := orch.RunTask("t-100", "implement feature", "high")
	if taskRes["status"] != "COMPLETED" {
		t.Fatalf("expected Orchestrator RunTask COMPLETED, got %+v", taskRes)
	}

	router := llm.NewLLMRouter()
	chunks, err := router.Stream(ctx, "z-ai/glm-5.3-flash", "hello world")
	if err != nil || len(chunks) == 0 {
		t.Fatalf("expected LLMRouter Stream chunks, got %v, %v", chunks, err)
	}
	batchVecs := router.EmbedBatch(ctx, []string{"one", "two"}, 384)
	if len(batchVecs) != 2 || len(batchVecs[0]) != 384 {
		t.Fatalf("expected EmbedBatch 2x384, got %d", len(batchVecs))
	}
}

func TestGroup5PortalAppsQualityV2AndCIAgents(t *testing.T) {
	p := portal.NewRAESuitePortal()
	up := p.KnowledgeHub.HandleFileUpload("ustawa-1", []byte("Art 1. System RAE zapewnia bezpieczenstwo ISO 42001 oraz ISO 27001."))
	if up["chunk_count"].(int) < 1 {
		t.Fatalf("expected PDKnowledgeHubApp to chunk uploaded document, got %+v", up)
	}
	hits := p.KnowledgeHub.SearchISAPAndWeb("ISO 42001")
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit in SearchISAPAndWeb, got %d", len(hits))
	}

	audit := p.Auditor.TriggerSentinelAudit("clean configuration")
	if audit["passed"] != true {
		t.Fatalf("expected CentralAuditorApp to pass clean config, got %+v", audit)
	}

	gw := quality.NewRAEQualityGatewayV2()
	if !gw.VerifyFix("package main\nfunc ok() {}\n") {
		t.Fatal("expected RAEQualityGatewayV2 VerifyFix to pass safe code")
	}

	fa := raesdk.NewFailureAnalyzer()
	if fa.ClassifyFailure("syntax error: build failed") != "BUILD_OR_COMPILE_ERROR" {
		t.Fatal("unexpected failure classification")
	}
	fixAgent := raesdk.NewCIFixAgent()
	patch := fixAgent.GenerateFix("BUILD_OR_COMPILE_ERROR", "func main() {}")
	fixAgent.SaveFix(patch)
	if fixAgent.GetCostSummary()["fixes_generated"].(int) != 1 {
		t.Fatal("expected CIFixAgent to record 1 generated fix")
	}
	_ = time.Now()
}
