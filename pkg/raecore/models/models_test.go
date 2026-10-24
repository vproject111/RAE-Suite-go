package models

import (
	"testing"
	"time"
)

func TestMemoryLayerAndArtifactModels(t *testing.T) {
	if len(AllMemoryLayers()) != 8 {
		t.Fatalf("expected 8 memory layers, got %d", len(AllMemoryLayers()))
	}
	if NormalizeLayer("stm") != LayerSensory || NormalizeLayer("wm") != LayerWorking || NormalizeLayer("ltm") != LayerLTM || NormalizeLayer("em") != LayerEpisodic {
		t.Fatalf("unexpected layer normalization")
	}

	future := time.Now().UTC().Add(12 * time.Hour)
	art := &EpistemicArtifact{
		ID:             GenerateUUIDv7(),
		TenantID:       GenerateULID(),
		Type:           ArtifactDecision,
		Status:         StatusDraft,
		Visibility:     VisibilityPrivate,
		InfoClass:      InfoInternal,
		GracePeriodEnd: &future,
	}
	if !art.IsInGracePeriod(time.Now().UTC()) {
		t.Fatalf("expected artifact to be in grace period")
	}
}

func TestNormalizeLayerAllVariantsAndCBOREncoding(t *testing.T) {
	cases := []struct {
		raw  string
		want MemoryLayer
	}{
		{"sensory", LayerSensory},
		{"STM", LayerSensory},
		{"working", LayerWorking},
		{"WM", LayerWorking},
		{"episodic", LayerEpisodic},
		{"EM", LayerEpisodic},
		{"semantic", LayerSemantic},
		{"SM", LayerSemantic},
		{"procedural", LayerProcedural},
		{"PM", LayerProcedural},
		{"reflective", LayerReflective},
		{"RM", LayerReflective},
		{"reflection", LayerReflective},
		{"ltm", LayerLTM},
		{"long_term", LayerLTM},
		{"longterm", LayerLTM},
		{"sealed", LayerLTM},
		{"graph", LayerGraph},
		{"kg", LayerGraph},
		{"knowledge_graph", LayerGraph},
		{"unknown_layer", LayerSemantic},
	}
	for _, tc := range cases {
		if got := NormalizeLayer(tc.raw); got != tc.want {
			t.Fatalf("NormalizeLayer(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	// Test IsInGracePeriod false cases
	pubArt := &EpistemicArtifact{Visibility: VisibilityPublic}
	if pubArt.IsInGracePeriod(time.Now().UTC()) {
		t.Fatalf("expected public artifact not to be in grace period")
	}
	past := time.Now().UTC().Add(-1 * time.Hour)
	expArt := &EpistemicArtifact{Visibility: VisibilityPrivate, GracePeriodEnd: &past}
	if expArt.IsInGracePeriod(time.Now().UTC()) {
		t.Fatalf("expected expired private artifact not to be in grace period")
	}

	// Test Deterministic CBOR & ComputeVersionCID across header size boundaries (<24, <=0xFF, <=0xFFFF, >0xFFFF)
	str30 := "abcdefghijklmnopqrstuvwxyz0123"
	str300 := make([]byte, 300)
	for i := range str300 {
		str300[i] = 'a'
	}
	str70000 := make([]byte, 70000)
	for i := range str70000 {
		str70000[i] = 'b'
	}
	record := map[string]string{
		"b":        "short",
		"a":        str30,
		"longer_k": string(str300),
		"huge_k":   string(str70000),
	}
	cbor1 := EncodeDeterministicCBOR(record)
	cbor2 := EncodeDeterministicCBOR(record)
	if string(cbor1) != string(cbor2) {
		t.Fatalf("expected deterministic CBOR output")
	}
	cid := ComputeVersionCID(record)
	if len(cid) < 10 || cid[:7] != "mh:1220" {
		t.Fatalf("unexpected ComputeVersionCID: %s", cid)
	}
}

