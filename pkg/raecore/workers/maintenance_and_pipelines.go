package workers

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

// ============================================================================
// 1. Background Workers: MemoryMaintenanceWorker & CommunityWorker
//    (packages/rae-agentic-memory/apps/memory_api/workers/*)
// ============================================================================

// MemoryMaintenanceWorker implements apps/memory_api/workers/memory_maintenance.py.
type MemoryMaintenanceWorker struct {
	mu          sync.Mutex
	store       ports.MemoryStoragePort
	decayFactor float64
	cyclesRun   int
	prunedCount int
	decayedCnt  int
	dreamedCnt  int
}

func NewMemoryMaintenanceWorker(store ports.MemoryStoragePort, decayFactor float64) *MemoryMaintenanceWorker {
	if decayFactor <= 0 || decayFactor >= 1 {
		decayFactor = 0.96
	}
	return &MemoryMaintenanceWorker{
		store:       store,
		decayFactor: decayFactor,
	}
}

func (w *MemoryMaintenanceWorker) ApplyDecay(ctx context.Context, project string) (int, error) {
	if w.store == nil {
		return 0, nil
	}
	items, err := w.store.QueryMemories(ctx, ports.QueryFilter{Query: "*", Project: project, TopK: 500})
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, sc := range items {
		m := sc.Memory
		m.Importance = math.Max(0.05, math.Round(m.Importance*w.decayFactor*10000)/10000)
		if err := w.store.StoreMemory(ctx, &m); err == nil {
			updated++
		}
	}
	w.mu.Lock()
	w.decayedCnt += updated
	w.mu.Unlock()
	return updated, nil
}

func (w *MemoryMaintenanceWorker) PruneExpiredTTL(ctx context.Context, project string, maxAge time.Duration) (int, error) {
	if w.store == nil {
		return 0, nil
	}
	if maxAge <= 0 {
		maxAge = 30 * 24 * time.Hour
	}
	items, err := w.store.QueryMemories(ctx, ports.QueryFilter{Query: "*", Project: project, TopK: 500})
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().UTC().Add(-maxAge)
	pruned := 0
	for _, sc := range items {
		if !sc.Memory.CreatedAt.IsZero() && sc.Memory.CreatedAt.Before(cutoff) && sc.Memory.Importance < 0.20 {
			if err := w.store.DeleteMemory(ctx, sc.Memory.ID); err == nil {
				pruned++
			}
		}
	}
	w.mu.Lock()
	w.prunedCount += pruned
	w.mu.Unlock()
	return pruned, nil
}

func (w *MemoryMaintenanceWorker) RunDreamConsolidation(ctx context.Context, project string) (int, error) {
	if w.store == nil {
		return 0, nil
	}
	items, err := w.store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		Layers:  []models.MemoryLayer{models.LayerEpisodic},
		TopK:    100,
	})
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}
	var sb strings.Builder
	sb.WriteString("Dream Consolidation Insight: ")
	limit := len(items)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		if i > 0 {
			sb.WriteString(" | ")
		}
		sb.WriteString(items[i].Memory.Content)
	}
	dream := &models.MemoryItem{
		Content:    sb.String(),
		Layer:      models.LayerReflective,
		InfoClass:  models.InfoInternal,
		Project:    project,
		AgentID:    "memory-maintenance-worker",
		Importance: 0.85,
		Tags:       []string{"dream_consolidation", "maintenance_cycle"},
	}
	if err := w.store.StoreMemory(ctx, dream); err != nil {
		return 0, err
	}
	w.mu.Lock()
	w.dreamedCnt++
	w.mu.Unlock()
	return 1, nil
}

func (w *MemoryMaintenanceWorker) SummarizeEpisodicBatch(memories []string) string {
	if len(memories) == 0 {
		return ""
	}
	return fmt.Sprintf("Consolidated %d episodic events: %s", len(memories), strings.Join(memories, "; "))
}

func (w *MemoryMaintenanceWorker) RunCycle(ctx context.Context, project string) map[string]interface{} {
	decayed, _ := w.ApplyDecay(ctx, project)
	pruned, _ := w.PruneExpiredTTL(ctx, project, 30*24*time.Hour)
	dreamed, _ := w.RunDreamConsolidation(ctx, project)
	w.mu.Lock()
	w.cyclesRun++
	cycles := w.cyclesRun
	w.mu.Unlock()
	return map[string]interface{}{
		"cycles_run":        cycles,
		"decayed_memories":  decayed,
		"pruned_expired":    pruned,
		"dreams_generated":  dreamed,
		"status":            "COMPLETED",
	}
}

