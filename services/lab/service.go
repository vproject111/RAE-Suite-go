package lab

import (
	"encoding/json"
	"net/http"
	"sync"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// CandidateGuardrail represents a self-evolving guardrail proposed from recurring scan failures.
type CandidateGuardrail struct {
	ID            string `json:"id"`
	RulePattern   string `json:"rule_pattern"`
	Status        string `json:"status"` // CANDIDATE, ACTIVE
	FailureSource string `json:"failure_source"`
}

// Service implements Module 7: RAE-Lab (Evolution Engine, MABTuner, Candidate Guardrails & Offline Replay).
type Service struct {
	mu         sync.RWMutex
	mathCtrl   *raemath.MathController
	candidates []CandidateGuardrail
	scansCount int
}

// NewService creates a new RAE-Lab service.
func NewService(mc *raemath.MathController) *Service {
	if mc == nil {
		mc = raemath.NewMathController(models.MathL3)
	}
	return &Service{
		mathCtrl: mc,
	}
}

// RegisterScan records a quality scan, updates MAB weights, and proposes CANDIDATE guardrails on failures.
func (s *Service) RegisterScan(passed bool, failureReason string) map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scansCount++
	reward := 0.90
	if !passed {
		reward = 0.20
		if failureReason != "" {
			s.candidates = append(s.candidates, CandidateGuardrail{
				ID:            models.GenerateUUIDv7(),
				RulePattern:   "GUARD_" + failureReason,
				Status:        "CANDIDATE",
				FailureSource: failureReason,
			})
		}
	}
	w := s.mathCtrl.UpdateReward("fluid_100", reward)
	return map[string]interface{}{
		"status":           "recorded",
		"scans_processed":  s.scansCount,
		"updated_weights":  w,
		"candidates_count": len(s.candidates),
	}
}

// GetWeights returns the current Multi-Armed Bandit weights.
func (s *Service) GetWeights() raemath.MABWeights {
	return s.mathCtrl.GetWeights()
}

// GetCandidates returns proposed candidate guardrails.
func (s *Service) GetCandidates(statusFilter string) []CandidateGuardrail {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []CandidateGuardrail
	for _, c := range s.candidates {
		if statusFilter == "" || c.Status == statusFilter {
			out = append(out, c)
		}
	}
	return out
}

// RegisterRoutes attaches RAE-Lab HTTP endpoints (:8011).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":     "fully_operational",
			"department": "lab",
		})
	})

	mux.HandleFunc("/api/mab/weights", func(w http.ResponseWriter, _ *http.Request) {
		weights := s.GetWeights()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]float64{
			"alpha": weights.Alpha,
			"beta":  weights.Beta,
			"gamma": weights.Gamma,
		})
	})

	mux.HandleFunc("/api/guardrails/candidates", func(w http.ResponseWriter, r *http.Request) {
		st := r.URL.Query().Get("status")
		cands := s.GetCandidates(st)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"candidates": cands})
	})

	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Passed        bool   `json:"passed"`
			FailureReason string `json:"failure_reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := s.RegisterScan(req.Passed, req.FailureReason)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/v2/lab/insights", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"weights":          s.GetWeights(),
			"candidates_count": len(s.GetCandidates("")),
		})
	})
}
