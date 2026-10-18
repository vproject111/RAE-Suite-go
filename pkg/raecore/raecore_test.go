package raecore_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/reflection"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/pkg/raemesh"
)

func TestAllEightMemoryLayersAndNormalization(t *testing.T) {
	layers := models.AllMemoryLayers()
	if len(layers) != 8 {
		t.Fatalf("expected 8 memory layers, got %d", len(layers))
	}
	if models.NormalizeLayer("Working") != models.LayerWorking {
		t.Fatal("expected Working -> working")
	}
	if models.NormalizeLayer("SEMANTIC") != models.LayerSemantic {
		t.Fatal("expected SEMANTIC -> semantic")
	}
	if models.NormalizeLayer("long_term") != models.LayerLTM {
		t.Fatal("expected long_term -> ltm")
	}
	if models.NormalizeLayer("procedural") != models.LayerProcedural {
		t.Fatal("expected procedural -> procedural")
	}
	if models.NormalizeLayer("knowledge_graph") != models.LayerGraph {
		t.Fatal("expected knowledge_graph -> graph")
	}
}

func TestUUIDv7AndULIDAndDeterministicCBOR(t *testing.T) {
	u1 := models.GenerateUUIDv7()
	if len(u1) != 36 {
		t.Fatalf("invalid UUIDv7 length: %s", u1)
	}
	ulid := models.GenerateULID()
	if len(ulid) != 26 {
		t.Fatalf("invalid ULID length: %s", ulid)
	}
	cid1 := models.ComputeVersionCID(map[string]string{"b": "2", "a": "1"})
	cid2 := models.ComputeVersionCID(map[string]string{"a": "1", "b": "2"})
	if cid1 != cid2 || !strings.HasPrefix(cid1, "mh:1220") {
		t.Fatalf("deterministic CBOR CID mismatch: %s vs %s", cid1, cid2)
	}
}

func TestHardFramesGuardAndRestrictedIsolation(t *testing.T) {
	guard := security.NewHardFramesGuard("test-secret")

	// 1. RESTRICTED outside working must fail
	bad := &models.MemoryItem{
		Content:   "Top secret API key",
		Layer:     models.LayerSemantic,
		InfoClass: models.InfoRestricted,
	}
	if err := guard.ValidateAndPrepareMemory(bad); err == nil {
		t.Fatal("expected error when storing RESTRICTED in semantic layer")
	}

	// 2. RESTRICTED inside working must succeed and encrypt with AES-256-GCM
	good := &models.MemoryItem{
		Content:   "Top secret session state",
		Layer:     models.LayerWorking,
		InfoClass: models.InfoRestricted,
	}
	if err := guard.ValidateAndPrepareMemory(good); err != nil {
		t.Fatalf("unexpected error for RESTRICTED in working layer: %v", err)
	}
	if !good.IsEncrypted || len(good.EncryptedBytes) == 0 {
		t.Fatal("expected RESTRICTED working memory to be encrypted")
	}
	dec, err := guard.DecryptAESGCM(good.EncryptedBytes)
	if err != nil || string(dec) != good.Content {
		t.Fatalf("AES-256-GCM decryption failed: %v", err)
	}

	// 3. C6 Path Validation
	if err := guard.ValidateProjectRelativePath("/home/grzegorz/secret.py"); err == nil {
		t.Fatal("expected absolute path to be rejected by C6 rule")
	}
	if err := guard.ValidateProjectRelativePath("pkg/raecore/models/models.go"); err != nil {
		t.Fatalf("valid relative path rejected: %v", err)
	}

	// 4. HMAC-SHA256 Token Signing
	sig := guard.SignTokenHMAC("req-1:approve:admin")
	if !guard.VerifyTokenHMAC("req-1:approve:admin", sig) {
		t.Fatal("expected HMAC token verification to succeed")
	}
}

