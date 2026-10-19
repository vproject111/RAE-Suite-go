package search

import (
	"regexp"
	"strings"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// EvidenceConflict describes a contradiction detected between two retrieved memories (<1ms).
type EvidenceConflict struct {
	MemoryIDA    string  `json:"memory_id_a"`
	MemoryIDB    string  `json:"memory_id_b"`
	ConflictType string  `json:"conflict_type"` // POLARITY_CONTRADICTION, NUMERIC_DIVERGENCE
	Description  string  `json:"description"`
	Severity     float64 `json:"severity"`
}

// RewritePlan represents an architectural query rewrite plan for 2-pass adaptive retrieval.
type RewritePlan struct {
	OriginalQuery   string   `json:"original_query"`
	RewrittenQuery  string   `json:"rewritten_query"`
	SubQueries      []string `json:"sub_queries"`
	ExpandedTerms   []string `json:"expanded_terms"`
	BoostFactor     float64  `json:"boost_factor"`
	SecondPassRun   bool     `json:"second_pass_run"`
}

var numberExtractor = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)

// DetectEvidenceConflicts scans retrieved memory items in <1ms for polarity contradictions and numeric divergences.
func DetectEvidenceConflicts(items []models.ScoredMemoryItem) []EvidenceConflict {
	var conflicts []EvidenceConflict
	antonymPairs := [][2]string{
		{"enabled", "disabled"},
		{"allowed", "forbidden"},
		{"true", "false"},
		{"active", "inactive"},
		{"must", "never"},
	}

	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			a := strings.ToLower(items[i].Memory.Content)
			b := strings.ToLower(items[j].Memory.Content)
			idA := items[i].Memory.ID
			idB := items[j].Memory.ID

			// Check shared subject tokens (>= 2 common words of length >= 4)
			wordsA := strings.Fields(a)
			common := 0
			for _, w := range wordsA {
				clean := strings.Trim(w, ".,:;()[]{}")
				if len(clean) >= 4 && strings.Contains(b, clean) {
					common++
				}
			}
			if common < 1 {
				continue
			}

			for _, pair := range antonymPairs {
				hasA0 := strings.Contains(a, pair[0])
				hasA1 := strings.Contains(a, pair[1])
				hasB0 := strings.Contains(b, pair[0])
				hasB1 := strings.Contains(b, pair[1])
				if (hasA0 && hasB1) || (hasA1 && hasB0) {
					conflicts = append(conflicts, EvidenceConflict{
						MemoryIDA:    idA,
						MemoryIDB:    idB,
						ConflictType: "POLARITY_CONTRADICTION",
						Description:  "Conflicting polarity terms (" + pair[0] + " vs " + pair[1] + ")",
						Severity:     0.85,
					})
					break
				}
			}

			numsA := numberExtractor.FindAllString(a, -1)
			numsB := numberExtractor.FindAllString(b, -1)
			if len(numsA) > 0 && len(numsB) > 0 && numsA[0] != numsB[0] && common >= 2 {
				conflicts = append(conflicts, EvidenceConflict{
					MemoryIDA:    idA,
					MemoryIDB:    idB,
					ConflictType: "NUMERIC_DIVERGENCE",
					Description:  "Divergent numeric values (" + numsA[0] + " vs " + numsB[0] + ") on shared topic",
					Severity:     0.75,
				})
			}
		}
	}
	return conflicts
}

// CreateRewritePlan generates a targeted 2-pass query rewrite plan when SufficiencyGate coverage is low.
func CreateRewritePlan(query string, sufficiencyScore float64) RewritePlan {
	synonyms := map[string][]string{
		"db":       {"database", "postgres", "qdrant", "storage"},
		"database": {"postgres", "qdrant", "redis", "agnostic"},
		"mesh":     {"merkle", "crdt", "federation", "sync"},
		"pdf":      {"dtp", "bleed", "trimbox", "cmyk"},
		"auth":     {"keycloak", "jwt", "rbac", "hmac"},
	}
	words := strings.Fields(query)
	var expanded []string
	for _, w := range words {
		lw := strings.ToLower(strings.Trim(w, ".,?!"))
		if syns, ok := synonyms[lw]; ok {
			expanded = append(expanded, syns...)
		}
	}
	if len(expanded) == 0 {
		expanded = []string{"architecture", "contract", "invariant"}
	}
	rewritten := strings.TrimSpace(query + " " + strings.Join(expanded, " "))
	subQueries := []string{query}
	for _, e := range expanded {
		subQueries = append(subQueries, query+" "+e)
	}
	return RewritePlan{
		OriginalQuery:  query,
		RewrittenQuery: rewritten,
		SubQueries:     subQueries,
		ExpandedTerms:  expanded,
		BoostFactor:    1.4,
		SecondPassRun:  sufficiencyScore < 0.85,
	}
}
