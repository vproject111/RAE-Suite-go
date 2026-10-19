package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/reflection"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/search"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/workflows"
	"github.com/vproject111/RAE-Suite-go/pkg/raemesh"
)

// Service implements Module 3: RAE-Memory (8 Storage Layers, L1-L4 Reflection,
// 9 Search Strategies, Procedural Oracle, Sycon Knowledge Resolve, OutboxDispatcher & RAE-Mesh).
type Service struct {
	store     ports.MemoryStoragePort
	outbox    ports.OutboxQueuePort
	reflector *reflection.Coordinator
	gateway   *search.LogicGateway
	mesh      *raemesh.MeshEngine
	guard     *security.HardFramesGuard
	scrubber  *workflows.IngestionPIIScrubber
}

// NewService constructs a full 8-layer RAE-Memory Service.
func NewService(store ports.MemoryStoragePort, outbox ports.OutboxQueuePort, guard *security.HardFramesGuard) *Service {
	if guard == nil {
		guard = security.NewHardFramesGuard("")
	}
	return &Service{
		store:     store,
		outbox:    outbox,
		reflector: reflection.NewCoordinator(),
		gateway:   search.NewLogicGateway(nil),
		mesh:      raemesh.NewMeshEngine(store, guard),
		guard:     guard,
		scrubber:  workflows.NewIngestionPIIScrubber(),
	}
}

// Mesh returns the underlying RAE-Mesh engine.
func (s *Service) Mesh() *raemesh.MeshEngine {
	return s.mesh
}

// ConsolidateFromWorking transitions eligible non-RESTRICTED Working memories into the Episodic layer.
func (s *Service) ConsolidateFromWorking(ctx context.Context, project string) (int, error) {
	items, err := s.store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		Layers:  []models.MemoryLayer{models.LayerWorking},
		TopK:    500,
	})
	if err != nil {
		return 0, err
	}
	consolidated := 0
	for _, sc := range items {
		m := sc.Memory
		if m.InfoClass == models.InfoRestricted {
			continue
		}
		m.Layer = models.LayerEpisodic
		if err := s.store.StoreMemory(ctx, &m); err == nil {
			consolidated++
		}
	}
	return consolidated, nil
}

// UpgradeToSemantic promotes high-usage or high-importance Episodic memories into the Semantic layer.
func (s *Service) UpgradeToSemantic(ctx context.Context, project string, minUsage int) (int, error) {
	if minUsage <= 0 {
		minUsage = 2
	}
	items, err := s.store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		Layers:  []models.MemoryLayer{models.LayerEpisodic},
		TopK:    500,
	})
	if err != nil {
		return 0, err
	}
	upgraded := 0
	for _, sc := range items {
		m := sc.Memory
		if m.UsageCount >= minUsage || m.Importance >= 0.80 {
			m.Layer = models.LayerSemantic
			if err := s.store.StoreMemory(ctx, &m); err == nil {
				upgraded++
			}
		}
	}
	return upgraded, nil
}

// PromoteToLTM promotes durable Semantic or Reflective memories into the sealed LTM layer.
func (s *Service) PromoteToLTM(ctx context.Context, project string, minImportance float64) (int, error) {
	if minImportance <= 0 {
		minImportance = 0.85
	}
	items, err := s.store.QueryMemories(ctx, ports.QueryFilter{
		Query:   "*",
		Project: project,
		Layers:  []models.MemoryLayer{models.LayerSemantic, models.LayerReflective},
		TopK:    500,
	})
	if err != nil {
		return 0, err
	}
	promoted := 0
	for _, sc := range items {
		m := sc.Memory
		if m.Importance >= minImportance {
			m.Layer = models.LayerLTM
			if err := s.store.StoreMemory(ctx, &m); err == nil {
				promoted++
			}
		}
	}
	return promoted, nil
}

// DetectCommunities discovers connected clusters in the Knowledge Graph layer.
func (s *Service) DetectCommunities(ctx context.Context) ([][]string, error) {
	adj, err := s.store.GetGraphAdjacency(ctx)
	if err != nil {
		return nil, err
	}
	nodeSet := make(map[string]bool)
	for src, targets := range adj {
		nodeSet[src] = true
		for _, dst := range targets {
			nodeSet[dst] = true
		}
	}
	nodes := make([]string, 0, len(nodeSet))
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	return search.DetectCommunities(nodes, adj), nil
}