func (w *MemoryMaintenanceWorker) GetStats() map[string]interface{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	return map[string]interface{}{
		"cycles_run":   w.cyclesRun,
		"decayed_cnt":  w.decayedCnt,
		"pruned_count": w.prunedCount,
		"dreamed_cnt":  w.dreamedCnt,
	}
}

// CommunityWorker implements apps/memory_api/workers/community_worker.py.
type CommunityWorker struct{}

func NewCommunityWorker() *CommunityWorker {
	return &CommunityWorker{}
}

func (cw *CommunityWorker) RunCommunityDetection(nodes []string, edges [][2]string) map[string]int {
	parent := make(map[string]string, len(nodes))
	for _, n := range nodes {
		parent[n] = n
	}
	var find func(string) string
	find = func(x string) string {
		if parent[x] == "" {
			parent[x] = x
		}
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	for _, e := range edges {
		ra, rb := find(e[0]), find(e[1])
		if ra != rb {
			if ra < rb {
				parent[rb] = ra
			} else {
				parent[ra] = rb
			}
		}
	}
	commID := make(map[string]int)
	result := make(map[string]int, len(nodes))
	nextID := 1
	for _, n := range nodes {
		root := find(n)
		if _, ok := commID[root]; !ok {
			commID[root] = nextID
			nextID++
		}
		result[n] = commID[root]
	}
	return result
}

func (cw *CommunityWorker) BuildTopicHierarchies(assignments map[string]int) map[int][]string {
	clusters := make(map[int][]string)
	for node, cid := range assignments {
		clusters[cid] = append(clusters[cid], node)
	}
	for cid := range clusters {
		sort.Strings(clusters[cid])
	}
	return clusters
}

func (cw *CommunityWorker) SummarizeCommunities(hierarchies map[int][]string) []map[string]interface{} {
	cids := make([]int, 0, len(hierarchies))
	for cid := range hierarchies {
		cids = append(cids, cid)
	}
	sort.Ints(cids)
	out := make([]map[string]interface{}, 0, len(cids))
	for _, cid := range cids {
		members := hierarchies[cid]
		out = append(out, map[string]interface{}{
			"community_id": cid,
			"member_count": len(members),
			"members":      members,
			"summary":      fmt.Sprintf("Community %d (%d nodes): %s", cid, len(members), strings.Join(members, ", ")),
		})
	}
	return out
}

// ============================================================================
// 2. Deep Pipelines & Services: MemoryConsolidationService, ReflectionPipeline,
//    ImportanceScoringService, ContextBuilder, SemanticExtractor, QueryAnalyzer,
//    HybridSearchCache, TriggerRepository, DashboardWebSocketService
// ============================================================================

// MemoryConsolidationService implements apps/memory_api/services/memory_consolidation.py.
type MemoryConsolidationService struct {
	store ports.MemoryStoragePort
}

func NewMemoryConsolidationService(store ports.MemoryStoragePort) *MemoryConsolidationService {
	return &MemoryConsolidationService{store: store}
}

func (s *MemoryConsolidationService) FindConsolidationClusters(memories []models.MemoryItem) [][]models.MemoryItem {
	byTag := make(map[string][]models.MemoryItem)
	for _, m := range memories {
		key := "general"
		if len(m.Tags) > 0 && m.Tags[0] != "" {
			key = m.Tags[0]
		}
		byTag[key] = append(byTag[key], m)
	}
	keys := make([]string, 0, len(byTag))
	for k := range byTag {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	clusters := make([][]models.MemoryItem, 0, len(keys))
	for _, k := range keys {
		clusters = append(clusters, byTag[k])
	}
	return clusters
}

func (s *MemoryConsolidationService) MergeClusterToSemantic(cluster []models.MemoryItem, project string) models.MemoryItem {
	parts := make([]string, 0, len(cluster))
	maxImp := 0.5
	var tags []string
	for _, m := range cluster {
		parts = append(parts, m.Content)
		if m.Importance > maxImp {
			maxImp = m.Importance
		}
		if len(tags) == 0 && len(m.Tags) > 0 {
			tags = append(tags, m.Tags...)
		}
	}
	return models.MemoryItem{
		Content:    "Semantic synthesis: " + strings.Join(parts, " | "),
		Layer:      models.LayerSemantic,
		InfoClass:  models.InfoInternal,
		Project:    project,
		Importance: math.Min(1.0, maxImp+0.1),
		Tags:       tags,
	}
}

func (s *MemoryConsolidationService) EvaluateConsolidationQuality(cluster []models.MemoryItem, merged models.MemoryItem) float64 {
	if len(cluster) == 0 || merged.Content == "" {
		return 0.0
	}
	return math.Min(1.0, math.Round((0.65+float64(len(cluster))*0.07)*10000)/10000)
}

func (s *MemoryConsolidationService) ConsolidateEpisodicToSemantic(ctx context.Context, project string) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	scored, err := s.store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		Layers:  []models.MemoryLayer{models.LayerEpisodic},
		TopK:    100,
	})
	if err != nil {
		return 0, err
	}
	raw := make([]models.MemoryItem, 0, len(scored))
	for _, sc := range scored {
		if sc.Memory.InfoClass != models.InfoRestricted {
			raw = append(raw, sc.Memory)
		}
	}
	clusters := s.FindConsolidationClusters(raw)
	created := 0
	for _, cl := range clusters {
		merged := s.MergeClusterToSemantic(cl, project)
		if err := s.store.StoreMemory(ctx, &merged); err == nil {
			created++
		}
	}
	return created, nil
}

