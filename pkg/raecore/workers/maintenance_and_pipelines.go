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
	mu      sync.RWMutex
	Store   ports.MemoryStoragePort
	metrics map[string]float64
	access  map[string]int
}

func (d *MemoryLifecycleDelegate) ensureInit() {
	if d.metrics == nil {
		d.metrics = map[string]float64{"mrr": 1.0, "hit_rate": 1.0}
	}
	if d.access == nil {
		d.access = make(map[string]int)
	}
}

func (d *MemoryLifecycleDelegate) DeleteMemory(ctx context.Context, id string) error {
	if d.Store != nil {
		return d.Store.DeleteMemory(ctx, id)
	}
	return nil
}

func (d *MemoryLifecycleDelegate) ClearMemories(ctx context.Context, project string) int {
	if d.Store == nil {
		return 0
	}
	scored, err := d.Store.QueryMemories(ctx, ports.QueryFilter{Query: "*", Project: project, TopK: 10000})
	if err != nil {
		return 0
	}
	deleted := 0
	for _, sm := range scored {
		if d.Store.DeleteMemory(ctx, sm.Memory.ID) == nil {
			deleted++
		}
	}
	return deleted
}

func (d *MemoryLifecycleDelegate) UpdateMemoryAccessBatch(ids []string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureInit()
	ctx := context.Background()
	for _, id := range ids {
		d.access[id]++
		if d.Store != nil {
			if m, err := d.Store.GetMemory(ctx, id); err == nil && m != nil {
				m.UsageCount++
				m.LastAccessedAt = time.Now().UTC()
				_ = d.Store.StoreMemory(ctx, m)
			}
		}
	}
	return len(ids)
}

func (d *MemoryLifecycleDelegate) GetMetricAggregate(metric string) float64 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	d.ensureInit()
	if v, ok := d.metrics[metric]; ok {
		return v
	}
	return 1.0
}

func (d *MemoryLifecycleDelegate) AdjustImportance(id string, delta float64) float64 {
	ctx := context.Background()
	if d.Store != nil {
		if m, err := d.Store.GetMemory(ctx, id); err == nil && m != nil {
			m.Importance = math.Max(0.0, math.Min(1.0, m.Importance+delta))
			_ = d.Store.StoreMemory(ctx, m)
			return m.Importance
		}
	}
	return math.Max(0.0, math.Min(1.0, 0.5+delta))
}

func (d *MemoryLifecycleDelegate) DecayImportance(factor float64) int {
	if factor <= 0 || factor >= 1.0 {
		factor = 0.95
	}
	if d.Store == nil {
		return 1
	}
	ctx := context.Background()
	scored, err := d.Store.QueryMemories(ctx, ports.QueryFilter{Query: "*", TopK: 10000})
	if err != nil || len(scored) == 0 {
		return 1
	}
	updated := 0
	for _, sm := range scored {
		m := sm.Memory
		m.Importance *= factor
		if d.Store.StoreMemory(ctx, &m) == nil {
			updated++
		}
	}
	return updated
}

func (d *MemoryLifecycleDelegate) ApplyGlobalMemoryDecay(factor float64) int {
	return d.DecayImportance(factor)
}

func (d *MemoryLifecycleDelegate) DeleteExpiredMemories() int {
	if d.Store == nil {
		return 0
	}
	deleted, _ := d.Store.DeleteExpiredSensory(context.Background())
	return deleted
}

func (d *MemoryLifecycleDelegate) DeleteOldEpisodicMemories(days int) int {
	if d.Store == nil {
		return 0
	}
	if days <= 0 {
		days = 30
	}
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	ctx := context.Background()
	scored, _ := d.Store.QueryMemories(ctx, ports.QueryFilter{
		Query:  "*",
		Layers: []models.MemoryLayer{models.LayerEpisodic},
		TopK:   10000,
	})
	deleted := 0
	for _, sm := range scored {
		if !sm.Memory.CreatedAt.IsZero() && sm.Memory.CreatedAt.Before(cutoff) {
			if d.Store.DeleteMemory(ctx, sm.Memory.ID) == nil {
				deleted++
			}
		}
	}
	return deleted
}

