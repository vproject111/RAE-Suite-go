package security

import (
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestHardFramesGuardAndAdmissionPolicy(t *testing.T) {
	guard := NewHardFramesGuard("test-secret-key-for-aes-256-gcm")

	// RESTRICTED outside Working must fail
	badRec := &models.MemoryItem{
		ID:        "r1",
		Content:   "secret token",
		Layer:     models.LayerSemantic,
		InfoClass: models.InfoRestricted,
	}
	if err := guard.ValidateAndPrepareMemory(badRec); err == nil {
		t.Fatalf("expected RESTRICTED policy violation on semantic layer")
	}

	// RESTRICTED inside Working must succeed and encrypt content
	goodRec := &models.MemoryItem{
		ID:        "r2",
		Content:   "classified payload",
		Layer:     models.LayerWorking,
		InfoClass: models.InfoRestricted,
	}
	if err := guard.ValidateAndPrepareMemory(goodRec); err != nil {
		t.Fatalf("expected RESTRICTED in working layer to pass, got %v", err)
	}
	if !goodRec.IsEncrypted || len(goodRec.EncryptedBytes) == 0 {
		t.Fatalf("expected IsEncrypted=true")
	}
	plain, err := guard.DecryptAESGCM(goodRec.EncryptedBytes)
	if err != nil || string(plain) != "classified payload" {
		t.Fatalf("DecryptAESGCM failed: %v, plain=%q", err, string(plain))
	}

	if err := guard.ValidateProjectRelativePath("/etc/passwd"); err == nil {
		t.Fatalf("expected C6 absolute path violation")
	}
	if err := guard.ValidateProjectRelativePath("pkg/raecore/models.go"); err != nil {
		t.Fatalf("expected relative path to pass, got %v", err)
	}

	// Negative tests for Guard: default secret, mesh transfer, epistemic grace period, short ciphertext, HMAC
	defGuard := NewHardFramesGuard("")
	sig := defGuard.SignTokenHMAC("payload-123")
	if !defGuard.VerifyTokenHMAC("payload-123", sig) || defGuard.VerifyTokenHMAC("payload-123", "bad-sig") {
		t.Fatalf("unexpected HMAC verification result")
	}
	if _, err := defGuard.DecryptAESGCM([]byte("short")); err == nil {
		t.Fatalf("expected error on short ciphertext")
	}
	if err := defGuard.ValidateMeshTransfer(models.MemoryItem{Layer: models.LayerWorking}); err != ErrRestrictedMeshEgress {
		t.Fatalf("expected ErrRestrictedMeshEgress for working layer, got %v", err)
	}
	if err := defGuard.ValidateMeshTransfer(models.MemoryItem{Layer: models.LayerSemantic, InfoClass: models.InfoPublic}); err != nil {
		t.Fatalf("expected public semantic item to pass mesh transfer, got %v", err)
	}
}

