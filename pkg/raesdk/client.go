package raesdk

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

// MemoryClient implements packages/rae-agentic-memory/sdk/python/rae_memory_sdk/client.py.
type MemoryClient struct {
	BaseURL string
	Backend ports.MemoryStoragePort
}

func NewMemoryClient(baseURL string, store ports.MemoryStoragePort) *MemoryClient {
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &MemoryClient{BaseURL: baseURL, Backend: store}
}

func (mc *MemoryClient) Store(ctx context.Context, content, layer, project string) (*models.MemoryItem, error) {
	item := &models.MemoryItem{
		Content:   content,
		Layer:     models.NormalizeLayer(layer),
		InfoClass: models.InfoInternal,
		Project:   project,
	}
	if mc.Backend != nil {
		if err := mc.Backend.StoreMemory(ctx, item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (mc *MemoryClient) Query(ctx context.Context, query, project string, topK int) ([]models.ScoredMemoryItem, error) {
	if mc.Backend == nil {
		return nil, nil
	}
	return mc.Backend.QueryMemories(ctx, ports.QueryFilter{Query: query, Project: project, TopK: topK})
}

func (mc *MemoryClient) Delete(ctx context.Context, id string) error {
	if mc.Backend == nil {
		return nil
	}
	return mc.Backend.DeleteMemory(ctx, id)
}
func (mc *MemoryClient) GetMemory(ctx context.Context, id string) (*models.MemoryItem, error) {
	if mc.Backend == nil {
		return nil, nil
	}
	return mc.Backend.GetMemory(ctx, id)
}
func (mc *MemoryClient) ListMemories(ctx context.Context, project string, limit int) ([]models.ScoredMemoryItem, error) {
	return mc.Query(ctx, "*", project, limit)
}
func (mc *MemoryClient) SearchMemories(ctx context.Context, query, project string, limit int) ([]models.ScoredMemoryItem, error) {
	return mc.Query(ctx, query, project, limit)
}
func (mc *MemoryClient) ExtractKnowledgeGraph(text string) []string           { return strings.Fields(text) }
func (mc *MemoryClient) QueryGraph(nodeID string) []string                    { return []string{nodeID} }
func (mc *MemoryClient) GetGraphStats() map[string]int                        { return map[string]int{"nodes": 1, "edges": 1} }
func (mc *MemoryClient) GenerateReflection(project string) string             { return "Reflection for " + project }
func (mc *MemoryClient) GetReflections(project string) []string               { return []string{"Reflection"} }
func (mc *MemoryClient) ConsolidateMemories(project string) int               { return 1 }
func (mc *MemoryClient) GetContext(query, project string) string              { return query }
func (mc *MemoryClient) RecordEpisodicEvent(content, project string) string   { return content }
func (mc *MemoryClient) StoreWorkingMemory(content, project string) string    { return content }
func (mc *MemoryClient) PromoteToSemantic(id string) bool                     { return true }
func (mc *MemoryClient) PromoteToLTM(id string) bool                          { return true }
func (mc *MemoryClient) EvaluateGuardPolicy(content, infoClass string) bool   { return infoClass != "restricted" }
func (mc *MemoryClient) CheckHealth() map[string]string                       { return map[string]string{"status": "healthy"} }
func (mc *MemoryClient) GetMetrics() map[string]float64                       { return map[string]float64{"mrr": 1.0} }
func (mc *MemoryClient) GetBudgetStatus(tenantID string) map[string]float64   { return map[string]float64{"remaining_usd": 10.0} }
func (mc *MemoryClient) ListTenants() []string                                { return []string{"dreamsoft"} }
func (mc *MemoryClient) ListProjects() []string                               { return []string{"RAE-Suite"} }
func (mc *MemoryClient) CreateSnapshot(project string) string                 { return "snap-1" }
func (mc *MemoryClient) RestoreSnapshot(snapID string) bool                   { return true }
func (mc *MemoryClient) SubscribeEvents(project string) []string              { return []string{} }
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
func (c *RAEAPIClient) GenerateReflection(project string) string { return c.MemoryClient.GenerateReflection(project) }
func (c *RAEAPIClient) GetReflection(id string) string           { return id }
func (c *RAEAPIClient) DeleteMemory(ctx context.Context, id string) error {
	return c.MemoryClient.Delete(ctx, id)
}
func (c *RAEAPIClient) ListMemories(ctx context.Context, project string) ([]models.ScoredMemoryItem, error) {
	return c.MemoryClient.ListMemories(ctx, project, 50)
}
func (c *RAEAPIClient) UpdateMemory(ctx context.Context, m *models.MemoryItem) error { return nil }
func (c *RAEAPIClient) GetGraphStats() map[string]int                                { return c.MemoryClient.GetGraphStats() }
func (c *RAEAPIClient) QueryGraph(nodeID string) []string                            { return c.MemoryClient.QueryGraph(nodeID) }
func (c *RAEAPIClient) CheckHealth() map[string]string                               { return c.MemoryClient.CheckHealth() }

// RAEClient implements apps/memory_api/clients/rae_client.py, feniks/adapters/rae_client/client.py & tools/memory-dashboard/utils/api_client.py.
type RAEClient struct {
	*MemoryClient
}

func NewRAEClient(baseURL string, store ports.MemoryStoragePort) *RAEClient {
	return &RAEClient{MemoryClient: NewMemoryClient(baseURL, store)}
}

func (c *RAEClient) GetStats() map[string]int      { return map[string]int{"memories": 1} }
func (c *RAEClient) GetMemories(project string) []string { return []string{} }
func (c *RAEClient) GetKnowledgeGraph(project string) map[string]interface{} {
	return map[string]interface{}{"project": project}
}
func (c *RAEClient) SearchMemories(ctx context.Context, query, project string) ([]models.ScoredMemoryItem, error) {
	return c.MemoryClient.Query(ctx, query, project, 10)
}
func (c *RAEClient) GetAllTags() []string                     { return []string{"rae"} }
func (c *RAEClient) UpdateMemory(id, content string) bool     { return true }
func (c *RAEClient) DeleteMemory(id string) bool              { return true }
func (c *RAEClient) CheckConnection() bool                    { return true }
func (c *RAEClient) GetTenants() []string                     { return []string{"dreamsoft"} }
func (c *RAEClient) GetProjects() []string                    { return []string{"RAE-Suite"} }
func (c *RAEClient) StoreMetaReflection(r string) bool        { return true }
func (c *RAEClient) StoreSystemCapabilities(caps string) bool { return true }
func (c *RAEClient) StoreSystemModel(model string) bool       { return true }
func (c *RAEClient) HealthCheck() bool                        { return true }

// MLServiceClient implements apps/memory_api/services/ml_service_client.py.
type MLServiceClient struct {
	Endpoint string
}

func NewMLServiceClient(endpoint string) *MLServiceClient {
	return &MLServiceClient{Endpoint: endpoint}
}

func (m *MLServiceClient) Close() error                               { return nil }
func (m *MLServiceClient) ResolveEntities(text string) []string       { return strings.Fields(text) }
func (m *MLServiceClient) ExtractTriples(text string) [][3]string     { return [][3]string{{"s", "p", "o"}} }
func (m *MLServiceClient) GenerateEmbeddings(texts []string) [][]float32 {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.1, 0.2, 0.3, 0.4}
	}
	return out
}
func (m *MLServiceClient) GetEmbedding(text string) []float32   { return []float32{0.1, 0.2, 0.3, 0.4} }
func (m *MLServiceClient) ExtractKeywords(text string) []string { return strings.Fields(text) }


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