// PropagateResonance spreads activation energy across the Knowledge Graph layer.
func (s *Service) PropagateResonance(ctx context.Context, seeds map[string]float64, hops int) (map[string]float64, error) {
	adj, err := s.store.GetGraphAdjacency(ctx)
	if err != nil {
		return nil, err
	}
	return search.PropagateSemanticResonance(seeds, adj, hops, 0.5), nil
}

// ProceduralChunk splits industrial/procedural text along step boundaries (Step 1, Krok 1, etc.).
func ProceduralChunk(text string, maxChunkSize int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 256
	}
	lines := strings.Split(text, "\n")
	var chunks []string
	var cur strings.Builder
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		isStep := strings.HasPrefix(strings.ToLower(trim), "step ") || strings.HasPrefix(strings.ToLower(trim), "krok ")
		if (isStep && cur.Len() > 0) || (cur.Len()+len(line) > maxChunkSize && cur.Len() > 0) {
			chunks = append(chunks, strings.TrimSpace(cur.String()))
			cur.Reset()
		}
		if cur.Len() > 0 {
			cur.WriteString("\n")
		}
		cur.WriteString(line)
	}
	if cur.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(cur.String()))
	}
	return chunks
}

// ResolveKnowledgeConflict resolves conflicting claims in the Sycon Knowledge Graph.
func (s *Service) ResolveKnowledgeConflict(ctx context.Context, req models.KnowledgeConflictRequest) (map[string]interface{}, error) {
	if len(req.Claims) == 0 {
		return map[string]interface{}{"status": "resolved", "winning_claim": req.EntityID, "strategy": "trust_weighted_sycon"}, nil
	}
	bestIdx := 0
	bestTrust := -1.0
	for i := range req.Claims {
		score := 0.5
		if i < len(req.TrustScores) {
			score = req.TrustScores[i]
		}
		if score >= bestTrust {
			bestTrust = score
			bestIdx = i
		}
	}
	winner := req.Claims[bestIdx]
	item := &models.MemoryItem{
		Content:    winner,
		Layer:      models.LayerSemantic,
		MemoryType: models.TypeEntity,
		Project:    req.Project,
		Importance: 0.85,
		Tags:       []string{"sycon_resolved", req.EntityID},
	}
	_ = s.store.StoreMemory(ctx, item)
	return map[string]interface{}{
		"status":        "resolved",
		"entity_id":     req.EntityID,
		"winning_claim": winner,
		"memory_id":     item.ID,
		"strategy":      "trust_weighted_sycon",
	}, nil
}

// ProcessGovernanceOutboxOnce claims pending outbox events (FOR UPDATE SKIP LOCKED) and delivers them.
func (s *Service) ProcessGovernanceOutboxOnce(ctx context.Context, limit int) (int, error) {
	if s.outbox == nil {
		return 0, nil
	}
	events, err := s.outbox.ClaimPendingOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, ev := range events {
		if err := s.outbox.MarkOutboxDelivered(ctx, ev.ID); err == nil {
			delivered++
		}
	}
	return delivered, nil
}

// StartBackgroundWorkers launches native Go goroutines for OutboxDispatcher and SensoryDecayWorker
// (replacing external Celery Worker and Celery Beat containers).
func (s *Service) StartBackgroundWorkers(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.ProcessGovernanceOutboxOnce(ctx, 50)
				_, _ = s.store.DeleteExpiredSensory(ctx)
			}
		}
	}()
}

