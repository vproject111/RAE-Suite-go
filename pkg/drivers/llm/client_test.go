package llm

import (
	"context"
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