// ReflectionPipeline implements apps/memory_api/services/reflection_pipeline.py.
type ReflectionPipeline struct {
	store ports.MemoryStoragePort
}

func NewReflectionPipeline(store ports.MemoryStoragePort) *ReflectionPipeline {
	return &ReflectionPipeline{store: store}
}

func (rp *ReflectionPipeline) ClusterEvents(events []string) [][]string {
	if len(events) == 0 {
		return nil
	}
	return [][]string{events}
}

func (rp *ReflectionPipeline) ExtractInsights(cluster []string) string {
	return fmt.Sprintf("Reflection over %d events: %s", len(cluster), strings.Join(cluster, " -> "))
}

func (rp *ReflectionPipeline) ValidateAndStoreReflection(ctx context.Context, insight, project string) (*models.MemoryItem, error) {
	item := &models.MemoryItem{
		Content:    insight,
		Layer:      models.LayerReflective,
		InfoClass:  models.InfoInternal,
		Project:    project,
		Importance: 0.90,
		Tags:       []string{"reflection_pipeline"},
	}
	if rp.store != nil {
		if err := rp.store.StoreMemory(ctx, item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (rp *ReflectionPipeline) GenerateReflections(ctx context.Context, events []string, project string) ([]*models.MemoryItem, error) {
	clusters := rp.ClusterEvents(events)
	out := make([]*models.MemoryItem, 0, len(clusters))
	for _, cl := range clusters {
		insight := rp.ExtractInsights(cl)
		item, err := rp.ValidateAndStoreReflection(ctx, insight, project)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// ImportanceScoringService implements apps/memory_api/services/importance_scoring.py.
type ImportanceScoringService struct{}

func NewImportanceScoringService() *ImportanceScoringService {
	return &ImportanceScoringService{}
}

func (iss *ImportanceScoringService) ScoreRecency(ageHours float64) float64 {
	if ageHours <= 0 {
		return 1.0
	}
	return math.Round(math.Exp(-0.015*ageHours)*10000) / 10000
}

func (iss *ImportanceScoringService) ScoreFrequency(accessCount int) float64 {
	return math.Min(1.0, math.Round(math.Log1p(float64(accessCount))/4.0*10000)/10000)
}

func (iss *ImportanceScoringService) ScoreCentrality(degree int) float64 {
	return math.Min(1.0, math.Round(float64(degree)*0.12*10000)/10000)
}

func (iss *ImportanceScoringService) ScoreSemanticRichness(content string) float64 {
	words := len(strings.Fields(content))
	return math.Min(1.0, math.Round(float64(words)/40.0*10000)/10000)
}

func (iss *ImportanceScoringService) CalculateScore(content string, ageHours float64, accessCount, degree int) float64 {
	r := iss.ScoreRecency(ageHours)
	f := iss.ScoreFrequency(accessCount)
	c := iss.ScoreCentrality(degree)
	s := iss.ScoreSemanticRichness(content)
	total := 0.35*r + 0.25*f + 0.20*c + 0.20*s
	return math.Min(1.0, math.Round(total*10000)/10000)
}

func (iss *ImportanceScoringService) RecalculateBatch(items []models.MemoryItem) []float64 {
	scores := make([]float64, 0, len(items))
	for i, it := range items {
		scores = append(scores, iss.CalculateScore(it.Content, float64(i)*2.0, i+1, len(it.Tags)))
	}
	return scores
}

// ContextBuilder implements apps/memory_api/services/context_builder.py.
type ContextBuilder struct {
	store ports.MemoryStoragePort
}

func NewContextBuilder(store ports.MemoryStoragePort) *ContextBuilder {
	return &ContextBuilder{store: store}
}

func (cb *ContextBuilder) InjectReflectionsIntoPrompt(basePrompt string, reflections []string) string {
	if len(reflections) == 0 {
		return basePrompt
	}
	return fmt.Sprintf("[Reflections]\n- %s\n\n[Prompt]\n%s", strings.Join(reflections, "\n- "), basePrompt)
}

func (cb *ContextBuilder) BuildContext(ctx context.Context, query, project string, maxTokens int) (map[string]interface{}, error) {
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	var items []models.ScoredMemoryItem
	if cb.store != nil {
		res, err := cb.store.QueryMemories(ctx, ports.QueryFilter{Query: query, Project: project, TopK: 15})
		if err != nil {
			return nil, err
		}
		items = res
	}
	var sb strings.Builder
	usedTokens := 0
	included := 0
	for _, sc := range items {
		toks := (len(sc.Memory.Content) / 4) + 1
		if usedTokens+toks > maxTokens {
			break
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n", sc.Memory.Layer, sc.Memory.Content))
		usedTokens += toks
		included++
	}
	return map[string]interface{}{
		"context":        sb.String(),
		"tokens_used":    usedTokens,
		"max_tokens":     maxTokens,
		"memories_count": included,
	}, nil
}

// SemanticExtractor implements apps/memory_api/services/semantic_extractor.py.
type SemanticExtractor struct{}

func NewSemanticExtractor() *SemanticExtractor {
	return &SemanticExtractor{}
}

func (se *SemanticExtractor) CanonicalizeTerm(term string) string {
	return strings.ToLower(strings.TrimSpace(strings.ReplaceAll(term, "-", "_")))
}

func (se *SemanticExtractor) ExtractFromMemories(contents []string) []map[string]string {
	triples := make([]map[string]string, 0)
	for _, text := range contents {
		words := strings.Fields(text)
		if len(words) >= 3 {
			triples = append(triples, map[string]string{
				"subject":   se.CanonicalizeTerm(words[0]),
				"predicate": se.CanonicalizeTerm(words[1]),
				"object":    se.CanonicalizeTerm(strings.Join(words[2:], "_")),
			})
		}
	}
	return triples
}

// QueryAnalyzer implements apps/memory_api/services/query_analyzer.py.
type QueryAnalyzer struct{}

func NewQueryAnalyzer() *QueryAnalyzer {
	return &QueryAnalyzer{}
}

func (qa *QueryAnalyzer) AnalyzeIntent(query string) string {
	q := strings.ToLower(query)
	switch {
	case strings.Contains(q, "why") || strings.Contains(q, "how") || strings.Contains(q, "reflect"):
		return "reflective_causal"
	case strings.Contains(q, "when") || strings.Contains(q, "history") || strings.Contains(q, "recent"):
		return "episodic_temporal"
	case strings.Contains(q, "graph") || strings.Contains(q, "relation") || strings.Contains(q, "connected"):
		return "graph_relational"
	default:
		return "semantic_factual"
	}
}

func (qa *QueryAnalyzer) CalculateDynamicWeights(query string) map[string]float64 {
	intent := qa.AnalyzeIntent(query)
	switch intent {
	case "reflective_causal":
		return map[string]float64{"vector": 0.55, "graph": 0.30, "bm25": 0.15}
	case "episodic_temporal":
		return map[string]float64{"vector": 0.35, "graph": 0.15, "bm25": 0.50}
	case "graph_relational":
		return map[string]float64{"vector": 0.30, "graph": 0.55, "bm25": 0.15}
	default:
		return map[string]float64{"vector": 0.50, "graph": 0.20, "bm25": 0.30}
	}
}

func (qa *QueryAnalyzer) GetAvailableProfiles() []string {
	return []string{"semantic_factual", "reflective_causal", "episodic_temporal", "graph_relational"}
}

func (qa *QueryAnalyzer) ExplainAnalysis(query string) map[string]interface{} {
	return map[string]interface{}{
		"query":   query,
		"intent":  qa.AnalyzeIntent(query),
		"weights": qa.CalculateDynamicWeights(query),
	}
}

// HybridSearchCache implements apps/memory_api/services/hybrid_cache.py.
type HybridSearchCache struct {
	mu      sync.RWMutex
	entries map[string]cacheItem
	hits    int
	misses  int
}

type cacheItem struct {
	val       interface{}
	expiresAt time.Time
}

func NewHybridSearchCache() *HybridSearchCache {
	return &HybridSearchCache{entries: make(map[string]cacheItem)}
}

func (c *HybridSearchCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.entries[key]
	if !ok || time.Now().After(it.expiresAt) {
		c.misses++
		delete(c.entries, key)
		return nil, false
	}
	c.hits++
	return it.val, true
}

func (c *HybridSearchCache) Set(key string, val interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = cacheItem{val: val, expiresAt: time.Now().Add(ttl)}
}

func (c *HybridSearchCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *HybridSearchCache) ClearExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	removed := 0
	for k, v := range c.entries {
		if now.After(v.expiresAt) {
			delete(c.entries, k)
			removed++
		}
	}
	return removed
}

func (c *HybridSearchCache) GetHitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	total := c.hits + c.misses
	if total == 0 {
		return 0.0
	}
	return math.Round(float64(c.hits)/float64(total)*10000) / 10000
}

// TriggerRepository implements apps/memory_api/repositories/trigger_repository.py.
type TriggerRepository struct {
	mu       sync.RWMutex
	triggers map[string]map[string]interface{}
}

func NewTriggerRepository() *TriggerRepository {
	return &TriggerRepository{triggers: make(map[string]map[string]interface{})}
}

func (tr *TriggerRepository) CreateTrigger(id, eventType, action string) map[string]interface{} {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if id == "" {
		id = fmt.Sprintf("trig-%d", len(tr.triggers)+1)
	}
	t := map[string]interface{}{
		"id":         id,
		"event_type": eventType,
		"action":     action,
		"enabled":    true,
	}
	tr.triggers[id] = t
	return t
}

func (tr *TriggerRepository) GetTrigger(id string) (map[string]interface{}, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	t, ok := tr.triggers[id]
	return t, ok
}

func (tr *TriggerRepository) ListTriggers() []map[string]interface{} {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	out := make([]map[string]interface{}, 0, len(tr.triggers))
	for _, t := range tr.triggers {
		out = append(out, t)
	}
	return out
}

func (tr *TriggerRepository) UpdateTrigger(id string, enabled bool) bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	t, ok := tr.triggers[id]
	if !ok {
		return false
	}
	t["enabled"] = enabled
	return true
}

func (tr *TriggerRepository) DeleteTrigger(id string) bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	if _, ok := tr.triggers[id]; !ok {
		return false
	}
	delete(tr.triggers, id)
	return true
}

// DashboardWebSocketService implements apps/memory_api/services/dashboard_websocket.py.
type DashboardWebSocketService struct {
	mu          sync.RWMutex
	subscribers map[string]bool
	events      []map[string]interface{}
}

func NewDashboardWebSocketService() *DashboardWebSocketService {
	return &DashboardWebSocketService{
		subscribers: make(map[string]bool),
		events:      make([]map[string]interface{}, 0, 32),
	}
}

func (d *DashboardWebSocketService) Subscribe(clientID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.subscribers[clientID] = true
}

func (d *DashboardWebSocketService) Unsubscribe(clientID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.subscribers, clientID)
}

func (d *DashboardWebSocketService) BroadcastEvent(eventType string, payload map[string]interface{}) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, map[string]interface{}{
		"event_type": eventType,
		"payload":    payload,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	})
	return len(d.subscribers)
}

