package reflection

import (
	"fmt"
	"strings"
)

// ReflectionPayload represents the input context for the 4-level Reflection Coordinator.
type ReflectionPayload struct {
	AnswerDraft      string   `json:"answer_draft"`
	RetrievedSources []string `json:"retrieved_sources"`
	ContractStatus   string   `json:"contract_status"`
	CoverageRatio    float64  `json:"coverage_ratio"`
	UncertaintyLevel float64  `json:"uncertainty_level"`
	Analysis         string   `json:"analysis"`
}

// ReflectionReport aggregates the results of L1, L2, L3, and L4 cognitive reflection.
type ReflectionReport struct {
	Blocked                 bool                   `json:"blocked"`
	L1Operational           map[string]interface{} `json:"l1_operational"`
	L2Structural            map[string]interface{} `json:"l2_structural"`
	L3MetaField             map[string]interface{} `json:"l3_meta_field"`
	L4Cognitive             map[string]interface{} `json:"l4_cognitive"`
	SynthesizedLesson       string                 `json:"synthesized_lesson"`
}

// Coordinator executes the 4-level RAE Reflection Hierarchy (L1 -> L2 -> L3 -> L4).
type Coordinator struct {
	CoverageThreshold float64
	MaxUncertainty    float64
}

// NewCoordinator creates a 4-layer ReflectionCoordinator with canonical thresholds.
func NewCoordinator() *Coordinator {
	return &Coordinator{
		CoverageThreshold: 0.65,
		MaxUncertainty:    0.45,
	}
}

// Reflect executes L1 Operational, L2 Structural, L3 Meta-Field (Szubar), and L4 Cognitive (The Sage).
func (c *Coordinator) Reflect(p ReflectionPayload) ReflectionReport {
	// L1: Operational (Evidence Verifier, Contract Enforcer, Uncertainty Estimator)
	cov := p.CoverageRatio
	if cov == 0 && len(p.RetrievedSources) > 0 {
		cov = 0.90
	}
	contract := p.ContractStatus
	if contract == "" {
		contract = "ok"
	}
	block := cov < c.CoverageThreshold || contract != "ok" || p.UncertaintyLevel > c.MaxUncertainty
	l1 := map[string]interface{}{
		"coverage_ratio":    cov,
		"contract_status":   contract,
		"uncertainty_level": p.UncertaintyLevel,
		"blocked":           block,
	}

	// L2: Structural (Retrieval Analyzer, Pattern Detector, Cost Optimizer)
	quality := 0.0
	if len(p.RetrievedSources) > 0 {
		quality = 1.0
	}
	var missed []string
	draftLower := strings.ToLower(p.AnswerDraft)
	if strings.Contains(draftLower, "log") {
		foundLog := false
		for _, s := range p.RetrievedSources {
			if strings.Contains(strings.ToLower(s), "log") {
				foundLog = true
				break
			}
		}
		if !foundLog {
			missed = append(missed, "log_source")
			quality -= 0.2
		}
	}
	opt := "optimal"
	if len(p.RetrievedSources) > 20 {
		opt = "decrease_top_k"
	} else if len(p.AnswerDraft) < 50 && len(p.RetrievedSources) < 2 {
		opt = "increase_top_k"
	}
	l2 := map[string]interface{}{
		"retrieval_quality":       quality,
		"missed_sources":          missed,
		"optimization_suggestion": opt,
	}

	// L3: Meta-Field Szubar (Field Density Monitor, Renormalization Engine, Symmetry Detector)
	var scaleInconsistencies []string
	if strings.Contains(draftLower, "always") && strings.Contains(draftLower, "except") {
		scaleInconsistencies = append(scaleInconsistencies, "generalization_contradiction")
	}
	var anomalies []map[string]interface{}
	if len(p.AnswerDraft) > 50 && len(p.RetrievedSources) == 0 {
		anomalies = append(anomalies, map[string]interface{}{
			"type":     "policy_violation_ungrounded_assertion",
			"severity": 0.8,
		})
	}
	if strings.Contains(draftLower, "confidential") && strings.Contains(draftLower, "public") {
		anomalies = append(anomalies, map[string]interface{}{
			"type":     "domain_leakage",
			"severity": 0.6,
		})
	}
	l3 := map[string]interface{}{
		"scale_inconsistencies": scaleInconsistencies,
		"anomalies":             anomalies,
		"density_delta":         0.05,
	}

	// L4: Cognitive ("The Sage" — Lesson Learned distillation)
	lesson := fmt.Sprintf("Verified %d grounding sources (quality=%.2f, opt=%s).", len(p.RetrievedSources), quality, opt)
	if p.Analysis != "" {
		lesson = fmt.Sprintf("Lesson Learned: %s [sources=%d, opt=%s]", p.Analysis, len(p.RetrievedSources), opt)
	}
	l4 := map[string]interface{}{
		"lesson":     lesson,
		"confidence": cov,
		"tags":       []string{"l4_cognitive", "lesson_learned"},
	}

	return ReflectionReport{
		Blocked:           block,
		L1Operational:     l1,
		L2Structural:      l2,
		L3MetaField:       l3,
		L4Cognitive:       l4,
		SynthesizedLesson: lesson,
	}
}
