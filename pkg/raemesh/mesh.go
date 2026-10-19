package raemesh

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

// Peer represents a registered RAE-Mesh node (e.g., rae-host, node1-lumina, node2-julka, node3-piotrek).
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

// MerkleTreeSnapshot holds the global Merkle root and 16 hex-prefix bucket hashes ("0".."f") for delta sync.
type MerkleTreeSnapshot struct {
	MerkleRoot      string            `json:"merkle_root"`
	BucketHashes    map[string]string `json:"bucket_hashes"`
	ExportableCount int               `json:"exportable_count"`
}

// MeshEngine manages database-agnostic multi-device Merkle-Tree delta synchronization and Consensual Memory Transfer (CMT).
type MeshEngine struct {
	mu     sync.RWMutex
	peers  map[string]Peer
	store  ports.MemoryStoragePort
	guard  *security.HardFramesGuard
	crypto *E2EEncryption
	client *http.Client
}

// NewMeshEngine creates a new RAE-Mesh engine with default Triple-Node cluster peers.
func NewMeshEngine(store ports.MemoryStoragePort, guard *security.HardFramesGuard) *MeshEngine {
	if guard == nil {
		guard = security.NewHardFramesGuard("")
	}
	m := &MeshEngine{
		peers:  make(map[string]Peer),
		store:  store,
		guard:  guard,
		crypto: NewE2EEncryption(""),
		client: &http.Client{Timeout: 350 * time.Millisecond},
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
	m.RegisterPeer(Peer{
		PeerID:    "node2-julka",
		Name:      "Node 2 Julka",
		URL:       "http://100.70.32.40:8000",
		Transport: "http",
		Status:    "active",
	})
	m.RegisterPeer(Peer{
		PeerID:    "node3-piotrek",
		Name:      "Node 3 Piotrek",
		URL:       "http://172.30.15.11:11434",
		Transport: "http",
		Status:    "active",
	})
	return m
}

// Crypto returns the E2EEncryption engine attached to this MeshEngine.
func (m *MeshEngine) Crypto() *E2EEncryption {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.crypto
}

// SetEncryptionPassword updates the PBKDF2-AES-256-GCM key used for E2E Mesh transfers.
func (m *MeshEngine) SetEncryptionPassword(password string) {
	m.mu.Lock()
	m.crypto = NewE2EEncryption(password)
	m.mu.Unlock()
}

// SetStorage hot-swaps the underlying database-agnostic storage port used by RAE-Mesh.
func (m *MeshEngine) SetStorage(store ports.MemoryStoragePort) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store = store
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

// ResolveCRDTLWW performs Last-Write-Wins Element-Set conflict resolution between two MemoryItems.
func ResolveCRDTLWW(existing, incoming models.MemoryItem) models.MemoryItem {
	if incoming.CreatedAt.After(existing.CreatedAt) {
		return incoming
	}
	if incoming.CreatedAt.Equal(existing.CreatedAt) && incoming.ContentHash > existing.ContentHash {
		return incoming
	}
	return existing
}

// ReceiveMemories implements the CMT (Consensual Memory Transfer) receiver protocol with E2E decryption and CRDT LWW resolution.
func (m *MeshEngine) ReceiveMemories(ctx context.Context, token string, items []models.MemoryItem) (int, int, error) {
	peerID, err := m.VerifyCapabilityToken(token, "mesh:write")
	if err != nil {
		return 0, 0, err
	}
	m.mu.RLock()
	store := m.store
	crypto := m.crypto
	m.mu.RUnlock()

	accepted, rejected := 0, 0
	for i := range items {
		item := items[i]
		if item.IsEncrypted && len(item.EncryptedBytes) > 0 && crypto != nil {
			if dec, err := crypto.DecryptMemory(item); err == nil {
				item = dec
			} else {
				rejected++
				continue
			}
		}
		if err := m.guard.ValidateMeshTransfer(item); err != nil {
			rejected++
			continue
		}
		if item.ID != "" {
			if existing, err := store.GetMemory(ctx, item.ID); err == nil && existing != nil {
				item = ResolveCRDTLWW(*existing, item)
			}
		}
		if item.Envelope == nil {
			item.Envelope = &models.ContextEnvelope{
				OriginDevice: peerID,
				TrustLevel:   models.TrustHigh,
				Version:      1,
			}
		}
		if err := store.StoreMemory(ctx, &item); err != nil {
			rejected++
			continue
		}
		accepted++
	}
	return accepted, rejected, nil
}

// ComputeMerkleTree computes both the global Merkle Root and 16 hex-bucket hashes ("0".."f") for delta sync.
func (m *MeshEngine) ComputeMerkleTree(ctx context.Context, project string) (MerkleTreeSnapshot, map[string][]models.MemoryItem, error) {
	m.mu.RLock()
	store := m.store
	m.mu.RUnlock()

	items, err := store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		TopK:    5000,
	})
	if err != nil {
		return MerkleTreeSnapshot{}, nil, err
	}

	hexDigits := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f"}
	bucketHashesList := make(map[string][]string, 16)
	bucketItems := make(map[string][]models.MemoryItem, 16)
	for _, d := range hexDigits {
		bucketHashesList[d] = nil
	}

	var allHashes []string
	exportableCount := 0
	for _, sc := range items {
		mem := sc.Memory
		if err := m.guard.ValidateMeshTransfer(mem); err != nil {
			continue
		}
		ch := mem.ContentHash
		if ch == "" {
			sum := sha256.Sum256([]byte(mem.Content))
			ch = hex.EncodeToString(sum[:])
			mem.ContentHash = ch
		}
		prefix := strings.ToLower(ch[:1])
		bucketHashesList[prefix] = append(bucketHashesList[prefix], ch)
		bucketItems[prefix] = append(bucketItems[prefix], mem)
		allHashes = append(allHashes, ch)
		exportableCount++
	}

	bucketDigests := make(map[string]string, 16)
	for _, d := range hexDigits {
		list := bucketHashesList[d]
		sort.Strings(list)
		bh := sha256.New()
		for _, ch := range list {
			_, _ = bh.Write([]byte(ch))
		}
		bucketDigests[d] = hex.EncodeToString(bh.Sum(nil))
	}

	sort.Strings(allHashes)
	rootHash := sha256.New()
	for _, ch := range allHashes {
		_, _ = rootHash.Write([]byte(ch))
	}

	return MerkleTreeSnapshot{
		MerkleRoot:      hex.EncodeToString(rootHash.Sum(nil)),
		BucketHashes:    bucketDigests,
		ExportableCount: exportableCount,
	}, bucketItems, nil
}

