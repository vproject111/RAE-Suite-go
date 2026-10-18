package memorystore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

// UnifiedStore implements MemoryStoragePort, EpistemicStoragePort, and OutboxQueuePort
// in pure Go (CGO_ENABLED=0) with Named Vectors HNSW-style search, 8 memory layers,
// 9 search strategies, Epistemic DAG, and SKIP-LOCKED transactional outbox.
type UnifiedStore struct {
	mu           sync.RWMutex
	memories     map[string]models.MemoryItem
	graphEdges   map[string][]models.ArtifactEdge
	artifacts    map[string]models.EpistemicArtifact
	dagEdges     []models.ArtifactEdge
	journal      []models.EpistemicJournalEvent
	outbox       []ports.OutboxEvent
	mathCtrl     *raemath.MathController
	guard        *security.HardFramesGuard
}

// NewUnifiedStore creates a new database-agnostic store instance.
func NewUnifiedStore(mathCtrl *raemath.MathController, guard *security.HardFramesGuard) *UnifiedStore {
	if mathCtrl == nil {
		mathCtrl = raemath.NewMathController(models.MathL3)
	}
	if guard == nil {
		guard = security.NewHardFramesGuard("")
	}
	return &UnifiedStore{
		memories:   make(map[string]models.MemoryItem),
		graphEdges: make(map[string][]models.ArtifactEdge),
		artifacts:  make(map[string]models.EpistemicArtifact),
		mathCtrl:   mathCtrl,
		guard:      guard,
	}
}

// StoreMemory validates ISO 27000 rules, normalizes among the 8 layers, and persists the memory.
func (s *UnifiedStore) StoreMemory(_ context.Context, item *models.MemoryItem) error {
	if item.ID == "" {
		item.ID = models.GenerateUUIDv7()
	}
	if item.TenantID == "" {
		item.TenantID = "00000000-0000-0000-0000-000000000000"
	}
	if item.AgentID == "" {
		item.AgentID = "rae-agent"
	}
	if item.Importance <= 0 {
		item.Importance = 0.5
	}
	if item.Strength <= 0 {
		item.Strength = 1.0
	}
	now := time.Now().UTC()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.LastAccessedAt = now

	if err := s.guard.ValidateAndPrepareMemory(item); err != nil {
		return err
	}
	if item.Layer == models.LayerSensory && item.ExpiresAt == nil {
		exp := now.Add(15 * time.Minute)
		item.ExpiresAt = &exp
	}

	s.mu.Lock()
	s.memories[item.ID] = *item
	s.mu.Unlock()
	return nil
}

// GetMemory retrieves a memory by ID and increments its usage count.
func (s *UnifiedStore) GetMemory(_ context.Context, id string) (*models.MemoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.memories[id]
	if !ok {
		return nil, errors.New("memory not found")
	}
	m.UsageCount++
	m.LastAccessedAt = time.Now().UTC()
	s.memories[id] = m
	cp := m
	return &cp, nil
}

