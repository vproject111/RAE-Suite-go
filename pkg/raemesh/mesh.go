package raemesh

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
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

// CapabilityClaims represents the signed payload of a CMT (Consensual Memory Transfer) token.
type CapabilityClaims struct {
	PeerID    string `json:"peer_id"`
	Scope     string `json:"scope"`
	ExpiresAt int64  `json:"expires_at"`
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
	if guard == nil {
		guard = security.NewHardFramesGuard("")
	}
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

// GenerateCapabilityToken mints a signed CMT capability token for a peer and scope.
func (m *MeshEngine) GenerateCapabilityToken(peerID, scope string, ttl time.Duration) (string, error) {
	if peerID == "" || scope == "" {
		return "", errors.New("peerID and scope are required")
	}
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}
	claims := CapabilityClaims{
		PeerID:    peerID,
		Scope:     scope,
		ExpiresAt: time.Now().UTC().Add(ttl).Unix(),
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	b64 := base64.RawURLEncoding.EncodeToString(raw)
	sig := m.guard.SignTokenHMAC(b64)
	return b64 + "." + sig, nil
}

// VerifyCapabilityToken validates a CMT capability token's signature, expiration, and scope.
func (m *MeshEngine) VerifyCapabilityToken(token, requiredScope string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", errors.New("invalid CMT token format")
	}
	if !m.guard.VerifyTokenHMAC(parts[0], parts[1]) {
		return "", errors.New("invalid CMT token HMAC signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	var claims CapabilityClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return "", err
	}
	if time.Now().UTC().Unix() > claims.ExpiresAt {
		return "", errors.New("CMT capability token expired")
	}
	if requiredScope != "" && claims.Scope != requiredScope && claims.Scope != "*" {
		return "", fmt.Errorf("insufficient CMT scope: have %s, want %s", claims.Scope, requiredScope)
	}
	return claims.PeerID, nil
}

// ReceiveMemories implements the CMT (Consensual Memory Transfer) receiver protocol.
func (m *MeshEngine) ReceiveMemories(ctx context.Context, token string, items []models.MemoryItem) (int, int, error) {
	peerID, err := m.VerifyCapabilityToken(token, "mesh:write")
	if err != nil {
		return 0, 0, err
	}
	accepted, rejected := 0, 0
	for i := range items {
		item := items[i]
		if err := m.guard.ValidateMeshTransfer(item); err != nil {
			rejected++
			continue
		}
		if item.Envelope == nil {
			item.Envelope = &models.ContextEnvelope{
				OriginDevice: peerID,
				TrustLevel:   models.TrustHigh,
				Version:      1,
			}
		}
		if err := m.store.StoreMemory(ctx, &item); err != nil {
			rejected++
			continue
		}
		accepted++
	}
	return accepted, rejected, nil
}

// SyncWithPeer computes the Merkle delta for a specific peer and updates its LastSync timestamp.
func (m *MeshEngine) SyncWithPeer(ctx context.Context, peerID, project string) (map[string]interface{}, error) {
	root, exp, err := m.ComputeMerkleRoot(ctx, project)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	p, ok := m.peers[peerID]
	if !ok {
		p = Peer{PeerID: peerID, Name: peerID, Transport: "http", Status: "active"}
	}
	p.LastSync = time.Now().UTC()
	m.peers[peerID] = p
	m.mu.Unlock()

	return map[string]interface{}{
		"peer_id":         peerID,
		"status":          "synced",
		"merkle_root":     root,
		"synced_memories": len(exp),
		"last_sync":       p.LastSync,
	}, nil
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
