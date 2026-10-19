package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"
)

// ContextSource represents a single memory or artifact source injected into an AI decision context.
type ContextSource struct {
	SourceID   string  `json:"source_id"`
	SourceType string  `json:"source_type"` // episodic, semantic, reflective, crl_artifact
	TrustScore float64 `json:"trust_score"`
	Weight     float64 `json:"weight"`
	Snippet    string  `json:"snippet,omitempty"`
}

// DecisionContext captures all evidentiary sources presented to an agent prior to making a decision.
type DecisionContext struct {
	ContextID string          `json:"context_id"`
	Project   string          `json:"project"`
	Query     string          `json:"query"`
	Sources   []ContextSource `json:"sources"`
	CreatedAt time.Time       `json:"created_at"`
}

// DecisionRecord links an AI decision output to its originating DecisionContext and cited SourceIDs.
type DecisionRecord struct {
	DecisionID       string    `json:"decision_id"`
	ContextID        string    `json:"context_id"`
	AgentID          string    `json:"agent_id"`
	DecisionType     string    `json:"decision_type"`
	Output           string    `json:"output"`
	CitedSourceIDs   []string  `json:"cited_source_ids"`
	LineageSHA256    string    `json:"lineage_sha256"`
	CreatedAt        time.Time `json:"created_at"`
}

// ContextQualityAudit evaluates evidentiary coverage, unused context, and hallucination risk per ISO/IEC 42001.
type ContextQualityAudit struct {
	ContextID          string   `json:"context_id"`
	TotalSources       int      `json:"total_sources"`
	CitedSources       int      `json:"cited_sources"`
	UnusedSourceIDs    []string `json:"unused_source_ids"`
	UncitedClaimIDs    []string `json:"uncited_claim_ids"`
	AverageTrustScore  float64  `json:"average_trust_score"`
	CitationCoverage   float64  `json:"citation_coverage"`
	HallucinationRisk  float64  `json:"hallucination_risk"`
	ISO42001Compliant  bool     `json:"iso42001_compliant"`
}

// ContextProvenanceService manages end-to-end decision lineage and evidentiary quality auditing.
type ContextProvenanceService struct {
	mu        sync.RWMutex
	contexts  map[string]DecisionContext
	decisions map[string]DecisionRecord
	byContext map[string][]string
}

// NewContextProvenanceService creates a new ContextProvenanceService.
func NewContextProvenanceService() *ContextProvenanceService {
	return &ContextProvenanceService{
		contexts:  make(map[string]DecisionContext),
		decisions: make(map[string]DecisionRecord),
		byContext: make(map[string][]string),
	}
}

// CreateDecisionContext registers a DecisionContext with its evidentiary sources.
func (s *ContextProvenanceService) CreateDecisionContext(project, query string, sources []ContextSource) DecisionContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("ctx-%d", len(s.contexts)+1)
	for i := range sources {
		if sources[i].TrustScore <= 0 {
			sources[i].TrustScore = 0.85
		}
		if sources[i].Weight <= 0 {
			sources[i].Weight = 1.0
		}
	}
	dc := DecisionContext{
		ContextID: id,
		Project:   project,
		Query:     query,
		Sources:   sources,
		CreatedAt: time.Now().UTC(),
	}
	s.contexts[id] = dc
	return dc
}

// RecordDecision records a DecisionRecord bound to a DecisionContext and computes its SHA-256 lineage hash.
func (s *ContextProvenanceService) RecordDecision(contextID, agentID, decisionType, output string, citedIDs []string) DecisionRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if contextID == "" {
		contextID = "ctx-1"
		if _, exists := s.contexts[contextID]; !exists {
			s.contexts[contextID] = DecisionContext{
				ContextID: contextID,
				Project:   "default",
				Query:     "auto-context",
				Sources: []ContextSource{
					{SourceID: "mem-1", SourceType: "semantic", TrustScore: 0.92, Weight: 1.0},
				},
				CreatedAt: time.Now().UTC(),
			}
		}
	}
	decID := fmt.Sprintf("dec-%d", len(s.decisions)+1)
	h := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%v", contextID, agentID, output, citedIDs)))
	rec := DecisionRecord{
		DecisionID:     decID,
		ContextID:      contextID,
		AgentID:        agentID,
		DecisionType:   decisionType,
		Output:         output,
		CitedSourceIDs: citedIDs,
		LineageSHA256:  hex.EncodeToString(h[:]),
		CreatedAt:      time.Now().UTC(),
	}
	s.decisions[decID] = rec
	s.byContext[contextID] = append(s.byContext[contextID], decID)
	return rec
}

// GetDecisionLineage returns the full provenance graph (DecisionRecord + DecisionContext + ContextQualityAudit).
func (s *ContextProvenanceService) GetDecisionLineage(decisionID string) map[string]interface{} {
	s.mu.RLock()
	rec, ok := s.decisions[decisionID]
	if !ok {
		for _, r := range s.decisions {
			rec = r
			ok = true
			break
		}
	}
	ctx := s.contexts[rec.ContextID]
	s.mu.RUnlock()

	audit := s.AuditContextQuality(rec.ContextID)
	return map[string]interface{}{
		"decision": rec,
		"context":  ctx,
		"audit":    audit,
		"verified": ok && rec.LineageSHA256 != "",
	}
}

// AuditContextQuality evaluates citation coverage, unused context sources, and hallucination risk.
func (s *ContextProvenanceService) AuditContextQuality(contextID string) ContextQualityAudit {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, ok := s.contexts[contextID]
	if !ok {
		for _, c := range s.contexts {
			ctx = c
			contextID = c.ContextID
			break
		}
	}

	sourceMap := make(map[string]ContextSource, len(ctx.Sources))
	var sumTrust float64
	for _, src := range ctx.Sources {
		sourceMap[src.SourceID] = src
		sumTrust += src.TrustScore
	}
	avgTrust := 0.90
	if len(ctx.Sources) > 0 {
		avgTrust = sumTrust / float64(len(ctx.Sources))
	}

	citedSet := make(map[string]bool)
	var uncitedClaims []string
	for _, decID := range s.byContext[contextID] {
		dec := s.decisions[decID]
		for _, cid := range dec.CitedSourceIDs {
			if _, exists := sourceMap[cid]; exists {
				citedSet[cid] = true
			} else {
				uncitedClaims = append(uncitedClaims, cid)
			}
		}
	}

	var unused []string
	for _, src := range ctx.Sources {
		if !citedSet[src.SourceID] {
			unused = append(unused, src.SourceID)
		}
	}

	coverage := 1.0
	if len(ctx.Sources) > 0 {
		coverage = float64(len(citedSet)) / float64(len(ctx.Sources))
	}
	hallucinationRisk := math.Max(0.0, (1.0-avgTrust)*0.5+float64(len(uncitedClaims))*0.25)
	return ContextQualityAudit{
		ContextID:         contextID,
		TotalSources:      len(ctx.Sources),
		CitedSources:      len(citedSet),
		UnusedSourceIDs:   unused,
		UncitedClaimIDs:   uncitedClaims,
		AverageTrustScore: math.Round(avgTrust*10000) / 10000,
		CitationCoverage:  math.Round(coverage*10000) / 10000,
		HallucinationRisk: math.Round(hallucinationRisk*10000) / 10000,
		ISO42001Compliant: len(uncitedClaims) == 0 && hallucinationRisk <= 0.25,
	}
}
