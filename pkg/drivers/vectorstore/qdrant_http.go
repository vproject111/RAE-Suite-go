package vectorstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// QdrantHTTPClient implements a pure-Go HTTP REST client for Qdrant (:6333) supporting
// Named Vectors ("dense" 384d, "nomic" 768d) upsert, search, and delete operations.
type QdrantHTTPClient struct {
	BaseURL    string
	Collection string
	Client     *http.Client
}

// NewQdrantHTTPClient creates a Qdrant REST client from environment variables or defaults.
func NewQdrantHTTPClient(baseURL, collection string) *QdrantHTTPClient {
	if baseURL == "" {
		host := os.Getenv("QDRANT_HOST")
		if host == "" {
			host = "127.0.0.1"
		}
		port := os.Getenv("QDRANT_PORT")
		if port == "" {
			port = "6333"
		}
		if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
			baseURL = host
		} else {
			baseURL = fmt.Sprintf("http://%s:%s", host, port)
		}
	}
	if collection == "" {
		collection = "memories"
	}
	return &QdrantHTTPClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Collection: collection,
		Client:     &http.Client{Timeout: 2 * time.Second},
	}
}

// EnsureCollection creates the Named Vectors collection in Qdrant if it does not already exist.
func (q *QdrantHTTPClient) EnsureCollection() error {
	url := fmt.Sprintf("%s/collections/%s", q.BaseURL, q.Collection)
	resp, err := q.Client.Get(url)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
	}
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"dense": map[string]interface{}{"size": 384, "distance": "Cosine"},
			"nomic": map[string]interface{}{"size": 768, "distance": "Cosine"},
		},
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp2, err := q.Client.Do(req)
	if err != nil {
		return err
	}
	_ = resp2.Body.Close()
	return nil
}

// UpsertPoint stores a memory point with Named Vectors and metadata payload in Qdrant.
func (q *QdrantHTTPClient) UpsertPoint(id string, vectorName string, vec []float32, payload map[string]interface{}) error {
	if vectorName == "" {
		if len(vec) == 768 {
			vectorName = "nomic"
		} else {
			vectorName = "dense"
		}
	}
	// Pad or trim to 384d for "dense" or 768d for "nomic"
	targetDim := 384
	if vectorName == "nomic" {
		targetDim = 768
	}
	normalized := make([]float32, targetDim)
	for i := 0; i < targetDim && i < len(vec); i++ {
		normalized[i] = vec[i]
	}

	url := fmt.Sprintf("%s/collections/%s/points?wait=false", q.BaseURL, q.Collection)
	reqBody := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id": id,
				"vector": map[string]interface{}{
					vectorName: normalized,
				},
				"payload": payload,
			},
		},
	}
	raw, _ := json.Marshal(reqBody)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.Client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant upsert status %d", resp.StatusCode)
	}
	return nil
}

// SearchPoints executes a nearest-neighbor cosine search in Qdrant over the specified Named Vector space.
func (q *QdrantHTTPClient) SearchPoints(vectorName string, queryVec []float32, topK int) ([]string, error) {
	if vectorName == "" {
		if len(queryVec) == 768 {
			vectorName = "nomic"
		} else {
			vectorName = "dense"
		}
	}
	targetDim := 384
	if vectorName == "nomic" {
		targetDim = 768
	}
	normalized := make([]float32, targetDim)
	for i := 0; i < targetDim && i < len(queryVec); i++ {
		normalized[i] = queryVec[i]
	}
	if topK <= 0 {
		topK = 10
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", q.BaseURL, q.Collection)
	reqBody := map[string]interface{}{
		"vector": map[string]interface{}{
			"name":   vectorName,
			"vector": normalized,
		},
		"limit":        topK,
		"with_payload": true,
	}
	raw, _ := json.Marshal(reqBody)
	resp, err := q.Client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant search status %d", resp.StatusCode)
	}
	var parsed struct {
		Result []struct {
			ID interface{} `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(parsed.Result))
	for _, r := range parsed.Result {
		ids = append(ids, fmt.Sprintf("%v", r.ID))
	}
	return ids, nil
}

// DeletePoint deletes a point by ID from Qdrant.
func (q *QdrantHTTPClient) DeletePoint(id string) error {
	url := fmt.Sprintf("%s/collections/%s/points/delete?wait=false", q.BaseURL, q.Collection)
	reqBody := map[string]interface{}{
		"points": []string{id},
	}
	raw, _ := json.Marshal(reqBody)
	resp, err := q.Client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

// LiveEmbeddingClient generates embeddings via live Ollama (/api/embeddings) or OpenRouter/LiteLLM (/v1/embeddings).
type LiveEmbeddingClient struct {
	OllamaURL string
	Model     string
	Client    *http.Client
}

func NewLiveEmbeddingClient() *LiveEmbeddingClient {
	ollamaURL := os.Getenv("OLLAMA_API_URL")
	if ollamaURL == "" {
		ollamaURL = "http://127.0.0.1:11434"
	}
	model := os.Getenv("RAE_EMBEDDING_MODEL")
	if model == "" {
		model = "nomic-embed-text"
	}
	return &LiveEmbeddingClient{
		OllamaURL: strings.TrimRight(ollamaURL, "/"),
		Model:     model,
		Client:    &http.Client{Timeout: 1500 * time.Millisecond},
	}
}

// Embed queries Ollama /api/embeddings if reachable.
func (e *LiveEmbeddingClient) Embed(text string) ([]float32, error) {
	url := e.OllamaURL + "/api/embeddings"
	raw, _ := json.Marshal(map[string]interface{}{
		"model":  e.Model,
		"prompt": text,
	})
	resp, err := e.Client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama status %d", resp.StatusCode)
	}
	var parsed struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding from ollama")
	}
	return parsed.Embedding, nil
}