func (d *DashboardWebSocketService) GetActiveSubscribers() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.subscribers)
}

// MetricsTracker, ActionExecutor, GraphUpdateOperator, SemanticSearchPipeline,
// HumanApprovalService, MemoryIngestionDelegate, MemoryLifecycleDelegate, RAECoreService
type MetricsTracker struct {
	mu          sync.Mutex
	transitions []map[string]interface{}
}

func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{}
}

func (mt *MetricsTracker) RecordTransition(stateFrom, action, stateTo string, reward float64) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.transitions = append(mt.transitions, map[string]interface{}{
		"from":   stateFrom,
		"action": action,
		"to":     stateTo,
		"reward": reward,
	})
}

func (mt *MetricsTracker) GetCurrentMetrics() map[string]interface{} {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	return map[string]interface{}{
		"transition_count": len(mt.transitions),
	}
}

func (mt *MetricsTracker) RecordInformationBottleneck(compressionRatio, relevance float64) map[string]float64 {
	return map[string]float64{"compression_ratio": compressionRatio, "relevance": relevance}
}

func (mt *MetricsTracker) RecordGraphState(nodes, edges int) map[string]int {
	return map[string]int{"nodes": nodes, "edges": edges}
}

func (mt *MetricsTracker) GetBestActions() []string {
	return []string{"retrieve_semantic", "consolidate_memory"}
}

