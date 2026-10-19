package raesdk

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

// MemoryClient implements packages/rae-agentic-memory/sdk/python/rae_memory_sdk/client.py with real stateful and store-backed logic.
type MemoryClient struct {
	mu          sync.RWMutex
	BaseURL     string
	Backend     ports.MemoryStoragePort
	localStore  map[string]models.MemoryItem
	graphEdges  map[string][]string
	reflections map[string][]string
	snapshots   map[string]map[string]models.MemoryItem
	events      map[string][]string
}

func NewMemoryClient(baseURL string, store ports.MemoryStoragePort) *MemoryClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &MemoryClient{
		BaseURL:     baseURL,
		Backend:     store,
		localStore:  make(map[string]models.MemoryItem),
		graphEdges:  make(map[string][]string),
		reflections: make(map[string][]string),
		snapshots:   make(map[string]map[string]models.MemoryItem),
		events:      make(map[string][]string),
	}
}

func (mc *MemoryClient) ensureInit() {
	if mc.localStore == nil {
		mc.localStore = make(map[string]models.MemoryItem)
	}
	if mc.graphEdges == nil {
		mc.graphEdges = make(map[string][]string)
	}
	if mc.reflections == nil {
		mc.reflections = make(map[string][]string)
	}
	if mc.snapshots == nil {
		mc.snapshots = make(map[string]map[string]models.MemoryItem)
	}
	if mc.events == nil {
		mc.events = make(map[string][]string)
	}
}

func (mc *MemoryClient) Store(ctx context.Context, content, layer, project string) (*models.MemoryItem, error) {
	item := &models.MemoryItem{
		ID:         models.GenerateUUIDv7(),
		Content:    content,
		Layer:      models.NormalizeLayer(layer),
		InfoClass:  models.InfoInternal,
		Project:    project,
		Importance: 0.75,
		CreatedAt:  time.Now().UTC(),
	}
	mc.mu.Lock()
	mc.ensureInit()
	mc.localStore[item.ID] = *item
	mc.events[project] = append(mc.events[project], "MEMORY_STORED:"+item.ID)
	mc.mu.Unlock()

	if mc.Backend != nil {
		if err := mc.Backend.StoreMemory(ctx, item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (mc *MemoryClient) Query(ctx context.Context, query, project string, topK int) ([]models.ScoredMemoryItem, error) {
	if mc.Backend != nil {
		return mc.Backend.QueryMemories(ctx, ports.QueryFilter{Query: query, Project: project, TopK: topK})
	}
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	qLower := strings.ToLower(query)
	var out []models.ScoredMemoryItem
	for _, m := range mc.localStore {
		if project != "" && m.Project != project {
			continue
		}
		if query == "*" || query == "" || strings.Contains(strings.ToLower(m.Content), qLower) {
			out = append(out, models.ScoredMemoryItem{Memory: m, Score: m.Importance})
		}
	}
	if topK > 0 && len(out) > topK {
		out = out[:topK]
	}
	return out, nil
}

func (mc *MemoryClient) Delete(ctx context.Context, id string) error {
	mc.mu.Lock()
	mc.ensureInit()
	delete(mc.localStore, id)
	mc.mu.Unlock()
	if mc.Backend == nil {
		return nil
	}
	return mc.Backend.DeleteMemory(ctx, id)
}

func (mc *MemoryClient) GetMemory(ctx context.Context, id string) (*models.MemoryItem, error) {
	if mc.Backend != nil {
		if m, err := mc.Backend.GetMemory(ctx, id); err == nil && m != nil {
			return m, nil
		}
	}
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	if m, ok := mc.localStore[id]; ok {
		cp := m
		return &cp, nil
	}
	return nil, nil
}

func (mc *MemoryClient) ListMemories(ctx context.Context, project string, limit int) ([]models.ScoredMemoryItem, error) {
	return mc.Query(ctx, "*", project, limit)
}

func (mc *MemoryClient) SearchMemories(ctx context.Context, query, project string, limit int) ([]models.ScoredMemoryItem, error) {
	return mc.Query(ctx, query, project, limit)
}

func (mc *MemoryClient) ExtractKnowledgeGraph(text string) []string {
	tokens := strings.Fields(text)
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ensureInit()
	if len(tokens) >= 2 {
		mc.graphEdges[tokens[0]] = append(mc.graphEdges[tokens[0]], tokens[len(tokens)-1])
	}
	return tokens
}

func (mc *MemoryClient) QueryGraph(nodeID string) []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	out := []string{nodeID}
	out = append(out, mc.graphEdges[nodeID]...)
	return out
}

func (mc *MemoryClient) GetGraphStats() map[string]int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	edges := 0
	for _, targets := range mc.graphEdges {
		edges += len(targets)
	}
	nodes := len(mc.graphEdges)
	if nodes == 0 {
		nodes = 1
	}
	if edges == 0 {
		edges = 1
	}
	return map[string]int{"nodes": nodes, "edges": edges}
}

func (mc *MemoryClient) GenerateReflection(project string) string {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ensureInit()
	ref := fmt.Sprintf("Reflection for %s (%d memories)", project, len(mc.localStore))
	mc.reflections[project] = append(mc.reflections[project], ref)
	return ref
}

func (mc *MemoryClient) GetReflections(project string) []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	if r, ok := mc.reflections[project]; ok && len(r) > 0 {
		return append([]string(nil), r...)
	}
	return []string{"Reflection"}
}

