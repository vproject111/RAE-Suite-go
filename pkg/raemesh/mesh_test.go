package raemesh

import (
	"context"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

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