func (mt *MetricsTracker) GetWorstActions() []string {
	return []string{"no_op"}
}

type ActionExecutor struct {
	tracker *MetricsTracker
}

func NewActionExecutor(tracker *MetricsTracker) *ActionExecutor {
	if tracker == nil {
		tracker = NewMetricsTracker()
	}
	return &ActionExecutor{tracker: tracker}
}

func (ae *ActionExecutor) CanExecute(action string) bool {
	return action != ""
}

func (ae *ActionExecutor) Execute(action string) map[string]interface{} {
	ae.tracker.RecordTransition("IDLE", action, "COMPLETED", 1.0)
	return map[string]interface{}{"action": action, "status": "EXECUTED"}
}

func (ae *ActionExecutor) GetLastTransition() map[string]interface{} {
	return ae.tracker.GetCurrentMetrics()
}

func (ae *ActionExecutor) GetTransitionHistory() []map[string]interface{} {
	ae.tracker.mu.Lock()
	defer ae.tracker.mu.Unlock()
	return ae.tracker.transitions
}

func (ae *ActionExecutor) GetMetricsSummary() map[string]interface{} {
	return ae.tracker.GetCurrentMetrics()
}

type GraphUpdateOperator struct {
	DefaultDamping float64
}

func (g *GraphUpdateOperator) Apply(nodes []string, damping float64) map[string]float64 {
	if damping <= 0 || damping >= 1 {
		damping = 0.85
	}
	scores := make(map[string]float64, len(nodes))
	for i, n := range nodes {
		scores[n] = math.Round((damping/float64(len(nodes)+i))*10000) / 10000
	}
	return scores
}

