package crl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/epistemic"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

// Service implements Module 9: RAE-CRL (Cognitive Research Layer v2.2).
// It manages the 7 closed epistemic artifact types, 24h Grace Period quarantine,
// FAILED_BUT_INFORMATIVE negative knowledge preservation, Epistemic Forks,
// PI Governance, Typed Provenance DAG, CBOR Event Journal, SBOM/VEX Evidence Packages,
// and bidirectional mapping to all 8 RAE-Memory layers.
type Service struct {
	epistemic ports.EpistemicStoragePort
	memory    ports.MemoryStoragePort
	guard     *security.HardFramesGuard
}

// NewService constructs a new RAE-CRL Service.
func NewService(ep ports.EpistemicStoragePort, mem ports.MemoryStoragePort, guard *security.HardFramesGuard) *Service {
	if guard == nil {
		guard = security.NewHardFramesGuard("")
	}
	return &Service{
		epistemic: ep,
		memory:    mem,
		guard:     guard,
	}
}

// QuickNote creates a private Epistemic Trace protected by a 24-hour Grace Period.
func (s *Service) QuickNote(ctx context.Context, content, project, author string) (*models.EpistemicArtifact, error) {
	if author == "" {
		author = "researcher"
	}
	title := content
	if len(title) > 50 {
		title = title[:50] + "..."
	}
	graceEnd := time.Now().UTC().Add(24 * time.Hour)
	trace := &models.EpistemicArtifact{
		ID:             models.GenerateUUIDv7(),
		TenantID:       models.GenerateULID(),
		Project:        project,
		Type:           models.ArtifactTrace,
		Title:          title,
		Description:    content,
		Status:         models.StatusDraft,
		Visibility:     models.VisibilityPrivate,
		InfoClass:      models.InfoRestricted,
		GracePeriodEnd: &graceEnd,
		Author:         author,
		MetadataBlob:   map[string]interface{}{"quarantine_hours": 24},
	}
	if err := s.epistemic.SaveArtifact(ctx, trace); err != nil {
		return nil, err
	}
	_ = s.epistemic.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{
		TenantID:   trace.TenantID,
		Project:    project,
		ArtifactID: trace.ID,
		Action:     "QUICK_NOTE_CREATED",
		Actor:      author,
	})
	if s.memory != nil {
		_ = s.memory.StoreMemory(ctx, &models.MemoryItem{
			Content:   content,
			Layer:     models.LayerWorking,
			InfoClass: models.InfoRestricted,
			Project:   project,
			AgentID:   author,
			Tags:      []string{"crl_trace", "grace_period"},
		})
	}
	return trace, nil
}

// RefineArtifact promotes a Trace into a formal Epistemic Artifact (Hypothesis, Experiment, etc.),
// ends its 24h Grace Period, and projects it to Semantic or Procedural memory.
func (s *Service) RefineArtifact(ctx context.Context, id string, newType models.ArtifactType, title, description string) (*models.EpistemicArtifact, error) {
	art, err := s.epistemic.GetArtifact(ctx, id)
	if err != nil {
		return nil, err
	}
	art.Type = newType
	if title != "" {
		art.Title = title
	}
	if description != "" {
		art.Description = description
	}
	art.GracePeriodEnd = nil
	art.Status = models.StatusActive
	art.Visibility = models.VisibilityTeam
	art.InfoClass = models.InfoInternal
	if err := s.epistemic.SaveArtifact(ctx, art); err != nil {
		return nil, err
	}
	_ = s.epistemic.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{
		TenantID:   art.TenantID,
		Project:    art.Project,
		ArtifactID: art.ID,
		Action:     "ARTIFACT_REFINED_" + strings.ToUpper(string(newType)),
		Actor:      art.Author,
	})
	if s.memory != nil {
		targetLayer := models.LayerSemantic
		if newType == models.ArtifactExperiment {
			targetLayer = models.LayerProcedural
		}
		_ = s.memory.StoreMemory(ctx, &models.MemoryItem{
			Content:    fmt.Sprintf("[%s] %s: %s", newType, art.Title, art.Description),
			Layer:      targetLayer,
			InfoClass:  models.InfoInternal,
			Project:    art.Project,
			AgentID:    art.Author,
			Importance: 0.75,
			Tags:       []string{"rae_crl", string(newType)},
		})
	}
	return art, nil
}

