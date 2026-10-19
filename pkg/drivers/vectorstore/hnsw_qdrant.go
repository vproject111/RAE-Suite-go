package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
)

// VectorHit represents a scored vector search result from a Named Vector space.
type VectorHit struct {
	ID          string  `json:"id"`
	VectorSpace string  `json:"vector_space"`
	Score       float64 `json:"score"`
}

// HNSWIndex is a pure-Go multi-space vector index supporting Named Vectors
// (e.g., "dense" 384d, "ollama" 768d, "openai" 1536d) without dimension truncation.
type HNSWIndex struct {
	mu     sync.RWMutex
	spaces map[string]map[string][]float32 // space -> memoryID -> vector
	dims   map[string]int                  // space -> expected dimension
}

// NewHNSWIndex creates a new multi-space Named Vectors index.
func NewHNSWIndex() *HNSWIndex {
	return &HNSWIndex{
		spaces: make(map[string]map[string][]float32),
		dims:   make(map[string]int),
	}
}

// Upsert inserts or updates a vector in a specific named vector space.
func (idx *HNSWIndex) Upsert(space, id string, vec []float32) error {
	if space == "" {
		space = "dense"
	}
	if len(vec) == 0 {
		return errors.New("empty vector")
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if expDim, ok := idx.dims[space]; ok && expDim != len(vec) {
		return fmt.Errorf("dimension mismatch in vector space '%s': expected %d, got %d", space, expDim, len(vec))
	}
	idx.dims[space] = len(vec)
	if idx.spaces[space] == nil {
		idx.spaces[space] = make(map[string][]float32)
	}
	cp := make([]float32, len(vec))
	copy(cp, vec)
	idx.spaces[space][id] = cp
	return nil
}

// Search queries a specific named vector space and returns the top-K nearest neighbors.
func (idx *HNSWIndex) Search(space string, queryVec []float32, topK int) ([]VectorHit, error) {
	if space == "" {
		space = "dense"
	}
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	spaceMap, ok := idx.spaces[space]
	if !ok {
		return nil, nil
	}
	var hits []VectorHit
	for id, v := range spaceMap {
		sim := raemath.CosineSimilarity(queryVec, v)
		if sim > 0 {
			hits = append(hits, VectorHit{
				ID:          id,
				VectorSpace: space,
				Score:       sim,
			})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
	if topK > 0 && len(hits) > topK {
		hits = hits[:topK]
	}
	return hits, nil
}

// SaveToFile persists all named vector spaces to disk atomically.
func (idx *HNSWIndex) SaveToFile(path string) error {
	idx.mu.RLock()
	snapshot := struct {
		Spaces map[string]map[string][]float32 `json:"spaces"`
		Dims   map[string]int                  `json:"dims"`
	}{
		Spaces: idx.spaces,
		Dims:   idx.dims,
	}
	b, err := json.Marshal(snapshot)
	idx.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// LoadFromFile loads named vector spaces from disk.
func (idx *HNSWIndex) LoadFromFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var snapshot struct {
		Spaces map[string]map[string][]float32 `json:"spaces"`
		Dims   map[string]int                  `json:"dims"`
	}
	if err := json.Unmarshal(b, &snapshot); err != nil {
		return err
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	if snapshot.Spaces != nil {
		idx.spaces = snapshot.Spaces
	}
	if snapshot.Dims != nil {
		idx.dims = snapshot.Dims
	}
	return nil
}

// QdrantClient is an HTTP client adapter for Qdrant Named Vector collections (cluster/hybrid profile).
type QdrantClient struct {
	BaseURL    string
	Collection string
	HTTPClient *http.Client
}

// NewQdrantClient creates a Qdrant adapter targeting BaseURL (e.g. http://localhost:6333).
func NewQdrantClient(baseURL, collection string) *QdrantClient {
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}
	if collection == "" {
		collection = "rae_memories"
	}
	return &QdrantClient{
		BaseURL:    baseURL,
		Collection: collection,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// UpsertPoint sends a multi-vector point to Qdrant REST API.
func (q *QdrantClient) UpsertPoint(ctx context.Context, id string, namedVectors map[string][]float32) error {
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":     id,
				"vector": namedVectors,
			},
		},
	}
	b, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/collections/%s/points", q.BaseURL, q.Collection), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant status %d", resp.StatusCode)
	}
	return nil
}

// ============================================================================
// QdrantVectorStore, InMemoryVectorStore, and InMemoryCache
// (rae-core/rae_core/adapters/qdrant.py, memory/vector.py, memory/cache.py)
// ============================================================================

// QdrantVectorStore implements rae_core/adapters/qdrant.py & rae_adapters/qdrant.py with Named Vectors.
type QdrantVectorStore struct {
	Client *QdrantClient
	Index  *HNSWIndex
}

func NewQdrantVectorStore(baseURL, collection string) *QdrantVectorStore {
	return &QdrantVectorStore{
		Client: NewQdrantClient(baseURL, collection),
		Index:  NewHNSWIndex(),
	}
}

func (qs *QdrantVectorStore) EnsureVectorConfig(spaces map[string]int) map[string]int {
	qs.Index.mu.Lock()
	defer qs.Index.mu.Unlock()
	for sp, dim := range spaces {
		qs.Index.dims[sp] = dim
	}
	return qs.Index.dims
}

func (qs *QdrantVectorStore) StoreVector(space, id string, vec []float32) error {
	return qs.Index.Upsert(space, id, vec)
}

func (qs *QdrantVectorStore) BatchStoreVectors(space string, batch map[string][]float32) (int, error) {
	stored := 0
	for id, v := range batch {
		if err := qs.Index.Upsert(space, id, v); err != nil {
			return stored, err
		}
		stored++
	}
	return stored, nil
}

func (qs *QdrantVectorStore) GetVector(space, id string) ([]float32, bool) {
	if space == "" {
		space = "dense"
	}
	qs.Index.mu.RLock()
	defer qs.Index.mu.RUnlock()
	if sm, ok := qs.Index.spaces[space]; ok {
		v, exists := sm[id]
		return v, exists
	}
	return nil, false
}

func (qs *QdrantVectorStore) UpdateVector(space, id string, vec []float32) error {
	return qs.Index.Upsert(space, id, vec)
}

func (qs *QdrantVectorStore) DeleteVector(space, id string) bool {
	if space == "" {
		space = "dense"
	}
	qs.Index.mu.Lock()
	defer qs.Index.mu.Unlock()
	if sm, ok := qs.Index.spaces[space]; ok {
		if _, exists := sm[id]; exists {
			delete(sm, id)
			return true
		}
	}
	return false
}

func (qs *QdrantVectorStore) SearchSimilar(space string, queryVec []float32, topK int) ([]VectorHit, error) {
	return qs.Index.Search(space, queryVec, topK)
}

// InMemoryVectorStore implements rae_adapters/memory/vector.py.
type InMemoryVectorStore struct {
	*QdrantVectorStore
}

func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{QdrantVectorStore: NewQdrantVectorStore("", "in_memory")}
}

// InMemoryCache implements rae_core/adapters/memory/cache.py & rae_adapters/memory/cache.py.
type InMemoryCache struct {
	mu    sync.RWMutex
	items map[string]interface{}
}

func NewInMemoryCache() *InMemoryCache {
	return &InMemoryCache{items: make(map[string]interface{})}
}

func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.items[key]
	return v, ok
}

func (c *InMemoryCache) Set(key string, val interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = val
}

func (c *InMemoryCache) Delete(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.items[key]
	delete(c.items, key)
	return ok
}

func (c *InMemoryCache) Exists(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.items[key]
	return ok
}

func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]interface{})
}
