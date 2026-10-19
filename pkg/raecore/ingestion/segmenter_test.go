package ingestion

import "testing"

func TestSegmentDocumentAndExtractGraphTriples(t *testing.T) {
	doc := `# Architecture Overview
RAE-Memory uses AgnosticStorageBroker and connects to Qdrant.
RAE-Mesh implements MerkleTree and replaces OpenClaw with Hermes.

## Security Policy
HardFramesGuard enforces ISO 42001 compliance and stores in PostgreSQL.
`
	res := SegmentDocument("RAE Spec", doc, 120, 20)
	if len(res.Chunks) < 2 {
		t.Fatalf("expected at least 2 semantic chunks, got %d", len(res.Chunks))
	}
	if len(res.Triples) < 3 {
		t.Fatalf("expected at least 3 extracted knowledge graph triples, got %+v", res.Triples)
	}
}