// RecordFailure preserves negative experimental outcomes as FAILED_BUT_INFORMATIVE.
func (s *Service) RecordFailure(ctx context.Context, experimentID, reason string) (*models.EpistemicArtifact, error) {
	exp, err := s.epistemic.GetArtifact(ctx, experimentID)
	if err != nil {
		return nil, err
	}
	exp.Status = models.StatusFailedButInformative
	if exp.MetadataBlob == nil {
		exp.MetadataBlob = make(map[string]interface{})
	}
	exp.MetadataBlob["failure_reason"] = reason
	if err := s.epistemic.SaveArtifact(ctx, exp); err != nil {
		return nil, err
	}
	_ = s.epistemic.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{
		TenantID:   exp.TenantID,
		Project:    exp.Project,
		ArtifactID: exp.ID,
		Action:     "FAILED_BUT_INFORMATIVE_RECORDED",
		Actor:      exp.Author,
	})
	if s.memory != nil {
		_ = s.memory.StoreMemory(ctx, &models.MemoryItem{
			Content:    fmt.Sprintf("FAILED_BUT_INFORMATIVE (%s): %s", exp.Title, reason),
			Layer:      models.LayerReflective,
			InfoClass:  models.InfoInternal,
			Project:    exp.Project,
			AgentID:    exp.Author,
			Importance: 0.90,
			Governance: models.GovernanceMeta{
				IsFailure:       true,
				FailureTrace:    reason,
				EpistemicStatus: string(models.StatusFailedButInformative),
				ReflectionTier:  "L3",
			},
			Tags: []string{"rae_crl", "informative_failure", "negative_knowledge"},
		})
	}
	return exp, nil
}

// ForkArtifact creates a competing epistemic interpretation linked by 'has_alternative_interpretation'.
func (s *Service) ForkArtifact(ctx context.Context, originalID, newAuthor, reason string) (*models.EpistemicArtifact, error) {
	orig, err := s.epistemic.GetArtifact(ctx, originalID)
	if err != nil {
		return nil, err
	}
	fork := &models.EpistemicArtifact{
		ID:          models.GenerateUUIDv7(),
		TenantID:    orig.TenantID,
		Project:     orig.Project,
		Type:        orig.Type,
		Title:       "Fork of: " + orig.Title,
		Description: fmt.Sprintf("Fork Reason: %s\n\nOriginal Content: %s", reason, orig.Description),
		Status:      models.StatusDraft,
		Visibility:  models.VisibilityTeam,
		InfoClass:   models.InfoInternal,
		Author:      newAuthor,
		MetadataBlob: map[string]interface{}{
			"forked_from": originalID,
			"reason":      reason,
		},
	}
	if err := s.epistemic.SaveArtifact(ctx, fork); err != nil {
		return nil, err
	}
	if err := s.epistemic.LinkArtifacts(ctx, orig.ID, fork.ID, "has_alternative_interpretation"); err != nil {
		return nil, err
	}
	if s.memory != nil {
		_ = s.memory.AddGraphEdge(ctx, orig.ID, fork.ID, "has_alternative_interpretation")
	}
	return fork, nil
}

// RequestApproval submits an artifact for Principal Investigator (PI) review.
func (s *Service) RequestApproval(ctx context.Context, artifactID, proposer string) (*models.EpistemicArtifact, error) {
	if proposer == "" {
		proposer = "researcher"
	}
	art, err := s.epistemic.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}
	art.ProposedBy = proposer
	art.Visibility = models.VisibilityTeam
	art.InfoClass = models.InfoInternal
	art.GracePeriodEnd = nil
	if err := s.epistemic.SaveArtifact(ctx, art); err != nil {
		return nil, err
	}
	_ = s.epistemic.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{
		TenantID:   art.TenantID,
		Project:    art.Project,
		ArtifactID: art.ID,
		Action:     "APPROVAL_REQUESTED",
		Actor:      proposer,
	})
	return art, nil
}

// ApproveArtifact performs Principal Investigator (PI) approval and unlocks team/public sync.
func (s *Service) ApproveArtifact(ctx context.Context, artifactID, approver string) (*models.EpistemicArtifact, error) {
	if approver == "" {
		return nil, errors.New("approver name required")
	}
	art, err := s.epistemic.GetArtifact(ctx, artifactID)
	if err != nil {
		return nil, err
	}
	art.ApprovedBy = approver
	art.Status = models.StatusActive
	art.Visibility = models.VisibilityPublic
	art.InfoClass = models.InfoPublic
	art.GracePeriodEnd = nil
	now := time.Now().UTC()
	art.LastSyncedAt = &now
	if err := s.epistemic.SaveArtifact(ctx, art); err != nil {
		return nil, err
	}
	_ = s.epistemic.AppendJournalEvent(ctx, &models.EpistemicJournalEvent{
		TenantID:   art.TenantID,
		Project:    art.Project,
		ArtifactID: art.ID,
		Action:     "ARTIFACT_APPROVED",
		Actor:      approver,
	})
	return art, nil
}