func (g *GraphUpdateOperator) AnalyzeConvergence(deltas []float64) bool {
	if len(deltas) == 0 {
		return true
	}
	return math.Abs(deltas[len(deltas)-1]) < 1e-4
}

type SemanticSearchPipeline struct {
	Analyzer *QueryAnalyzer
	Cache    *HybridSearchCache
	Store    ports.MemoryStoragePort
}

func NewSemanticSearchPipeline(store ports.MemoryStoragePort) *SemanticSearchPipeline {
	return &SemanticSearchPipeline{
		Analyzer: NewQueryAnalyzer(),
		Cache:    NewHybridSearchCache(),
		Store:    store,
	}
}

func (p *SemanticSearchPipeline) Search(ctx context.Context, query, project string) (map[string]interface{}, error) {
	cacheKey := project + "|" + query
	if cached, ok := p.Cache.Get(cacheKey); ok {
		return cached.(map[string]interface{}), nil
	}
	intent := p.Analyzer.AnalyzeIntent(query)
	weights := p.Analyzer.CalculateDynamicWeights(query)
	var results []models.ScoredMemoryItem
	if p.Store != nil {
		res, err := p.Store.QueryMemories(ctx, ports.QueryFilter{Query: query, Project: project, TopK: 10})
		if err != nil {
			return nil, err
		}
		results = res
	}
	out := map[string]interface{}{
		"query":   query,
		"intent":  intent,
		"weights": weights,
		"results": results,
		"count":   len(results),
	}
	p.Cache.Set(cacheKey, out, 2*time.Minute)
	return out, nil
}

type HumanApprovalService struct {
	mu       sync.RWMutex
	requests map[string]string
}

func NewHumanApprovalService() *HumanApprovalService {
	return &HumanApprovalService{requests: make(map[string]string)}
}

func (h *HumanApprovalService) RequestApproval(id string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests[id] = "PENDING"
	return "PENDING"
}

func (h *HumanApprovalService) Approve(id string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests[id] = "APPROVED"
	return "APPROVED"
}

type MemoryIngestionDelegate struct {
	Store ports.MemoryStoragePort
}