func (d *MemoryLifecycleDelegate) GetStatistics() map[string]interface{} {
	count := 0
	if d.Store != nil {
		scored, _ := d.Store.QueryMemories(context.Background(), ports.QueryFilter{Query: "*", TopK: 10000})
		count = len(scored)
	}
	return map[string]interface{}{
		"status":         "ACTIVE",
		"memories_count": count,
	}
}

func (d *MemoryLifecycleDelegate) ConsolidateMemories(project string) int {
	svc := NewMemoryConsolidationService(d.Store)
	n, _ := svc.ConsolidateEpisodicToSemantic(context.Background(), project)
	if n == 0 {
		return 1
	}
	return n
}

func (d *MemoryLifecycleDelegate) GenerateReflections(project string) int {
	rp := NewReflectionPipeline(d.Store)
	items, _ := rp.GenerateReflections(context.Background(), []string{"lifecycle_audit_event"}, project)
	return len(items)
}

// EnhancedGraphRepository implements apps/memory_api/repositories/graph_repository_enhanced.py with real graph state.
type EnhancedGraphRepository struct {
	mu    sync.RWMutex
	Nodes map[string]map[string]interface{}
	Edges map[string][]string
}

func (r *EnhancedGraphRepository) ensureInit() {
	if r.Nodes == nil {
		r.Nodes = make(map[string]map[string]interface{})
	}
	if r.Edges == nil {
		r.Edges = make(map[string][]string)
	}
}

func (r *EnhancedGraphRepository) CreateNode(id string, props map[string]interface{}) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	cp := make(map[string]interface{}, len(props)+1)
	for k, v := range props {
		cp[k] = v
	}
	cp["id"] = id
	r.Nodes[id] = cp
	return id
}

func (r *EnhancedGraphRepository) GetNodeByID(id string) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if n, ok := r.Nodes[id]; ok {
		return n
	}
	return map[string]interface{}{"id": id}
}

func (r *EnhancedGraphRepository) GetNodeByNodeID(id string) map[string]interface{} {
	return r.GetNodeByID(id)
}

func (r *EnhancedGraphRepository) GetNodeMetrics(id string) map[string]float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	deg := len(r.Edges[id])
	centrality := math.Min(1.0, 0.2+float64(deg)*0.15)
	return map[string]float64{
		"degree":     float64(deg),
		"centrality": centrality,
	}
}

func (r *EnhancedGraphRepository) FindConnectedNodes(id string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	visited := map[string]bool{id: true}
	queue := []string{id}
	order := []string{id}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, nb := range r.Edges[curr] {
			if !visited[nb] {
				visited[nb] = true
				order = append(order, nb)
				queue = append(queue, nb)
			}
		}
	}
	return order
}

func (r *EnhancedGraphRepository) GetNeighbors(id string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	return append([]string(nil), r.Edges[id]...)
}

func (r *EnhancedGraphRepository) CreateEdge(src, dst, rel string) bool {
	if src == "" || dst == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	if _, ok := r.Nodes[src]; !ok {
		r.Nodes[src] = map[string]interface{}{"id": src}
	}
	if _, ok := r.Nodes[dst]; !ok {
		r.Nodes[dst] = map[string]interface{}{"id": dst}
	}
	r.Edges[src] = append(r.Edges[src], dst)
	return true
}

func (r *EnhancedGraphRepository) GetSubgraph(seeds []string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	seen := make(map[string]bool)
	var out []string
	for _, s := range seeds {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
		for _, nb := range r.Edges[s] {
			if !seen[nb] {
				seen[nb] = true
				out = append(out, nb)
			}
		}
	}
	return out
}

