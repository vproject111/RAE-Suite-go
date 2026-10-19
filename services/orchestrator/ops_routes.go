package orchestrator

import (
	"encoding/json"
	"net/http"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
)

func (s *Service) registerOpsRoutes(mux *http.ServeMux) {
	// Seed default base prompt in FederatedPromptRegistry
	_ = s.promptRegistry.RegisterBase("You are a governed RAE autonomous agent.")

	// 1. Self-Improvement Plane: Hypothesis, Experiment/Shadow, Canary Rollout & Promotion Gate
	mux.HandleFunc("/v2/improvement/hypothesis", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			HypothesisID string `json:"hypothesis_id"`
			Statement    string `json:"statement"`
			Motivation   string `json:"motivation"`
			TargetMetric string `json:"target_metric"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.HypothesisID == "" {
			req.HypothesisID = "hyp-" + models.GenerateULID()
		}
		hyp := s.improvementPlane.CreateHypothesis(req.HypothesisID, req.Statement, req.Motivation, req.TargetMetric)
		writeJSON(w, http.StatusOK, hyp)
	})

	mux.HandleFunc("/v2/improvement/experiment", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ExperimentID    string                 `json:"experiment_id"`
			Name            string                 `json:"name"`
			HypothesisID    string                 `json:"hypothesis_id"`
			Candidate       string                 `json:"candidate"`
			SuccessCriteria string                 `json:"success_criteria"`
			Mode            string                 `json:"mode"`
			Result          string                 `json:"result"`
			Metrics         map[string]interface{} `json:"metrics"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.ExperimentID == "" {
			req.ExperimentID = "exp-" + models.GenerateULID()
		}
		if req.Mode == "" {
			req.Mode = "shadow"
		}
		if req.Result == "" {
			req.Result = "pass"
		}
		exp := s.improvementPlane.CreateExperiment(req.ExperimentID, req.Name, req.HypothesisID, req.Candidate, req.SuccessCriteria)
		run := s.improvementPlane.RecordRun(ops.ExperimentRun{
			RunID:        "run-" + models.GenerateULID(),
			ExperimentID: exp.ExperimentID,
			Mode:         req.Mode,
			Result:       req.Result,
			Metrics:      req.Metrics,
		})
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"experiment": exp,
			"run":        run,
		})
	})

	mux.HandleFunc("/v2/improvement/canary", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ExperimentID string  `json:"experiment_id"`
			ErrorRate    float64 `json:"error_rate"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		dec := s.improvementPlane.EvaluateCanaryHealth(req.ExperimentID, req.ErrorRate)
		writeJSON(w, http.StatusOK, dec)
	})

	mux.HandleFunc("/v2/improvement/promote", func(w http.ResponseWriter, r *http.Request) {
		var prop ops.ImprovementProposal
		_ = json.NewDecoder(r.Body).Decode(&prop)
		if prop.ProposalID == "" {
			prop.ProposalID = "prop-" + models.GenerateULID()
		}
		dec := s.improvementPlane.EvaluatePromotion(prop)
		writeJSON(w, http.StatusOK, dec)
	})

	// 2. Cluster Manager (Node1 Lumina, Node2 Julka, Node3 Piotrek)
	mux.HandleFunc("/v2/ops/cluster", func(w http.ResponseWriter, r *http.Request) {
		workload := r.URL.Query().Get("workload")
		if workload == "" {
			workload = r.URL.Query().Get("role")
		}
		nodes := s.clusterMgr.ListNodes()
		selected := s.clusterMgr.RouteTask(workload)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"nodes":         nodes,
			"selected_node": selected,
		})
	})

	// 3. Transport Circuit Breaker (CLOSED -> OPEN -> HALF_OPEN)
	mux.HandleFunc("/v2/ops/circuit-breaker", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action string `json:"action"` // allow, success, failure, reset
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		allowed := true
		switch req.Action {
		case "failure":
			s.circuitBreaker.RecordFailure()
			allowed = s.circuitBreaker.AllowExecution()
		case "success":
			s.circuitBreaker.RecordSuccess()
			allowed = s.circuitBreaker.AllowExecution()
		case "reset":
			s.circuitBreaker = ops.NewTransportCircuitBreaker(3, 5.0)
		default:
			allowed = s.circuitBreaker.AllowExecution()
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"state":         s.circuitBreaker.State,
			"failure_count": s.circuitBreaker.FailureCount,
			"allowed":       allowed,
		})
	})

	// 4. Distributed Rate Limiter (Token Bucket)
	mux.HandleFunc("/v2/ops/rate-limit", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ClientID string `json:"client_id"`
			TenantID string `json:"tenant_id"`
			Cost     int    `json:"cost"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		cid := req.ClientID
		if cid == "" {
			cid = req.TenantID
		}
		if cid == "" {
			cid = "default-client"
		}
		writeJSON(w, http.StatusOK, s.rateLimiter.CheckRateLimit(cid, req.Cost))
	})

	// 5. Rollback SLA Manager (< 30s git_worktree_revert, < 120s db_schema_rollback, R4-R6 sandbox check)
	mux.HandleFunc("/v2/ops/rollback-sla", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PlanID              string  `json:"plan_id"`
			ActionType          string  `json:"action_type"`
			SLAThresholdSeconds float64 `json:"sla_threshold_seconds"`
			VerifiedInSandbox   bool    `json:"verified_in_sandbox"`
			RiskClass           string  `json:"risk_class"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.ActionType == "" {
			req.ActionType = "git_worktree_revert"
		}
		valid, msg := s.rollbackSLA.VerifyPlan(ops.RollbackPlan{
			PlanID:              req.PlanID,
			ActionType:          req.ActionType,
			SLAThresholdSeconds: req.SLAThresholdSeconds,
			VerifiedInSandbox:   req.VerifiedInSandbox,
		}, req.RiskClass)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":      valid,
			"message":    msg,
			"risk_class": req.RiskClass,
		})
	})

	// 6. Dual Escalation Engine (HERMES / OPENCODE / HUMAN_OPERATOR)
	mux.HandleFunc("/v2/ops/escalate", func(w http.ResponseWriter, r *http.Request) {
		var req ops.EscalationRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.TaskID == "" {
			req.TaskID = "task-" + models.GenerateULID()
		}
		writeJSON(w, http.StatusOK, s.escalator.EvaluateEscalation(req))
	})

	// 7. Infra Reconciler (Micro-Restarts <= 3 before L3 Escalation)
	mux.HandleFunc("/v2/ops/reconcile-infra", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ServiceName string `json:"service_name"`
			Healthy     bool   `json:"healthy"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.ServiceName == "" {
			req.ServiceName = "rae-memory"
		}
		writeJSON(w, http.StatusOK, s.infraReconciler.CheckAndRecoverService(req.ServiceName, req.Healthy))
	})

	// 8. Federated Prompt Registry (Immutable Base + Org/Team/Feature Overlay + SHA-256)
	mux.HandleFunc("/v2/ops/prompts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Layer   string `json:"layer"`
			Scope   string `json:"scope"`
			Section string `json:"section"`
			Content string `json:"content"`
			Org     string `json:"org"`
			Team    string `json:"team"`
			Feature string `json:"feature"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Layer != "" && req.Scope != "" && req.Content != "" {
			if req.Section == "" {
				req.Section = "rules"
			}
			_ = s.promptRegistry.RegisterLayer(req.Layer, req.Scope, req.Section, req.Content)
		}
		compiled, sha := s.promptRegistry.CompilePrompt(req.Org, req.Team, req.Feature)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"compiled_prompt": compiled,
			"sha256":          sha,
		})
	})

	// 9. Tool Gateway & EmptyRunCache (Echo-Loop Breaker)
	mux.HandleFunc("/v2/ops/tool-gateway", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ToolName    string `json:"tool_name"`
			ContextHash string `json:"context_hash"`
			IsEmptyRun  bool   `json:"is_empty_run"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, s.toolGateway.ExecuteTool(req.ToolName, req.ContextHash, req.IsEmptyRun))
	})

	// 10. Embedding Drift Detector (PSI >= 0.25)
	mux.HandleFunc("/v2/ops/drift-psi", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Baseline []float64 `json:"baseline"`
			Current  []float64 `json:"current"`
			Buckets  int       `json:"buckets"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		det := ops.NewEmbeddingDriftDetector(0.25)
		psi, drifted := det.CalculatePSI(req.Baseline, req.Current, req.Buckets)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"psi":            psi,
			"drift_detected": drifted,
			"threshold":      det.PSIAlertThreshold,
		})
	})

	// 11. W3C TraceContext Propagator
	mux.HandleFunc("/v2/ops/traceparent", func(w http.ResponseWriter, r *http.Request) {
		traceID := r.URL.Query().Get("trace_id")
		spanID := r.URL.Query().Get("span_id")
		if traceID == "" {
			traceID = "4bf92f3577b34da6a3ce929d0e0e4736"
		}
		if spanID == "" {
			spanID = "00f067aa0ba902b7"
		}
		tp := ops.InjectW3CTraceContext(traceID, spanID, true)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"traceparent": tp,
			"trace_id":    traceID,
			"span_id":     spanID,
		})
	})

	// 12. Keycloak JWT HS256 Auth Validator
	mux.HandleFunc("/v2/ops/keycloak-verify", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token              string   `json:"token"`
			MintUser           string   `json:"mint_user"`
			MintTenant         string   `json:"mint_tenant"`
			MintRoles          []string `json:"mint_roles"`
			MintScopes         []string `json:"mint_scopes"`
			RequiredCapability string   `json:"required_capability"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Token == "" && req.MintUser != "" {
			req.Token = s.keycloakAuth.CreateJWT(req.MintUser, req.MintTenant, req.MintRoles, req.MintScopes)
		}
		claims, err := s.keycloakAuth.VerifyJWT(req.Token, req.RequiredCapability)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":  true,
			"token":  req.Token,
			"claims": claims,
		})
	})

	// 13. Autonomy Evaluation Harness (Prompt Injection + ISO Retention)
	mux.HandleFunc("/v2/ops/eval-harness", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Prompt      string  `json:"prompt"`
			CurrentCost float64 `json:"current_cost"`
			MaxBudget   float64 `json:"max_budget"`
			RiskClass   string  `json:"risk_class"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.MaxBudget <= 0 {
			req.MaxBudget = 1.0
		}
		if req.RiskClass == "" {
			req.RiskClass = "R2"
		}
		writeJSON(w, http.StatusOK, s.evalHarness.EvaluatePrompt(req.Prompt, req.CurrentCost, req.MaxBudget, req.RiskClass))
	})

	// 14. Hermes + OpenCode Direct Bridge (Replaces OpenClaw)
	hermesBridge := ops.NewHermesOpenCodeBridge()
	mux.HandleFunc("/v2/ops/hermes-opencode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"openclaw_disabled": true,
				"hermes_endpoint":   hermesBridge.HermesEndpoint,
				"opencode_bin":      hermesBridge.OpenCodeBin,
				"status":            "active",
			})
			return
		}
		var req ops.HermesOpenCodeRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, hermesBridge.Dispatch(req))
	})

	// 15. GitOps Daemon (Branch, Signed Commit & Pull Request Spec)
	gitopsDaemon := ops.NewGitOpsDaemon()
	mux.HandleFunc("/v2/ops/gitops", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TaskID    string `json:"task_id"`
			Summary   string `json:"summary"`
			Rationale string `json:"rationale"`
			Author    string `json:"author"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, gitopsDaemon.ExecuteGitOpsWorkflow(req.TaskID, req.Summary, req.Rationale, req.Author))
	})

	// 16. Project Phase Analyzer (SDLC Phase Cost/Token/Latency Aggregation wired to live AuditLedger)
	mux.HandleFunc("/v2/ops/project-phases", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Project string           `json:"project"`
			Entries []ops.PhaseEntry `json:"entries"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Entries) == 0 && s.auditLedger != nil {
			req.Entries = ops.ConvertLedgerReceiptsToPhaseEntries(s.auditLedger.ListReceipts())
		}
		writeJSON(w, http.StatusOK, ops.AnalyzeProjectPhases(req.Project, req.Entries))
	})

	// 17. RAE Global Search Engine (Federated across live Memory, CRL, AuditLedger, Quality)
	mux.HandleFunc("/v2/portal/global-search", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query   string `json:"query"`
			Project string `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Query == "" {
			req.Query = r.URL.Query().Get("query")
		}
		var crlArts []models.EpistemicArtifact
		if s.crlSvc != nil {
			crlArts, _ = s.crlSvc.ListTraces(r.Context(), req.Project, "")
		}
		var receipts = s.auditLedger.ListReceipts()
		writeJSON(w, http.StatusOK, ops.ExecuteLiveGlobalSearch(
			req.Query,
			req.Project,
			nil,
			crlArts,
			receipts,
			[]string{
				"ISO/IEC 42001 AI Management System Governance",
				"Zero Fake Success SHA-256 Receipt Verification",
				"HardFramesGuard RESTRICTED Layer Isolation",
				"PDF/X-4 DTP 3mm Bleed & TAC <= 300% Audit",
			},
		))
	})
}
