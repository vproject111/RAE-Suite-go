package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLLMClientNamedVectorsAndTriad(t *testing.T) {
	c := NewClient()
	ctx := context.Background()

	vecs := c.GenerateNamedVectors(ctx, "RAE-Go multi-vector semantic test")
	if len(vecs["dense"]) != 384 || len(vecs["ollama"]) != 768 || len(vecs["openai"]) != 1536 {
		t.Fatalf("unexpected Named Vector dimensions: dense=%d, ollama=%d, openai=%d",
			len(vecs["dense"]), len(vecs["ollama"]), len(vecs["openai"]))
	}

	triad := c.ExecuteLayerTriad(ctx, "tier1_generate", "Optimize BM25 scoring")
	if triad["status"] != "completed" {
		t.Fatalf("unexpected ExecuteLayerTriad result: %+v", triad)
	}
}

func TestLLMFullCoverageWithMockHTTPAndAllProviders(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatal("expected RegisteredPackageTypes")
	}

	ctx := context.Background()

	// GenerateDeterministicVector edge cases
	vDef := GenerateDeterministicVector("   ", 0)
	if len(vDef) != 384 {
		t.Fatalf("expected default dim 384, got %d", len(vDef))
	}

	// TierModels all cases
	for _, tier := range []string{"tier0_local", "tier1_generate", "tier2_review_repair", "tier3_judge", "default_other"} {
		if len(TierModels(tier)) != 3 {
			t.Fatalf("expected 3 models for tier %s", tier)
		}
	}

	// Mock server for Ollama embeddings and OpenAI chat completions
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/embeddings":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"embedding":[0.6, 0.8]}`))
		case "/chat/completions":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"mocked llm completion"}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	_ = os.Setenv("OLLAMA_API_URL", srv.URL)
	_ = os.Setenv("OPENAI_API_BASE", srv.URL)
	_ = os.Setenv("GEMINI_API_KEY", "test-key")
	_ = os.Setenv("RAE_LLM_NETWORK", "true")
	netClient := NewClient()
	_ = os.Unsetenv("OLLAMA_API_URL")
	_ = os.Unsetenv("OPENAI_API_BASE")
	_ = os.Unsetenv("GEMINI_API_KEY")
	_ = os.Unsetenv("RAE_LLM_NETWORK")

	nv := netClient.GenerateNamedVectors(ctx, "live network vector test")
	if len(nv["ollama"]) != 2 {
		t.Fatalf("expected mocked 2d ollama vector, got %d", len(nv["ollama"]))
	}
	ans, err := netClient.ChatCompletion(ctx, "gpt-5.6-sol", "sys", "usr")
	if err != nil || ans != "mocked llm completion" {
		t.Fatalf("expected mocked llm completion, got %q (%v)", ans, err)
	}

	// Fallback when server returns invalid JSON or is unreachable
	badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid`))
	}))
	defer badSrv.Close()
	badClient := &Client{
		OllamaURL:     badSrv.URL,
		OpenAIBaseURL: badSrv.URL,
		APIKey:        "key",
		HTTPClient:    badSrv.Client(),
		AllowNetwork:  true,
	}
	embFallback, _ := badClient.EmbedOllama(ctx, "nomic", "test")
	if len(embFallback) != 768 {
		t.Fatalf("expected 768d fallback vector, got %d", len(embFallback))
	}
	chatFallback, _ := badClient.ChatCompletion(ctx, "glm-5.3", "sys", "hello")
	if chatFallback == "" {
		t.Fatal("expected fallback chat completion")
	}

	offlineEmb, _ := NewClient().EmbedOllama(ctx, "nomic", "test")
	if len(offlineEmb) != 768 {
		t.Fatal("expected 768d offline embedding")
	}

	// Providers: Qwen, Gemini, Anthropic, Ollama, DeepSeek, OpenAI
	qwen := NewQwenProvider()
	_, _ = qwen.Complete(ctx, "qwen", "hi")
	_, _ = qwen.Stream(ctx, "qwen", "hi")
	_ = qwen.Embed(ctx, "hi", 16)
	_ = qwen.EmbedBatch(ctx, []string{"a", "b"}, 16)

	gemini := NewGeminiProvider()
	if gemini.Name() != "gemini" || !gemini.IsAvailable() || len(gemini.AvailableModels()) == 0 {
		t.Fatal("GeminiProvider metadata mismatch")
	}
	_, _ = gemini.Generate(ctx, "hi")

	anth := NewAnthropicProvider()
	_, _ = anth.Complete(ctx, "claude", "hi")
	_, _ = anth.Stream(ctx, "claude", "hi")
	_ = anth.Embed(ctx, "hi", 16)
	_ = anth.EmbedBatch(ctx, []string{"a", "b"}, 16)

	ollama := NewOllamaProvider()
	if ollama.Name() != "ollama" || !ollama.IsAvailable() || len(ollama.AvailableModels()) == 0 {
		t.Fatal("OllamaProvider metadata mismatch")
	}
	_, _ = ollama.Generate(ctx, "hi")
	_, _ = ollama.Complete(ctx, "deepseek-r1:8b", "hi")
	_, _ = ollama.Stream(ctx, "deepseek-r1:8b", "hi")
	_ = ollama.Embed(ctx, "hi", 16)
	_ = ollama.EmbedBatch(ctx, []string{"a", "b"}, 16)

	ds := NewDeepSeekProvider()
	_, _ = ds.Complete(ctx, "deepseek", "hi")
	_, _ = ds.Stream(ctx, "deepseek", "hi")
	_ = ds.Embed(ctx, "hi", 16)
	_ = ds.EmbedBatch(ctx, []string{"a", "b"}, 16)

	oai := NewOpenAIProvider()
	_, _ = oai.Complete(ctx, "gpt", "hi")
	_, _ = oai.Stream(ctx, "gpt", "hi")
	_ = oai.Embed(ctx, "hi", 16)
	_ = oai.EmbedBatch(ctx, []string{"a", "b"}, 16)
}

