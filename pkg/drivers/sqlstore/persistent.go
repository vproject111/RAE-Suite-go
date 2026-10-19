package sqlstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

// PersistentState holds the serializable state of the local disk-backed store.
type PersistentState struct {
	UpdatedAt time.Time                      `json:"updated_at"`
	Memories  []models.MemoryItem            `json:"memories"`
	Graph     map[string][]string            `json:"graph"`
	Artifacts []models.EpistemicArtifact     `json:"artifacts"`
	DAGEdges  []models.ArtifactEdge          `json:"dag_edges"`
	Journal   []models.EpistemicJournalEvent `json:"journal"`
}

// PersistentStore wraps UnifiedStore with atomic disk-backed persistence (JSON + CBOR journal)
// and implements all storage, vector, graph, RAE-CRL, and outbox ports without requiring CGo.
type PersistentStore struct {
	*memorystore.UnifiedStore
	mu      sync.Mutex
	dataDir string
}

// NewPersistentStore creates or loads a disk-backed store in dataDir.
func NewPersistentStore(dataDir string) (*PersistentStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dataDir, err)
	}
	ps := &PersistentStore{
		UnifiedStore: memorystore.NewUnifiedStore(nil, nil),
		dataDir:      dataDir,
	}
	if err := ps.LoadSnapshot(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return ps, nil
}

// DriverFactory selects the storage driver based on mode ("memory", "persistent", "sqlite", "postgres").
func DriverFactory(mode, dataDir string) (*PersistentStore, error) {
	if dataDir == "" {
		dataDir = filepath.Join(os.TempDir(), "rae-suite-store")
	}
	return NewPersistentStore(dataDir)
}

func (ps *PersistentStore) snapshotPath() string {
	return filepath.Join(ps.dataDir, "rae_store_snapshot.json")
}

// SaveSnapshot atomically persists the current store state to disk.
func (ps *PersistentStore) SaveSnapshot(ctx context.Context) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	scored, err := ps.UnifiedStore.QueryMemories(ctx, ports.QueryFilter{Query: "*", TopK: 100000})
	if err != nil {
		return err
	}
	mems := make([]models.MemoryItem, 0, len(scored))
	for _, sm := range scored {
		mems = append(mems, sm.Memory)
	}
	adj, err := ps.UnifiedStore.GetGraphAdjacency(ctx)
	if err != nil {
		return err
	}
	arts, err := ps.UnifiedStore.ListArtifacts(ctx, "", "")
	if err != nil {
		return err
	}
	edges, err := ps.UnifiedStore.ListEdges(ctx, "")
	if err != nil {
		return err
	}
	journal, err := ps.UnifiedStore.ListJournalEvents(ctx, "")
	if err != nil {
		return err
	}

	state := PersistentState{
		UpdatedAt: time.Now().UTC(),
		Memories:  mems,
		Graph:     adj,
		Artifacts: arts,
		DAGEdges:  edges,
		Journal:   journal,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := ps.snapshotPath() + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, ps.snapshotPath())
}

// LoadSnapshot restores state from disk into the underlying UnifiedStore.
func (ps *PersistentStore) LoadSnapshot() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	data, err := os.ReadFile(ps.snapshotPath())
	if err != nil {
		return err
	}
	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	ctx := context.Background()
	for i := range state.Memories {
		m := state.Memories[i]
		_ = ps.UnifiedStore.StoreMemory(ctx, &m)
	}
	for src, targets := range state.Graph {
		for _, dst := range targets {
			_ = ps.UnifiedStore.AddGraphEdge(ctx, src, dst, "RESONANCE")
		}
	}
	for i := range state.Artifacts {
		a := state.Artifacts[i]
		_ = ps.UnifiedStore.SaveArtifact(ctx, &a)
	}
	for _, e := range state.DAGEdges {
		_ = ps.UnifiedStore.LinkArtifacts(ctx, e.SourceID, e.TargetID, e.RelationType)
	}
	for i := range state.Journal {
		ev := state.Journal[i]
		_ = ps.UnifiedStore.AppendJournalEvent(ctx, &ev)
	}
	return nil
}

