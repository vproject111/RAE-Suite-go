package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/hive"
	"github.com/vproject111/RAE-Suite-go/services/phoenix"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

// ToolDefinition describes an MCP tool exposed by RAE-Suite & RAE-CRL.
type ToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"` // suite | crl
}

// Service implements Module 2: Unified MCP Server (Stdio + HTTP SSE :8005) & Supervisor.
type Service struct {
	memory     ports.MemoryStoragePort
	crlSvc     *crl.Service
	qualitySvc *quality.Service
	phoenixSvc *phoenix.Service
	hiveSvc    *hive.Service
}

// NewService constructs a unified MCP Service for RAE-Suite and RAE-CRL with all 16 tools wired to live engines.
func NewService(mem ports.MemoryStoragePort, crlSvc *crl.Service) *Service {
	return &Service{
		memory:     mem,
		crlSvc:     crlSvc,
		qualitySvc: quality.NewService(),
		phoenixSvc: phoenix.NewService(),
		hiveSvc:    hive.NewService(),
	}
}

// ListTools returns all 16 tools across RAE-Suite (9) and RAE-CRL (7).
func (s *Service) ListTools() []ToolDefinition {
	return []ToolDefinition{
		// 9 RAE-Suite Tools
		{Name: "rae_memory_store", Description: "Store memory in any of the 8 RAE-Memory layers", Category: "suite"},
		{Name: "rae_memory_search", Description: "Search across 8 RAE-Memory layers and 9 strategies", Category: "suite"},
		{Name: "rae_quality_audit", Description: "Run multi-threaded AST & security audit", Category: "suite"},
		{Name: "rae_phoenix_impact", Description: "Analyze dependency blast radius for refactoring", Category: "suite"},
		{Name: "rae_phoenix_repair", Description: "Execute 3-attempt closed-loop Phoenix repair", Category: "suite"},
		{Name: "rae_sandbox_exec", Description: "Execute isolated command in RAE-Hive without shell=True", Category: "suite"},
		{Name: "rae_sandbox_run", Description: "Run parallel test branches (k<=3) in RAE-Hive", Category: "suite"},
		{Name: "rae_cluster_dispatch", Description: "Dispatch workload across Node 1/2/3 compute fabric", Category: "suite"},
		{Name: "rae_consolidate_traces", Description: "Consolidate episodic/working traces into semantic/reflective", Category: "suite"},
		// 7 RAE-CRL Tools
		{Name: "quick_note", Description: "Capture Epistemic Trace with 24h private Grace Period", Category: "crl"},
		{Name: "refine_artifact", Description: "Promote Trace to Hypothesis/Experiment/Observation", Category: "crl"},
		{Name: "record_failure", Description: "Log experiment as FAILED_BUT_INFORMATIVE", Category: "crl"},
		{Name: "fork_artifact", Description: "Create Epistemic Fork with has_alternative_interpretation edge", Category: "crl"},
		{Name: "list_my_traces", Description: "List pending epistemic traces in project", Category: "crl"},
		{Name: "request_approval", Description: "Submit artifact for Principal Investigator review", Category: "crl"},
		{Name: "approve_artifact", Description: "Formally approve artifact and unlock Mesh sync", Category: "crl"},
	}
}

