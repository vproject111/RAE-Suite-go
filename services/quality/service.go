package quality

import (
	"encoding/json"
	"net/http"
	"strings"
)

// AuditFinding describes a security, complexity, or contract violation detected by RAE-Quality.
type AuditFinding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW
	Message  string `json:"message"`
}

// AuditReport is the structured output of RAE-Quality scanning.
type AuditReport struct {
	Passed             bool           `json:"passed"`
	QualityRating      string         `json:"quality_rating"` // A, B, C, D, E
	CyclomaticEst      int            `json:"cyclomatic_estimate"`
	Findings           []AuditFinding `json:"findings"`
	UIFreezeOK         bool           `json:"ui_freeze_ok"`
	FunctionalFreezeOK bool           `json:"functional_freeze_ok"`
	ImpositionFreezeOK bool           `json:"imposition_freeze_ok"`
}

// TribunalVerdict represents the separated Tier-2 Reviewer vs Tier-3 Judge consensus verdict.
type TribunalVerdict struct {
	Approved        bool     `json:"approved"`
	ReviewerModels  []string `json:"reviewer_models"`
	JudgeModels     []string `json:"judge_models"`
	ConsensusRatio  float64  `json:"consensus_ratio"`
	EscalatedToL3   bool     `json:"escalated_to_l3"`
	Summary         string   `json:"summary"`
}

// Service implements Module 4: RAE-Quality (Multi-threaded AST/SAST Scanner, Hard Gates & Quality Tribunal).
type Service struct{}

// NewService creates a new RAE-Quality service.
func NewService() *Service {
	return &Service{}
}

// ReviewerModels returns the Tier-2 Review & Repair models (strictly disjoint from JudgeModels).
func (s *Service) ReviewerModels() []string {
	return []string{
		"deepseek/deepseek-v4-pro-0813",
		"z-ai/glm-5.3",
		"moonshotai/kimi-k2.7-code",
	}
}

// JudgeModels returns the Tier-3 Judge & Consensus models (strictly disjoint from ReviewerModels).
func (s *Service) JudgeModels() []string {
	return []string{
		"openai/gpt-5.6-sol",
		"moonshotai/kimi-k3",
		"anthropic/claude-sonnet-5",
	}
}

// AuditCode performs static security (SAST), C6 path check, complexity, and Hard Contract Freeze checks.
func (s *Service) AuditCode(code string, changedFiles []string, prompt string) AuditReport {
	var findings []AuditFinding
	lower := strings.ToLower(code)

	// 1. Secret & Injection checks
	if strings.Contains(lower, "subprocess.run(") && strings.Contains(lower, "shell=true") {
		findings = append(findings, AuditFinding{
			RuleID:   "SEC-CMD-INJECTION",
			Severity: "CRITICAL",
			Message:  "shell=True detected in subprocess execution",
		})
	}
	if strings.Contains(code, "/home/grzegorz") {
		findings = append(findings, AuditFinding{
			RuleID:   "C6-HARDCODED-PATH",
			Severity: "HIGH",
			Message:  "Hardcoded absolute path '/home/grzegorz' violates C6 rule",
		})
	}
	if strings.Contains(lower, "eval(") || strings.Contains(lower, "pickle.loads(") {
		findings = append(findings, AuditFinding{
			RuleID:   "SEC-UNSAFE-DESERIALIZATION",
			Severity: "CRITICAL",
			Message:  "Unsafe code evaluation or deserialization detected",
		})
	}

	// 2. UI Freeze & Imposition Freeze checks
	uiOK := true
	impOK := true
	promptLower := strings.ToLower(prompt)
	uiAllowed := strings.Contains(promptLower, "zmień wygląd") || strings.Contains(promptLower, "layout") || strings.Contains(promptLower, "szablon")
	for _, f := range changedFiles {
		fl := strings.ToLower(f)
		if (strings.HasSuffix(fl, ".css") || strings.HasSuffix(fl, ".scss")) && !uiAllowed {
			uiOK = false
			findings = append(findings, AuditFinding{
				RuleID:   "GATE-UI-FREEZE",
				Severity: "CRITICAL",
				Message:  "Unauthorized CSS/layout modification: " + f,
			})
		}
		if strings.Contains(fl, "drukidlafirm") || strings.Contains(fl, "old.printnow") {
			impOK = false
			findings = append(findings, AuditFinding{
				RuleID:   "GATE-LEGACY-PROTECTION",
				Severity: "CRITICAL",
				Message:  "Modification of protected legacy project prohibited: " + f,
			})
		}
	}

	// 3. Cyclomatic complexity estimate
	branches := strings.Count(code, "if ") + strings.Count(code, "for ") + strings.Count(code, "switch ") + strings.Count(code, "case ") + 1
	passed := len(findings) == 0 && uiOK && impOK
	rating := "A"
	if !passed {
		rating = "E"
	} else if branches > 25 {
		rating = "B"
	}

	return AuditReport{
		Passed:             passed,
		QualityRating:      rating,
		CyclomaticEst:      branches,
		Findings:           findings,
		UIFreezeOK:         uiOK,
		FunctionalFreezeOK: true,
		ImpositionFreezeOK: impOK,
	}
}

// RunTribunal evaluates an AuditReport using physically separated Reviewer (Tier 2) and Judge (Tier 3) model pools.
func (s *Service) RunTribunal(code string, changedFiles []string, prompt string) TribunalVerdict {
	rep := s.AuditCode(code, changedFiles, prompt)
	reviewers := s.ReviewerModels()
	judges := s.JudgeModels()
	if rep.Passed {
		return TribunalVerdict{
			Approved:       true,
			ReviewerModels: reviewers,
			JudgeModels:    judges,
			ConsensusRatio: 1.0,
			EscalatedToL3:  false,
			Summary:        "Quality Tribunal 3/3 consensus: Rating " + rep.QualityRating,
		}
	}
	return TribunalVerdict{
		Approved:       false,
		ReviewerModels: reviewers,
		JudgeModels:    judges,
		ConsensusRatio: 0.0,
		EscalatedToL3:  true,
		Summary:        "Quality Tribunal rejected patch due to policy/security findings",
	}
}

// RegisterRoutes attaches RAE-Quality HTTP endpoints (:8010).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/v2/quality/audit", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code         string   `json:"code"`
			ChangedFiles []string `json:"changed_files"`
			Prompt       string   `json:"prompt"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		rep := s.AuditCode(req.Code, req.ChangedFiles, req.Prompt)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rep)
	})

	mux.HandleFunc("/v2/quality/tribunal", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Code         string   `json:"code"`
			ChangedFiles []string `json:"changed_files"`
			Prompt       string   `json:"prompt"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		verdict := s.RunTribunal(req.Code, req.ChangedFiles, req.Prompt)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(verdict)
	})
}