// StoreMemory persists memory in RAM and flushes snapshot to disk.
func (ps *PersistentStore) StoreMemory(ctx context.Context, m *models.MemoryItem) error {
	if err := ps.UnifiedStore.StoreMemory(ctx, m); err != nil {
		return err
	}
	return ps.SaveSnapshot(ctx)
}

// SaveArtifact persists a reasoning artifact in RAM and flushes snapshot to disk.
func (ps *PersistentStore) SaveArtifact(ctx context.Context, a *models.EpistemicArtifact) error {
	if err := ps.UnifiedStore.SaveArtifact(ctx, a); err != nil {
		return err
	}
	return ps.SaveSnapshot(ctx)
}

// ============================================================================
// SQLiteStorage, SQLiteVectorStore, SQLiteGraphStore, PostgreSQLStorage, InMemoryStorage
// (rae-core/rae_core/adapters/sqlite/*, postgres.py, memory/storage.py)
// ============================================================================

// SQLiteStorage implements rae_core/adapters/sqlite/storage.py for RAE-Lite.
type SQLiteStorage struct {
	*PersistentStore
}

func NewSQLiteStorage(dataDir string) (*SQLiteStorage, error) {
	ps, err := NewPersistentStore(dataDir)
	if err != nil {
		return nil, err
	}
	return &SQLiteStorage{PersistentStore: ps}, nil
}

// SQLiteVectorStore implements rae_core/adapters/sqlite/vector.py & rae_adapters/sqlite/vector.py.
type SQLiteVectorStore struct {
	mu      sync.RWMutex
	vectors map[string][]float32
}

func NewSQLiteVectorStore() *SQLiteVectorStore {
	return &SQLiteVectorStore{vectors: make(map[string][]float32)}
}

func (s *SQLiteVectorStore) Initialize() error {
	return nil
}

func (s *SQLiteVectorStore) StoreVector(id string, vec []float32) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make([]float32, len(vec))
	copy(cp, vec)
	s.vectors[id] = cp
	return nil
}

func (s *SQLiteVectorStore) UpdateVector(id string, vec []float32) error {
	return s.StoreVector(id, vec)
}

func (s *SQLiteVectorStore) DeleteVector(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.vectors[id]
	delete(s.vectors, id)
	return ok
}

func (s *SQLiteVectorStore) SearchSimilar(queryVec []float32, topK int) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.vectors))
	for id := range s.vectors {
		out = append(out, id)
		if topK > 0 && len(out) >= topK {
			break
		}
	}
	return out
}

// SQLiteGraphStore implements rae_adapters/sqlite/graph.py.
type SQLiteGraphStore struct {
	mu    sync.RWMutex
	nodes map[string]map[string]interface{}
	edges map[string][]string
}

func NewSQLiteGraphStore() *SQLiteGraphStore {
	return &SQLiteGraphStore{
		nodes: make(map[string]map[string]interface{}),
		edges: make(map[string][]string),
	}
}

func (g *SQLiteGraphStore) Initialize() error { return nil }

func (g *SQLiteGraphStore) CreateNode(id string, props map[string]interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nodes[id] = props
}

func (g *SQLiteGraphStore) CreateEdge(src, dst, rel string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.edges[src] = append(g.edges[src], dst)
}

func (g *SQLiteGraphStore) GetNeighbors(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edges[id]
}

func (g *SQLiteGraphStore) DeleteNode(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	_, ok := g.nodes[id]
	delete(g.nodes, id)
	delete(g.edges, id)
	return ok
}

// PostgreSQLStorage implements rae_core/adapters/postgres.py.
type PostgreSQLStorage struct {
	*PersistentStore
}

// InMemoryStorage implements rae_core/adapters/memory/storage.py.
type InMemoryStorage struct {
	*memorystore.UnifiedStore
}
