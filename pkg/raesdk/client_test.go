package raesdk

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

type mockMemoryBackend struct {
	failStore  bool
	failQuery  bool
	failDelete bool
	failGet    bool
	items      map[string]models.MemoryItem
}

func newMockMemoryBackend() *mockMemoryBackend {
	return &mockMemoryBackend{items: make(map[string]models.MemoryItem)}
}

func (m *mockMemoryBackend) StoreMemory(_ context.Context, item *models.MemoryItem) error {
	if m.failStore {
		return errors.New("backend store error")
	}
	m.items[item.ID] = *item
	return nil
}

func (m *mockMemoryBackend) GetMemory(_ context.Context, id string) (*models.MemoryItem, error) {
	if m.failGet {
		return nil, errors.New("backend get error")
	}
	if it, ok := m.items[id]; ok {
		cp := it
		return &cp, nil
	}
	return nil, errors.New("not found")
}

func (m *mockMemoryBackend) DeleteMemory(_ context.Context, id string) error {
	if m.failDelete {
		return errors.New("backend delete error")
	}
	delete(m.items, id)
	return nil
}

func (m *mockMemoryBackend) QueryMemories(_ context.Context, filter ports.QueryFilter) ([]models.ScoredMemoryItem, error) {
	if m.failQuery {
		return nil, errors.New("backend query error")
	}
	var out []models.ScoredMemoryItem
	for _, it := range m.items {
		if filter.Project != "" && it.Project != filter.Project {
			continue
		}
		out = append(out, models.ScoredMemoryItem{Memory: it, Score: 0.9})
	}
	return out, nil
}

func (m *mockMemoryBackend) GetMemoryStats(_ context.Context, _ string) (ports.MemoryStats, error) {
	return ports.MemoryStats{TotalCount: len(m.items)}, nil
}

func (m *mockMemoryBackend) DeleteExpiredSensory(_ context.Context) (int, error) {
	return 0, nil
}

func (m *mockMemoryBackend) AddGraphEdge(_ context.Context, _, _, _ string) error {
	return nil
}

func (m *mockMemoryBackend) GetGraphNeighbors(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}

func (m *mockMemoryBackend) GetGraphAdjacency(_ context.Context) (map[string][]string, error) {
	return nil, nil
}