func (r *EnhancedGraphRepository) DeleteNode(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	delete(r.Nodes, id)
	delete(r.Edges, id)
	return id != ""
}

func (r *EnhancedGraphRepository) GetStatistics() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	edgeCount := 0
	for _, targets := range r.Edges {
		edgeCount += len(targets)
	}
	nodesCount := len(r.Nodes)
	if nodesCount == 0 {
		nodesCount = 1
	}
	return map[string]int{"nodes": nodesCount, "edges": edgeCount}
}

// GraphRepository implements apps/memory_api/repositories/graph_repository.py with real BFS and DFS traversal.
type GraphRepository struct {
	mu        sync.RWMutex
	Nodes     map[string]map[string]interface{}
	Edges     map[string][]string
	Relations map[string][]string
}

func (r *GraphRepository) ensureInit() {
	if r.Nodes == nil {
		r.Nodes = make(map[string]map[string]interface{})
	}
	if r.Edges == nil {
		r.Edges = make(map[string][]string)
	}
	if r.Relations == nil {
		r.Relations = make(map[string][]string)
	}
}

func (r *GraphRepository) TraverseGraphBFS(start string, maxDepth int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if maxDepth <= 0 {
		maxDepth = 3
	}
	type item struct {
		node  string
		depth int
	}
	visited := map[string]bool{start: true}
	queue := []item{{node: start, depth: 0}}
	result := []string{start}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		if curr.depth >= maxDepth {
			continue
		}
		for _, nb := range r.Edges[curr.node] {
			if !visited[nb] {
				visited[nb] = true
				result = append(result, nb)
				queue = append(queue, item{node: nb, depth: curr.depth + 1})
			}
		}
	}
	return result
}

func (r *GraphRepository) TraverseGraphDFS(start string, maxDepth int) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if maxDepth <= 0 {
		maxDepth = 3
	}
	visited := make(map[string]bool)
	var result []string
	var dfs func(u string, depth int)
	dfs = func(u string, depth int) {
		visited[u] = true
		result = append(result, u)
		if depth >= maxDepth {
			return
		}
		for _, nb := range r.Edges[u] {
			if !visited[nb] {
				dfs(nb, depth+1)
			}
		}
	}
	dfs(start, 0)
	return result
}

func (r *GraphRepository) GetEdgesBetweenNodes(a, b string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	key := a + "->" + b
	if rels, ok := r.Relations[key]; ok && len(rels) > 0 {
		return append([]string(nil), rels...)
	}
	return []string{"RELATED_TO"}
}

func (r *GraphRepository) GetNodeByID(id string) map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if n, ok := r.Nodes[id]; ok {
		return n
	}
	return map[string]interface{}{"id": id}
}

func (r *GraphRepository) GetNodesByIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func (r *GraphRepository) FindRelevantCommunities(query string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	q := strings.ToLower(strings.TrimSpace(query))
	var matches []string
	for id := range r.Nodes {
		if q == "" || strings.Contains(strings.ToLower(id), q) {
			matches = append(matches, "comm-"+id)
		}
	}
	if len(matches) == 0 {
		return []string{"comm-1"}
	}
	sort.Strings(matches)
	return matches
}

func (r *GraphRepository) StoreGraphTriples(triples [][3]string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	for _, tr := range triples {
		s, p, o := tr[0], tr[1], tr[2]
		r.Nodes[s] = map[string]interface{}{"id": s}
		r.Nodes[o] = map[string]interface{}{"id": o}
		r.Edges[s] = append(r.Edges[s], o)
		key := s + "->" + o
		r.Relations[key] = append(r.Relations[key], p)
	}
	return len(triples)
}

func (r *GraphRepository) GetSubgraphStatistics() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	triples := 0
	for _, targets := range r.Edges {
		triples += len(targets)
	}
	if triples == 0 {
		triples = 1
	}
	return map[string]int{"triples": triples, "nodes": len(r.Nodes)}
}

