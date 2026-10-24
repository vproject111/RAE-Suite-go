package raemesh

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

type failingMeshStore struct {
	failQuery bool
	failStore bool
}

func (f *failingMeshStore) StoreMemory(_ context.Context, _ *models.MemoryItem) error {
	if f.failStore {
		return errors.New("store failure")
	}
	return nil
}
func (f *failingMeshStore) GetMemory(_ context.Context, _ string) (*models.MemoryItem, error) {
	return nil, nil
}
func (f *failingMeshStore) DeleteMemory(_ context.Context, _ string) error { return nil }
func (f *failingMeshStore) QueryMemories(_ context.Context, _ ports.QueryFilter) ([]models.ScoredMemoryItem, error) {
	if f.failQuery {
		return nil, errors.New("query failure")
	}
	return []models.ScoredMemoryItem{{Memory: models.MemoryItem{ID: "1", Content: "no hash content", Layer: models.LayerSemantic}}}, nil
}
func (f *failingMeshStore) GetMemoryStats(_ context.Context, _ string) (ports.MemoryStats, error) {
	return ports.MemoryStats{}, nil
}
func (f *failingMeshStore) DeleteExpiredSensory(_ context.Context) (int, error) { return 0, nil }
func (f *failingMeshStore) AddGraphEdge(_ context.Context, _, _, _ string) error { return nil }
func (f *failingMeshStore) GetGraphNeighbors(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (f *failingMeshStore) GetGraphAdjacency(_ context.Context) (map[string][]string, error) {
	return nil, nil
}

func TestCMTCapabilityTokensAndMeshReceive(t *testing.T) {
	guard := security.NewHardFramesGuard("mesh-test-secret")
	store := memorystore.NewUnifiedStore(nil, guard)
	engine := NewMeshEngine(store, guard)

	token, err := engine.GenerateCapabilityToken("node1-lumina", "mesh:write", 10*time.Minute)
	if err != nil {
		t.Fatalf("GenerateCapabilityToken error: %v", err)
	}
	peerID, err := engine.VerifyCapabilityToken(token, "mesh:write")
	if err != nil || peerID != "node1-lumina" {
		t.Fatalf("VerifyCapabilityToken failed: peer=%s, err=%v", peerID, err)
	}

	ctx := context.Background()
	accepted, rejected, err := engine.ReceiveMemories(ctx, token, []models.MemoryItem{
		{
			ID:        "mesh-ok-1",
			Content:   "Exportable semantic fact",
			Layer:     models.LayerSemantic,
			InfoClass: models.InfoInternal,
			Project:   "p1",
		},
		{
			ID:        "mesh-bad-1",
			Content:   "Restricted working secret",
			Layer:     models.LayerWorking,
			InfoClass: models.InfoRestricted,
			Project:   "p1",
		},
	})
	if err != nil || accepted != 1 || rejected != 1 {
		t.Fatalf("expected accepted=1, rejected=1, got accepted=%d, rejected=%d, err=%v", accepted, rejected, err)
	}

	syncRes, err := engine.SyncWithPeer(ctx, "node1-lumina", "p1")
	if err != nil || syncRes["synced_memories"] != 1 {
		t.Fatalf("SyncWithPeer failed: %+v, err=%v", syncRes, err)
	}

	// Test 16-bucket Merkle delta diffing
	changed, delta, snap, err := engine.DiffMerkleBuckets(ctx, "p1", map[string]string{})
	if err != nil || len(changed) != 1 || len(delta) != 1 || len(snap.BucketHashes) != 16 {
		t.Fatalf("DiffMerkleBuckets unexpected result: changed=%v delta=%d buckets=%d err=%v", changed, len(delta), len(snap.BucketHashes), err)
	}
}

func TestMeshEngineFullCoverageAndEdgeCases(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	store := memorystore.NewUnifiedStore(nil, nil)
	engine := NewMeshEngine(store, nil)
	engine.SetEncryptionPassword("new-mesh-secret")
	engine.SetStorage(store)
	if engine.Crypto() == nil {
		t.Fatalf("expected non-nil Crypto()")
	}
	engine.RegisterPeer(Peer{PeerID: "custom-peer", URL: "http://127.0.0.1:1"})
	peers := engine.ListPeers()
	if len(peers) < 5 {
		t.Fatalf("expected at least 5 peers, got %d", len(peers))
	}

	// Token validation negative cases
	if _, err := engine.GenerateCapabilityToken("", "mesh:write", 0); err == nil {
		t.Fatalf("expected error for empty peerID")
	}
	tokDefTTL, err := engine.GenerateCapabilityToken("p1", "mesh:read", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := engine.VerifyCapabilityToken("malformed", "mesh:read"); err == nil {
		t.Fatalf("expected error on malformed token")
	}
	if _, err := engine.VerifyCapabilityToken("part1.badsig", "mesh:read"); err == nil {
		t.Fatalf("expected error on bad HMAC signature")
	}
	if _, err := engine.VerifyCapabilityToken(tokDefTTL, "mesh:write"); err == nil {
		t.Fatalf("expected error on insufficient scope")
	}
	// Craft an expired token signed with guard
	expClaims, _ := json.Marshal(CapabilityClaims{PeerID: "p1", Scope: "mesh:write", ExpiresAt: 100})
	expB64 := base64.RawURLEncoding.EncodeToString(expClaims)
	expTok := expB64 + "." + engine.guard.SignTokenHMAC(expB64)
	if _, err := engine.VerifyCapabilityToken(expTok, "mesh:write"); err == nil {
		t.Fatalf("expected error on expired token")
	}

	// Test SyncWithPeer HTTP push against a live httptest server with >50 exportable items
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	engine.RegisterPeer(Peer{PeerID: "live-peer", URL: srv.URL})
	for i := 0; i < 55; i++ {
		_ = store.StoreMemory(context.Background(), &models.MemoryItem{
			ID:        fmt.Sprintf("m-%d", i),
			Content:   fmt.Sprintf("Exportable content %d", i),
			Layer:     models.LayerSemantic,
			InfoClass: models.InfoPublic,
		})
	}
	syncLive, err := engine.SyncWithPeer(context.Background(), "live-peer", "")
	if err != nil || syncLive["remote_pushed"] != true {
		t.Fatalf("expected remote_pushed=true with httptest peer, got %+v (%v)", syncLive, err)
	}

	// CRDT LWW resolution
	now := time.Now().UTC()
	mOld := models.MemoryItem{ID: "1", Content: "old", CreatedAt: now.Add(-time.Minute), ContentHash: "aaa"}
	mNew := models.MemoryItem{ID: "1", Content: "new", CreatedAt: now, ContentHash: "bbb"}
	if ResolveCRDTLWW(mOld, mNew).Content != "new" {
		t.Fatalf("expected newer item to win LWW")
	}
	if ResolveCRDTLWW(mNew, mOld).Content != "new" {
		t.Fatalf("expected existing newer item to win LWW")
	}
	mTie := models.MemoryItem{ID: "1", Content: "tie", CreatedAt: now, ContentHash: "zzz"}
	if ResolveCRDTLWW(mNew, mTie).Content != "tie" {
		t.Fatalf("expected higher ContentHash to win LWW tie")
	}

	// ReceiveMemories with existing memory and encrypted memory, plus ComputeMerkleRoot
	ctx := context.Background()
	writeTok, _ := engine.GenerateCapabilityToken("p1", "mesh:write", time.Hour)
	encMem, _ := engine.Crypto().EncryptMemory(models.MemoryItem{
		ID:             "mesh-ok-1",
		Content:        "Updated via E2E mesh",
		Layer:          models.LayerSemantic,
		InfoClass:      models.InfoInternal,
		IsEncrypted:    true,
		EncryptedBytes: []byte("dummy"),
		CreatedAt:      now.Add(time.Minute),
	})
	encMem.IsEncrypted = true
	encMem.EncryptedBytes = []byte("non-empty")
	acc, _, err := engine.ReceiveMemories(ctx, writeTok, []models.MemoryItem{encMem})
	if err != nil || acc != 1 {
		t.Fatalf("expected encrypted memory receive to succeed, got acc=%d err=%v", acc, err)
	}
	// Receive again to trigger ResolveCRDTLWW against existing stored memory
	_, _, _ = engine.ReceiveMemories(ctx, writeTok, []models.MemoryItem{{
		ID:        "mesh-ok-1",
		Content:   "Second update",
		Layer:     models.LayerSemantic,
		InfoClass: models.InfoInternal,
		CreatedAt: now.Add(2 * time.Minute),
	}})

	root, expItems, err := engine.ComputeMerkleRoot(ctx, "")
	if err != nil || root == "" || len(expItems) == 0 {
		t.Fatalf("ComputeMerkleRoot failed: root=%s items=%d err=%v", root, len(expItems), err)
	}
	_, _ = engine.SyncWithPeer(ctx, "unregistered-peer", "")

	// PBKDF2 defaults and Decrypt error cases
	k := DeriveKeyPBKDF2([]byte("pw"), []byte("salt"), 0, 0)
	if len(k) != 32 {
		t.Fatalf("expected 32-byte key from DeriveKeyPBKDF2")
	}
	e2eDef := NewE2EEncryption("")
	if _, err := e2eDef.Decrypt("not-base64!!!"); err == nil {
		t.Fatalf("expected base64 error")
	}
	if _, err := e2eDef.Decrypt("c2hvcnQ="); err == nil {
		t.Fatalf("expected short ciphertext error")
	}
	if _, err := e2eDef.DecryptBatch([]models.MemoryItem{{Content: "bad"}}); err == nil {
		t.Fatalf("expected DecryptBatch error on invalid ciphertext")
	}

	// VerifyCapabilityToken invalid base64 and invalid JSON
	badB64 := "!!!not_b64!!!"
	if _, err := engine.VerifyCapabilityToken(badB64+"."+engine.guard.SignTokenHMAC(badB64), "mesh:write"); err == nil {
		t.Fatalf("expected base64 decode error")
	}
	notJSONB64 := base64.RawURLEncoding.EncodeToString([]byte("not-json"))
	if _, err := engine.VerifyCapabilityToken(notJSONB64+"."+engine.guard.SignTokenHMAC(notJSONB64), "mesh:write"); err == nil {
		t.Fatalf("expected json unmarshal error")
	}

	// ReceiveMemories with bad token, corrupt encrypted item, and store error
	if _, _, err := engine.ReceiveMemories(ctx, "bad.tok", nil); err == nil {
		t.Fatalf("expected ReceiveMemories error on invalid token")
	}
	_, rej, _ := engine.ReceiveMemories(ctx, writeTok, []models.MemoryItem{{
		ID:             "bad-enc",
		Content:        "not-encrypted-properly",
		IsEncrypted:    true,
		EncryptedBytes: []byte("123"),
	}})
	if rej != 1 {
		t.Fatalf("expected corrupt encrypted item to be rejected")
	}

	// Test failing store branches in ReceiveMemories, ComputeMerkleTree, DiffMerkleBuckets, SyncWithPeer, ComputeMerkleRoot
	fStore := &failingMeshStore{failStore: true}
	engine.SetStorage(fStore)
	_, rejStore, _ := engine.ReceiveMemories(ctx, writeTok, []models.MemoryItem{{
		ID:        "s1",
		Content:   "ok",
		Layer:     models.LayerSemantic,
		InfoClass: models.InfoPublic,
	}})
	if rejStore != 1 {
		t.Fatalf("expected rejected=1 when StoreMemory fails")
	}
	// QueryMemories returning item with empty ContentHash
	_, _, _ = engine.ComputeMerkleTree(ctx, "")

	fStore.failQuery = true
	if _, _, _, err := engine.DiffMerkleBuckets(ctx, "", nil); err == nil {
		t.Fatalf("expected DiffMerkleBuckets error when QueryMemories fails")
	}
	if _, err := engine.SyncWithPeer(ctx, "p1", ""); err == nil {
		t.Fatalf("expected SyncWithPeer error when QueryMemories fails")
	}
	if _, _, err := engine.ComputeMerkleRoot(ctx, ""); err == nil {
		t.Fatalf("expected ComputeMerkleRoot error when QueryMemories fails")
	}
}

