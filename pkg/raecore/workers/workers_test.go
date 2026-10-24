package workers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

type mockWorkerStore struct {
	failQuery  bool
	failStore  bool
	failDelete bool
	items      map[string]models.MemoryItem
}

func newMockWorkerStore() *mockWorkerStore {
	return &mockWorkerStore{items: make(map[string]models.MemoryItem)}
}

func (m *mockWorkerStore) StoreMemory(ctx context.Context, item *models.MemoryItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.failStore {
		return errors.New("store error")
	}
	if item.ID == "" {
		item.ID = models.GenerateUUIDv7()
	}
	m.items[item.ID] = *item
	return nil
}

func (m *mockWorkerStore) GetMemory(ctx context.Context, id string) (*models.MemoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if it, ok := m.items[id]; ok {
		cp := it
		return &cp, nil
	}
	return nil, errors.New("not found")
}

func (m *mockWorkerStore) DeleteMemory(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.failDelete {
		return errors.New("delete error")
	}
	delete(m.items, id)
	return nil
}

func (m *mockWorkerStore) QueryMemories(ctx context.Context, filter ports.QueryFilter) ([]models.ScoredMemoryItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m.failQuery {
		return nil, errors.New("query error")
	}
	var out []models.ScoredMemoryItem
	for _, it := range m.items {
		if filter.Project != "" && it.Project != filter.Project {
			continue
		}
		out = append(out, models.ScoredMemoryItem{Memory: it, Score: it.Importance})
	}
	return out, nil
}

func (m *mockWorkerStore) GetMemoryStats(_ context.Context, _ string) (ports.MemoryStats, error) {
	return ports.MemoryStats{TotalCount: len(m.items)}, nil
}

func (m *mockWorkerStore) DeleteExpiredSensory(_ context.Context) (int, error) {
	return 2, nil
}

func (m *mockWorkerStore) AddGraphEdge(_ context.Context, _, _, _ string) error { return nil }
func (m *mockWorkerStore) GetGraphNeighbors(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockWorkerStore) GetGraphAdjacency(_ context.Context) (map[string][]string, error) {
	return nil, nil
}

func TestGraphAlgorithmsAndRepositories(t *testing.T) {
	repo := &GraphRepository{}
	n := repo.StoreGraphTriples([][3]string{
		{"A", "CALLS", "B"},
		{"B", "CALLS", "C"},
		{"A", "DEPENDS_ON", "C"},
	})
	if n != 3 {
		t.Fatalf("expected 3 triples stored, got %d", n)
	}

	bfs := repo.TraverseGraphBFS("A", 3)
	if len(bfs) != 3 {
		t.Fatalf("expected BFS to visit 3 nodes, got %v", bfs)
	}
	dfs := repo.TraverseGraphDFS("A", 3)
	if len(dfs) != 3 {
		t.Fatalf("expected DFS to visit 3 nodes, got %v", dfs)
	}

	alg := &GraphAlgorithmsService{Damping: 0.85}
	alg.AddEdge("A", "B")
	alg.AddEdge("B", "C")
	alg.AddEdge("A", "C")

	pr := alg.PageRank([]string{"A", "B", "C"})
	if pr["C"] <= pr["A"] {
		t.Fatalf("expected sink node C to have higher PageRank than A, got %v", pr)
	}
	sp := alg.ShortestPath("A", "C")
	if len(sp) != 2 || sp[0] != "A" || sp[1] != "C" {
		t.Fatalf("expected shortest path [A C], got %v", sp)
	}
	allPaths := alg.FindAllPaths("A", "C")
	if len(allPaths) < 2 {
		t.Fatalf("expected at least 2 paths from A to C, got %v", allPaths)
	}

	core := &RAECoreService{}
	core.EnableSzubarMode(true)
	stats := core.GetStatistics()
	if stats["szubar_mode"] != true {
		t.Fatalf("expected szubar_mode=true")
	}
	_ = core.DeleteMemory(context.Background(), "nonexistent")
}