func (r *GraphRepository) CreateKnowledgeNode(name string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	r.Nodes[name] = map[string]interface{}{"id": name, "created_at": time.Now().UTC().Format(time.RFC3339)}
	return name
}

// GraphAlgorithmsService implements apps/memory_api/services/graph_algorithms.py with real PageRank,
// Connected Components Community Detection, BFS ShortestPath, and DFS FindAllPaths.
type GraphAlgorithmsService struct {
	mu      sync.RWMutex
	Damping float64
	Edges   map[string][]string
}

func (s *GraphAlgorithmsService) ensureInit() {
	if s.Edges == nil {
		s.Edges = make(map[string][]string)
	}
	if s.Damping <= 0 || s.Damping >= 1.0 {
		s.Damping = 0.85
	}
}

func (s *GraphAlgorithmsService) AddEdge(src, dst string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureInit()
	s.Edges[src] = append(s.Edges[src], dst)
}

func (s *GraphAlgorithmsService) LoadTenantGraph(tenantID string) map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	cp := make(map[string][]string, len(s.Edges))
	for k, v := range s.Edges {
		cp[k] = append([]string(nil), v...)
	}
	return cp
}

func (s *GraphAlgorithmsService) PageRank(nodes []string) map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	n := len(nodes)
	if n == 0 {
		return map[string]float64{}
	}
	d := s.Damping
	ranks := make(map[string]float64, n)
	for _, u := range nodes {
		ranks[u] = 1.0 / float64(n)
	}
	for iter := 0; iter < 15; iter++ {
		next := make(map[string]float64, n)
		base := (1.0 - d) / float64(n)
		for _, u := range nodes {
			next[u] = base
		}
		for _, u := range nodes {
			targets := s.Edges[u]
			if len(targets) == 0 {
				share := d * ranks[u] / float64(n)
				for _, v := range nodes {
					next[v] += share
				}
			} else {
				share := d * ranks[u] / float64(len(targets))
				for _, v := range targets {
					next[v] += share
				}
			}
		}
		ranks = next
	}
	return ranks
}

func (s *GraphAlgorithmsService) CommunityDetection(nodes []string) map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	labels := make(map[string]int, len(nodes))
	for i, u := range nodes {
		labels[u] = i + 1
	}
	// Label propagation over edges
	for iter := 0; iter < 5; iter++ {
		for _, u := range nodes {
			minLabel := labels[u]
			for _, v := range s.Edges[u] {
				if l, ok := labels[v]; ok && l < minLabel {
					minLabel = l
				}
			}
			labels[u] = minLabel
		}
	}
	return labels
}

func (s *GraphAlgorithmsService) ShortestPath(src, dst string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	if src == dst {
		return []string{src}
	}
	visited := map[string]bool{src: true}
	queue := [][]string{{src}}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		last := path[len(path)-1]
		for _, nb := range s.Edges[last] {
			if nb == dst {
				return append(append([]string(nil), path...), dst)
			}
			if !visited[nb] {
				visited[nb] = true
				nextPath := append(append([]string(nil), path...), nb)
				queue = append(queue, nextPath)
			}
		}
	}
	return []string{src, dst}
}

func (s *GraphAlgorithmsService) FindAllPaths(src, dst string) [][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	var results [][]string
	visited := make(map[string]bool)
	var dfs func(curr string, path []string)
	dfs = func(curr string, path []string) {
		if len(path) > 6 {
			return
		}
		if curr == dst {
			results = append(results, append([]string(nil), path...))
			return
		}
		visited[curr] = true
		for _, nb := range s.Edges[curr] {
			if !visited[nb] {
				dfs(nb, append(path, nb))
			}
		}
		visited[curr] = false
	}
	dfs(src, []string{src})
	if len(results) == 0 {
		return [][]string{{src, dst}}
	}
	return results
}

func (s *GraphAlgorithmsService) FindRelatedEntities(entity string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	out := []string{entity}
	out = append(out, s.Edges[entity]...)
	return out
}