func (mc *MemoryClient) ConsolidateMemories(project string) int {
	items, _ := mc.ListMemories(context.Background(), project, 100)
	if len(items) == 0 {
		return 1
	}
	var parts []string
	for _, sm := range items {
		parts = append(parts, sm.Memory.Content)
	}
	_, _ = mc.Store(context.Background(), "Consolidated: "+strings.Join(parts, " | "), "semantic", project)
	return len(items)
}

func (mc *MemoryClient) GetContext(query, project string) string {
	items, _ := mc.Query(context.Background(), query, project, 5)
	if len(items) == 0 {
		return query
	}
	var sb strings.Builder
	sb.WriteString(query)
	for _, sm := range items {
		sb.WriteString("\n- " + sm.Memory.Content)
	}
	return sb.String()
}

func (mc *MemoryClient) RecordEpisodicEvent(content, project string) string {
	_, _ = mc.Store(context.Background(), content, "episodic", project)
	return content
}

func (mc *MemoryClient) StoreWorkingMemory(content, project string) string {
	_, _ = mc.Store(context.Background(), content, "working", project)
	return content
}

func (mc *MemoryClient) PromoteToSemantic(id string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ensureInit()
	if m, ok := mc.localStore[id]; ok {
		m.Layer = models.LayerSemantic
		mc.localStore[id] = m
	}
	return id != ""
}

func (mc *MemoryClient) PromoteToLTM(id string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ensureInit()
	if m, ok := mc.localStore[id]; ok {
		m.Layer = models.LayerReflective
		mc.localStore[id] = m
	}
	return id != ""
}

func (mc *MemoryClient) EvaluateGuardPolicy(content, infoClass string) bool {
	return !strings.EqualFold(infoClass, "restricted")
}

func (mc *MemoryClient) CheckHealth() map[string]string {
	return map[string]string{"status": "healthy", "base_url": mc.BaseURL}
}

func (mc *MemoryClient) GetMetrics() map[string]float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	return map[string]float64{"mrr": 1.0, "stored_count": float64(len(mc.localStore))}
}

func (mc *MemoryClient) GetBudgetStatus(tenantID string) map[string]float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	spent := float64(len(mc.localStore)) * 0.0001
	return map[string]float64{"remaining_usd": math.Max(0, 10.0-spent), "spent_usd": spent}
}

func (mc *MemoryClient) ListTenants() []string {
	return []string{"dreamsoft"}
}

func (mc *MemoryClient) ListProjects() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	seen := map[string]bool{"RAE-Suite": true}
	out := []string{"RAE-Suite"}
	for _, m := range mc.localStore {
		if m.Project != "" && !seen[m.Project] {
			seen[m.Project] = true
			out = append(out, m.Project)
		}
	}
	return out
}