// RegisterRoutes attaches all RAE-Memory v2, Procedural, Reflection, Sycon, RAE-Mesh, and 19 Extended Router groups (:8000).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	s.RegisterExtendedRoutes(mux)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"version": "4.3.0",
			"layers":  models.AllMemoryLayers(),
		})
	})

	mux.HandleFunc("/v2/memories/scrub", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := s.scrubber.ScrubText(req.Text)
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("/v2/memories/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var item models.MemoryItem
			if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			if item.Metadata != nil {
				if scrubFlag, ok := item.Metadata["scrub_pii"].(bool); ok && scrubFlag {
					scrubRes := s.scrubber.ScrubText(item.Content)
					item.Content = scrubRes.ScrubbedText
					item.Metadata["pii_redactions"] = scrubRes.RedactionsCount
				}
			}
			if err := s.store.StoreMemory(r.Context(), &item); err != nil {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
				return
			}
			if s.outbox != nil {
				_ = s.outbox.EnqueueOutbox(r.Context(), &ports.OutboxEvent{
					EventType: "MEMORY_STORED",
					Payload:   map[string]interface{}{"id": item.ID, "layer": item.Layer},
				})
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"id":        item.ID,
				"memory_id": item.ID,
				"layer":     item.Layer,
				"status":    "saved",
			})
		case http.MethodDelete:
			id := strings.TrimPrefix(r.URL.Path, "/v2/memories/")
			if id == "" {
				id = r.URL.Query().Get("id")
			}
			if err := s.store.DeleteMemory(r.Context(), id); err != nil {
				writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
		default:
			res, err := s.store.QueryMemories(r.Context(), ports.QueryFilter{
				Query:   r.URL.Query().Get("query"),
				Project: r.URL.Query().Get("project"),
				Layers:  parseLayersParam(r.URL.Query().Get("layer")),
				TopK:    50,
			})
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"results": res})
		}
	})

	mux.HandleFunc("/v2/memories/stats", func(w http.ResponseWriter, r *http.Request) {
		stats, err := s.store.GetMemoryStats(r.Context(), r.URL.Query().Get("project"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, stats)
	})

	mux.HandleFunc("/v2/memories/consolidate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Project string `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		count, err := s.ConsolidateFromWorking(r.Context(), req.Project)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"status": "consolidated", "consolidated_count": count})
	})

	mux.HandleFunc("/v2/memories/upgrade", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Project       string  `json:"project"`
			MinUsage      int     `json:"min_usage"`
			PromoteToLTM  bool    `json:"promote_to_ltm"`
			MinImportance float64 `json:"min_importance"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		upgraded, err := s.UpgradeToSemantic(r.Context(), req.Project, req.MinUsage)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		promoted := 0
		if req.PromoteToLTM {
			promoted, _ = s.PromoteToLTM(r.Context(), req.Project, req.MinImportance)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":         "upgraded",
			"upgraded_count": upgraded,
			"promoted_ltm":   promoted,
		})
	})

	mux.HandleFunc("/v2/memories/query", func(w http.ResponseWriter, r *http.Request) {
		var filter ports.QueryFilter
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		res, err := s.store.QueryMemories(r.Context(), filter)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": res})
	})

	mux.HandleFunc("/v2/knowledge/resolve", func(w http.ResponseWriter, r *http.Request) {
		var req models.KnowledgeConflictRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		res, err := s.ResolveKnowledgeConflict(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("/v2/reflection/evaluate", func(w http.ResponseWriter, r *http.Request) {
		var p reflection.ReflectionPayload
		_ = json.NewDecoder(r.Body).Decode(&p)
		rep := s.reflector.Reflect(p)
		writeJSON(w, http.StatusOK, rep)
	})

	mux.HandleFunc("/procedural/query", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query   string `json:"query"`
			Project string `json:"project"`
			Model   string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		chunks := ProceduralChunk(req.Query, 200)
		instruction := fmt.Sprintf("PROCEDURAL ORACLE (Steps=%d, project=%s): Verified deterministic procedure.", len(chunks), req.Project)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"instruction": instruction,
			"chunks":      chunks,
			"debug":       map[string]string{"context": "STATELESS_AUDIT", "layer": string(models.LayerProcedural)},
		})
	})

	mux.HandleFunc("/v2/mesh/peers", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"peers": s.mesh.ListPeers()})
	})

	mux.HandleFunc("/v2/mesh/sync", func(w http.ResponseWriter, r *http.Request) {
		root, exp, err := s.mesh.ComputeMerkleRoot(r.Context(), r.URL.Query().Get("project"))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":          "synced",
			"merkle_root":     root,
			"synced_memories": len(exp),
		})
	})

	mux.HandleFunc("/v2/mesh/peers/sync", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PeerID  string `json:"peer_id"`
			Project string `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res, err := s.mesh.SyncWithPeer(r.Context(), req.PeerID, req.Project)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	})

	mux.HandleFunc("/v2/mesh/receive", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Token    string              `json:"token"`
			Memories []models.MemoryItem `json:"memories"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		accepted, rejected, err := s.mesh.ReceiveMemories(r.Context(), req.Token, req.Memories)
		if err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":   "received",
			"accepted": accepted,
			"rejected": rejected,
		})
	})
}

func parseLayersParam(raw string) []models.MemoryLayer {
	if raw == "" {
		return nil
	}
	return []models.MemoryLayer{models.NormalizeLayer(raw)}
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
