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

func TestIngestionSignaturesAndEdgeCases(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	if sig := DetectContentSignature("package main\nfunc hello() {}"); sig != "code" {
		t.Fatalf("expected code signature, got %s", sig)
	}
	if sig := DetectContentSignature("ISO 27001 security compliance policy"); sig != "policy" {
		t.Fatalf("expected policy signature, got %s", sig)
	}
	if sig := DetectContentSignature("# Architecture\nDatabase endpoint spec"); sig != "technical_doc" {
		t.Fatalf("expected technical_doc signature, got %s", sig)
	}
	if sig := DetectContentSignature("just a simple casual note"); sig != "general" {
		t.Fatalf("expected general signature, got %s", sig)
	}

	// Test default title, maxChars <= 0, overlapChars < 0, duplicate triples, and long paragraph splitting
	longText := "RAECore uses Postgres. RAECore uses Postgres.\n" +
		"This is a long paragraph with many words to trigger maxChars chunk flushing across boundaries without headings."
	res := SegmentDocument("", longText, 0, -1)
	if res.DocumentTitle != "IngestedDocument" || len(res.Chunks) == 0 {
		t.Fatalf("unexpected default SegmentDocument result: %+v", res)
	}
	if len(res.Triples) != 1 {
		t.Fatalf("expected duplicate triple to be deduplicated to 1, got %d", len(res.Triples))
	}
	resSmall := SegmentDocument("Small", longText, 50, 10)
	if len(resSmall.Chunks) < 2 {
		t.Fatalf("expected multiple chunks with maxChars=50, got %d", len(resSmall.Chunks))
	}
}