// ListTraces returns artifacts for a project and optional type.
func (s *Service) ListTraces(ctx context.Context, project string, filterType models.ArtifactType) ([]models.EpistemicArtifact, error) {
	return s.epistemic.ListArtifacts(ctx, project, filterType)
}

// DetectLogicalConflicts scans project artifacts for contradictory claims or divergent forks.
func (s *Service) DetectLogicalConflicts(ctx context.Context, project string) ([]map[string]interface{}, error) {
	arts, err := s.epistemic.ListArtifacts(ctx, project, "")
	if err != nil {
		return nil, err
	}
	edges, _ := s.epistemic.ListEdges(ctx, project)
	var conflicts []map[string]interface{}
	for _, e := range edges {
		if e.RelationType == "contradicts" || e.RelationType == "has_alternative_interpretation" {
			conflicts = append(conflicts, map[string]interface{}{
				"source_id":     e.SourceID,
				"target_id":     e.TargetID,
				"relation_type": e.RelationType,
				"status":        "CANDIDATE_CONFLICT",
			})
		}
	}
	for i := 0; i < len(arts); i++ {
		for j := i + 1; j < len(arts); j++ {
			a, b := arts[i], arts[j]
			if a.Type == models.ArtifactHypothesis && b.Type == models.ArtifactHypothesis {
				if (strings.Contains(strings.ToLower(a.Description), "increase") && strings.Contains(strings.ToLower(b.Description), "decrease")) ||
					(strings.Contains(strings.ToLower(a.Description), "always") && strings.Contains(strings.ToLower(b.Description), "never")) {
					conflicts = append(conflicts, map[string]interface{}{
						"source_id":     a.ID,
						"target_id":     b.ID,
						"relation_type": "semantic_contradiction",
						"status":        "CANDIDATE_CONFLICT",
					})
				}
			}
		}
	}
	return conflicts, nil
}

// SealEvidencePackage seals non-private project artifacts into a signed EvidencePackage (with SBOM/VEX)
// and archives the sealed snapshot into the RAE-Memory LTM (Long-Term Memory) layer.
func (s *Service) SealEvidencePackage(ctx context.Context, project, sbomRef, vexStatus string, mlbom []string) (*models.EvidencePackage, error) {
	if sbomRef == "" {
		return nil, errors.New("RAE-CRL Section 13 violation: SBOM reference is mandatory for sealing")
	}
	if vexStatus == "" {
		vexStatus = "NOT_AFFECTED"
	}
	arts, err := s.epistemic.ListArtifacts(ctx, project, "")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var exportable []models.EpistemicArtifact
	cborMap := map[string]string{
		"project":   project,
		"sbom_ref":  sbomRef,
		"vex":       vexStatus,
		"sealed_at": now.Format(time.RFC3339),
	}
	for _, a := range arts {
		if a.IsInGracePeriod(now) {
			continue
		}
		exportable = append(exportable, a)
		cborMap["art_"+a.ID] = a.VersionCID
	}
	edges, _ := s.epistemic.ListEdges(ctx, project)
	versionCID := models.ComputeVersionCID(cborMap)
	sig := s.guard.SignTokenHMAC(versionCID + "|" + sbomRef + "|" + vexStatus)

	pkg := &models.EvidencePackage{
		PackageID:     models.GenerateUUIDv7(),
		Project:       project,
		VersionCID:    versionCID,
		Artifacts:     exportable,
		Edges:         edges,
		SBOMReference: sbomRef,
		VEXStatus:     vexStatus,
		MLBOMModels:   mlbom,
		SignatureHMAC: sig,
		SealedAt:      now,
	}
	if s.memory != nil {
		_ = s.memory.StoreMemory(ctx, &models.MemoryItem{
			Content:    fmt.Sprintf("Sealed Evidence Package %s for project %s (CID=%s, SBOM=%s)", pkg.PackageID, project, versionCID, sbomRef),
			Layer:      models.LayerLTM,
			InfoClass:  models.InfoPublic,
			Project:    project,
			Importance: 1.0,
			Tags:       []string{"sealed_version", "evidence_package", "ltm"},
		})
	}
	return pkg, nil
}

