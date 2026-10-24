package vectorstore_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestVectorStoreFullCoverage(t *testing.T) {
	if len(vectorstore.RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes")
	}

	// HNSW edge cases
	idx := vectorstore.NewHNSWIndex()
	if err := idx.Upsert("", "m1", []float32{}); err == nil {
		t.Fatal("expected error for empty vector")
	}
	_ = idx.Upsert("", "m1", []float32{1, 0})
	_ = idx.Upsert("", "m2", []float32{0.9, 0.1})
	hits, _ := idx.Search("", []float32{1, 0}, 1)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit with topK=1, got %d", len(hits))
	}
	if none, _ := idx.Search("missing_space", []float32{1, 0}, 5); none != nil {
		t.Fatal("expected nil for missing space")
	}
	if err := idx.LoadFromFile("/nonexistent/file.json"); err == nil {
		t.Fatal("expected error loading nonexistent file")
	}
	badFile := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(badFile, []byte("{bad"), 0o600)
	if err := idx.LoadFromFile(badFile); err == nil {
		t.Fatal("expected error loading invalid JSON")
	}

	// QdrantClient defaults and error status
	qcDef := vectorstore.NewQdrantClient("", "")
	if qcDef.Collection != "rae_memories" {
		t.Fatal("unexpected default collection")
	}
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errSrv.Close()
	qcErr := vectorstore.NewQdrantClient(errSrv.URL, "")
	if err := qcErr.UpsertPoint(context.Background(), "m1", map[string][]float32{"dense": {1}}); err == nil {
		t.Fatal("expected error on 500 status")
	}

	// QdrantVectorStore & InMemoryVectorStore
	qs := vectorstore.NewQdrantVectorStore("", "")
	qs.EnsureVectorConfig(map[string]int{"dense": 2})
	if _, err := qs.BatchStoreVectors("dense", map[string][]float32{"a": {1, 0}, "b": {0, 1}}); err != nil {
		t.Fatalf("BatchStoreVectors failed: %v", err)
	}
	if _, err := qs.BatchStoreVectors("dense", map[string][]float32{"bad": {1, 0, 0}}); err == nil {
		t.Fatal("expected error on batch dimension mismatch")
	}
	if _, ok := qs.GetVector("", "a"); !ok {
		t.Fatal("GetVector failed")
	}
	if _, ok := qs.GetVector("missing", "a"); ok {
		t.Fatal("expected false for missing space")
	}
	_ = qs.UpdateVector("dense", "a", []float32{0.8, 0.2})
	_, _ = qs.SearchSimilar("dense", []float32{1, 0}, 5)
	if !qs.DeleteVector("", "a") || qs.DeleteVector("", "a") || qs.DeleteVector("missing", "a") {
		t.Fatal("DeleteVector behavior mismatch")
	}

	imv := vectorstore.NewInMemoryVectorStore()
	_ = imv.StoreVector("dense", "id1", []float32{1, 0})
	_ = imv.UpdateVector("dense", "id1", []float32{0.9, 0.1})
	_, _ = imv.GetVector("dense", "id1")
	_, _ = imv.BatchStoreVectors("dense", map[string][]float32{"id2": {0, 1}})
	_, _ = imv.SearchSimilar("dense", []float32{1, 0}, 2)
	_ = imv.DeleteVector("dense", "id2")
	imv.ClearAll()

	// InMemoryCache
	cache := vectorstore.NewInMemoryCache()
	cache.Set("k1", "val1")
	if !cache.Exists("k1") {
		t.Fatal("Exists failed")
	}
	if v, ok := cache.Get("k1"); !ok || v != "val1" {
		t.Fatal("Get failed")
	}
	if !cache.Delete("k1") || cache.Delete("k1") {
		t.Fatal("Delete failed")
	}
	cache.Set("k2", "val2")
	cache.Clear()
	if cache.Exists("k2") {
		t.Fatal("Clear failed")
	}

	// QdrantHTTPClient & LiveEmbeddingClient with mock server
	getCollectionCalled := 0
	mockQdrantAndOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/collections/"):
			getCollectionCalled++
			if getCollectionCalled == 1 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/collections/"):
			if strings.Contains(r.URL.Path, "fail_col") {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":{"status":"acknowledged"}}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/points/search"):
			if strings.Contains(r.URL.Path, "fail_col") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":[{"id":"pt-1"},{"id":"pt-2"}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/points/delete"):
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && r.URL.Path == "/api/embeddings":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"embedding":[0.25, 0.75]}`))
		}
	}))
	defer mockQdrantAndOllama.Close()

	_ = os.Setenv("QDRANT_HOST", mockQdrantAndOllama.URL)
	qhc := vectorstore.NewQdrantHTTPClient("", "")
	_ = os.Unsetenv("QDRANT_HOST")

	// First call: GET 404 -> PUT 200
	if err := qhc.EnsureCollection(); err != nil {
		t.Fatalf("EnsureCollection 1st call failed: %v", err)
	}
	// Second call: GET 200
	if err := qhc.EnsureCollection(); err != nil {
		t.Fatalf("EnsureCollection 2nd call failed: %v", err)
	}

	vec768 := make([]float32, 768)
	vec768[0] = 1.0
	if err := qhc.UpsertPoint("p1", "", []float32{1, 0, 0}, map[string]interface{}{"k": "v"}); err != nil {
		t.Fatalf("UpsertPoint 384d failed: %v", err)
	}
	if err := qhc.UpsertPoint("p2", "", vec768, nil); err != nil {
		t.Fatalf("UpsertPoint 768d failed: %v", err)
	}
	found, err := qhc.SearchPoints("", []float32{1, 0}, 0)
	if err != nil || len(found) != 2 {
		t.Fatalf("SearchPoints 384d failed: %v, %v", found, err)
	}
	if _, err := qhc.SearchPoints("", vec768, 5); err != nil {
		t.Fatalf("SearchPoints 768d failed: %v", err)
	}
	if err := qhc.DeletePoint("p1"); err != nil {
		t.Fatalf("DeletePoint failed: %v", err)
	}

	qhcFail := vectorstore.NewQdrantHTTPClient(mockQdrantAndOllama.URL, "fail_col")
	if err := qhcFail.UpsertPoint("p1", "dense", []float32{1}, nil); err == nil {
		t.Fatal("expected UpsertPoint error on fail_col")
	}
	if _, err := qhcFail.SearchPoints("dense", []float32{1}, 5); err == nil {
		t.Fatal("expected SearchPoints error on fail_col")
	}

	// LiveEmbeddingClient
	_ = os.Setenv("OLLAMA_API_URL", mockQdrantAndOllama.URL)
	lec := vectorstore.NewLiveEmbeddingClient()
	_ = os.Unsetenv("OLLAMA_API_URL")
	emb, err := lec.Embed("hello")
	if err != nil || len(emb) != 2 {
		t.Fatalf("LiveEmbeddingClient.Embed failed: %v, %v", emb, err)
	}

	emptyOllama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"embedding":[]}`))
	}))
	defer emptyOllama.Close()
	lecEmpty := &vectorstore.LiveEmbeddingClient{OllamaURL: emptyOllama.URL, Model: "nomic", Client: emptyOllama.Client()}
	if _, err := lecEmpty.Embed("hello"); err == nil {
		t.Fatal("expected error for empty embedding")
	}
	lecErr := &vectorstore.LiveEmbeddingClient{OllamaURL: errSrv.URL, Model: "nomic", Client: errSrv.Client()}
	if _, err := lecErr.Embed("hello"); err == nil {
		t.Fatal("expected error for 500 status from ollama")
	}
}