func TestAllWorkersPipelinesDelegatesAndContextCancellation(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes to return non-empty slice")
	}

	ctx := context.Background()
	canceledCtx, cancel := context.WithCancel(ctx)
	cancel()

	// 1. MemoryMaintenanceWorker (nil store, populated store, canceled ctx)
	mwNil := NewMemoryMaintenanceWorker(nil, 0)
	if mwNil.decayFactor != 0.96 {
		t.Fatalf("expected default decayFactor 0.96, got %f", mwNil.decayFactor)
	}
	_, _ = mwNil.ApplyDecay(ctx, "p1")
	_, _ = mwNil.PruneExpiredTTL(ctx, "p1", 0)
	_, _ = mwNil.RunDreamConsolidation(ctx, "p1")
	if mwNil.SummarizeEpisodicBatch(nil) != "" {
		t.Fatal("expected empty summary for nil batch")
	}
	if !strings.Contains(mwNil.SummarizeEpisodicBatch([]string{"e1", "e2"}), "Consolidated 2") {
		t.Fatal("unexpected SummarizeEpisodicBatch output")
	}

	store := newMockWorkerStore()
	mw := NewMemoryMaintenanceWorker(store, 0.9)
	// Empty dream consolidation
	if d, err := mw.RunDreamConsolidation(ctx, "p1"); err != nil || d != 0 {
		t.Fatalf("expected 0 dreams on empty store, got %d, %v", d, err)
	}

	// Populate 6 memories (including old low-importance one for pruning)
	oldTime := time.Now().UTC().Add(-60 * 24 * time.Hour)
	for i := 0; i < 6; i++ {
		imp := 0.8
		created := time.Now().UTC()
		if i == 0 {
			imp = 0.10
			created = oldTime
		}
		_ = store.StoreMemory(ctx, &models.MemoryItem{
			ID:         models.GenerateUUIDv7(),
			Content:    "episodic event content",
			Layer:      models.LayerEpisodic,
			Project:    "p1",
			Importance: imp,
			CreatedAt:  created,
		})
	}
	cycle := mw.RunCycle(ctx, "p1")
	if cycle["status"] != "COMPLETED" || cycle["pruned_expired"] != 1 || cycle["dreams_generated"] != 1 {
		t.Fatalf("unexpected RunCycle result: %+v", cycle)
	}
	if mw.GetStats()["cycles_run"] != 1 {
		t.Fatalf("unexpected GetStats: %+v", mw.GetStats())
	}
	_, _ = mw.PruneExpiredTTL(ctx, "p1", 0)

	// Canceled context / error branches
	if _, err := mw.ApplyDecay(canceledCtx, "p1"); err == nil {
		t.Fatal("expected ApplyDecay error on canceled context")
	}
	if _, err := mw.PruneExpiredTTL(canceledCtx, "p1", time.Hour); err == nil {
		t.Fatal("expected PruneExpiredTTL error on canceled context")
	}
	if _, err := mw.RunDreamConsolidation(canceledCtx, "p1"); err == nil {
		t.Fatal("expected RunDreamConsolidation error on canceled context")
	}
	store.failStore = true
	_, _ = mw.RunDreamConsolidation(ctx, "p1")
	store.failStore = false

	// 2. CommunityWorker
	cw := NewCommunityWorker()
	comm := cw.RunCommunityDetection([]string{"N1", "N2", "N3"}, [][2]string{{"N2", "N1"}, {"N1", "N3"}, {"X", "Y"}})
	hier := cw.BuildTopicHierarchies(comm)
	sums := cw.SummarizeCommunities(hier)
	if len(sums) == 0 {
		t.Fatal("expected community summaries")
	}

	// 3. MemoryConsolidationService
	mcsNil := NewMemoryConsolidationService(nil)
	_, _ = mcsNil.ConsolidateEpisodicToSemantic(ctx, "p1")
	if mcsNil.EvaluateConsolidationQuality(nil, models.MemoryItem{}) != 0.0 {
		t.Fatal("expected 0 quality for empty cluster")
	}
	mcs := NewMemoryConsolidationService(store)
	_ = store.StoreMemory(ctx, &models.MemoryItem{
		ID:         "m-tagged",
		Content:    "tagged memory",
		Layer:      models.LayerEpisodic,
		InfoClass:  models.InfoInternal,
		Project:    "p1",
		Importance: 0.9,
		Tags:       []string{"arch"},
	})
	cCount, err := mcs.ConsolidateEpisodicToSemantic(ctx, "p1")
	if err != nil || cCount == 0 {
		t.Fatalf("ConsolidateEpisodicToSemantic failed: %d, %v", cCount, err)
	}
	if mcs.EvaluateConsolidationQuality([]models.MemoryItem{{Content: "a"}}, models.MemoryItem{Content: "merged"}) <= 0 {
		t.Fatal("expected positive consolidation quality")
	}
	_, _ = mcs.ConsolidateEpisodicToSemantic(canceledCtx, "p1")

	// 4. ReflectionPipeline
	rp := NewReflectionPipeline(store)
	if rp.ClusterEvents(nil) != nil {
		t.Fatal("expected nil clusters for empty events")
	}
	refs, err := rp.GenerateReflections(ctx, []string{"ev1", "ev2"}, "p1")
	if err != nil || len(refs) != 1 {
		t.Fatalf("GenerateReflections failed: %v", err)
	}
	if _, err := rp.GenerateReflections(canceledCtx, []string{"ev1"}, "p1"); err == nil {
		t.Fatal("expected GenerateReflections error on canceled context")
	}

	// 5. ImportanceScoringService
	iss := NewImportanceScoringService()
	if iss.ScoreRecency(0) != 1.0 || iss.ScoreRecency(10) >= 1.0 {
		t.Fatal("ScoreRecency failed")
	}
	batchScores := iss.RecalculateBatch([]models.MemoryItem{
		{Content: "one two three", Tags: []string{"t1"}},
		{Content: "four five six seven", Tags: []string{"t1", "t2"}},
	})
	if len(batchScores) != 2 {
		t.Fatal("RecalculateBatch failed")
	}

	// 6. ContextBuilder
	cb := NewContextBuilder(store)
	if cb.InjectReflectionsIntoPrompt("base", nil) != "base" {
		t.Fatal("InjectReflectionsIntoPrompt empty failed")
	}
	if !strings.Contains(cb.InjectReflectionsIntoPrompt("base", []string{"r1"}), "[Reflections]") {
		t.Fatal("InjectReflectionsIntoPrompt non-empty failed")
	}
	ctxMap, err := cb.BuildContext(ctx, "*", "p1", 0)
	if err != nil || ctxMap["memories_count"].(int) == 0 {
		t.Fatalf("BuildContext failed: %+v, %v", ctxMap, err)
	}
	// Small maxTokens to trigger token budget break
	_, _ = cb.BuildContext(ctx, "*", "p1", 1)
	if _, err := cb.BuildContext(canceledCtx, "*", "p1", 100); err == nil {
		t.Fatal("expected BuildContext error on canceled context")
	}

	// 7. SemanticExtractor
	se := NewSemanticExtractor()
	triples := se.ExtractFromMemories([]string{"Agent-One executes Task-Two now", "short"})
	if len(triples) != 1 || triples[0]["subject"] != "agent_one" {
		t.Fatalf("ExtractFromMemories failed: %+v", triples)
	}

	// 8. QueryAnalyzer
	qa := NewQueryAnalyzer()
	for _, q := range []string{"why did this happen", "when was recent history", "graph relation connected", "factual lookup"} {
		exp := qa.ExplainAnalysis(q)
		if exp["intent"] == "" {
			t.Fatalf("ExplainAnalysis empty intent for %s", q)
		}
	}
	if len(qa.GetAvailableProfiles()) != 4 {
		t.Fatal("expected 4 profiles")
	}

	// 9. HybridSearchCache
	hc := NewHybridSearchCache()
	if hc.GetHitRate() != 0.0 {
		t.Fatal("expected 0 hit rate initially")
	}
	hc.Set("k1", "v1", 0)
	if v, ok := hc.Get("k1"); !ok || v != "v1" {
		t.Fatal("expected cache hit for k1")
	}
	hc.Invalidate("k1")
	if _, ok := hc.Get("k1"); ok {
		t.Fatal("expected cache miss after Invalidate")
	}
	hc.entries["expired"] = cacheItem{val: "old", expiresAt: time.Now().Add(-time.Minute)}
	if hc.ClearExpired() != 1 {
		t.Fatal("expected 1 expired item cleared")
	}
	if hc.GetHitRate() <= 0 {
		t.Fatal("expected positive hit rate")
	}

	// 10. TriggerRepository
	tr := NewTriggerRepository()
	trig := tr.CreateTrigger("", "MEMORY_CREATED", "consolidate")
	tid := trig["id"].(string)
	if _, ok := tr.GetTrigger(tid); !ok {
		t.Fatal("GetTrigger failed")
	}
	if len(tr.ListTriggers()) != 1 {
		t.Fatal("ListTriggers failed")
	}
	if !tr.UpdateTrigger(tid, false) || tr.UpdateTrigger("missing", false) {
		t.Fatal("UpdateTrigger failed")
	}
	if !tr.DeleteTrigger(tid) || tr.DeleteTrigger(tid) {
		t.Fatal("DeleteTrigger failed")
	}

	// 11. DashboardWebSocketService
	ws := NewDashboardWebSocketService()
	ws.Subscribe("c1")
	ws.Subscribe("c2")
	if ws.GetActiveSubscribers() != 2 {
		t.Fatal("expected 2 subscribers")
	}
	if ws.BroadcastEvent("UPDATE", map[string]interface{}{"ok": true}) != 2 {
		t.Fatal("BroadcastEvent failed")
	}
	ws.Unsubscribe("c1")
	if ws.GetActiveSubscribers() != 1 {
		t.Fatal("expected 1 subscriber after Unsubscribe")
	}

	// 12. MetricsTracker & ActionExecutor
	ae := NewActionExecutor(nil)
	if !ae.CanExecute("act1") || ae.CanExecute("") {
		t.Fatal("CanExecute failed")
	}
	ae.Execute("act1")
	_ = ae.GetLastTransition()
	_ = ae.GetTransitionHistory()
	_ = ae.GetMetricsSummary()
	mt := ae.tracker
	_ = mt.RecordInformationBottleneck(0.5, 0.9)
	_ = mt.RecordGraphState(10, 20)
	_ = mt.GetBestActions()
	_ = mt.GetWorstActions()

	// 13. GraphUpdateOperator
	guo := &GraphUpdateOperator{}
	scores := guo.Apply([]string{"n1", "n2"}, 0)
	if len(scores) != 2 {
		t.Fatal("GraphUpdateOperator.Apply failed")
	}
	if !guo.AnalyzeConvergence(nil) || !guo.AnalyzeConvergence([]float64{1e-6}) || guo.AnalyzeConvergence([]float64{0.5}) {
		t.Fatal("AnalyzeConvergence failed")
	}

	// 14. SemanticSearchPipeline
	ssp := NewSemanticSearchPipeline(store)
	if _, err := ssp.Search(ctx, "factual query", "p1"); err != nil {
		t.Fatalf("SemanticSearchPipeline.Search failed: %v", err)
	}
	if _, err := ssp.Search(ctx, "factual query", "p1"); err != nil {
		t.Fatalf("SemanticSearchPipeline cached Search failed: %v", err)
	}
	if _, err := ssp.Search(canceledCtx, "uncached query", "p1"); err == nil {
		t.Fatal("expected error on canceled context")
	}

	// 15. HumanApprovalService
	has := NewHumanApprovalService()
	if has.RequestApproval("req-1") != "PENDING" || has.Approve("req-1") != "APPROVED" {
		t.Fatal("HumanApprovalService failed")
	}

	// 16. MemoryIngestionDelegate
	mid := &MemoryIngestionDelegate{Store: store}
	if mid.EnforceSecurityPolicy("restricted", "semantic") || !mid.EnforceSecurityPolicy("restricted", "working") {
		t.Fatal("EnforceSecurityPolicy failed")
	}
	if mid.DetectAgenticPatterns("meta reflection text")[0] != "meta_reflection" {
		t.Fatal("DetectAgenticPatterns reflection failed")
	}
	if mid.DetectAgenticPatterns("normal text")[0] != "standard_observation" {
		t.Fatal("DetectAgenticPatterns standard failed")
	}
	_ = mid.HandleHardFrames("frame")
	if mid.ResolveProjectContext("") != "RAE-Suite" || mid.ResolveProjectContext("custom") != "custom" {
		t.Fatal("ResolveProjectContext failed")
	}
	if !mid.ContainsCode("func test() {}") || mid.ContainsCode("plain text") {
		t.Fatal("ContainsCode failed")
	}
	_ = mid.StoreMemory(ctx, &models.MemoryItem{ID: "mid-1", Content: "c", Project: "p1"})
	midNil := &MemoryIngestionDelegate{}
	_ = midNil.StoreMemory(ctx, &models.MemoryItem{ID: "mid-2", Content: "c"})

	// 17. MemoryLifecycleDelegate
	mldNil := &MemoryLifecycleDelegate{}
	_ = mldNil.DeleteMemory(ctx, "x")
	_ = mldNil.ClearMemories(ctx, "p1")
	_ = mldNil.AdjustImportance("x", 0.2)
	_ = mldNil.DecayImportance(0)
	_ = mldNil.DeleteExpiredMemories()
	_ = mldNil.DeleteOldEpisodicMemories(0)
	_ = mldNil.ConsolidateMemories("empty")

	mld := &MemoryLifecycleDelegate{Store: store}
	_ = mld.UpdateMemoryAccessBatch([]string{"mid-1"})
	if mld.GetMetricAggregate("mrr") != 1.0 || mld.GetMetricAggregate("unknown") != 1.0 {
		t.Fatal("GetMetricAggregate failed")
	}
	_ = mld.AdjustImportance("mid-1", 0.1)
	_ = mld.ApplyGlobalMemoryDecay(0)
	_ = mld.DeleteExpiredMemories()
	_ = store.StoreMemory(ctx, &models.MemoryItem{
		ID:        "old-ep",
		Content:   "old",
		Layer:     models.LayerEpisodic,
		Project:   "p1",
		CreatedAt: time.Now().UTC().Add(-40 * 24 * time.Hour),
	})
	if mld.DeleteOldEpisodicMemories(0) == 0 {
		t.Fatal("expected DeleteOldEpisodicMemories to delete old-ep")
	}
	_ = mld.GetStatistics()
	_ = mld.ConsolidateMemories("p1")
	_ = mld.GenerateReflections("p1")
	_ = mld.DeleteMemory(ctx, "mid-1")
	_ = mld.ClearMemories(ctx, "p1")

	// 18. EnhancedGraphRepository
	egr := &EnhancedGraphRepository{}
	_ = egr.GetStatistics()
	egr.CreateNode("en1", map[string]interface{}{"type": "service"})
	_ = egr.GetNodeByID("en1")
	_ = egr.GetNodeByNodeID("missing")
	if egr.CreateEdge("", "en2", "REL") {
		t.Fatal("expected CreateEdge with empty src to return false")
	}
	egr.CreateEdge("en1", "en2", "REL")
	egr.CreateEdge("en2", "en3", "REL")
	_ = egr.GetNodeMetrics("en1")
	if len(egr.FindConnectedNodes("en1")) != 3 {
		t.Fatal("FindConnectedNodes failed")
	}
	if len(egr.GetNeighbors("en1")) != 1 {
		t.Fatal("GetNeighbors failed")
	}
	if len(egr.GetSubgraph([]string{"en1"})) != 2 {
		t.Fatal("GetSubgraph failed")
	}
	_ = egr.GetStatistics()
	egr.DeleteNode("en1")

	// 19. GraphRepository extra methods
	gr := &GraphRepository{}
	_ = gr.GetSubgraphStatistics()
	_ = gr.FindRelevantCommunities("nonexistent")
	gr.CreateKnowledgeNode("AuthService")
	gr.StoreGraphTriples([][3]string{{"AuthService", "USES", "TokenDB"}})
	_ = gr.TraverseGraphBFS("AuthService", 0)
	_ = gr.TraverseGraphDFS("AuthService", 0)
	if gr.GetEdgesBetweenNodes("AuthService", "TokenDB")[0] != "USES" {
		t.Fatal("GetEdgesBetweenNodes failed")
	}
	if gr.GetEdgesBetweenNodes("X", "Y")[0] != "RELATED_TO" {
		t.Fatal("GetEdgesBetweenNodes fallback failed")
	}
	_ = gr.GetNodeByID("AuthService")
	_ = gr.GetNodeByID("Missing")
	_ = gr.GetNodesByIDs([]string{"AuthService", ""})
	_ = gr.FindRelevantCommunities("auth")
	_ = gr.GetSubgraphStatistics()

	// 20. GraphAlgorithmsService extra methods
	gas := &GraphAlgorithmsService{}
	_ = gas.PageRank(nil)
	gas.AddEdge("N1", "N2")
	_ = gas.LoadTenantGraph("t1")
	_ = gas.CommunityDetection([]string{"N1", "N2"})
	_ = gas.ShortestPath("N1", "N1")
	_ = gas.ShortestPath("N1", "Unreachable")
	_ = gas.FindAllPaths("N1", "Unreachable")
	_ = gas.FindRelatedEntities("N1")

	// 21. RAECoreService full coverage
	core := &RAECoreService{}
	if core.ExecuteAction(" run ") != "executed:run" {
		t.Fatal("ExecuteAction failed")
	}
	if err := core.StoreMemory(ctx, nil); err == nil {
		t.Fatal("expected error for nil memory")
	}
	if err := core.StoreMemory(ctx, &models.MemoryItem{Content: "sec", InfoClass: models.InfoRestricted, Layer: models.LayerSemantic}); err == nil {
		t.Fatal("expected ISO 27000 violation error")
	}
	memItem := &models.MemoryItem{Content: "valid core memory", InfoClass: models.InfoInternal, Layer: models.LayerWorking, Project: "p-core"}
	if err := core.StoreMemory(ctx, memItem); err != nil {
		t.Fatalf("RAECoreService.StoreMemory failed: %v", err)
	}
	if got, err := core.GetMemory(ctx, memItem.ID); err != nil || got == nil {
		t.Fatal("RAECoreService.GetMemory failed")
	}
	if got, _ := core.GetMemory(ctx, "missing"); got != nil {
		t.Fatal("expected nil for missing memory")
	}
	if res, err := core.QueryMemories(ctx, "valid"); err != nil || len(res) != 1 {
		t.Fatal("RAECoreService.QueryMemories failed")
	}
	_ = core.UpdateMemory(ctx, memItem.ID)
	if core.CountMemories("p-core") != 1 {
		t.Fatal("CountMemories failed")
	}
	_ = core.ConsolidateMemories("p-core")
	_ = core.GenerateReflections("p-core")
	if !core.VerifyConsistency() {
		t.Fatal("VerifyConsistency failed")
	}
	_ = core.ExtractKnowledgeGraph("one two")
	_ = core.ExtractKnowledgeGraph("   ")
	_ = core.DecayMemories(0.95)
	_ = core.PruneExpired()
	_ = core.DeleteMemory(ctx, memItem.ID)

	// RAECoreService with Store backing
	coreBacked := &RAECoreService{Store: newMockWorkerStore()}
	_ = coreBacked.StoreMemory(ctx, &models.MemoryItem{ID: "cb-1", Content: "backed", Layer: models.LayerWorking, Project: "p-cb"})
	_, _ = coreBacked.GetMemory(ctx, "cb-1")
	_, _ = coreBacked.QueryMemories(ctx, "backed")
	_ = coreBacked.ListMemories("p-cb")
	_ = coreBacked.DeleteMemory(ctx, "cb-1")

	// 22. MetricsRepository
	mRepo := &MetricsRepository{}
	_ = mRepo.GetAvailableMetrics()
	_ = mRepo.GetLatestMetricValue("none")
	_ = mRepo.GetMetricStatistics("none")
	mRepo.RecordMetricsBatch(map[string]float64{"mrr": 0.95, "latency": 12.0})
	mRepo.RecordMetric("mrr", 1.0)
	if mRepo.GetLatestMetricValue("mrr") != 1.0 {
		t.Fatal("GetLatestMetricValue failed")
	}
	if mRepo.GetMetricStatistics("mrr")["count"] != 2 {
		t.Fatal("GetMetricStatistics failed")
	}
	if len(mRepo.GetAvailableMetrics()) != 2 {
		t.Fatal("GetAvailableMetrics failed")
	}

	// 23. BenchmarkRepository & RewardFunction
	bRepo := &BenchmarkRepository{}
	_ = bRepo.ListSuites()
	_ = bRepo.ListExecutions("s1")
	_ = bRepo.GetSuite("missing")
	_ = bRepo.GetExecution("missing")
	bRepo.CreateSuite("s1", "Academic Lite")
	if bRepo.GetSuite("s1") != "Academic Lite" {
		t.Fatal("GetSuite failed")
	}
	bRepo.UpdateSuiteStatus("s1", "READY")
	_ = bRepo.ListSuites()
	execID := bRepo.CreateExecution("s1")
	bRepo.UpdateExecution(execID, "COMPLETED")
	if bRepo.GetExecution(execID) != "COMPLETED" {
		t.Fatal("GetExecution failed")
	}
	_ = bRepo.ListExecutions("s1")

	rf := &RewardFunction{Lambda: 0.5}
	if rf.ComputeReward(1.0, 100, 50) <= 0 {
		t.Fatal("ComputeReward failed")
	}
}