// DiffMerkleBuckets compares local 16-bucket hashes with remote 16-bucket hashes and returns only the delta MemoryItems.
func (m *MeshEngine) DiffMerkleBuckets(ctx context.Context, project string, remoteBuckets map[string]string) ([]string, []models.MemoryItem, MerkleTreeSnapshot, error) {
	snap, bucketItems, err := m.ComputeMerkleTree(ctx, project)
	if err != nil {
		return nil, nil, MerkleTreeSnapshot{}, err
	}
	var changedBuckets []string
	var deltaItems []models.MemoryItem
	for bKey, localHash := range snap.BucketHashes {
		remoteHash, exists := remoteBuckets[bKey]
		if !exists || remoteHash != localHash {
			if len(bucketItems[bKey]) > 0 {
				changedBuckets = append(changedBuckets, bKey)
				deltaItems = append(deltaItems, bucketItems[bKey]...)
			}
		}
	}
	sort.Strings(changedBuckets)
	return changedBuckets, deltaItems, snap, nil
}

// SyncWithPeer computes the 16-bucket Merkle delta for a specific peer, attempts a non-blocking push if reachable, and updates LastSync.
func (m *MeshEngine) SyncWithPeer(ctx context.Context, peerID, project string) (map[string]interface{}, error) {
	snap, bucketItems, err := m.ComputeMerkleTree(ctx, project)
	if err != nil {
		return nil, err
	}
	var allExportable []models.MemoryItem
	for _, items := range bucketItems {
		allExportable = append(allExportable, items...)
	}

	m.mu.Lock()
	p, ok := m.peers[peerID]
	if !ok {
		p = Peer{PeerID: peerID, Name: peerID, Transport: "http", Status: "active"}
	}
	p.LastSync = time.Now().UTC()
	m.peers[peerID] = p
	m.mu.Unlock()

	remotePushed := false
	if strings.HasPrefix(p.URL, "http://127.0.0.1") || strings.HasPrefix(p.URL, "http://localhost") {
		tok, _ := m.GenerateCapabilityToken("local-mesh-node", "mesh:write", 10*time.Minute)
		body, _ := json.Marshal(map[string]interface{}{
			"token":    tok,
			"memories": allExportable,
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.URL, "/")+"/v2/mesh/sync/receive", bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err := m.client.Do(req); err == nil {
				_ = resp.Body.Close()
				remotePushed = resp.StatusCode == http.StatusOK
			}
		}
	}

	return map[string]interface{}{
		"peer_id":         peerID,
		"status":          "synced",
		"merkle_root":     snap.MerkleRoot,
		"bucket_hashes":   snap.BucketHashes,
		"synced_memories": snap.ExportableCount,
		"remote_pushed":   remotePushed,
		"last_sync":       p.LastSync,
	}, nil
}

// ComputeMerkleRoot computes the SHA-256 Merkle root over all exportable (non-RESTRICTED, non-Working) memories.
func (m *MeshEngine) ComputeMerkleRoot(ctx context.Context, project string) (string, []models.MemoryItem, error) {
	snap, bucketItems, err := m.ComputeMerkleTree(ctx, project)
	if err != nil {
		return "", nil, err
	}
	var exportable []models.MemoryItem
	for _, items := range bucketItems {
		exportable = append(exportable, items...)
	}
	return snap.MerkleRoot, exportable, nil
}