func TestMathControllerAndReflectionL1ToL4(t *testing.T) {
	mc := raemath.NewMathController(models.MathL3)
	w := mc.UpdateReward("hyper_37", 0.95)
	if w.ActiveArm != "hyper_37" {
		t.Fatalf("expected active arm hyper_37, got %s", w.ActiveArm)
	}

	coord := reflection.NewCoordinator()
	rep := coord.Reflect(reflection.ReflectionPayload{
		AnswerDraft:      "System log shows always high throughput except during GC.",
		RetrievedSources: []string{"doc_architecture.md"},
		CoverageRatio:    0.85,
		UncertaintyLevel: 0.15,
		Analysis:         "Tune GC target percentage",
	})
	if rep.Blocked {
		t.Fatal("expected valid reflection payload not to be blocked")
	}
	if !strings.Contains(rep.SynthesizedLesson, "Lesson Learned") {
		t.Fatalf("expected L4 synthesized lesson, got %s", rep.SynthesizedLesson)
	}
}

func TestUnifiedStoreNamedVectorsAndMeshAndCRL(t *testing.T) {
	ctx := context.Background()
	mc := raemath.NewMathController(models.MathL3)
	guard := security.NewHardFramesGuard("mesh-secret")
	store := memorystore.NewUnifiedStore(mc, guard)

	// Store across multiple layers including Named Vectors
	_ = store.StoreMemory(ctx, &models.MemoryItem{
		ID:        "mem-sem-1",
		Content:   "Dreamsoft calculation engine pricing rules",
		Layer:     models.LayerSemantic,
		InfoClass: models.InfoInternal,
		Project:   "RAE-Suite",
		NamedVectors: map[string][]float32{
			"dense":  {1.0, 0.0, 0.0},
			"ollama": {0.0, 1.0, 0.0, 0.0},
		},
	})
	_ = store.StoreMemory(ctx, &models.MemoryItem{
		ID:        "mem-work-1",
		Content:   "Private session token",
		Layer:     models.LayerWorking,
		InfoClass: models.InfoRestricted,
		Project:   "RAE-Suite",
	})

	// Query Named Vector space "ollama"
	res, err := store.QueryMemories(ctx, ports.QueryFilter{
		Project:     "RAE-Suite",
		Strategy:    "multi_vector",
		VectorSpace: "ollama",
		QueryVector: []float32{0.0, 1.0, 0.0, 0.0},
		TopK:        5,
	})
	if err != nil || len(res) != 1 || res[0].Memory.ID != "mem-sem-1" {
		t.Fatalf("expected multi_vector search on 'ollama' space to match mem-sem-1, got %v", res)
	}

	// Verify MeshEngine excludes RESTRICTED Working memory
	mesh := raemesh.NewMeshEngine(store, guard)
	root, exp, err := mesh.ComputeMerkleRoot(ctx, "RAE-Suite")
	if err != nil || root == "" || len(exp) != 1 {
		t.Fatalf("expected 1 exportable memory in Mesh Merkle tree, got %d", len(exp))
	}

	// Verify RAE-CRL Epistemic DAG cycle rejection & Grace Period
	graceEnd := time.Now().UTC().Add(24 * time.Hour)
	trace := &models.EpistemicArtifact{
		ID:             "art-1",
		Project:        "doktorat-kubus",
		Type:           models.ArtifactTrace,
		Title:          "Initial tensor idea",
		Status:         models.StatusDraft,
		Visibility:     models.VisibilityPrivate,
		GracePeriodEnd: &graceEnd,
	}
	if err := store.SaveArtifact(ctx, trace); err != nil {
		t.Fatalf("SaveArtifact failed: %v", err)
	}
	if err := guard.ValidateEpistemicMeshTransfer(*trace, time.Now().UTC()); err == nil {
		t.Fatal("expected trace in 24h Grace Period to be blocked from Mesh transfer")
	}
	if err := store.LinkArtifacts(ctx, "art-1", "art-1", "refines"); err == nil {
		t.Fatal("expected self-loop in Epistemic DAG to be rejected")
	}
}