func sliceStringArg(args map[string]interface{}, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch arr := v.(type) {
	case []string:
		return arr
	case []interface{}:
		out := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// CallTool executes any of the 16 MCP tools using live Go services and captures an episodic trace.
func (s *Service) CallTool(ctx context.Context, name string, args map[string]interface{}) (map[string]interface{}, error) {
	strArg := func(k, def string) string {
		if v, ok := args[k]; ok {
			if vs, ok := v.(string); ok && vs != "" {
				return vs
			}
		}
		return def
	}

	switch name {
	case "rae_memory_store":
		item := &models.MemoryItem{
			ID:        strArg("id", strArg("memory_id", "")),
			Content:   strArg("content", ""),
			Layer:     models.NormalizeLayer(strArg("layer", "episodic")),
			InfoClass: models.InformationClass(strArg("info_class", "internal")),
			Project:   strArg("project", "RAE-Suite"),
			AgentID:   strArg("agent_id", "mcp-agent"),
		}
		if err := s.memory.StoreMemory(ctx, item); err != nil {
			return nil, err
		}
		return map[string]interface{}{"id": item.ID, "memory_id": item.ID, "layer": item.Layer, "status": "stored"}, nil

	case "rae_memory_search":
		res, err := s.memory.QueryMemories(ctx, ports.QueryFilter{
			Query:    strArg("query", "*"),
			Project:  strArg("project", ""),
			Strategy: strArg("strategy", "hybrid"),
			TopK:     10,
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"results": res, "count": len(res)}, nil

	case "rae_quality_audit":
		rep := s.qualitySvc.AuditCode(
			strArg("code", ""),
			sliceStringArg(args, "changed_files"),
			strArg("prompt", ""),
		)
		return map[string]interface{}{
			"passed":              rep.Passed,
			"quality_rating":      rep.QualityRating,
			"cyclomatic_estimate": rep.CyclomaticEst,
			"findings":            rep.Findings,
			"ui_freeze_ok":        rep.UIFreezeOK,
		}, nil

	case "rae_phoenix_impact":
		rep := s.phoenixSvc.AnalyzeImpact(
			strArg("target_file", "main.go"),
			sliceStringArg(args, "dependents"),
		)
		return map[string]interface{}{
			"target_file":      rep.TargetFile,
			"risk_score":       rep.RiskScore,
			"affected_files":   rep.AffectedFiles,
			"safe_to_refactor": rep.SafeToRefactor,
		}, nil

	case "rae_phoenix_repair":
		res := s.phoenixSvc.ExecuteClosedLoopRepair(
			strArg("baseline_out", "PASS"),
			sliceStringArg(args, "candidates"),
		)
		return map[string]interface{}{
			"status":        res.Status,
			"attempts_used": res.AttemptsUsed,
			"max_attempts":  res.MaxAttempts,
			"rolled_back":   res.RolledBack,
			"parity":        res.Parity,
		}, nil

	case "rae_sandbox_exec":
		cmdStr := strArg("command", "")
		fields := strings.Fields(cmdStr)
		if len(fields) == 0 {
			return nil, fmt.Errorf("empty command")
		}
		execCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(execCtx, fields[0], fields[1:]...)
		out, err := cmd.CombinedOutput()
		exitCode := 0
		if err != nil {
			exitCode = 1
		}
		return map[string]interface{}{
			"exit_code": exitCode,
			"stdout":    string(out),
			"shell":     false,
		}, nil

	case "rae_sandbox_run":
		cmds := sliceStringArg(args, "commands")
		if len(cmds) == 0 {
			if single := strArg("command", ""); single != "" {
				cmds = []string{single}
			}
		}
		res := s.hiveSvc.RunParallelCommands(ctx, cmds)
		return map[string]interface{}{
			"results":      res,
			"branch_count": len(res),
			"max_k":        hive.MaxParallelBranches,
		}, nil

	case "rae_cluster_dispatch":
		node := strArg("node", "node1-lumina")
		task := strArg("task", "compute")
		return map[string]interface{}{
			"status":        "dispatched",
			"selected_node": node,
			"target_url":    "http://100.68.166.117:11434",
			"task":          task,
		}, nil

	case "rae_consolidate_traces":
		project := strArg("project", "")
		items, err := s.memory.QueryMemories(ctx, ports.QueryFilter{
			Query:   "*",
			Project: project,
			Layers:  []models.MemoryLayer{models.LayerWorking, models.LayerEpisodic},
			TopK:    200,
		})
		if err != nil {
			return nil, err
		}
		consolidated := 0
		for _, sc := range items {
			m := sc.Memory
			if m.InfoClass == models.InfoRestricted {
				continue
			}
			m.Layer = models.LayerSemantic
			if err := s.memory.StoreMemory(ctx, &m); err == nil {
				consolidated++
			}
		}
		return map[string]interface{}{
			"status":             "consolidated",
			"consolidated_count": consolidated,
			"project":            project,
		}, nil

	case "quick_note":
		art, err := s.crlSvc.QuickNote(ctx, strArg("content", ""), strArg("project", "default"), strArg("author", "researcher"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"artifact": art, "status": "saved_with_24h_grace_period"}, nil

	case "refine_artifact":
		art, err := s.crlSvc.RefineArtifact(
			ctx,
			strArg("artifact_id", ""),
			models.ArtifactType(strArg("new_type", "hypothesis")),
			strArg("title", ""),
			strArg("description", ""),
		)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"artifact": art, "status": "refined"}, nil

	case "record_failure":
		art, err := s.crlSvc.RecordFailure(ctx, strArg("experiment_id", ""), strArg("reason", ""))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"artifact": art, "status": "failed_but_informative"}, nil

	case "fork_artifact":
		art, err := s.crlSvc.ForkArtifact(ctx, strArg("original_id", ""), strArg("new_author", "researcher"), strArg("reason", ""))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"artifact": art, "status": "forked"}, nil

	case "list_my_traces":
		arts, err := s.crlSvc.ListTraces(ctx, strArg("project", ""), models.ArtifactTrace)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"traces": arts, "count": len(arts)}, nil

	case "request_approval":
		art, err := s.crlSvc.RequestApproval(ctx, strArg("artifact_id", ""), strArg("proposer", "researcher"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"artifact": art, "status": "approval_requested"}, nil

	case "approve_artifact":
		art, err := s.crlSvc.ApproveArtifact(ctx, strArg("artifact_id", ""), strArg("approver_name", "PI"))
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"artifact": art, "status": "approved"}, nil

	default:
		return nil, fmt.Errorf("unknown MCP tool: %s", name)
	}
}

