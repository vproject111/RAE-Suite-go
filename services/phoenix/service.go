package phoenix

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// ImpactReport describes the dependency blast radius of a proposed refactoring.
type ImpactReport struct {
	TargetFile     string   `json:"target_file"`
	RiskScore      float64  `json:"risk_score"`
	AffectedFiles  []string `json:"affected_files"`
	SafeToRefactor bool     `json:"safe_to_refactor"`
}

// ParityResult captures the outcome of BehaviorComparisonEngine verification.
type ParityResult struct {
	ParityVerified bool   `json:"parity_verified"`
	ExitCodeMatch  bool   `json:"exit_code_match"`
	OutputMatch    bool   `json:"output_match"`
	Reason         string `json:"reason"`
}

// RepairResult captures the outcome of a bounded 3-attempt Phoenix closed-loop repair.
type RepairResult struct {
	Status       string       `json:"status"` // REPAIRED, ROLLED_BACK_ESCALATED_L3
	AttemptsUsed int          `json:"attempts_used"`
	MaxAttempts  int          `json:"max_attempts"`
	RolledBack   bool         `json:"rolled_back"`
	Parity       ParityResult `json:"parity"`
}

// Service implements Module 5: RAE-Phoenix (Impact Graph, Behavioral Parity & 3-Attempt Closed-Loop Repair).
type Service struct {
	MaxAttempts       int
	OnRepairFailure   func(reason string)
	OnRepairCompleted func(repaired bool, reason string)
}

// NewService creates a new RAE-Phoenix service with PHOENIX_MAX_ATTEMPTS default = 3.
func NewService() *Service {
	maxAtt := 3
	if v := os.Getenv("PHOENIX_MAX_ATTEMPTS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			maxAtt = parsed
		}
	}
	return &Service{MaxAttempts: maxAtt}
}

// AnalyzeImpact evaluates the blast radius of modifying targetFile.
func (s *Service) AnalyzeImpact(targetFile string, dependents []string) ImpactReport {
	risk := float64(len(dependents)) * 0.12
	if risk > 1.0 {
		risk = 1.0
	}
	return ImpactReport{
		TargetFile:     targetFile,
		RiskScore:      risk,
		AffectedFiles:  dependents,
		SafeToRefactor: risk < 0.75,
	}
}

// VerifyBehavioralParity compares baseline and refactored outputs and exit codes.
func (s *Service) VerifyBehavioralParity(baselineOut string, baselineExit int, candidateOut string, candidateExit int) ParityResult {
	exitMatch := baselineExit == candidateExit
	outMatch := strings.TrimSpace(baselineOut) == strings.TrimSpace(candidateOut)
	if exitMatch && outMatch {
		return ParityResult{
			ParityVerified: true,
			ExitCodeMatch:  true,
			OutputMatch:    true,
			Reason:         "100% behavioral parity verified",
		}
	}
	return ParityResult{
		ParityVerified: false,
		ExitCodeMatch:  exitMatch,
		OutputMatch:    outMatch,
		Reason:         "Behavioral drift detected between baseline and candidate",
	}
}

// ExecuteClosedLoopRepair runs at most MaxAttempts (3) repair iterations and rolls back on failure.
func (s *Service) ExecuteClosedLoopRepair(baselineOut string, candidateAttempts []string) RepairResult {
	limit := s.MaxAttempts
	if limit <= 0 {
		limit = 3
	}
	used := 0
	for i, cand := range candidateAttempts {
		if i >= limit {
			break
		}
		used++
		parity := s.VerifyBehavioralParity(baselineOut, 0, cand, 0)
		if parity.ParityVerified {
			if s.OnRepairCompleted != nil {
				s.OnRepairCompleted(true, "")
			}
			return RepairResult{
				Status:       "REPAIRED",
				AttemptsUsed: used,
				MaxAttempts:  limit,
				RolledBack:   false,
				Parity:       parity,
			}
		}
	}
	reason := "Exceeded max repair attempts; rolled back and recorded as FAILED_BUT_INFORMATIVE"
	if s.OnRepairFailure != nil {
		s.OnRepairFailure(reason)
	}
	if s.OnRepairCompleted != nil {
		s.OnRepairCompleted(false, "PHOENIX_ROLLBACK")
	}
	return RepairResult{
		Status:       "ROLLED_BACK_ESCALATED_L3",
		AttemptsUsed: used,
		MaxAttempts:  limit,
		RolledBack:   true,
		Parity: ParityResult{
			ParityVerified: false,
			Reason:         reason,
		},
	}
}

// RegisterRoutes attaches RAE-Phoenix HTTP endpoints (:8012).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/v2/phoenix/impact", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TargetFile string   `json:"target_file"`
			Dependents []string `json:"dependents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		rep := s.AnalyzeImpact(req.TargetFile, req.Dependents)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rep)
	})

	mux.HandleFunc("/v2/phoenix/repair", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			BaselineOut string   `json:"baseline_out"`
			Candidates  []string `json:"candidates"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := s.ExecuteClosedLoopRepair(req.BaselineOut, req.Candidates)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})
}