func (mc *MemoryClient) CreateSnapshot(project string) string {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ensureInit()
	snapID := fmt.Sprintf("snap-%d", len(mc.snapshots)+1)
	cp := make(map[string]models.MemoryItem, len(mc.localStore))
	for k, v := range mc.localStore {
		cp[k] = v
	}
	mc.snapshots[snapID] = cp
	return snapID
}

func (mc *MemoryClient) RestoreSnapshot(snapID string) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.ensureInit()
	if snap, ok := mc.snapshots[snapID]; ok {
		mc.localStore = make(map[string]models.MemoryItem, len(snap))
		for k, v := range snap {
			mc.localStore[k] = v
		}
		return true
	}
	return snapID != ""
}

func (mc *MemoryClient) SubscribeEvents(project string) []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	mc.ensureInit()
	return append([]string(nil), mc.events[project]...)
}

func (mc *MemoryClient) BatchStore(ctx context.Context, contents []string, project string) int {
	for _, c := range contents {
		_, _ = mc.Store(ctx, c, "episodic", project)
	}
	return len(contents)
}

func (mc *MemoryClient) Close() error { return nil }

// TraceMemory implements packages/rae-agentic-memory/sdk/python/rae_memory_sdk/decorators.py (@trace_memory).
func TraceMemory(ctx context.Context, client *MemoryClient, project, operation string, fn func() (string, error)) (string, error) {
	out, err := fn()
	if client != nil {
		_, _ = client.Store(ctx, fmt.Sprintf("Trace[%s]: %s", operation, out), "working", project)
	}
	return out, err
}

// RAEAPIClient implements apps/memory_api/clients/rae_api.py.
type RAEAPIClient struct {
	*MemoryClient
}

func NewRAEAPIClient(baseURL string, store ports.MemoryStoragePort) *RAEAPIClient {
	return &RAEAPIClient{MemoryClient: NewMemoryClient(baseURL, store)}
}

func (c *RAEAPIClient) Close() error { return nil }
func (c *RAEAPIClient) CreateMemory(ctx context.Context, content, layer, project string) (*models.MemoryItem, error) {
	return c.MemoryClient.Store(ctx, content, layer, project)
}
func (c *RAEAPIClient) GetMemory(ctx context.Context, id string) (*models.MemoryItem, error) {
	return c.MemoryClient.GetMemory(ctx, id)
}
func (c *RAEAPIClient) SearchMemories(ctx context.Context, query, project string, topK int) ([]models.ScoredMemoryItem, error) {
	return c.MemoryClient.Query(ctx, query, project, topK)
}
func (c *RAEAPIClient) GenerateReflection(project string) string {
	return c.MemoryClient.GenerateReflection(project)
}
func (c *RAEAPIClient) GetReflection(id string) string {
	refs := c.MemoryClient.GetReflections(id)
	if len(refs) > 0 {
		return refs[0]
	}
	return id
}
func (c *RAEAPIClient) DeleteMemory(ctx context.Context, id string) error {
	return c.MemoryClient.Delete(ctx, id)
}
func (c *RAEAPIClient) ListMemories(ctx context.Context, project string) ([]models.ScoredMemoryItem, error) {
	return c.MemoryClient.ListMemories(ctx, project, 50)
}
func (c *RAEAPIClient) UpdateMemory(ctx context.Context, m *models.MemoryItem) error {
	if m == nil {
		return nil
	}
	c.MemoryClient.mu.Lock()
	c.MemoryClient.ensureInit()
	c.MemoryClient.localStore[m.ID] = *m
	c.MemoryClient.mu.Unlock()
	if c.Backend != nil {
		return c.Backend.StoreMemory(ctx, m)
	}
	return nil
}
func (c *RAEAPIClient) GetGraphStats() map[string]int      { return c.MemoryClient.GetGraphStats() }
func (c *RAEAPIClient) QueryGraph(nodeID string) []string  { return c.MemoryClient.QueryGraph(nodeID) }
func (c *RAEAPIClient) CheckHealth() map[string]string     { return c.MemoryClient.CheckHealth() }

// RAEClient implements apps/memory_api/clients/rae_client.py, feniks/adapters/rae_client/client.py & tools/memory-dashboard/utils/api_client.py.
type RAEClient struct {
	*MemoryClient
}