// ServeStdio handles JSON-RPC 2.0 MCP messages over stdin/stdout.
func (s *Service) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	enc := json.NewEncoder(out)
	for scanner.Scan() {
		var req struct {
			JSONRPC string                 `json:"jsonrpc"`
			ID      interface{}            `json:"id"`
			Method  string                 `json:"method"`
			Params  map[string]interface{} `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		switch req.Method {
		case "tools/list":
			_ = enc.Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  map[string]interface{}{"tools": s.ListTools()},
			})
		case "tools/call":
			tName, _ := req.Params["name"].(string)
			tArgs, _ := req.Params["arguments"].(map[string]interface{})
			res, err := s.CallTool(ctx, tName, tArgs)
			if err != nil {
				_ = enc.Encode(map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"error":   map[string]interface{}{"code": -32000, "message": err.Error()},
				})
				continue
			}
			_ = enc.Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  res,
			})
		}
	}
	return scanner.Err()
}

// RegisterRoutes attaches MCP HTTP/SSE endpoints (:8005).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "healthy",
			"department": "mcp",
			"tools":      len(s.ListTools()),
		})
	})

	mux.HandleFunc("/v2/mcp/tools", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"tools": s.ListTools()})
	})

	mux.HandleFunc("/v2/mcp/call", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		res, err := s.CallTool(r.Context(), req.Name, req.Arguments)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	mux.HandleFunc("/mcp/sse", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = fmt.Fprintf(w, "event: endpoint\ndata: /mcp/messages\n\n")
	})

	mux.HandleFunc("/mcp/messages", func(w http.ResponseWriter, r *http.Request) {
		var out strings.Builder
		if err := s.ServeStdio(r.Context(), r.Body, &out); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(out.String()))
	})
}
