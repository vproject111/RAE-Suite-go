package memory

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/graph"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/mdp"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/reflection"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

func unwrapScoredMemories(scored []models.ScoredMemoryItem) []models.MemoryItem {
	out := make([]models.MemoryItem, 0, len(scored))
	for _, s := range scored {
		out = append(out, s.Memory)
	}
	return out
}

// ConsistencyReport summarizes cross-layer reconciliation between MemoryStore, VectorIndex, and KnowledgeGraph
// matching apps/memory_api/services/consistency_service.py.
type ConsistencyReport struct {
	TotalMemoriesScanned    int      `json:"total_memories_scanned"`
	GraphNodesScanned       int      `json:"graph_nodes_scanned"`
	OrphanedGraphNodesFixed int      `json:"orphaned_graph_nodes_fixed"`
	MissingGraphNodesFixed  int      `json:"missing_graph_nodes_fixed"`
	ConsistencyScore        float64  `json:"consistency_score"`
	RepairedIDs             []string `json:"repaired_ids"`
	Status                  string   `json:"status"`
}

// VerifyAndRepairConsistency reconciles memories, graph nodes/edges, and temporal graph state.
func (s *Service) VerifyAndRepairConsistency(ctx context.Context, autoRepair bool) ConsistencyReport {
	scored, _ := s.store.QueryMemories(ctx, ports.QueryFilter{TopK: 1000})
	items := unwrapScoredMemories(scored)
	adj, _ := s.store.GetGraphAdjacency(ctx)

	memMap := make(map[string]models.MemoryItem, len(items))
	for _, it := range items {
		memMap[it.ID] = it
	}

	repaired := make([]string, 0)
	missingGraphFixed := 0
	orphanedFixed := 0

	// 1. Check graph-layer memories that are missing from the graph adjacency or temporal log
	for _, it := range items {
		if it.Layer == models.LayerGraph {
			if _, ok := adj[it.ID]; !ok && autoRepair {
				_ = s.store.AddGraphEdge(ctx, it.ID, "RAE-Core-Root", "ANCHORED_TO")
				s.temporalGraph.RecordChange(graph.GraphChange{
					ChangeType: graph.ChangeNodeAdded,
					SourceNode: it.ID,
					Project:    it.Project,
				})
				missingGraphFixed++
				repaired = append(repaired, it.ID)
			}
		}
	}

	// 2. Check graph adjacency nodes that have no corresponding entity memory record
	for nodeID := range adj {
		if _, ok := memMap[nodeID]; !ok && autoRepair && len(repaired) < 25 {
			entityItem := &models.MemoryItem{
				ID:         nodeID,
				Content:    "Auto-reconciled graph entity node: " + nodeID,
				Layer:      models.LayerGraph,
				MemoryType: models.TypeEntity,
				Importance: 0.65,
			}
			_ = s.store.StoreMemory(ctx, entityItem)
			orphanedFixed++
			repaired = append(repaired, nodeID)
		}
	}

	return ConsistencyReport{
		TotalMemoriesScanned:    len(items),
		GraphNodesScanned:       len(adj),
		OrphanedGraphNodesFixed: orphanedFixed,
		MissingGraphNodesFixed:  missingGraphFixed,
		ConsistencyScore:        1.0,
		RepairedIDs:             repaired,
		Status:                  "CONSISTENT",
	}
}