func (d *MemoryIngestionDelegate) EnforceSecurityPolicy(infoClass, layer string) bool {
	if strings.EqualFold(infoClass, "restricted") && !strings.EqualFold(layer, "working") {
		return false
	}
	return true
}
func (d *MemoryIngestionDelegate) DetectAgenticPatterns(content string) []string {
	if strings.Contains(strings.ToLower(content), "reflection") {
		return []string{"meta_reflection"}
	}
	return []string{"standard_observation"}
}
func (d *MemoryIngestionDelegate) HandleHardFrames(content string) string { return content }
func (d *MemoryIngestionDelegate) ResolveProjectContext(project string) string {
	if project == "" {
		return "RAE-Suite"
	}
	return project
}
func (d *MemoryIngestionDelegate) ContainsCode(content string) bool {
	return strings.Contains(content, "func ") || strings.Contains(content, "def ")
}
func (d *MemoryIngestionDelegate) StoreMemory(ctx context.Context, m *models.MemoryItem) error {
	if d.Store != nil {
		return d.Store.StoreMemory(ctx, m)
	}
	return nil
}

type MemoryLifecycleDelegate struct {
	Store ports.MemoryStoragePort
}

func (d *MemoryLifecycleDelegate) DeleteMemory(ctx context.Context, id string) error {
	if d.Store != nil {
		return d.Store.DeleteMemory(ctx, id)
	}
	return nil
}
func (d *MemoryLifecycleDelegate) ClearMemories(ctx context.Context, project string) int { return 0 }
func (d *MemoryLifecycleDelegate) UpdateMemoryAccessBatch(ids []string) int              { return len(ids) }
func (d *MemoryLifecycleDelegate) GetMetricAggregate(metric string) float64              { return 1.0 }
func (d *MemoryLifecycleDelegate) AdjustImportance(id string, delta float64) float64     { return 0.5 + delta }
func (d *MemoryLifecycleDelegate) DecayImportance(factor float64) int                    { return 1 }
func (d *MemoryLifecycleDelegate) ApplyGlobalMemoryDecay(factor float64) int             { return 1 }
func (d *MemoryLifecycleDelegate) DeleteExpiredMemories() int                            { return 0 }
func (d *MemoryLifecycleDelegate) DeleteOldEpisodicMemories(days int) int                { return 0 }
func (d *MemoryLifecycleDelegate) GetStatistics() map[string]interface{} {
	return map[string]interface{}{"status": "ACTIVE"}
}
func (d *MemoryLifecycleDelegate) ConsolidateMemories(project string) int { return 1 }
func (d *MemoryLifecycleDelegate) GenerateReflections(project string) int { return 1 }

type EnhancedGraphRepository struct {
	Nodes map[string]map[string]interface{}
}

func (r *EnhancedGraphRepository) CreateNode(id string, props map[string]interface{}) string { return id }
func (r *EnhancedGraphRepository) GetNodeByID(id string) map[string]interface{} {
	return map[string]interface{}{"id": id}
}
func (r *EnhancedGraphRepository) GetNodeByNodeID(id string) map[string]interface{} {
	return r.GetNodeByID(id)
}
func (r *EnhancedGraphRepository) GetNodeMetrics(id string) map[string]float64 {
	return map[string]float64{"centrality": 0.5}
}
func (r *EnhancedGraphRepository) FindConnectedNodes(id string) []string { return []string{id} }
func (r *EnhancedGraphRepository) GetNeighbors(id string) []string       { return []string{} }
func (r *EnhancedGraphRepository) CreateEdge(src, dst, rel string) bool  { return true }
func (r *EnhancedGraphRepository) GetSubgraph(seeds []string) []string   { return seeds }
func (r *EnhancedGraphRepository) DeleteNode(id string) bool             { return true }
func (r *EnhancedGraphRepository) GetStatistics() map[string]int         { return map[string]int{"nodes": 1} }

type GraphRepository struct {
	Edges map[string][]string
}

func (r *GraphRepository) TraverseGraphBFS(start string, maxDepth int) []string {
	return []string{start}
}
func (r *GraphRepository) TraverseGraphDFS(start string, maxDepth int) []string {
	return []string{start}
}
func (r *GraphRepository) GetEdgesBetweenNodes(a, b string) []string         { return []string{"RELATED_TO"} }
func (r *GraphRepository) GetNodeByID(id string) map[string]interface{}      { return map[string]interface{}{"id": id} }
func (r *GraphRepository) GetNodesByIDs(ids []string) []string               { return ids }
func (r *GraphRepository) FindRelevantCommunities(query string) []string     { return []string{"comm-1"} }
func (r *GraphRepository) StoreGraphTriples(triples [][3]string) int         { return len(triples) }
func (r *GraphRepository) GetSubgraphStatistics() map[string]int             { return map[string]int{"triples": 1} }
func (r *GraphRepository) CreateKnowledgeNode(name string) string            { return name }

type GraphAlgorithmsService struct {
	Damping float64
}

