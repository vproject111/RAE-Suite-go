package orchestrator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/accounting"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/cognition"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/economy"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/workflows"
)

func (s *Service) registerCoreEngineRoutes(mux *http.ServeMux) {
	// 1. Execution Audit & Cost Ledger (KTO, CO, DLACZEGO, ZA ILE, ILE TRWAŁO, JAKI WPŁYW/EFEKT)
	mux.HandleFunc("/v2/audit/ledger/record", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			ModelName          string  `json:"model_name"`
			Provider           string  `json:"provider"`
			UserIdentity       string  `json:"user_identity"`
			TaskIntent         string  `json:"task_intent"`
			Rationale          string  `json:"rationale"`
			Decision           string  `json:"decision"`
			PromptTokens       int     `json:"prompt_tokens"`
			CompletionTokens   int     `json:"completion_tokens"`
			CostUSD            float64 `json:"cost_usd"`
			LatencyMs          float64 `json:"latency_ms"`
			QualityEffect      string  `json:"quality_effect"`
			ASTViolationsCount int     `json:"ast_violations_count"`
			Project            string  `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		receipt := s.auditLedger.RecordInvocation(
			in.ModelName,
			in.Provider,
			in.UserIdentity,
			in.TaskIntent,
			in.Rationale,
			in.Decision,
			in.PromptTokens,
			in.CompletionTokens,
			in.CostUSD,
			in.LatencyMs,
			in.QualityEffect,
			in.ASTViolationsCount,
		)
		fivePillar := accounting.BuildFivePillarMemory(receipt, in.Project)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":             "recorded",
			"receipt":            receipt,
			"five_pillar_memory": fivePillar,
		})
	})

	mux.HandleFunc("/v2/audit/ledger/summary", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.auditLedger.GetSummaryStats())
	})

	mux.HandleFunc("/v2/audit/ledger/receipts", func(w http.ResponseWriter, _ *http.Request) {
		receipts := s.auditLedger.ListReceipts()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"count":    len(receipts),
			"receipts": receipts,
		})
	})

	mux.HandleFunc("/v2/audit/receipt/verify", func(w http.ResponseWriter, r *http.Request) {
		var rec accounting.ExecutionReceipt
		_ = json.NewDecoder(r.Body).Decode(&rec)
		if rec.SHA256Hash == "" {
			rec.Finalize()
		}
		valid, msg := accounting.VerifyExecutionReceipt(rec)
		if !valid {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
				"valid":   false,
				"error":   msg,
				"receipt": rec,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid":   true,
			"message": msg,
			"receipt": rec,
		})
	})

	mux.HandleFunc("/v2/audit/project-summary", func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		if project == "" {
			project = "RAE-Suite-go"
		}
		summary := s.auditLedger.AnalyzeProject(project)
		writeJSON(w, http.StatusOK, summary)
	})

	// 2. Batch Optimization Engine & Knowledge Scale Effect
	mux.HandleFunc("/v2/batch/tasks", func(w http.ResponseWriter, r *http.Request) {
		var task economy.BatchTask
		_ = json.NewDecoder(r.Body).Decode(&task)
		if task.ActionType == "" {
			task.ActionType = "refactor"
		}
		if err := s.batchEngine.AddTask(task); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "queued",
			"task_id": task.TaskID,
			"module":  task.Module,
		})
	})

	mux.HandleFunc("/v2/batch/optimize", func(w http.ResponseWriter, _ *http.Request) {
		batches := s.batchEngine.OptimizeAndGroup()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":        "optimized",
			"batches_count": len(batches),
			"batches":       batches,
		})
	})

	mux.HandleFunc("/v2/batch/dispatch", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Batch   economy.Batch `json:"batch"`
			AgentID string        `json:"agent_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.AgentID == "" {
			req.AgentID = "agent-hive-coder-1"
		}
		tokensSaved, durationSaved := s.batchEngine.DispatchBatch(req.Batch)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":          "dispatched",
			"batch_id":        req.Batch.BatchID,
			"agent_id":        req.AgentID,
			"tasks_completed": len(req.Batch.Tasks),
			"tokens_saved":    tokensSaved,
			"duration_saved":  durationSaved,
		})
	})

	mux.HandleFunc("/v2/batch/savings", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, s.batchEngine.GetSavingsReport())
	})

	// 3. Cognitive Planner (ToT + MCTS + PRM), Dynamic Council & Swarm Consensus
	mux.HandleFunc("/v2/cognition/plan", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Intent    string `json:"intent"`
			Context   string `json:"context"`
			RiskClass string `json:"risk_class"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		plan := s.planner.CreatePlan(req.Intent, req.Context, req.RiskClass)
		writeJSON(w, http.StatusOK, plan)
	})

	mux.HandleFunc("/v2/cognition/council", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TaskClass    string  `json:"task_class"`
			RiskTier     int     `json:"risk_tier"`
			MaxCostPer1M float64 `json:"max_cost_per_1m"`
			MaxLatencyMs float64 `json:"max_latency_ms"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.TaskClass == "" {
			req.TaskClass = "code_edit"
		}
		quorum, err := s.council.SelectCouncilQuorum(req.TaskClass, req.RiskTier, req.MaxCostPer1M, req.MaxLatencyMs)
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"task_class": req.TaskClass,
			"risk_tier":  req.RiskTier,
			"quorum":     quorum,
		})
	})

	mux.HandleFunc("/v2/cognition/consensus", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Votes []cognition.AgentVote `json:"votes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := s.consensus.EvaluateConsensus(req.Votes)
		writeJSON(w, http.StatusOK, res)
	})

	// 4. Context Economy & Trust Evaluation
	mux.HandleFunc("/v2/context/depth", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			InitialScores []float64 `json:"initial_scores"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		k, rerank, _ := s.contextBroker.DetermineRetrievalDepth(req.InitialScores)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"k":            k,
			"run_reranker": rerank,
		})
	})

	mux.HandleFunc("/v2/context/prune", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Constitution string                        `json:"constitution"`
			Envelopes    []economy.ContextEnvelopeItem `json:"envelopes"`
			BudgetTokens int                           `json:"budget_tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		broker := s.contextBroker
		if req.BudgetTokens > 0 {
			broker = economy.NewContextBroker(req.BudgetTokens, 0.4)
		}
		pruned := broker.PruneContext(req.Envelopes, req.Constitution)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"pruned_context": pruned,
			"length_chars":   len(pruned),
		})
	})

	mux.HandleFunc("/v2/context/trust", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Metadata map[string]interface{} `json:"metadata"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		score, classification := s.trustEval.EvaluateTrust(req.Metadata)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"trust_score":    score,
			"classification": classification,
		})
	})

	// 5. Workflows: Saga, Streaming Composer, Speculative Exec, ClaimCheck, SafeReplay, A2A
	mux.HandleFunc("/v2/workflows/saga", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Steps      []string `json:"steps"`
			FailAtStep string   `json:"fail_at_step"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Steps) == 0 {
			req.Steps = []string{"reserve_resources", "apply_migration", "publish_mesh_state"}
		}
		saga := workflows.NewSagaCoordinator()
		for _, stepName := range req.Steps {
			sn := stepName
			saga.AddStep(sn, func() (any, error) {
				if req.FailAtStep != "" && sn == req.FailAtStep {
					return nil, fmt.Errorf("simulated failure at %s", sn)
				}
				return "ok", nil
			}, func() error {
				return nil
			})
		}
		writeJSON(w, http.StatusOK, saga.ExecuteSaga())
	})

	mux.HandleFunc("/v2/workflows/stream", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Chunks []string `json:"chunks"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		composer := workflows.NewStreamingFunctionComposer(map[string]workflows.ToolHandler{
			"search_code": func(args map[string]interface{}) (interface{}, error) {
				return fmt.Sprintf("match:%v", args["query"]), nil
			},
			"verify_match": func(args map[string]interface{}) (interface{}, error) {
				return fmt.Sprintf("verified(%v)", args["target"]), nil
			},
		})
		res := composer.PipeSteps(req.Chunks)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"steps_executed": len(res),
			"results":        res,
		})
	})

	mux.HandleFunc("/v2/workflows/speculative", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Commands []string `json:"commands"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := s.specExec.ExecuteSpeculatively(req.Commands)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"results": res,
		})
	})

	mux.HandleFunc("/v2/workflows/claim-check", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Payload  string `json:"payload"`
			Retrieve bool   `json:"retrieve"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		pointer := s.claimCheck.OffloadIfNeeded(req.Payload)
		out := map[string]interface{}{
			"pointer_or_payload": pointer,
			"offloaded":          pointer != req.Payload,
		}
		if req.Retrieve {
			restored, err := s.claimCheck.Retrieve(pointer)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			out["restored_length"] = len(restored)
			out["integrity_verified"] = restored == req.Payload
		}
		writeJSON(w, http.StatusOK, out)
	})

	mux.HandleFunc("/v2/workflows/replay", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			StepData           map[string]interface{} `json:"step_data"`
			Mode               string                 `json:"mode"`
			AuthorizationToken string                 `json:"authorization_token"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Mode == "" {
			req.Mode = string(workflows.ReplaySimulate)
		}
		res, err := s.replayEngine.ReplayStep(req.StepData, workflows.ReplayMode(req.Mode), req.AuthorizationToken)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("/v2/a2a/send", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SenderID  string                 `json:"sender_id"`
			Secret    string                 `json:"secret"`
			Recipient string                 `json:"recipient"`
			Algorithm string                 `json:"alg"`
			Nonce     string                 `json:"nonce"`
			Payload   map[string]interface{} `json:"payload"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.SenderID == "" {
			req.SenderID = "rae-suite"
		}
		if req.Recipient == "" {
			req.Recipient = "rae-hive"
		}
		if req.Secret != "" {
			s.a2aAdapter.RegisterAgent(req.SenderID, req.Secret)
		}
		env, err := s.a2aAdapter.SignMessage(req.SenderID, req.Recipient, req.Algorithm, req.Nonce, time.Now().Unix(), req.Payload)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		verified, err := s.a2aAdapter.VerifyAndReceive(env)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, verified)
	})

	// 6. Autonomy: Shadow Evaluator, Curiosity Engine, Tool Forge & Thin Device Runtime
	mux.HandleFunc("/v2/autonomy/shadow", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FailurePattern    string  `json:"failure_pattern"`
			TestInput         string  `json:"test_input"`
			Promote           bool    `json:"promote"`
			HoursActive       float64 `json:"hours_active"`
			FalsePositiveRate float64 `json:"false_positive_rate"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		g := s.shadowEval.MineFailure(req.FailurePattern)
		matched := s.shadowEval.EvaluateShadowGuardrails(req.TestInput)
		var promoErr string
		if req.Promote {
			promoted, err := s.shadowEval.PromoteGuardrail(g.RuleID, req.HoursActive, req.FalsePositiveRate)
			if err != nil {
				promoErr = err.Error()
			} else {
				g = promoted
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"guardrail":       g,
			"matched_rules":   matched,
			"promotion_error": promoErr,
		})
	})

	mux.HandleFunc("/v2/autonomy/curiosity", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID         string `json:"id"`
			Hypothesis string `json:"hypothesis"`
			Module     string `json:"module"`
			RunIdle    bool   `json:"run_idle"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.ID == "" {
			req.ID = "cur-" + strconv.FormatInt(time.Now().UnixNano(), 10)
		}
		item := s.curiosity.EnqueueIdleVerification(req.ID, req.Hypothesis, req.Module)
		var promoted []workflows.CuriosityItem
		if req.RunIdle {
			promoted = s.curiosity.RunIdleCycle()
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"enqueued": item,
			"promoted": promoted,
		})
	})

	mux.HandleFunc("/v2/autonomy/tool-forge", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ToolName string `json:"tool_name"`
			Code     string `json:"code"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, s.toolForge.SynthesizeTool(req.ToolName, req.Code))
	})

	mux.HandleFunc("/v2/runtime/device-profile", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TotalRAMMB  int  `json:"total_ram_mb"`
			HasLocalGPU bool `json:"has_local_gpu"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.TotalRAMMB <= 0 {
			if q := r.URL.Query().Get("ram_mb"); q != "" {
				req.TotalRAMMB, _ = strconv.Atoi(q)
			}
		}
		if req.TotalRAMMB <= 0 {
			req.TotalRAMMB = 16384
		}
		writeJSON(w, http.StatusOK, s.thinRuntime.DetectHardwareProfile(req.TotalRAMMB, req.HasLocalGPU))
	})
}