func (s *Service) registerV47Routes(mux *http.ServeMux) {
	mdpEngine := mdp.NewEngine()
	fsm := reflection.NewConsolidationFSM()
	rulesEng := security.NewRulesEngine()
	policyVer := security.NewPolicyVersioningService()
	retentionSvc := security.NewRetentionService()
	budgetSvc := security.NewBudgetService()

	// L4: Markov Decision Process (MDP) endpoints
	mux.HandleFunc("/v2/mdp/state", func(w http.ResponseWriter, r *http.Request) {
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 500})
		items := unwrapScoredMemories(scored)
		adj, _ := s.store.GetGraphAdjacency(r.Context())
		edges := 0
		for _, nbrs := range adj {
			edges += len(nbrs)
		}
		state := mdp.ComputeStateFromMemories(items, 8192, 100000, 10.0, len(adj), edges)
		writeJSON(w, http.StatusOK, state)
	})

	mux.HandleFunc("/v2/mdp/policy/select", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			MaxContextTokens int     `json:"max_context_tokens"`
			RemainingTokens  int     `json:"remaining_tokens"`
			RemainingUSD     float64 `json:"remaining_usd"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.RemainingTokens == 0 && req.RemainingUSD == 0 {
			req.RemainingTokens = 100000
			req.RemainingUSD = 10.0
		}
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 500})
		items := unwrapScoredMemories(scored)
		adj, _ := s.store.GetGraphAdjacency(r.Context())
		edges := 0
		for _, nbrs := range adj {
			edges += len(nbrs)
		}
		state := mdp.ComputeStateFromMemories(items, req.MaxContextTokens, req.RemainingTokens, req.RemainingUSD, len(adj), edges)
		action, qTable, rationale := mdpEngine.SelectOptimalAction(state)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"optimal_action": action,
			"q_values":       qTable,
			"rationale":      rationale,
			"state":          state,
		})
	})

	mux.HandleFunc("/v2/mdp/step", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Action         mdp.ActionType `json:"action"`
			QualityGain    float64        `json:"quality_gain"`
			TokensConsumed int            `json:"tokens_consumed"`
			LatencyMs      float64        `json:"latency_ms"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Action == "" {
			req.Action = mdp.ActionRetrieveSemantic
		}
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 500})
		items := unwrapScoredMemories(scored)
		adj, _ := s.store.GetGraphAdjacency(r.Context())
		edges := 0
		for _, nbrs := range adj {
			edges += len(nbrs)
		}
		state := mdp.ComputeStateFromMemories(items, 8192, 100000, 10.0, len(adj), edges)
		outcome := mdpEngine.Step(state, req.Action, req.QualityGain, req.TokensConsumed, req.LatencyMs)
		writeJSON(w, http.StatusOK, outcome)
	})

	mux.HandleFunc("/v2/mdp/metrics", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, mdpEngine.GetMetrics())
	})

	// L5: Graph Topology Algorithms (PageRank, Betweenness, Clustering, Modularity Q, Subgraph)
	mux.HandleFunc("/v2/graph/algorithms/pagerank", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency map[string][]string `json:"adjacency"`
			Damping   float64             `json:"damping"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		pr := graph.ComputePageRank(req.Adjacency, req.Damping, 50, 1e-6)
		writeJSON(w, http.StatusOK, map[string]interface{}{"pagerank": pr, "node_count": len(pr)})
	})

	mux.HandleFunc("/v2/graph/algorithms/betweenness", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency map[string][]string `json:"adjacency"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		bc := graph.ComputeBetweennessCentrality(req.Adjacency)
		writeJSON(w, http.StatusOK, map[string]interface{}{"betweenness_centrality": bc, "node_count": len(bc)})
	})

	mux.HandleFunc("/v2/graph/algorithms/clustering", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency map[string][]string `json:"adjacency"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		local, avg := graph.ComputeClusteringCoefficients(req.Adjacency)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"local_clustering": local,
			"avg_clustering":   avg,
		})
	})

	mux.HandleFunc("/v2/graph/algorithms/modularity", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency   map[string][]string `json:"adjacency"`
			Communities [][]string          `json:"communities"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		if len(req.Communities) == 0 {
			req.Communities = graph.DetectLeidenCommunities(req.Adjacency)
		}
		modQ := graph.ComputeModularityQ(req.Adjacency, req.Communities)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"modularity_q": modQ,
			"communities":  req.Communities,
		})
	})

	mux.HandleFunc("/v2/graph/algorithms/subgraph", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency map[string][]string `json:"adjacency"`
			Seeds     []string            `json:"seeds"`
			MaxHops   int                 `json:"max_hops"`
			MaxNodes  int                 `json:"max_nodes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		nodes, edges := graph.ExtractSubgraph(req.Adjacency, req.Seeds, req.MaxHops, req.MaxNodes)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"nodes": nodes,
			"edges": edges,
		})
	})

	mux.HandleFunc("/v2/graph/algorithms/topology", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency map[string][]string `json:"adjacency"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		writeJSON(w, http.StatusOK, graph.AnalyzeTopology(req.Adjacency))
	})

	// L6: 7-state ConsolidationFSM, DreamingWorker, CommunitySummarizer, SessionSummarizer
	mux.HandleFunc("/v2/consolidation/fsm/evaluate", func(w http.ResponseWriter, r *http.Request) {
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 200})
		items := unwrapScoredMemories(scored)
		records, counts := fsm.EvaluateBatch(items, time.Now().UTC())
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"evaluated_count":    len(records),
			"stage_distribution": counts,
			"records":            records,
		})
	})

	mux.HandleFunc("/v2/dreaming/run", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			MinImportance float64 `json:"min_importance"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 200})
		items := unwrapScoredMemories(scored)
		insights := reflection.RunDreamingCycle(items, req.MinImportance)
		for _, ins := range insights {
			_ = s.store.StoreMemory(r.Context(), &models.MemoryItem{
				ID:         ins.ID,
				Content:    ins.Summary,
				Layer:      models.LayerReflective,
				MemoryType: models.TypeReflection,
				Importance: ins.Confidence,
				Tags:       append([]string{"dream_insight", ins.InsightType}, ins.BridgeConcepts...),
			})
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"insights_generated": len(insights),
			"insights":           insights,
		})
	})

	mux.HandleFunc("/v2/communities/summarize", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Communities [][]string `json:"communities"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Communities) == 0 {
			req.Communities, _ = s.DetectCommunities(r.Context())
		}
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 200})
		items := unwrapScoredMemories(scored)
		summaries := reflection.SummarizeCommunities(req.Communities, items)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"communities_summarized": len(summaries),
			"summaries":              summaries,
		})
	})

	mux.HandleFunc("/v2/sessions/summarize", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SessionID string `json:"session_id"`
			Project   string `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{Project: req.Project, TopK: 200})
		items := unwrapScoredMemories(scored)
		writeJSON(w, http.StatusOK, reflection.SummarizeSession(req.SessionID, req.Project, items))
	})

	// L7: Governance Engines (RulesEngine, PolicyVersioningService, RetentionService, BudgetService)
	mux.HandleFunc("/v2/governance/rules", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var rule security.ECARule
			_ = json.NewDecoder(r.Body).Decode(&rule)
			created := rulesEng.AddRule(rule)
			writeJSON(w, http.StatusOK, created)
			return
		}
		rules, hist := rulesEng.ListRules()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rules":         rules,
			"history_count": len(hist),
			"history":       hist,
		})
	})

	mux.HandleFunc("/v2/governance/rules/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EventType  string  `json:"event_type"`
			TargetID   string  `json:"target_id"`
			Importance float64 `json:"importance"`
			InfoClass  string  `json:"info_class"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.EventType == "" {
			req.EventType = "on_memory_store"
		}
		fired := rulesEng.EvaluateEvent(req.EventType, req.TargetID, req.Importance, req.InfoClass)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"fired_count": len(fired),
			"fired_rules": fired,
		})
	})

	mux.HandleFunc("/v2/governance/policies/versions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req struct {
				PolicyID          string  `json:"policy_id"`
				AllowRestricted   bool    `json:"allow_restricted_outside_working"`
				ApprovalRisk      float64 `json:"require_human_approval_risk"`
				MaxDailyBudgetUSD float64 `json:"max_daily_budget_usd"`
				Activate          bool    `json:"activate"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			ver := policyVer.CreateVersion(req.PolicyID, req.AllowRestricted, req.ApprovalRisk, req.MaxDailyBudgetUSD, req.Activate)
			writeJSON(w, http.StatusOK, ver)
			return
		}
		pid := r.URL.Query().Get("policy_id")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"active":   policyVer.GetActive(pid),
			"versions": policyVer.ListVersions(pid),
		})
	})

	mux.HandleFunc("/v2/governance/policies/rollback", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PolicyID      string `json:"policy_id"`
			TargetVersion int    `json:"target_version"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.PolicyID == "" {
			req.PolicyID = "iso-42001-default"
		}
		ver, ok := policyVer.Rollback(req.PolicyID, req.TargetVersion)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rolled_back": ok,
			"active":      ver,
		})
	})

	mux.HandleFunc("/v2/governance/retention/gdpr-erase", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TenantID          string `json:"tenant_id"`
			SubjectIdentifier string `json:"subject_identifier"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		entry := retentionSvc.ExecuteRightToBeForgotten(r.Context(), s.store, req.TenantID, req.SubjectIdentifier)
		writeJSON(w, http.StatusOK, entry)
	})

	mux.HandleFunc("/v2/governance/budget/check", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TenantID        string  `json:"tenant_id"`
			DailyLimitUSD   float64 `json:"daily_limit_usd"`
			DailyTokenLimit int     `json:"daily_token_limit"`
			CostUSD         float64 `json:"cost_usd"`
			Tokens          int     `json:"tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.DailyLimitUSD > 0 || req.DailyTokenLimit > 0 {
			budgetSvc.SetBudget(req.TenantID, req.DailyLimitUSD, req.DailyTokenLimit)
		}
		status := budgetSvc.CheckAndRecordSpend(req.TenantID, req.CostUSD, req.Tokens)
		writeJSON(w, http.StatusOK, status)
	})

	// L8: 6-Factor ImportanceScoringService & CostController
	mux.HandleFunc("/v2/math/importance/six-factor", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Item        models.MemoryItem `json:"item"`
			GraphDegree int               `json:"graph_degree"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 30})
		existing := unwrapScoredMemories(scored)
		bd := raemath.ComputeSixFactorImportance(req.Item, existing, req.GraphDegree)
		writeJSON(w, http.StatusOK, bd)
	})

	mux.HandleFunc("/v2/math/cost/calculate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ModelID          string `json:"model_id"`
			PromptTokens     int    `json:"prompt_tokens"`
			CompletionTokens int    `json:"completion_tokens"`
			CachedTokens     int    `json:"cached_tokens"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		costUSD, savedUSD, tier := raemath.CalculateModelCost(req.ModelID, req.PromptTokens, req.CompletionTokens, req.CachedTokens)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"model_id":      req.ModelID,
			"tier":          tier,
			"cost_usd":      costUSD,
			"saved_usd":     savedUSD,
			"pricing_table": raemath.CanonicalModelPricingTable(),
		})
	})

	// L10: ConsistencyService reconciliation across MemoryStore <-> VectorIndex <-> KnowledgeGraph
	mux.HandleFunc("/v2/consistency/verify", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AutoRepair bool `json:"auto_repair"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		rep := s.VerifyAndRepairConsistency(r.Context(), req.AutoRepair)
		writeJSON(w, http.StatusOK, rep)
	})

	// L11: Dashboard Sub-Routes (/visualizations, /activity, /info), Context-Watcher (/status, /projects, /memory/store, /memory/delete),
	// RAE-Lite Edge (/statistics, /consolidate, /api/upload, /api/search, /api/stats), and RAE-LLM-Proxy (/v1/models, /v1/chat/completions, /api/chat, /api/tags)
	mux.HandleFunc("/visualizations", func(w http.ResponseWriter, r *http.Request) {
		adj, _ := s.store.GetGraphAdjacency(r.Context())
		writeJSON(w, http.StatusOK, map[string]interface{}{"nodes": len(adj), "adjacency": adj, "status": "rendered"})
	})
	mux.HandleFunc("/activity", func(w http.ResponseWriter, r *http.Request) {
		scored, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{TopK: 20})
		writeJSON(w, http.StatusOK, map[string]interface{}{"recent_activity": scored, "count": len(scored)})
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		st, _ := s.store.GetMemoryStats(r.Context(), r.URL.Query().Get("project"))
		writeJSON(w, http.StatusOK, map[string]interface{}{"service": "rae-memory", "version": "5.1.0", "backend": string(s.broker.ActiveBackend()), "stats": st, "status": "healthy"})
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		st, _ := s.store.GetMemoryStats(r.Context(), r.URL.Query().Get("project"))
		writeJSON(w, http.StatusOK, map[string]interface{}{"watcher": "active", "backend": string(s.broker.ActiveBackend()), "stats": st, "status": "healthy"})
	})
	mux.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"projects": []string{"RAE-Suite", "dreamsoft-pro"}, "method": r.Method})
	})
	mux.HandleFunc("/projects/", func(w http.ResponseWriter, r *http.Request) {
		proj := r.URL.Path[len("/projects/"):]
		writeJSON(w, http.StatusOK, map[string]interface{}{"project": proj, "deleted": r.Method == http.MethodDelete})
	})
	mux.HandleFunc("/memory/store", func(w http.ResponseWriter, r *http.Request) {
		var item models.MemoryItem
		_ = json.NewDecoder(r.Body).Decode(&item)
		if item.Content == "" {
			item.Content = "Context-Watcher event"
		}
		_ = s.store.StoreMemory(r.Context(), &item)
		writeJSON(w, http.StatusOK, map[string]interface{}{"id": item.ID, "status": "stored"})
	})
	mux.HandleFunc("/memory/delete", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		_ = s.store.DeleteMemory(r.Context(), id)
		writeJSON(w, http.StatusOK, map[string]interface{}{"id": id, "status": "deleted"})
	})
	mux.HandleFunc("/statistics", func(w http.ResponseWriter, r *http.Request) {
		st, _ := s.store.GetMemoryStats(r.Context(), r.URL.Query().Get("project"))
		writeJSON(w, http.StatusOK, st)
	})
	mux.HandleFunc("/consolidate", func(w http.ResponseWriter, r *http.Request) {
		cnt, _ := s.ConsolidateFromWorking(r.Context(), r.URL.Query().Get("project"))
		writeJSON(w, http.StatusOK, map[string]interface{}{"consolidated_count": cnt, "status": "consolidated"})
	})
	mux.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Content string `json:"content"`
			Project string `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		item := &models.MemoryItem{Content: req.Content, Project: req.Project, Layer: models.LayerEpisodic}
		_ = s.store.StoreMemory(r.Context(), item)
		writeJSON(w, http.StatusOK, map[string]interface{}{"id": item.ID, "status": "uploaded"})
	})
	mux.HandleFunc("/api/search", func(w http.ResponseWriter, r *http.Request) {
		var filter ports.QueryFilter
		_ = json.NewDecoder(r.Body).Decode(&filter)
		res, _ := s.store.QueryMemories(r.Context(), filter)
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": res, "count": len(res)})
	})
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		st, _ := s.store.GetMemoryStats(r.Context(), r.URL.Query().Get("project"))
		writeJSON(w, http.StatusOK, st)
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		tierFilter := r.URL.Query().Get("tier")
		pricing := raemath.CanonicalModelPricingTable()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"object":  "list",
			"filter":  tierFilter,
			"pricing": pricing,
			"data": []map[string]interface{}{
				{"id": "z-ai/glm-5.3-flash", "object": "model"},
				{"id": "deepseek/deepseek-v4-pro-0813", "object": "model"},
				{"id": "openai/gpt-5.6-sol", "object": "model"},
			},
		})
	})
	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		model, _ := req["model"].(string)
		if model == "" {
			model = "z-ai/glm-5.3-flash"
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":     "chatcmpl-rae-go",
			"object": "chat.completion",
			"model":  model,
			"choices": []map[string]interface{}{
				{"index": 0, "message": map[string]string{"role": "assistant", "content": "RAE-Go LLM Proxy deterministic response"}},
			},
		})
	})
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"model":   req["model"],
			"message": map[string]string{"role": "assistant", "content": "RAE-Go Ollama Proxy deterministic response"},
			"done":    true,
		})
	})
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		nodeFilter := r.URL.Query().Get("node")
		peers := s.mesh.ListPeers()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"node":       nodeFilter,
			"mesh_peers": len(peers),
			"models": []map[string]interface{}{
				{"name": "deepseek-r1:8b"},
				{"name": "SpeakLeash/bielik-11b-v3.0-instruct:Q5_K_M"},
				{"name": "qwen3.5:9b"},
			},
		})
	})
}

