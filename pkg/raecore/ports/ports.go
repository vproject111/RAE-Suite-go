package ports

import (
	"context"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// QueryFilter specifies multi-layer search parameters across all 9 search strategies.
type QueryFilter struct {
	Query       string               `json:"query"`
	TenantID    string               `json:"tenant_id"`
	AgentID     string               `json:"agent_id"`
	Project     string               `json:"project"`
	Layers      []models.MemoryLayer `json:"layers"`
	Strategy    string               `json:"strategy"`     // anchor, fulltext, sparse, vector, multi_vector, graph, graph_lite, visual, hybrid
	VectorSpace string               `json:"vector_space"` // e.g., "dense", "ollama", "openai"
	QueryVector []float32            `json:"query_vector,omitempty"`
	TopK        int                  `json:"k"`
}

// MemoryStoragePort defines the database-agnostic contract for all 8 RAE-Memory layers.
type MemoryStoragePort interface {
	StoreMemory(ctx context.Context, item *models.MemoryItem) error
	GetMemory(ctx context.Context, id string) (*models.MemoryItem, error)
	QueryMemories(ctx context.Context, filter QueryFilter) ([]models.ScoredMemoryItem, error)
	DeleteExpiredSensory(ctx context.Context) (int, error)
	AddGraphEdge(ctx context.Context, sourceID, targetID, relation string) error
	GetGraphNeighbors(ctx context.Context, nodeID string) ([]string, error)
}

// EpistemicStoragePort defines the database-agnostic contract for RAE-CRL artifacts, DAG, and CBOR journal.
type EpistemicStoragePort interface {
	SaveArtifact(ctx context.Context, art *models.EpistemicArtifact) error
	GetArtifact(ctx context.Context, id string) (*models.EpistemicArtifact, error)
	ListArtifacts(ctx context.Context, project string, filterType models.ArtifactType) ([]models.EpistemicArtifact, error)
	LinkArtifacts(ctx context.Context, sourceID, targetID, relationType string) error
	ListEdges(ctx context.Context, project string) ([]models.ArtifactEdge, error)
	AppendJournalEvent(ctx context.Context, ev *models.EpistemicJournalEvent) error
	ListJournalEvents(ctx context.Context, project string) ([]models.EpistemicJournalEvent, error)
}

// OutboxEvent represents a transactional outbox message for RAE-Mesh / Governance propagation.
type OutboxEvent struct {
	ID          string                 `json:"id"`
	EventType   string                 `json:"event_type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      string                 `json:"status"` // PENDING, PROCESSING, DELIVERED, FAILED
	Attempts    int                    `json:"attempts"`
}

// OutboxQueuePort defines the transactional outbox contract (SKIP LOCKED / WAL).
type OutboxQueuePort interface {
	EnqueueOutbox(ctx context.Context, ev *OutboxEvent) error
	ClaimPendingOutbox(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkOutboxDelivered(ctx context.Context, id string) error
}
