package ingestion

import (
	"regexp"
	"strings"
)

// DocumentChunk represents a semantically segmented chunk of an ingested document.
type DocumentChunk struct {
	Index            int      `json:"index"`
	Heading          string   `json:"heading"`
	Content          string   `json:"content"`
	ContentSignature string   `json:"content_signature"` // technical_doc, code, policy, general
	TokenEstimate    int      `json:"token_estimate"`
	Keywords         []string `json:"keywords"`
}

// GraphTriple represents a (Subject, Relation, Object) triple extracted from text.
type GraphTriple struct {
	Subject  string  `json:"subject"`
	Relation string  `json:"relation"`
	Object   string  `json:"object"`
	Weight   float64 `json:"weight"`
}

// IngestionResult summarizes semantic document segmentation and knowledge graph extraction.
type IngestionResult struct {
	DocumentTitle    string          `json:"document_title"`
	ContentSignature string          `json:"content_signature"`
	Chunks           []DocumentChunk `json:"chunks"`
	Triples          []GraphTriple   `json:"triples"`
}

var (
	headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	tripleRegex  = regexp.MustCompile(`(?i)\b([A-Z][A-Za-z0-9_\-\.]{1,32})\s+(uses|depends on|depends_on|implements|replaces|connects to|connects_to|stores in|stores_in|integrates with|calls)\s+([A-Z][A-Za-z0-9_\-\.]{1,32})\b`)
)

// DetectContentSignature classifies document text into technical_doc, code, policy, or general.
func DetectContentSignature(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "func ") || strings.Contains(lower, "def ") || strings.Contains(lower, "package ") || strings.Contains(lower, "```") {
		return "code"
	}
	if strings.Contains(lower, "iso 42001") || strings.Contains(lower, "iso 27001") || strings.Contains(lower, "policy") || strings.Contains(lower, "compliance") {
		return "policy"
	}
	if strings.Contains(lower, "architecture") || strings.Contains(lower, "endpoint") || strings.Contains(lower, "database") || strings.Contains(lower, "# ") {
		return "technical_doc"
	}
	return "general"
}

// SegmentDocument splits a raw document by Markdown headings and paragraph boundaries with overlap.
func SegmentDocument(title, text string, maxChars, overlapChars int) IngestionResult {
	if maxChars <= 0 {
		maxChars = 400
	}
	if overlapChars < 0 {
		overlapChars = 40
	}
	if title == "" {
		title = "IngestedDocument"
	}
	sig := DetectContentSignature(text)
	lines := strings.Split(text, "\n")

	currentHeading := title
	var buf strings.Builder
	var chunks []DocumentChunk

	flush := func() {
		content := strings.TrimSpace(buf.String())
		if content == "" {
			return
		}
		words := strings.Fields(content)
		var kw []string
		seen := make(map[string]bool)
		for _, w := range words {
			clean := strings.Trim(w, ".,:;()[]{}\"'`#")
			if len(clean) >= 4 && !seen[strings.ToLower(clean)] {
				seen[strings.ToLower(clean)] = true
				kw = append(kw, clean)
				if len(kw) >= 6 {
					break
				}
			}
		}
		chunks = append(chunks, DocumentChunk{
			Index:            len(chunks),
			Heading:          currentHeading,
			Content:          content,
			ContentSignature: DetectContentSignature(content),
			TokenEstimate:    len(words),
			Keywords:         kw,
		})
		buf.Reset()
		if overlapChars > 0 && len(content) > overlapChars {
			tail := content[len(content)-overlapChars:]
			buf.WriteString(strings.TrimSpace(tail))
			buf.WriteString("\n")
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if m := headingRegex.FindStringSubmatch(trimmed); len(m) == 3 {
			flush()
			currentHeading = strings.TrimSpace(m[2])
			continue
		}
		if buf.Len()+len(line)+1 > maxChars && buf.Len() > 0 {
			flush()
		}
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(line)
	}
	flush()

	triples := ExtractGraphTriples(text)
	return IngestionResult{
		DocumentTitle:    title,
		ContentSignature: sig,
		Chunks:           chunks,
		Triples:          triples,
	}
}

// ExtractGraphTriples extracts (Subject, Relation, Object) triples from unstructured text.
func ExtractGraphTriples(text string) []GraphTriple {
	matches := tripleRegex.FindAllStringSubmatch(text, -1)
	var triples []GraphTriple
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) != 4 {
			continue
		}
		subj := strings.TrimSpace(m[1])
		rel := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(m[2]), " ", "_"))
		obj := strings.TrimSpace(m[3])
		key := subj + "|" + rel + "|" + obj
		if seen[key] {
			continue
		}
		seen[key] = true
		triples = append(triples, GraphTriple{
			Subject:  subj,
			Relation: rel,
			Object:   obj,
			Weight:   1.0,
		})
	}
	return triples
}