// VerifyPackageOffline verifies a sealed EvidencePackage's CBOR VersionCID, SBOM/VEX, and HMAC-SHA256 signature offline.
func (s *Service) VerifyPackageOffline(pkg *models.EvidencePackage) error {
	return epistemic.VerifyEvidencePackageOffline(pkg, s.guard)
}

// StartSyncEngine launches the background RAE-CRL SyncEngine that sweeps expired Grace Periods.
func (s *Service) StartSyncEngine(ctx context.Context, interval time.Duration) {
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
				arts, err := s.epistemic.ListArtifacts(ctx, "", "")
				if err != nil {
					continue
				}
				now := time.Now().UTC()
				for i := range arts {
					a := arts[i]
					if a.GracePeriodEnd != nil && now.After(*a.GracePeriodEnd) {
						a.GracePeriodEnd = nil
						_ = s.epistemic.SaveArtifact(ctx, &a)
					}
				}
			}
		}
	}()
}

// RegisterRoutes attaches RAE-CRL REST API and embedded WCAG Web Lab Desk routes (:8503).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":     "healthy",
			"department": "crl",
			"version":    "4.1.0",
		})
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><title>RAE-CRL Cognitive Research Layer</title></head><body style="background:#0f172a;color:#f8fafc;font-family:sans-serif;padding:2rem;"><h1>RAE-CRL v4.1 (Go Native Lab Desk)</h1><p>Epistemic Continuity, 24h Grace Period Traces, Typed Provenance DAG &amp; SBOM/VEX Evidence Sealing.</p></body></html>`))
	})

	mux.HandleFunc("/v2/crl/traces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var req struct {
				Content string `json:"content"`
				Project string `json:"project"`
				Author  string `json:"author"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			art, err := s.QuickNote(r.Context(), req.Content, req.Project, req.Author)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, art)
			return
		}
		project := r.URL.Query().Get("project")
		arts, err := s.ListTraces(r.Context(), project, "")
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"results": arts})
	})

	mux.HandleFunc("/v2/crl/artifacts/refine", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ArtifactID  string              `json:"artifact_id"`
			NewType     models.ArtifactType `json:"new_type"`
			Title       string              `json:"title"`
			Description string              `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		art, err := s.RefineArtifact(r.Context(), req.ArtifactID, req.NewType, req.Title, req.Description)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, art)
	})

	mux.HandleFunc("/v2/crl/experiments/failure", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ExperimentID string `json:"experiment_id"`
			Reason       string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		art, err := s.RecordFailure(r.Context(), req.ExperimentID, req.Reason)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, art)
	})

	mux.HandleFunc("/v2/crl/artifacts/fork", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OriginalID string `json:"original_id"`
			NewAuthor  string `json:"new_author"`
			Reason     string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		art, err := s.ForkArtifact(r.Context(), req.OriginalID, req.NewAuthor, req.Reason)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, art)
	})

	mux.HandleFunc("/v2/crl/artifacts/request-approval", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ArtifactID string `json:"artifact_id"`
			Proposer   string `json:"proposer"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		art, err := s.RequestApproval(r.Context(), req.ArtifactID, req.Proposer)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, art)
	})

	mux.HandleFunc("/v2/crl/artifacts/approve", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ArtifactID   string `json:"artifact_id"`
			ApproverName string `json:"approver_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		art, err := s.ApproveArtifact(r.Context(), req.ArtifactID, req.ApproverName)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, art)
	})

	mux.HandleFunc("/v2/crl/conflicts", func(w http.ResponseWriter, r *http.Request) {
		project := r.URL.Query().Get("project")
		conflicts, err := s.DetectLogicalConflicts(r.Context(), project)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"conflicts": conflicts})
	})

	mux.HandleFunc("/v2/crl/seal", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Project       string   `json:"project"`
			SBOMReference string   `json:"sbom_reference"`
			VEXStatus     string   `json:"vex_status"`
			MLBOMModels   []string `json:"mlbom_models"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		pkg, err := s.SealEvidencePackage(r.Context(), req.Project, req.SBOMReference, req.VEXStatus, req.MLBOMModels)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, pkg)
	})

	mux.HandleFunc("/v2/crl/verify", func(w http.ResponseWriter, r *http.Request) {
		var pkg models.EvidencePackage
		if err := json.NewDecoder(r.Body).Decode(&pkg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := s.VerifyPackageOffline(&pkg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]interface{}{"verified": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"verified":    true,
			"package_id":  pkg.PackageID,
			"version_cid": pkg.VersionCID,
		})
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