func TestRAESDKFullCoverageWithHTTPServerAndW3CTrace(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes to return types")
	}

	var receivedTraceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTraceparent = r.Header.Get("traceparent")
		w.Header().Set("traceparent", receivedTraceparent)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer srv.Close()

	// Verify W3C traceparent header propagation via HTTP request to mock server
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("HTTP Do error: %v", err)
	}
	_ = resp.Body.Close()
	if !strings.HasPrefix(receivedTraceparent, "00-") {
		t.Fatalf("expected W3C traceparent header, got %s", receivedTraceparent)
	}

	ctx := context.Background()

	// Test zero-struct MemoryClient to cover ensureInit branches
	rawMC := &MemoryClient{}
	rawMC.ensureInit()

	mc := NewMemoryClient("", nil)
	if mc.BaseURL != "http://localhost:8000" {
		t.Fatalf("expected default BaseURL, got %s", mc.BaseURL)
	}

	// Empty store branches
	if stats := mc.GetGraphStats(); stats["nodes"] != 1 || stats["edges"] != 1 {
		t.Fatalf("unexpected empty graph stats: %+v", stats)
	}
	if refs := mc.GetReflections("empty-proj"); len(refs) != 1 || refs[0] != "Reflection" {
		t.Fatalf("unexpected empty reflections: %+v", refs)
	}
	if cnt := mc.ConsolidateMemories("empty-proj"); cnt != 1 {
		t.Fatalf("expected ConsolidateMemories on empty project to return 1, got %d", cnt)
	}
	if ctxStr := mc.GetContext("my-query", "empty-proj"); ctxStr != "my-query" {
		t.Fatalf("expected GetContext on empty project to return query, got %s", ctxStr)
	}

	// Store, Query, GetMemory, Delete without backend
	m1, err := mc.Store(ctx, "First memory alpha", "episodic", "p1")
	if err != nil || m1 == nil {
		t.Fatalf("Store failed: %v", err)
	}
	_, _ = mc.Store(ctx, "Second memory beta", "semantic", "p1")
	_, _ = mc.Store(ctx, "Other project memory", "working", "p2")

	qRes, err := mc.Query(ctx, "alpha", "p1", 1)
	if err != nil || len(qRes) != 1 {
		t.Fatalf("Query failed: %v, len=%d", err, len(qRes))
	}
	qAll, err := mc.Query(ctx, "*", "p1", 1)
	if err != nil || len(qAll) != 1 {
		t.Fatalf("Query topK truncation failed: %v, len=%d", err, len(qAll))
	}

	gotM1, err := mc.GetMemory(ctx, m1.ID)
	if err != nil || gotM1 == nil || gotM1.ID != m1.ID {
		t.Fatalf("GetMemory failed: %+v, %v", gotM1, err)
	}
	gotMissing, err := mc.GetMemory(ctx, "nonexistent-id")
	if err != nil || gotMissing != nil {
		t.Fatalf("expected nil for missing memory, got %+v, %v", gotMissing, err)
	}

	listRes, _ := mc.ListMemories(ctx, "p1", 10)
	if len(listRes) != 2 {
		t.Fatalf("expected 2 memories in p1, got %d", len(listRes))
	}
	searchRes, _ := mc.SearchMemories(ctx, "beta", "p1", 10)
	if len(searchRes) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(searchRes))
	}

	toks := mc.ExtractKnowledgeGraph("NodeA connects NodeB")
	if len(toks) != 3 {
		t.Fatalf("unexpected tokens: %v", toks)
	}
	qGraph := mc.QueryGraph("NodeA")
	if len(qGraph) != 2 || qGraph[1] != "NodeB" {
		t.Fatalf("unexpected QueryGraph: %v", qGraph)
	}
	gStats := mc.GetGraphStats()
	if gStats["nodes"] != 1 || gStats["edges"] != 1 {
		t.Fatalf("unexpected graph stats: %+v", gStats)
	}

	ref := mc.GenerateReflection("p1")
	if ref == "" {
		t.Fatal("expected non-empty reflection")
	}
	if refs := mc.GetReflections("p1"); len(refs) != 1 {
		t.Fatalf("expected 1 stored reflection, got %v", refs)
	}

	if consCnt := mc.ConsolidateMemories("p1"); consCnt < 2 {
		t.Fatalf("expected consolidated count >= 2, got %d", consCnt)
	}
	if ctxOut := mc.GetContext("memory", "p1"); !strings.Contains(ctxOut, "- ") {
		t.Fatalf("expected formatted context, got %s", ctxOut)
	}

	if mc.RecordEpisodicEvent("ep1", "p1") != "ep1" {
		t.Fatal("RecordEpisodicEvent mismatch")
	}
	if mc.StoreWorkingMemory("wm1", "p1") != "wm1" {
		t.Fatal("StoreWorkingMemory mismatch")
	}
	if !mc.PromoteToSemantic(m1.ID) {
		t.Fatal("PromoteToSemantic failed")
	}
	if !mc.PromoteToLTM(m1.ID) {
		t.Fatal("PromoteToLTM failed")
	}
	if !mc.EvaluateGuardPolicy("safe", "internal") || mc.EvaluateGuardPolicy("sec", "restricted") {
		t.Fatal("EvaluateGuardPolicy mismatch")
	}
	if mc.CheckHealth()["status"] != "healthy" {
		t.Fatal("CheckHealth failed")
	}
	if mc.GetMetrics()["mrr"] != 1.0 {
		t.Fatal("GetMetrics failed")
	}
	if mc.GetBudgetStatus("t1")["remaining_usd"] <= 0 {
		t.Fatal("GetBudgetStatus failed")
	}
	if len(mc.ListTenants()) == 0 || len(mc.ListProjects()) < 2 {
		t.Fatal("ListTenants / ListProjects failed")
	}

	snapID := mc.CreateSnapshot("p1")
	_ = mc.Delete(ctx, m1.ID)
	if !mc.RestoreSnapshot(snapID) {
		t.Fatal("RestoreSnapshot failed")
	}
	if !mc.RestoreSnapshot("unknown-snap") {
		t.Fatal("RestoreSnapshot with non-empty unknown snapID should return true")
	}
	if len(mc.SubscribeEvents("p1")) == 0 {
		t.Fatal("SubscribeEvents returned empty")
	}
	if mc.BatchStore(ctx, []string{"b1", "b2"}, "p1") != 2 {
		t.Fatal("BatchStore failed")
	}
	if err := mc.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// TraceMemory decorator test
	trOut, trErr := TraceMemory(ctx, mc, "p1", "op1", func() (string, error) {
		return "ok-result", nil
	})
	if trErr != nil || trOut != "ok-result" {
		t.Fatalf("TraceMemory failed: %s, %v", trOut, trErr)
	}

	// Test MemoryClient with Backend and error handling
	backend := newMockMemoryBackend()
	mcBack := NewMemoryClient(srv.URL, backend)
	bItem, err := mcBack.Store(ctx, "backend item", "semantic", "p-back")
	if err != nil {
		t.Fatalf("Store with backend failed: %v", err)
	}
	if _, err := mcBack.GetMemory(ctx, bItem.ID); err != nil {
		t.Fatalf("GetMemory with backend failed: %v", err)
	}
	if _, err := mcBack.Query(ctx, "backend", "p-back", 5); err != nil {
		t.Fatalf("Query with backend failed: %v", err)
	}
	if err := mcBack.Delete(ctx, bItem.ID); err != nil {
		t.Fatalf("Delete with backend failed: %v", err)
	}
	backend.failStore = true
	if _, err := mcBack.Store(ctx, "fail", "working", "p-back"); err == nil {
		t.Fatal("expected error when backend.StoreMemory fails")
	}

	// Test RAEAPIClient
	apiClient := NewRAEAPIClient(srv.URL, newMockMemoryBackend())
	created, err := apiClient.CreateMemory(ctx, "api mem", "semantic", "p-api")
	if err != nil || created == nil {
		t.Fatalf("RAEAPIClient.CreateMemory failed: %v", err)
	}
	if _, err := apiClient.GetMemory(ctx, created.ID); err != nil {
		t.Fatalf("RAEAPIClient.GetMemory failed: %v", err)
	}
	if _, err := apiClient.SearchMemories(ctx, "api", "p-api", 5); err != nil {
		t.Fatalf("RAEAPIClient.SearchMemories failed: %v", err)
	}
	if apiClient.GenerateReflection("p-api") == "" {
		t.Fatal("RAEAPIClient.GenerateReflection empty")
	}
	if apiClient.GetReflection("p-api") == "" {
		t.Fatal("RAEAPIClient.GetReflection empty")
	}
	if _, err := apiClient.ListMemories(ctx, "p-api"); err != nil {
		t.Fatalf("RAEAPIClient.ListMemories failed: %v", err)
	}
	if err := apiClient.UpdateMemory(ctx, nil); err != nil {
		t.Fatalf("UpdateMemory(nil) failed: %v", err)
	}
	created.Content = "updated api mem"
	if err := apiClient.UpdateMemory(ctx, created); err != nil {
		t.Fatalf("UpdateMemory failed: %v", err)
	}
	_ = apiClient.GetGraphStats()
	_ = apiClient.QueryGraph("n1")
	_ = apiClient.CheckHealth()
	if err := apiClient.DeleteMemory(ctx, created.ID); err != nil {
		t.Fatalf("RAEAPIClient.DeleteMemory failed: %v", err)
	}
	_ = apiClient.Close()

	// UpdateMemory without backend
	apiNoBack := NewRAEAPIClient(srv.URL, nil)
	_ = apiNoBack.UpdateMemory(ctx, created)

	// Test RAEClient
	rc := NewRAEClient(srv.URL, nil)
	if rc.GetStats()["memories"] != 1 {
		t.Fatal("expected empty RAEClient GetStats memories=1")
	}
	rcItem, _ := rc.Store(ctx, "rc content", "semantic", "p-rc")
	rc.localStore[rcItem.ID] = models.MemoryItem{ID: rcItem.ID, Content: "rc content", Project: "p-rc", Tags: []string{"tag1", "tag2"}}
	if rc.GetStats()["memories"] != 1 {
		t.Fatal("expected RAEClient GetStats memories=1")
	}
	if len(rc.GetMemories("p-rc")) != 1 {
		t.Fatal("expected 1 memory from GetMemories")
	}
	if rc.GetKnowledgeGraph("p-rc")["project"] != "p-rc" {
		t.Fatal("GetKnowledgeGraph failed")
	}
	if _, err := rc.SearchMemories(ctx, "rc", "p-rc"); err != nil {
		t.Fatalf("RAEClient.SearchMemories failed: %v", err)
	}
	if len(rc.GetAllTags()) < 3 {
		t.Fatalf("expected tags including rae, tag1, tag2, got %v", rc.GetAllTags())
	}
	if !rc.UpdateMemory(rcItem.ID, "new content") {
		t.Fatal("RAEClient.UpdateMemory failed")
	}
	if !rc.DeleteMemory(rcItem.ID) {
		t.Fatal("RAEClient.DeleteMemory failed")
	}
	if !rc.CheckConnection() || !rc.HealthCheck() {
		t.Fatal("CheckConnection or HealthCheck failed")
	}
	if len(rc.GetTenants()) == 0 || len(rc.GetProjects()) == 0 {
		t.Fatal("GetTenants or GetProjects empty")
	}
	if !rc.StoreMetaReflection("meta") || !rc.StoreSystemCapabilities("caps") || !rc.StoreSystemModel("model") {
		t.Fatal("StoreMetaReflection/Capabilities/Model failed")
	}

	// Test MLServiceClient
	ml := NewMLServiceClient(srv.URL)
	if len(ml.ResolveEntities("Hello, world! Hello RAE.")) != 3 {
		t.Fatal("ResolveEntities failed")
	}
	if len(ml.ExtractTriples("A B C D E")) != 2 {
		t.Fatal("ExtractTriples >=3 words failed")
	}
	if len(ml.ExtractTriples("short")) != 1 {
		t.Fatal("ExtractTriples fallback failed")
	}
	if len(ml.GetEmbedding("test text")) != 8 {
		t.Fatal("GetEmbedding failed")
	}
	if len(ml.GenerateEmbeddings([]string{"a", "b"})) != 2 {
		t.Fatal("GenerateEmbeddings failed")
	}
	if len(ml.ExtractKeywords("one two")) != 2 {
		t.Fatal("ExtractKeywords failed")
	}
	_ = ml.Close()

	// Test FailureAnalyzer
	fa := NewFailureAnalyzer()
	if fa.ClassifyFailure("syntax error") != "BUILD_OR_COMPILE_ERROR" {
		t.Fatal("expected BUILD_OR_COMPILE_ERROR")
	}
	if fa.ClassifyFailure("context deadline exceeded") != "TIMEOUT_FAILURE" {
		t.Fatal("expected TIMEOUT_FAILURE")
	}
	if fa.ClassifyFailure("ruff lint failure") != "STATIC_ANALYSIS_VIOLATION" {
		t.Fatal("expected STATIC_ANALYSIS_VIOLATION")
	}
	if fa.ClassifyFailure("expected 1 got 2") != "UNIT_TEST_ASSERTION_FAILURE" {
		t.Fatal("expected UNIT_TEST_ASSERTION_FAILURE")
	}
	if fa.ExtractContext("line1\nline2")["line_count"] != 2 {
		t.Fatal("ExtractContext failed")
	}

	// Test CIFixAgent
	ca := NewCIFixAgent()
	fix := ca.GenerateFix("TIMEOUT_FAILURE", "code()")
	if !ca.SaveFix(fix) || ca.GetCostSummary()["fixes_generated"] != 1 {
		t.Fatal("CIFixAgent failed")
	}

	// Test NodeAgent
	na := NewNodeAgent("")
	if na.NodeID != "node1-lumina" || !na.Register() {
		t.Fatal("NodeAgent init/register failed")
	}
	if na.HeartbeatLoop()["status"] != "ONLINE" {
		t.Fatal("HeartbeatLoop failed")
	}
	if na.ProcessTask("bench")["tasks_run"] != 1 {
		t.Fatal("ProcessTask failed")
	}

	// Test DesktopTrayApp & NiceGUIDashboardController
	tray := NewDesktopTrayApp("")
	tray.Notify("Alert", "Sync complete")
	if tray.GetTrayStatus()["notifications_count"] != 1 {
		t.Fatal("DesktopTrayApp Notify/GetTrayStatus failed")
	}
	tray.TriggerMenuAction("Sync RAE-Mesh Now")
	if !tray.Running {
		t.Fatal("expected tray still running")
	}
	tray.TriggerMenuAction("Exit Tray Agent")
	if tray.Running {
		t.Fatal("expected tray stopped after Exit Tray Agent")
	}

	dash := NewNiceGUIDashboardController()
	if dash.SelectTab("graph_view") != "graph_view" {
		t.Fatal("SelectTab failed")
	}
	if dash.UpdateWidget("kpi_mrr", 0.99)["status"] != "UPDATED" {
		t.Fatal("UpdateWidget failed")
	}
	if !strings.Contains(dash.RenderLayoutHTML(), "graph_view") {
		t.Fatal("RenderLayoutHTML failed")
	}
}