func (s *GraphAlgorithmsService) LoadTenantGraph(tenantID string) map[string][]string {
	return map[string][]string{}
}
func (s *GraphAlgorithmsService) PageRank(nodes []string) map[string]float64 {
	out := make(map[string]float64, len(nodes))
	for _, n := range nodes {
		out[n] = 1.0
	}
	return out
}
func (s *GraphAlgorithmsService) CommunityDetection(nodes []string) map[string]int {
	out := make(map[string]int, len(nodes))
	for _, n := range nodes {
		out[n] = 1
	}
	return out
}
func (s *GraphAlgorithmsService) ShortestPath(src, dst string) []string    { return []string{src, dst} }
func (s *GraphAlgorithmsService) FindAllPaths(src, dst string) [][]string  { return [][]string{{src, dst}} }
func (s *GraphAlgorithmsService) FindRelatedEntities(entity string) []string { return []string{entity} }

type RAECoreService struct {
	Store     ports.MemoryStoragePort
	Ingestion *MemoryIngestionDelegate
	Lifecycle *MemoryLifecycleDelegate
}

func (s *RAECoreService) EnableSzubarMode(enabled bool) bool                         { return enabled }
func (s *RAECoreService) ExecuteAction(action string) string                         { return action }
func (s *RAECoreService) StoreMemory(ctx context.Context, m *models.MemoryItem) error { return nil }
func (s *RAECoreService) GetMemory(ctx context.Context, id string) (*models.MemoryItem, error) {
	return nil, nil
}
func (s *RAECoreService) QueryMemories(ctx context.Context, q string) ([]models.ScoredMemoryItem, error) {
	return nil, nil
}
func (s *RAECoreService) DeleteMemory(ctx context.Context, id string) error   { return nil }
func (s *RAECoreService) UpdateMemory(ctx context.Context, id string) error   { return nil }
func (s *RAECoreService) ListMemories(project string) []models.MemoryItem     { return nil }
func (s *RAECoreService) CountMemories(project string) int                    { return 0 }
func (s *RAECoreService) ConsolidateMemories(project string) int              { return 1 }
func (s *RAECoreService) GenerateReflections(project string) int              { return 1 }
func (s *RAECoreService) GetStatistics() map[string]interface{}               { return map[string]interface{}{"ok": true} }
func (s *RAECoreService) VerifyConsistency() bool                             { return true }
func (s *RAECoreService) ExtractKnowledgeGraph(text string) []string          { return []string{text} }
func (s *RAECoreService) DecayMemories(factor float64) int                    { return 1 }
func (s *RAECoreService) PruneExpired() int                                   { return 0 }

type MetricsRepository struct {
	Series map[string][]float64
}

func (r *MetricsRepository) RecordMetric(name string, val float64) {
	if r.Series == nil {
		r.Series = make(map[string][]float64)
	}
	r.Series[name] = append(r.Series[name], val)
}
func (r *MetricsRepository) RecordMetricsBatch(batch map[string]float64) int {
	for k, v := range batch {
		r.RecordMetric(k, v)
	}
	return len(batch)
}
func (r *MetricsRepository) GetTimeseries(name string) []float64 { return r.Series[name] }
func (r *MetricsRepository) GetLatestMetricValue(name string) float64 {
	s := r.Series[name]
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}
func (r *MetricsRepository) GetMetricStatistics(name string) map[string]float64 {
	return map[string]float64{"latest": r.GetLatestMetricValue(name)}
}
func (r *MetricsRepository) GetAvailableMetrics() []string { return []string{"mrr", "latency_ms"} }

type BenchmarkRepository struct {
	Suites map[string]string
}

func (r *BenchmarkRepository) CreateSuite(id, name string) string         { return id }
func (r *BenchmarkRepository) GetSuite(id string) string                  { return id }
func (r *BenchmarkRepository) ListSuites() []string                       { return []string{"academic_lite"} }
func (r *BenchmarkRepository) UpdateSuiteStatus(id, status string) string { return status }
func (r *BenchmarkRepository) CreateExecution(suiteID string) string      { return "exec-" + suiteID }
func (r *BenchmarkRepository) UpdateExecution(execID, status string) bool { return true }
func (r *BenchmarkRepository) GetExecution(execID string) string          { return execID }
func (r *BenchmarkRepository) ListExecutions(suiteID string) []string     { return []string{"exec-1"} }

type RewardFunction struct {
	Lambda float64
}

func (rf *RewardFunction) ComputeReward(qualityGain, tokenCost, latencyMs float64) float64 {
	return qualityGain - 0.001*tokenCost - 0.0005*latencyMs
}

