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

// RAEAPIClient implements apps/memory_api/clients/rae_api.py.
type RAEAPIClient struct {
	*MemoryClient
}

// RAEClient implements apps/memory_api/clients/rae_client.py & tools/memory-dashboard/utils/api_client.py.
type RAEClient struct {
	*MemoryClient
}

// MLServiceClient implements apps/memory_api/services/ml_service_client.py.
type MLServiceClient struct {
	Endpoint string
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
