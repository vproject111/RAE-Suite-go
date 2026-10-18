package vectorstore_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/vectorstore"
)

func TestHNSWNamedVectorsAndPersistenceAndQdrantClient(t *testing.T) {
	idx := vectorstore.NewHNSWIndex()
	if err := idx.Upsert("dense", "m1", []float32{1.0, 0.0, 0.0}); err != nil {
		t.Fatalf("Upsert dense failed: %v", err)
	}
	if err := idx.Upsert("ollama", "m1", []float32{0.0, 1.0, 0.0, 0.0}); err != nil {
		t.Fatalf("Upsert ollama failed: %v", err)
	}
	// Dimension mismatch in 'dense' must fail
	if err := idx.Upsert("dense", "m2", []float32{1.0, 0.0}); err == nil {
		t.Fatal("expected dimension mismatch in 'dense' space to return error")
	}

	tmp := filepath.Join(t.TempDir(), "test.hnsw.json")
	if err := idx.SaveToFile(tmp); err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	idx2 := vectorstore.NewHNSWIndex()
	if err := idx2.LoadFromFile(tmp); err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	hits, err := idx2.Search("ollama", []float32{0.0, 1.0, 0.0, 0.0}, 5)
	if err != nil || len(hits) != 1 || hits[0].ID != "m1" {
		t.Fatalf("expected loaded HNSW index to return m1, got %v", hits)
	}

	// Test QdrantClient HTTP adapter against mock server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	qc := vectorstore.NewQdrantClient(srv.URL, "rae_memories")
	if err := qc.UpsertPoint(context.Background(), "m1", map[string][]float32{"dense": {1, 0, 0}}); err != nil {
		t.Fatalf("QdrantClient UpsertPoint failed: %v", err)
	}
}
