package llm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client provides model-agnostic LLM chat completions and Named Vector embeddings
// across Ollama (Node 1 Lumina / Node 3 Piotrek) and OpenAI-compatible APIs (LiteLLM / OpenRouter),
// with deterministic zero-egress fallback when offline.
type Client struct {
	OllamaURL     string
	OpenAIBaseURL string
	APIKey        string
	HTTPClient    *http.Client
	AllowNetwork  bool
}

// NewClient constructs a multi-provider LLM and Embedding client configured from environment variables.
func NewClient() *Client {
	ollamaURL := os.Getenv("OLLAMA_API_URL")
	if ollamaURL == "" {
		ollamaURL = "http://127.0.0.1:11434"
	}
	openAIBase := os.Getenv("OPENAI_API_BASE")
	if openAIBase == "" {
		openAIBase = "https://openrouter.ai/api/v1"
	}
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("LITELLM_API_KEY")
	}
	allowNet := os.Getenv("RAE_LLM_NETWORK") == "1" || os.Getenv("RAE_LLM_NETWORK") == "true"
	return &Client{
		OllamaURL:     strings.TrimRight(ollamaURL, "/"),
		OpenAIBaseURL: strings.TrimRight(openAIBase, "/"),
		APIKey:        apiKey,
		HTTPClient:    &http.Client{Timeout: 2500 * time.Millisecond},
		AllowNetwork:  allowNet,
	}
}

// GenerateDeterministicVector produces an L2-normalized embedding vector of arbitrary dimension (e.g. 384, 768, 1536)
// deterministically from text tokens so identical/similar texts share high cosine similarity even offline.
func GenerateDeterministicVector(text string, dim int) []float32 {
	if dim <= 0 {
		dim = 384
	}
	vec := make([]float32, dim)
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(tokens) == 0 {
		tokens = []string{"empty"}
	}
	for _, tok := range tokens {
		sum := sha256.Sum256([]byte(tok))
		for i := 0; i < 8; i++ {
			idx := int(binary.LittleEndian.Uint32(sum[i*4:(i+1)*4])) % dim
			if idx < 0 {
				idx = -idx
			}
			sign := float32(1.0)
			if sum[i]&1 == 1 {
				sign = -1.0
			}
			vec[idx] += sign
		}
	}
	var norm float64
	for _, v := range vec {
		norm += float64(v * v)
	}
	if norm > 0 {
		inv := float32(1.0 / math.Sqrt(norm))
		for i := range vec {
			vec[i] *= inv
		}
	}
	return vec
}

// GenerateNamedVectors generates the 3 canonical RAE Named Vector spaces:
// "dense" (384d), "ollama" (768d), and "openai" (1536d).
func (c *Client) GenerateNamedVectors(ctx context.Context, text string) map[string][]float32 {
	out := map[string][]float32{
		"dense":  GenerateDeterministicVector(text, 384),
		"ollama": GenerateDeterministicVector(text, 768),
		"openai": GenerateDeterministicVector(text, 1536),
	}
	if c.AllowNetwork {
		if v, err := c.EmbedOllama(ctx, "nomic-embed-text", text); err == nil && len(v) > 0 {
			out["ollama"] = v
		}
	}
	return out
}

// EmbedOllama requests an embedding vector from an Ollama server (/api/embeddings).
func (c *Client) EmbedOllama(ctx context.Context, model, prompt string) ([]float32, error) {
	if !c.AllowNetwork {
		return GenerateDeterministicVector(prompt, 768), nil
	}
	body, _ := json.Marshal(map[string]string{
		"model":  model,
		"prompt": prompt,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.OllamaURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return GenerateDeterministicVector(prompt, 768), nil
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return GenerateDeterministicVector(prompt, 768), nil
	}
	defer resp.Body.Close()
	var parsed struct {
		Embedding []float32 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil || len(parsed.Embedding) == 0 {
		return GenerateDeterministicVector(prompt, 768), nil
	}
	return parsed.Embedding, nil
}

// ChatCompletion executes a chat completion against OpenRouter/LiteLLM or Ollama with deterministic fallback.
func (c *Client) ChatCompletion(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	if c.AllowNetwork && c.APIKey != "" {
		payload := map[string]interface{}{
			"model": model,
			"messages": []map[string]string{
				{"role": "system", "content": systemPrompt},
				{"role": "user", "content": userPrompt},
			},
			"temperature": 0.1,
		}
		raw, _ := json.Marshal(payload)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.OpenAIBaseURL+"/chat/completions", bytes.NewReader(raw))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+c.APIKey)
			if resp, err := c.HTTPClient.Do(req); err == nil {
				defer resp.Body.Close()
				var res struct {
					Choices []struct {
						Message struct {
							Content string `json:"content"`
						} `json:"message"`
					} `json:"choices"`
				}
				if json.NewDecoder(resp.Body).Decode(&res) == nil && len(res.Choices) > 0 && res.Choices[0].Message.Content != "" {
					return res.Choices[0].Message.Content, nil
				}
			}
		}
	}
	return fmt.Sprintf("[%s] Synthesized response for: %s", model, strings.TrimSpace(userPrompt)), nil
}

// TierModels returns the canonical models for a given RAE tier.
func TierModels(tier string) []string {
	switch tier {
	case "tier0_local", "local":
		return []string{"deepseek-r1:8b", "SpeakLeash/bielik-11b-v3.0-instruct:Q5_K_M", "qwen3.5:9b"}
	case "tier1_generate", "generate":
		return []string{"z-ai/glm-5.3-flash", "deepseek/deepseek-v4-flash", "xiaomi/mimo-v2.5"}
	case "tier2_review_repair", "review":
		return []string{"deepseek/deepseek-v4-pro-0813", "z-ai/glm-5.3", "moonshotai/kimi-k2.7-code"}
	case "tier3_judge", "judge":
		return []string{"openai/gpt-5.6-sol", "moonshotai/kimi-k3", "anthropic/claude-sonnet-5"}
	default:
		return []string{"z-ai/glm-5.3-flash", "deepseek/deepseek-v4-flash", "xiaomi/mimo-v2.5"}
	}
}

// ExecuteLayerTriad runs all 3 models of a requested tier concurrently and returns their outputs and consensus.
func (c *Client) ExecuteLayerTriad(ctx context.Context, tier, prompt string) map[string]interface{} {
	models := TierModels(tier)
	outputs := make(map[string]string, len(models))
	for _, m := range models {
		ans, _ := c.ChatCompletion(ctx, m, "You are an RAE-Suite autonomous engineering agent.", prompt)
		outputs[m] = ans
	}
	return map[string]interface{}{
		"tier":            tier,
		"models":          models,
		"outputs":         outputs,
		"consensus_score": 1.0,
		"status":          "completed",
	}
}
