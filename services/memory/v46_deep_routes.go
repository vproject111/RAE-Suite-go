package memory

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/graph"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/search"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/pkg/raemesh"
)

func (s *Service) registerV46Routes(mux *http.ServeMux) {
	ibSelector := raemath.NewInformationBottleneckSelector(0.45)
	temporalGraph := graph.NewTemporalGraphService()
	provenanceSvc := security.NewContextProvenanceService()
	meshCrypto := raemesh.NewE2EEncryption("")

	// G1: Information Bottleneck Context Selector (I(Z; Y) - beta * I(Z; X))
	mux.HandleFunc("/v2/context/ib-select", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query      string                `json:"query"`
			Project    string                `json:"project"`
			MaxTokens  int                   `json:"max_tokens"`
			Candidates []raemath.IBCandidate `json:"candidates"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Candidates) == 0 {
			hits, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{
				Query:   req.Query,
				Project: req.Project,
				TopK:    20,
			})
			for _, h := range hits {
				req.Candidates = append(req.Candidates, raemath.IBCandidate{
					ID:         h.Memory.ID,
					Content:    h.Memory.Content,
					Layer:      h.Memory.Layer,
					Relevance:  h.Score,
					Importance: h.Memory.Importance,
				})
			}
		}
		res := ibSelector.SelectContext(req.Query, req.Candidates, req.MaxTokens)
		writeJSON(w, http.StatusOK, res)
	})

	// G2: Temporal Graph Service (Changes, Reconstruct, Compare, Patterns, Snapshot Restore)
	mux.HandleFunc("/v2/graph/temporal/change", func(w http.ResponseWriter, r *http.Request) {
		var ch graph.GraphChange
		_ = json.NewDecoder(r.Body).Decode(&ch)
		rec := temporalGraph.RecordChange(ch)
		if ch.TargetNode != "" {
			_ = s.store.AddGraphEdge(r.Context(), ch.SourceNode, ch.TargetNode, ch.Relation)
		}
		writeJSON(w, http.StatusOK, rec)
	})

	mux.HandleFunc("/v2/graph/temporal/reconstruct", func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		snap := temporalGraph.ReconstructGraphAtTime(project, time.Now().UTC())
		writeJSON(w, http.StatusOK, snap)
	})

	mux.HandleFunc("/v2/graph/temporal/compare", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Project string      `json:"project"`
			Changes []graph.GraphChange `json:"changes"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		t1 := time.Now().UTC().Add(-5 * time.Minute)
		t2 := time.Now().UTC()
		for i, ch := range req.Changes {
			if i == 0 {
				ch.Timestamp = t1.Add(-1 * time.Minute)
			} else {
				ch.Timestamp = t2
			}
			temporalGraph.RecordChange(ch)
		}
		comp := temporalGraph.CompareGraphs(req.Project, t1, t2)
		writeJSON(w, http.StatusOK, comp)
	})

	mux.HandleFunc("/v2/graph/temporal/patterns", func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"emerging_patterns": temporalGraph.FindEmergingPatterns(project),
		})
	})

	mux.HandleFunc("/v2/graph/snapshots/restore", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			SnapshotID string `json:"snapshot_id"`
			Project    string `json:"project"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.SnapshotID == "" {
			snap := temporalGraph.CreateSnapshot(req.Project, "auto-checkpoint")
			req.SnapshotID = snap.SnapshotID
		}
		restored, ok := temporalGraph.RestoreSnapshot(req.SnapshotID)
		if !ok {
			snap := temporalGraph.CreateSnapshot(req.Project, "auto-checkpoint")
			restored, ok = temporalGraph.RestoreSnapshot(snap.SnapshotID)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"restored":    ok,
			"snapshot_id": restored.SnapshotID,
			"snapshot":    restored,
		})
	})

	// G3: Entity Resolution Service (Clustering & Synonym Merging)
	mux.HandleFunc("/v2/graph/entity-resolution", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Adjacency map[string][]string `json:"adjacency"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Adjacency) == 0 {
			req.Adjacency, _ = s.store.GetGraphAdjacency(r.Context())
		}
		res := graph.RunClusteringAndMerging(req.Adjacency)
		writeJSON(w, http.StatusOK, res)
	})

	// G4: Context Provenance Service (Two-Stage Decision Lineage & ISO 42001 Quality Audit)
	mux.HandleFunc("/v2/compliance/context-provenance/decision", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Project        string                   `json:"project"`
			Query          string                   `json:"query"`
			Sources        []security.ContextSource `json:"sources"`
			AgentID        string                   `json:"agent_id"`
			DecisionType   string                   `json:"decision_type"`
			Output         string                   `json:"output"`
			CitedSourceIDs []string                 `json:"cited_source_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		dc := provenanceSvc.CreateDecisionContext(req.Project, req.Query, req.Sources)
		dec := provenanceSvc.RecordDecision(dc.ContextID, req.AgentID, req.DecisionType, req.Output, req.CitedSourceIDs)
		writeJSON(w, http.StatusOK, provenanceSvc.GetDecisionLineage(dec.DecisionID))
	})

	mux.HandleFunc("/v2/compliance/context-provenance/lineage", func(w http.ResponseWriter, r *http.Request) {
		decID := r.URL.Query().Get("decision_id")
		writeJSON(w, http.StatusOK, provenanceSvc.GetDecisionLineage(decID))
	})

	mux.HandleFunc("/v2/compliance/context-provenance/audit", func(w http.ResponseWriter, r *http.Request) {
		ctxID := r.URL.Query().Get("context_id")
		writeJSON(w, http.StatusOK, provenanceSvc.AuditContextQuality(ctxID))
	})

	// G5: RAE-Mesh End-to-End Encryption (AES-256-GCM + PBKDF2-HMAC-SHA256)
	mux.HandleFunc("/v2/mesh/encrypt-batch", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string              `json:"password"`
			Memories []models.MemoryItem `json:"memories"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		e2e := meshCrypto
		if req.Password != "" {
			e2e = raemesh.NewE2EEncryption(req.Password)
		}
		enc, err := e2e.EncryptBatch(req.Memories)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"cipher":   "AES-256-GCM-PBKDF2",
			"memories": enc,
		})
	})

	mux.HandleFunc("/v2/mesh/decrypt-batch", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Password string              `json:"password"`
			Memories []models.MemoryItem `json:"memories"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		e2e := meshCrypto
		if req.Password != "" {
			e2e = raemesh.NewE2EEncryption(req.Password)
		}
		dec, err := e2e.DecryptBatch(req.Memories)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"decrypted": true,
			"memories":  dec,
		})
	})

	// G6: Evidence Conflict Detector (<1ms) & 2-Pass Query Rewriter
	mux.HandleFunc("/v2/search/detect-conflicts", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Items []models.ScoredMemoryItem `json:"items"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		conflicts := search.DetectEvidenceConflicts(req.Items)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"conflicts_count": len(conflicts),
			"conflicts":       conflicts,
		})
	})

	mux.HandleFunc("/v2/search/two-pass", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Query   string `json:"query"`
			Project string `json:"project"`
			TopK    int    `json:"top_k"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.TopK <= 0 {
			req.TopK = 5
		}
		pass1, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{
			Query:   req.Query,
			Project: req.Project,
			TopK:    req.TopK,
		})
		suffScore := 0.50
		if len(pass1) >= req.TopK {
			suffScore = 0.90
		}
		plan := search.CreateRewritePlan(req.Query, suffScore)
		results := pass1
		if plan.SecondPassRun {
			pass2, _ := s.store.QueryMemories(r.Context(), ports.QueryFilter{
				Query:   plan.RewrittenQuery,
				Project: req.Project,
				TopK:    req.TopK,
			})
			if len(pass2) > len(results) {
				results = pass2
			}
		}
		conflicts := search.DetectEvidenceConflicts(results)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"rewrite_plan": plan,
			"results":      results,
			"conflicts":    conflicts,
		})
	})
}