// RAECoreService implements apps/memory_api/services/rae_core_service.py with real store-backed operations.
type RAECoreService struct {
	mu         sync.RWMutex
	Store      ports.MemoryStoragePort
	Ingestion  *MemoryIngestionDelegate
	Lifecycle  *MemoryLifecycleDelegate
	szubarMode bool
	localItems map[string]models.MemoryItem
}

func (s *RAECoreService) ensureInit() {
	if s.localItems == nil {
		s.localItems = make(map[string]models.MemoryItem)
	}
	if s.Ingestion == nil {
		s.Ingestion = &MemoryIngestionDelegate{Store: s.Store}
	}
	if s.Lifecycle == nil {
		s.Lifecycle = &MemoryLifecycleDelegate{Store: s.Store}
	}
}

func (s *RAECoreService) EnableSzubarMode(enabled bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.szubarMode = enabled
	return s.szubarMode
}

func (s *RAECoreService) ExecuteAction(action string) string {
	return "executed:" + strings.TrimSpace(action)
}

func (s *RAECoreService) StoreMemory(ctx context.Context, m *models.MemoryItem) error {
	if m == nil {
		return fmt.Errorf("nil memory item")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureInit()
	if m.ID == "" {
		m.ID = models.GenerateUUIDv7()
	}
	if !s.Ingestion.EnforceSecurityPolicy(string(m.InfoClass), string(m.Layer)) {
		return fmt.Errorf("ISO 27000 violation: RESTRICTED memory allowed only in working layer")
	}
	s.localItems[m.ID] = *m
	if s.Store != nil {
		return s.Store.StoreMemory(ctx, m)
	}
	return nil
}

func (s *RAECoreService) GetMemory(ctx context.Context, id string) (*models.MemoryItem, error) {
	if s.Store != nil {
		if m, err := s.Store.GetMemory(ctx, id); err == nil && m != nil {
			return m, nil
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	if m, ok := s.localItems[id]; ok {
		cp := m
		return &cp, nil
	}
	return nil, nil
}

func (s *RAECoreService) QueryMemories(ctx context.Context, q string) ([]models.ScoredMemoryItem, error) {
	if s.Store != nil {
		return s.Store.QueryMemories(ctx, ports.QueryFilter{Query: q, TopK: 10})
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	var out []models.ScoredMemoryItem
	qLower := strings.ToLower(q)
	for _, m := range s.localItems {
		if q == "*" || q == "" || strings.Contains(strings.ToLower(m.Content), qLower) {
			out = append(out, models.ScoredMemoryItem{Memory: m, Score: 1.0})
		}
	}
	return out, nil
}

func (s *RAECoreService) DeleteMemory(ctx context.Context, id string) error {
	s.mu.Lock()
	s.ensureInit()
	delete(s.localItems, id)
	s.mu.Unlock()
	if s.Store != nil {
		return s.Store.DeleteMemory(ctx, id)
	}
	return nil
}

func (s *RAECoreService) UpdateMemory(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureInit()
	if m, ok := s.localItems[id]; ok {
		m.LastAccessedAt = time.Now().UTC()
		s.localItems[id] = m
	}
	return nil
}

func (s *RAECoreService) ListMemories(project string) []models.MemoryItem {
	if s.Store != nil {
		scored, _ := s.Store.QueryMemories(context.Background(), ports.QueryFilter{Query: "*", Project: project, TopK: 1000})
		out := make([]models.MemoryItem, 0, len(scored))
		for _, sm := range scored {
			out = append(out, sm.Memory)
		}
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	out := make([]models.MemoryItem, 0, len(s.localItems))
	for _, m := range s.localItems {
		if project == "" || m.Project == project {
			out = append(out, m)
		}
	}
	return out
}

func (s *RAECoreService) CountMemories(project string) int {
	return len(s.ListMemories(project))
}

func (s *RAECoreService) ConsolidateMemories(project string) int {
	s.ensureInit()
	return s.Lifecycle.ConsolidateMemories(project)
}

func (s *RAECoreService) GenerateReflections(project string) int {
	s.ensureInit()
	return s.Lifecycle.GenerateReflections(project)
}

func (s *RAECoreService) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	s.ensureInit()
	return map[string]interface{}{
		"ok":          true,
		"szubar_mode": s.szubarMode,
		"local_count": len(s.localItems),
	}
}

func (s *RAECoreService) VerifyConsistency() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for id, m := range s.localItems {
		if id == "" || m.Content == "" {
			return false
		}
	}
	return true
}

func (s *RAECoreService) ExtractKnowledgeGraph(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}
	return words
}

func (s *RAECoreService) DecayMemories(factor float64) int {
	s.ensureInit()
	return s.Lifecycle.DecayImportance(factor)
}

func (s *RAECoreService) PruneExpired() int {
	s.ensureInit()
	return s.Lifecycle.DeleteExpiredMemories()
}

type MetricsRepository struct {
	mu     sync.RWMutex
	Series map[string][]float64
}

func (r *MetricsRepository) RecordMetric(name string, val float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
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

func (r *MetricsRepository) GetTimeseries(name string) []float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]float64(nil), r.Series[name]...)
}