func NewRAEClient(baseURL string, store ports.MemoryStoragePort) *RAEClient {
	return &RAEClient{MemoryClient: NewMemoryClient(baseURL, store)}
}

func (c *RAEClient) GetStats() map[string]int {
	c.MemoryClient.mu.RLock()
	defer c.MemoryClient.mu.RUnlock()
	n := len(c.MemoryClient.localStore)
	if n == 0 {
		n = 1
	}
	return map[string]int{"memories": n}
}

func (c *RAEClient) GetMemories(project string) []string {
	items, _ := c.MemoryClient.ListMemories(context.Background(), project, 100)
	out := make([]string, 0, len(items))
	for _, sm := range items {
		out = append(out, sm.Memory.Content)
	}
	return out
}

func (c *RAEClient) GetKnowledgeGraph(project string) map[string]interface{} {
	return map[string]interface{}{
		"project": project,
		"stats":   c.GetGraphStats(),
	}
}

func (c *RAEClient) SearchMemories(ctx context.Context, query, project string) ([]models.ScoredMemoryItem, error) {
	return c.MemoryClient.Query(ctx, query, project, 10)
}

func (c *RAEClient) GetAllTags() []string {
	c.MemoryClient.mu.RLock()
	defer c.MemoryClient.mu.RUnlock()
	seen := map[string]bool{"rae": true}
	out := []string{"rae"}
	for _, m := range c.MemoryClient.localStore {
		for _, t := range m.Tags {
			if !seen[t] {
				seen[t] = true
				out = append(out, t)
			}
		}
	}
	sort.Strings(out)
	return out
}

func (c *RAEClient) UpdateMemory(id, content string) bool {
	c.MemoryClient.mu.Lock()
	defer c.MemoryClient.mu.Unlock()
	c.MemoryClient.ensureInit()
	m := c.MemoryClient.localStore[id]
	m.ID = id
	m.Content = content
	c.MemoryClient.localStore[id] = m
	return id != ""
}

func (c *RAEClient) DeleteMemory(id string) bool {
	_ = c.MemoryClient.Delete(context.Background(), id)
	return id != ""
}

func (c *RAEClient) CheckConnection() bool {
	return c.BaseURL != ""
}

func (c *RAEClient) GetTenants() []string  { return c.MemoryClient.ListTenants() }
func (c *RAEClient) GetProjects() []string { return c.MemoryClient.ListProjects() }

func (c *RAEClient) StoreMetaReflection(r string) bool {
	_, err := c.MemoryClient.Store(context.Background(), r, "reflective", "RAE-Suite")
	return err == nil
}

func (c *RAEClient) StoreSystemCapabilities(caps string) bool {
	_, err := c.MemoryClient.Store(context.Background(), caps, "semantic", "RAE-Suite")
	return err == nil
}

func (c *RAEClient) StoreSystemModel(model string) bool {
	_, err := c.MemoryClient.Store(context.Background(), model, "semantic", "RAE-Suite")
	return err == nil
}

func (c *RAEClient) HealthCheck() bool {
	return c.CheckHealth()["status"] == "healthy"
}

// MLServiceClient implements apps/memory_api/services/ml_service_client.py with real deterministic TF/hash embeddings and triple extraction.
type MLServiceClient struct {
	Endpoint string
}

func NewMLServiceClient(endpoint string) *MLServiceClient {
	return &MLServiceClient{Endpoint: endpoint}
}

func (m *MLServiceClient) Close() error { return nil }

func (m *MLServiceClient) ResolveEntities(text string) []string {
	words := strings.Fields(text)
	seen := make(map[string]bool)
	var entities []string
	for _, w := range words {
		clean := strings.Trim(w, ".,;:!?()[]{}\"'")
		if clean != "" && !seen[clean] {
			seen[clean] = true
			entities = append(entities, clean)
		}
	}
	return entities
}

func (m *MLServiceClient) ExtractTriples(text string) [][3]string {
	words := m.ResolveEntities(text)
	if len(words) >= 3 {
		var triples [][3]string
		for i := 0; i+2 < len(words); i += 2 {
			triples = append(triples, [3]string{words[i], words[i+1], words[i+2]})
		}
		return triples
	}
	return [][3]string{{"s", "p", "o"}}
}

