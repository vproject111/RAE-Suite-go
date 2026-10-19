package neural

import (
	"testing"
)

func TestONNXRuntimeEngineAndCrossEncoder(t *testing.T) {
	eng := NewONNXRuntimeEngine("all-MiniLM-L6-v2.onnx", 32)
	if !eng.Initialized {
		t.Fatalf("expected ONNXRuntimeEngine to be initialized")
	}

	v1 := eng.ForwardPass("RAE Agnostic Memory Core with Named Vectors")
	v2 := eng.ForwardPass("RAE Agnostic Memory Core with Named Vectors")
	v3 := eng.ForwardPass("PostgreSQL wire protocol socket framing")

	if len(v1) != 32 {
		t.Fatalf("expected 32-dim vector, got %d", len(v1))
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			t.Fatalf("expected deterministic forward pass at index %d", i)
		}
	}
	same := true
	for i := range v1 {
		if v1[i] != v3[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatalf("expected distinct texts to produce distinct embeddings")
	}

	ranked := eng.CrossEncoderRerank("Named Vectors Qdrant", []string{
		"Unrelated billing receipt log",
		"RAE Named Vectors in Qdrant multi-vector space",
	})
	if len(ranked) != 2 {
		t.Fatalf("expected 2 ranked candidates, got %d", len(ranked))
	}
	if ranked[0]["original_index"] != 1 {
		t.Fatalf("expected candidate 1 to rank first, got %+v", ranked[0])
	}
}