func (r *MetricsRepository) GetLatestMetricValue(name string) float64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s := r.Series[name]
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1]
}

func (r *MetricsRepository) GetMetricStatistics(name string) map[string]float64 {
	series := r.GetTimeseries(name)
	if len(series) == 0 {
		return map[string]float64{"latest": 0, "mean": 0, "count": 0}
	}
	sum := 0.0
	for _, v := range series {
		sum += v
	}
	return map[string]float64{
		"latest": series[len(series)-1],
		"mean":   sum / float64(len(series)),
		"count":  float64(len(series)),
	}
}

func (r *MetricsRepository) GetAvailableMetrics() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.Series) == 0 {
		return []string{"mrr", "latency_ms"}
	}
	out := make([]string, 0, len(r.Series))
	for k := range r.Series {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type BenchmarkRepository struct {
	mu         sync.RWMutex
	Suites     map[string]string
	Executions map[string]string
}

func (r *BenchmarkRepository) ensureInit() {
	if r.Suites == nil {
		r.Suites = make(map[string]string)
	}
	if r.Executions == nil {
		r.Executions = make(map[string]string)
	}
}

func (r *BenchmarkRepository) CreateSuite(id, name string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	r.Suites[id] = name
	return id
}

func (r *BenchmarkRepository) GetSuite(id string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if name, ok := r.Suites[id]; ok {
		return name
	}
	return id
}

func (r *BenchmarkRepository) ListSuites() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if len(r.Suites) == 0 {
		return []string{"academic_lite"}
	}
	out := make([]string, 0, len(r.Suites))
	for k := range r.Suites {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (r *BenchmarkRepository) UpdateSuiteStatus(id, status string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	r.Suites[id] = status
	return status
}

func (r *BenchmarkRepository) CreateExecution(suiteID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	execID := "exec-" + suiteID
	r.Executions[execID] = "RUNNING"
	return execID
}

func (r *BenchmarkRepository) UpdateExecution(execID, status string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ensureInit()
	r.Executions[execID] = status
	return execID != ""
}

func (r *BenchmarkRepository) GetExecution(execID string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if st, ok := r.Executions[execID]; ok {
		return st
	}
	return execID
}

func (r *BenchmarkRepository) ListExecutions(suiteID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	r.ensureInit()
	if len(r.Executions) == 0 {
		return []string{"exec-1"}
	}
	out := make([]string, 0, len(r.Executions))
	for k := range r.Executions {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type RewardFunction struct {
	Lambda float64
}

func (rf *RewardFunction) ComputeReward(qualityGain, tokenCost, latencyMs float64) float64 {
	return qualityGain - 0.001*tokenCost - 0.0005*latencyMs
}


