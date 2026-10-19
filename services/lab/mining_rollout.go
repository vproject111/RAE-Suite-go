package lab

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strings"
)

// MinedFailureCluster represents a recurring failure signature discovered by FailureMiningEngine
// (RAE-Lab/src/lab/mining/engine.py).
type MinedFailureCluster struct {
	Signature       string   `json:"signature"`
	Occurrences     int      `json:"occurrences"`
	RootCauseClass  string   `json:"root_cause_class"`
	ProposedRuleID  string   `json:"proposed_rule_id"`
	SampleTraces    []string `json:"sample_traces"`
	Severity        float64  `json:"severity"`
}

// MineFailurePatterns clusters raw failure traces into root-cause failure signatures and synthesizes guardrails.
func (s *Service) MineFailurePatterns(traces []string) []MinedFailureCluster {
	if len(traces) == 0 {
		for _, c := range s.GetCandidates("") {
			traces = append(traces, c.FailureSource)
		}
	}
	if len(traces) == 0 {
		traces = []string{
			"AST_COMPLEXITY_EXCEEDED in service handler",
			"AST_COMPLEXITY_EXCEEDED in workflow loop",
			"HARDCODED_PATH_DETECTED in config loader",
		}
	}

	buckets := make(map[string][]string)
	for _, tr := range traces {
		sig := classifyTraceSignature(tr)
		buckets[sig] = append(buckets[sig], tr)
	}

	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	clusters := make([]MinedFailureCluster, 0, len(keys))
	for _, sig := range keys {
		samples := buckets[sig]
		sev := math.Min(1.0, math.Round((0.45+float64(len(samples))*0.15)*10000)/10000)
		clusters = append(clusters, MinedFailureCluster{
			Signature:      sig,
			Occurrences:    len(samples),
			RootCauseClass: "recurring_" + strings.ToLower(sig),
			ProposedRuleID: "GUARD_MINED_" + sig,
			SampleTraces:   samples,
			Severity:       sev,
		})
	}
	return clusters
}

func classifyTraceSignature(trace string) string {
	u := strings.ToUpper(trace)
	switch {
	case strings.Contains(u, "COMPLEXITY"):
		return "AST_COMPLEXITY_SPIKE"
	case strings.Contains(u, "PATH"):
		return "HARDCODED_FILESYSTEM_PATH"
	case strings.Contains(u, "SECRET") || strings.Contains(u, "KEY"):
		return "SECRET_EXPOSURE_RISK"
	case strings.Contains(u, "TIMEOUT") || strings.Contains(u, "DEADLINE"):
		return "LATENCY_DEADLINE_BREACH"
	default:
		return "INVARIANT_DRIFT_SIGNATURE"
	}
}

// RolloutStageDecision represents a canary rollout evaluation by SafeRolloutManager
// (RAE-Lab/src/lab/rollout/manager.py).
type RolloutStageDecision struct {
	FeatureID        string  `json:"feature_id"`
	CurrentStagePct  int     `json:"current_stage_pct"` // 5, 25, 50, 100
	NextStagePct     int     `json:"next_stage_pct"`
	Action           string  `json:"action"` // PROMOTE_STAGE, HOLD, AUTO_ROLLBACK
	ErrorRate        float64 `json:"error_rate"`
	MaxAllowedErr    float64 `json:"max_allowed_error_rate"`
	RegressionPass   bool    `json:"regression_pass"`
}

// EvaluateSafeRollout evaluates canary telemetry (5% -> 25% -> 50% -> 100%) and triggers automatic rollback on regression.
func EvaluateSafeRollout(featureID string, currentStagePct int, observedErrorRate float64, maxAllowedErrorRate float64, latencyDeltaPct float64) RolloutStageDecision {
	if featureID == "" {
		featureID = "canary-mab-hyper37"
	}
	if maxAllowedErrorRate <= 0 {
		maxAllowedErrorRate = 0.02
	}
	if currentStagePct <= 0 {
		currentStagePct = 5
	}

	if observedErrorRate > maxAllowedErrorRate || latencyDeltaPct > 25.0 {
		return RolloutStageDecision{
			FeatureID:       featureID,
			CurrentStagePct: currentStagePct,
			NextStagePct:    0,
			Action:          "AUTO_ROLLBACK",
			ErrorRate:       observedErrorRate,
			MaxAllowedErr:   maxAllowedErrorRate,
			RegressionPass:  false,
		}
	}

	nextPct := 100
	switch {
	case currentStagePct < 5:
		nextPct = 5
	case currentStagePct < 25:
		nextPct = 25
	case currentStagePct < 50:
		nextPct = 50
	default:
		nextPct = 100
	}

	return RolloutStageDecision{
		FeatureID:       featureID,
		CurrentStagePct: currentStagePct,
		NextStagePct:    nextPct,
		Action:          "PROMOTE_STAGE",
		ErrorRate:       observedErrorRate,
		MaxAllowedErr:   maxAllowedErrorRate,
		RegressionPass:  true,
	}
}

func (s *Service) registerV47LabRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/mining/failures", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Traces []string `json:"traces"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		clusters := s.MineFailurePatterns(req.Traces)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"clusters":      clusters,
			"cluster_count": len(clusters),
		})
	})

	mux.HandleFunc("/v1/rollout/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FeatureID           string  `json:"feature_id"`
			CurrentStagePct     int     `json:"current_stage_pct"`
			ObservedErrorRate   float64 `json:"observed_error_rate"`
			MaxAllowedErrorRate float64 `json:"max_allowed_error_rate"`
			LatencyDeltaPct     float64 `json:"latency_delta_pct"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		dec := EvaluateSafeRollout(req.FeatureID, req.CurrentStagePct, req.ObservedErrorRate, req.MaxAllowedErrorRate, req.LatencyDeltaPct)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(dec)
	})
}
