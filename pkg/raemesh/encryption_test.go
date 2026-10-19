package raemesh

import (
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestE2EEncryptionBatchRoundTrip(t *testing.T) {
	e2e := NewE2EEncryption("cluster-mesh-passphrase")
	batch := []models.MemoryItem{
		{ID: "m-1", Content: "Confidential architecture invariant for Node1 Lumina"},
		{ID: "m-2", Content: "Qdrant Named Vectors 768d configuration"},
	}
	enc, err := e2e.EncryptBatch(batch)
	if err != nil {
		t.Fatalf("EncryptBatch failed: %v", err)
	}
	if enc[0].Content == batch[0].Content {
		t.Fatalf("expected encrypted content to differ from plaintext")
	}
	dec, err := e2e.DecryptBatch(enc)
	if err != nil {
		t.Fatalf("DecryptBatch failed: %v", err)
	}
	if dec[0].Content != batch[0].Content || dec[1].Content != batch[1].Content {
		t.Fatalf("expected decrypted content to match original, got %+v", dec)
	}
}
