package raemesh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

// Peer represents a registered RAE-Mesh node (e.g., rae-host, node1-lumina).
type Peer struct {
	PeerID    string    `json:"peer_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Transport string    `json:"transport"`
	Status    string    `json:"status"`
	LastSync  time.Time `json:"last_sync"`
}

// MeshEngine manages multi-device Merkle-Tree delta synchronization and Consensual Memory Transfer (CMT).
type MeshEngine struct {
	mu    sync.RWMutex
	peers map[string]Peer
	store ports.MemoryStoragePort
	guard *security.HardFramesGuard
}

// NewMeshEngine creates a new RAE-Mesh engine with default Triple-Node cluster peers.
func NewMeshEngine(store ports.MemoryStoragePort, guard *security.HardFramesGuard) *MeshEngine {
	m := &MeshEngine{
		peers: make(map[string]Peer),
		store: store,
		guard: guard,
	}
	m.RegisterPeer(Peer{
		PeerID:    "rae-host",
		Name:      "Host Cloud",
		URL:       "https://rae.dreamsoft.pro",
		Transport: "http",
		Status:    "active",
	})
	m.RegisterPeer(Peer{
		PeerID:    "node1-lumina",
		Name:      "Node 1 Lumina",
		URL:       "http://100.68.166.117:8000",
		Transport: "http",
		Status:    "active",
	})
	return m
}

// RegisterPeer adds or updates a peer in the Mesh registry.
func (m *MeshEngine) RegisterPeer(p Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.Status == "" {
		p.Status = "active"
	}
	m.peers[p.PeerID] = p
}

// ListPeers returns all registered RAE-Mesh peers.
func (m *MeshEngine) ListPeers() []Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Peer, 0, len(m.peers))
	for _, p := range m.peers {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PeerID < out[j].PeerID
	})
	return out
}

// ComputeMerkleRoot computes the SHA-256 Merkle root over all exportable (non-RESTRICTED, non-Working) memories.
func (m *MeshEngine) ComputeMerkleRoot(ctx context.Context, project string) (string, []models.MemoryItem, error) {
	items, err := m.store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		TopK:    1000,
	})
	if err != nil {
		return "", nil, err
	}
	var exportable []models.MemoryItem
	var hashes []string
	for _, sc := range items {
		if err := m.guard.ValidateMeshTransfer(sc.Memory); err != nil {
			continue
		}
		exportable = append(exportable, sc.Memory)
		hashes = append(hashes, sc.Memory.ContentHash)
	}
	sort.Strings(hashes)
	h := sha256.New()
	for _, ch := range hashes {
		_, _ = h.Write([]byte(ch))
	}
	return hex.EncodeToString(h.Sum(nil)), exportable, nil
}
