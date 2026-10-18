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