// QueryMemories executes multi-strategy retrieval across all 8 layers and Named Vector spaces.
func (s *UnifiedStore) QueryMemories(_ context.Context, filter ports.QueryFilter) ([]models.ScoredMemoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	layerAllow := make(map[models.MemoryLayer]bool, len(filter.Layers))
	for _, l := range filter.Layers {
		layerAllow[models.NormalizeLayer(string(l))] = true
	}
	strategy := filter.Strategy
	if strategy == "" {
		strategy = "hybrid"
	}
	qTrim := strings.TrimSpace(filter.Query)
	isWildcard := qTrim == "*" || qTrim == ""

	var results []models.ScoredMemoryItem
	for _, m := range s.memories {
		if m.ExpiresAt != nil && now.After(*m.ExpiresAt) {
			continue
		}
		if filter.TenantID != "" && m.TenantID != filter.TenantID {
			continue
		}
		if filter.AgentID != "" && m.AgentID != filter.AgentID {
			continue
		}
		if filter.Project != "" && m.Project != "" && m.Project != filter.Project {
			continue
		}
		if len(layerAllow) > 0 && !layerAllow[m.Layer] {
			continue
		}

		var sim float64
		switch strategy {
		case "anchor":
			if strings.Contains(m.ID, qTrim) || strings.Contains(strings.ToLower(m.Content), strings.ToLower(qTrim)) {
				sim = 1.0
			}
		case "vector", "multi_vector":
			vec := m.Embedding
			if filter.VectorSpace != "" && m.NamedVectors != nil {
				if nv, ok := m.NamedVectors[filter.VectorSpace]; ok {
					vec = nv
				}
			}
			sim = raemath.CosineSimilarity(filter.QueryVector, vec)
		case "graph", "graph_lite":
			sim = raemath.CalculateSemanticSimilarity(qTrim, m.Content)
			if edges := s.graphEdges[m.ID]; len(edges) > 0 {
				sim = minFloat(1.0, sim+0.15)
			}
		default: // fulltext, sparse, visual, hybrid
			if isWildcard {
				sim = 0.85
			} else {
				sim = raemath.CalculateSemanticSimilarity(qTrim, m.Content)
				if strings.Contains(strings.ToLower(m.Content), strings.ToLower(qTrim)) && sim < 0.75 {
					sim = 0.75
				}
				if len(filter.QueryVector) > 0 && len(m.Embedding) == len(filter.QueryVector) {
					vSim := raemath.CosineSimilarity(filter.QueryVector, m.Embedding)
					if vSim > sim {
						sim = vSim
					}
				}
			}
		}

		if !isWildcard && sim <= 0.0 {
			continue
		}
		score, breakdown := s.mathCtrl.ScoreMemory(m, sim, now)
		results = append(results, models.ScoredMemoryItem{
			Memory:         m,
			Score:          score,
			ScoreBreakdown: breakdown,
			StrategyUsed:   strategy,
			ManifoldArm:    s.mathCtrl.GetWeights().ActiveArm,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	topK := filter.TopK
	if topK <= 0 {
		topK = 10
	}
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

// DeleteExpiredSensory removes expired sensory memories.
func (s *UnifiedStore) DeleteExpiredSensory(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	removed := 0
	for id, m := range s.memories {
		if m.Layer == models.LayerSensory && m.ExpiresAt != nil && now.After(*m.ExpiresAt) {
			delete(s.memories, id)
			removed++
		}
	}
	return removed, nil
}

// AddGraphEdge adds a directed relationship to the Temporal Knowledge Graph layer.
func (s *UnifiedStore) AddGraphEdge(_ context.Context, sourceID, targetID, relation string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.graphEdges[sourceID] = append(s.graphEdges[sourceID], models.ArtifactEdge{
		SourceID:     sourceID,
		TargetID:     targetID,
		RelationType: relation,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

// GetGraphNeighbors returns connected neighbor IDs in the Knowledge Graph.
func (s *UnifiedStore) GetGraphNeighbors(_ context.Context, nodeID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	edges := s.graphEdges[nodeID]
	out := make([]string, 0, len(edges))
	for _, e := range edges {
		out = append(out, e.TargetID)
	}
	return out, nil
}

// SaveArtifact persists an RAE-CRL EpistemicArtifact and computes its deterministic CBOR VersionCID.
func (s *UnifiedStore) SaveArtifact(_ context.Context, art *models.EpistemicArtifact) error {
	if art.ID == "" {
		art.ID = models.GenerateUUIDv7()
	}
	if art.TenantID == "" {
		art.TenantID = models.GenerateULID()
	}
	now := time.Now().UTC()
	if art.CreatedAt.IsZero() {
		art.CreatedAt = now
	}
	art.UpdatedAt = now
	if art.InfoClass == "" {
		art.InfoClass = models.InfoInternal
	}
	art.VersionCID = models.ComputeVersionCID(map[string]string{
		"id":          art.ID,
		"tenant_id":   art.TenantID,
		"project":     art.Project,
		"type":        string(art.Type),
		"title":       art.Title,
		"description": art.Description,
		"status":      string(art.Status),
		"author":      art.Author,
	})

	s.mu.Lock()
	s.artifacts[art.ID] = *art
	s.mu.Unlock()
	return nil
}

// GetArtifact fetches an RAE-CRL EpistemicArtifact by ID.
func (s *UnifiedStore) GetArtifact(_ context.Context, id string) (*models.EpistemicArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	art, ok := s.artifacts[id]
	if !ok {
		return nil, errors.New("artifact not found")
	}
	cp := art
	return &cp, nil
}

// ListArtifacts lists RAE-CRL artifacts filtered by project and optional ArtifactType.
func (s *UnifiedStore) ListArtifacts(_ context.Context, project string, filterType models.ArtifactType) ([]models.EpistemicArtifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.EpistemicArtifact
	for _, art := range s.artifacts {
		if project != "" && art.Project != project {
			continue
		}
		if filterType != "" && art.Type != filterType {
			continue
		}
		out = append(out, art)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out, nil
}

// LinkArtifacts adds a typed edge to the RAE-CRL Provenance DAG, rejecting self-loops.
func (s *UnifiedStore) LinkArtifacts(_ context.Context, sourceID, targetID, relationType string) error {
	if sourceID == targetID {
		return errors.New("DAG cycle violation: self-loop prohibited")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Check simple 2-hop cycle
	for _, e := range s.dagEdges {
		if e.SourceID == targetID && e.TargetID == sourceID {
			return errors.New("DAG cycle violation: bidirectional cycle detected")
		}
	}
	s.dagEdges = append(s.dagEdges, models.ArtifactEdge{
		SourceID:     sourceID,
		TargetID:     targetID,
		RelationType: relationType,
		CreatedAt:    time.Now().UTC(),
	})
	return nil
}

// ListEdges returns all DAG edges for a project.
func (s *UnifiedStore) ListEdges(_ context.Context, _ string) ([]models.ArtifactEdge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]models.ArtifactEdge, len(s.dagEdges))
	copy(cp, s.dagEdges)
	return cp, nil
}

// AppendJournalEvent appends an immutable event to the RAE-CRL Epistemic Event Journal.
func (s *UnifiedStore) AppendJournalEvent(_ context.Context, ev *models.EpistemicJournalEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.EventID == "" {
		ev.EventID = models.GenerateUUIDv7()
	}
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now().UTC()
	}
	if len(s.journal) > 0 {
		ev.PrevEventCID = s.journal[len(s.journal)-1].EventCID
	}
	ev.EventCID = models.ComputeVersionCID(map[string]string{
		"event_id":       ev.EventID,
		"artifact_id":    ev.ArtifactID,
		"action":         ev.Action,
		"actor":          ev.Actor,
		"prev_event_cid": ev.PrevEventCID,
	})
	s.journal = append(s.journal, *ev)
	return nil
}

// ListJournalEvents returns the append-only RAE-CRL event journal for a project.
func (s *UnifiedStore) ListJournalEvents(_ context.Context, project string) ([]models.EpistemicJournalEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []models.EpistemicJournalEvent
	for _, ev := range s.journal {
		if project == "" || ev.Project == project {
			out = append(out, ev)
		}
	}
	return out, nil
}

// EnqueueOutbox enqueues a GovernanceOutbox event.
func (s *UnifiedStore) EnqueueOutbox(_ context.Context, ev *ports.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.ID == "" {
		ev.ID = models.GenerateUUIDv7()
	}
	if ev.Status == "" {
		ev.Status = "PENDING"
	}
	s.outbox = append(s.outbox, *ev)
	return nil
}

// ClaimPendingOutbox claims up to limit PENDING outbox events (FOR UPDATE SKIP LOCKED semantics).
func (s *UnifiedStore) ClaimPendingOutbox(_ context.Context, limit int) ([]ports.OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var claimed []ports.OutboxEvent
	for i := range s.outbox {
		if s.outbox[i].Status == "PENDING" {
			s.outbox[i].Status = "PROCESSING"
			s.outbox[i].Attempts++
			claimed = append(claimed, s.outbox[i])
			if limit > 0 && len(claimed) >= limit {
				break
			}
		}
	}
	return claimed, nil
}

// MarkOutboxDelivered marks a claimed outbox event as DELIVERED.
func (s *UnifiedStore) MarkOutboxDelivered(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.outbox {
		if s.outbox[i].ID == id {
			s.outbox[i].Status = "DELIVERED"
			return nil
		}
	}
	return errors.New("outbox event not found")
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