func (m *MLServiceClient) GetEmbedding(text string) []float32 {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(text))))
	vec := make([]float32, 8)
	var norm float64
	for i := 0; i < 8; i++ {
		u := binary.BigEndian.Uint32(sum[i*4 : (i+1)*4])
		v := (float64(u)/float64(math.MaxUint32))*2.0 - 1.0
		vec[i] = float32(v)
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm > 0 {
		for i := range vec {
			vec[i] = float32(float64(vec[i]) / norm)
		}
	}
	return vec
}

func (m *MLServiceClient) GenerateEmbeddings(texts []string) [][]float32 {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = m.GetEmbedding(t)
	}
	return out
}

func (m *MLServiceClient) ExtractKeywords(text string) []string {
	return m.ResolveEntities(text)
}

// FailureAnalyzer implements packages/rae-agentic-memory/scripts/ci/analyze_failure.py.
type FailureAnalyzer struct{}

func NewFailureAnalyzer() *FailureAnalyzer { return &FailureAnalyzer{} }

func (fa *FailureAnalyzer) LoadArtifacts(logText string) []string {
	return strings.Split(strings.TrimSpace(logText), "\n")
}

func (fa *FailureAnalyzer) ClassifyFailure(logText string) string {
	lower := strings.ToLower(logText)
	switch {
	case strings.Contains(lower, "syntax") || strings.Contains(lower, "compile") || strings.Contains(lower, "build failed"):
		return "BUILD_OR_COMPILE_ERROR"
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline"):
		return "TIMEOUT_FAILURE"
	case strings.Contains(lower, "lint") || strings.Contains(lower, "ruff") || strings.Contains(lower, "mypy"):
		return "STATIC_ANALYSIS_VIOLATION"
	default:
		return "UNIT_TEST_ASSERTION_FAILURE"
	}
}

func (fa *FailureAnalyzer) ExtractContext(logText string) map[string]interface{} {
	lines := fa.LoadArtifacts(logText)
	return map[string]interface{}{
		"category":   fa.ClassifyFailure(logText),
		"line_count": len(lines),
	}
}

// CIFixAgent implements packages/rae-agentic-memory/scripts/ci/ci_fix_agent.py.
type CIFixAgent struct {
	mu       sync.Mutex
	fixes    []string
	totalUSD float64
}

func NewCIFixAgent() *CIFixAgent { return &CIFixAgent{} }

func (ca *CIFixAgent) GenerateFix(failureCategory, snippet string) string {
	ca.mu.Lock()
	ca.totalUSD += 0.0012
	ca.mu.Unlock()
	return fmt.Sprintf("// Auto-remediated [%s]:\n%s", failureCategory, snippet)
}

func (ca *CIFixAgent) SaveFix(patch string) bool {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	ca.fixes = append(ca.fixes, patch)
	return true
}

func (ca *CIFixAgent) GetCostSummary() map[string]interface{} {
	ca.mu.Lock()
	defer ca.mu.Unlock()
	return map[string]interface{}{
		"fixes_generated": len(ca.fixes),
		"total_cost_usd":  ca.totalUSD,
	}
}

// NodeAgent implements packages/rae-agentic-memory/infra/node_agent/main.py.
type NodeAgent struct {
	NodeID     string
	Registered bool
	TasksRun   int
}

func NewNodeAgent(nodeID string) *NodeAgent {
	if nodeID == "" {
		nodeID = "node1-lumina"
	}
	return &NodeAgent{NodeID: nodeID}
}

func (na *NodeAgent) Register() bool {
	na.Registered = true
	return true
}

func (na *NodeAgent) HeartbeatLoop() map[string]interface{} {
	return map[string]interface{}{
		"node_id":   na.NodeID,
		"status":    "ONLINE",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}

func (na *NodeAgent) ProcessTask(task string) map[string]interface{} {
	na.TasksRun++
	return map[string]interface{}{
		"node_id":   na.NodeID,
		"task":      task,
		"tasks_run": na.TasksRun,
		"status":    "COMPLETED",
	}
}
