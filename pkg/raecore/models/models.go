package models

import (
	"strings"
	"time"
)

// MemoryLayer represents the 8 storage & retention layers of RAE-Memory.
type MemoryLayer string

const (
	LayerSensory    MemoryLayer = "sensory"
	LayerWorking    MemoryLayer = "working"
	LayerEpisodic   MemoryLayer = "episodic"
	LayerSemantic   MemoryLayer = "semantic"
	LayerProcedural MemoryLayer = "procedural"
	LayerReflective MemoryLayer = "reflective"
	LayerLTM        MemoryLayer = "ltm"
	LayerGraph      MemoryLayer = "graph"
)

// AllMemoryLayers returns all 8 canonical memory layers.
func AllMemoryLayers() []MemoryLayer {
	return []MemoryLayer{
		LayerSensory,
		LayerWorking,
		LayerEpisodic,
		LayerSemantic,
		LayerProcedural,
		LayerReflective,
		LayerLTM,
		LayerGraph,
	}
}

// NormalizeLayer maps legacy or case-variant layer names to the 8 canonical layers.
func NormalizeLayer(raw string) MemoryLayer {
	clean := strings.TrimSpace(strings.ToLower(raw))
	switch clean {
	case "sensory", "stm":
		return LayerSensory
	case "working", "wm":
		return LayerWorking
	case "episodic", "em":
		return LayerEpisodic
	case "semantic", "sm":
		return LayerSemantic
	case "procedural", "pm":
		return LayerProcedural
	case "reflective", "rm", "reflection":
		return LayerReflective
	case "ltm", "long_term", "longterm", "sealed":
		return LayerLTM
	case "graph", "kg", "knowledge_graph":
		return LayerGraph
	default:
		return LayerSemantic
	}
}

// MemoryType represents the functional content type of a memory record.
type MemoryType string

const (
	TypeText         MemoryType = "text"
	TypeCode         MemoryType = "code"
	TypeImage        MemoryType = "image"
	TypeDocument     MemoryType = "document"
	TypeConversation MemoryType = "conversation"
	TypeReflection   MemoryType = "reflection"
	TypeEntity       MemoryType = "entity"
	TypeRelationship MemoryType = "relationship"
)

// InformationClass represents ISO/IEC 27000 classification levels.
type InformationClass string

const (
	InfoPublic       InformationClass = "public"
	InfoInternal     InformationClass = "internal"
	InfoConfidential InformationClass = "confidential"
	InfoRestricted   InformationClass = "restricted"
)

// SourceTrustLevel represents ISO/IEC 42001 source reliability classification.
type SourceTrustLevel string

const (
	TrustHigh       SourceTrustLevel = "high"
	TrustMedium     SourceTrustLevel = "medium"
	TrustLow        SourceTrustLevel = "low"
	TrustUnverified SourceTrustLevel = "unverified"
)

// OperationRiskLevel represents ISO/IEC 42001 Section 9 risk levels.
type OperationRiskLevel string

const (
	RiskCritical OperationRiskLevel = "critical"
	RiskHigh     OperationRiskLevel = "high"
	RiskMedium   OperationRiskLevel = "medium"
	RiskLow      OperationRiskLevel = "low"
	RiskNone     OperationRiskLevel = "none"
)

// ReflectionType represents the category of generated reflection.
type ReflectionType string

const (
	ReflectionConsolidation ReflectionType = "consolidation"
	ReflectionPattern       ReflectionType = "pattern"
	ReflectionAnomaly       ReflectionType = "anomaly"
	ReflectionMeta          ReflectionType = "meta"
)

// MathLevel represents the 3-tier mathematical controller level (L1, L2, L3).
type MathLevel int

const (
	MathL1 MathLevel = 1
	MathL2 MathLevel = 2
	MathL3 MathLevel = 3
)

// ContextEnvelope encapsulates provenance and governance without polluting raw content.
type ContextEnvelope struct {
	OriginDevice      string           `json:"origin_device,omitempty"`
	SourceOwner       string           `json:"source_owner,omitempty"`
	TrustLevel        SourceTrustLevel `json:"trust_level"`
	LastVerifiedAt    *time.Time       `json:"last_verified_at,omitempty"`
	VerificationNotes string           `json:"verification_notes,omitempty"`
	Version           int              `json:"version"`
	SupersedesID      string           `json:"supersedes_id,omitempty"`
}

// GovernanceMeta holds ISO 42001 governance and Szubar Mode failure/fork metadata.
type GovernanceMeta struct {
	DecisionRationale string             `json:"decision_rationale,omitempty"`
	Confidence        float64            `json:"confidence"`
	RiskLevel         OperationRiskLevel `json:"risk_level"`
	IsFailure         bool               `json:"is_failure"`
	FailureTrace      string             `json:"failure_trace,omitempty"`
	DecisionForks     []string           `json:"decision_forks,omitempty"`
	EpistemicStatus   string             `json:"epistemic_status,omitempty"`
	ReflectionTier    string             `json:"reflection_tier,omitempty"` // L1, L2, L3, L4
}

// MemoryItem is the fundamental storage-agnostic unit of memory across all 8 layers.
type MemoryItem struct {
	ID             string                 `json:"id"`
	Content        string                 `json:"content"`
	EncryptedBytes []byte                 `json:"encrypted_bytes,omitempty"`
	IsEncrypted    bool                   `json:"is_encrypted,omitempty"`
	Layer          MemoryLayer            `json:"layer"`
	MemoryType     MemoryType             `json:"memory_type"`
	InfoClass      InformationClass       `json:"info_class"`
	Governance     GovernanceMeta         `json:"governance"`
	TenantID       string                 `json:"tenant_id"`
	AgentID        string                 `json:"agent_id"`
	Project        string                 `json:"project,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	NamedVectors   map[string][]float32   `json:"named_vectors,omitempty"` // e.g., "dense" (384d), "ollama" (768d), "openai" (1536d)
	Embedding      []float32              `json:"embedding,omitempty"`
	Importance     float64                `json:"importance"`
	Strength       float64                `json:"strength"`
	UsageCount     int                    `json:"usage_count"`
	CreatedAt      time.Time              `json:"created_at"`
	LastAccessedAt time.Time              `json:"last_accessed_at"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	Source         string                 `json:"source,omitempty"`
	Envelope       *ContextEnvelope       `json:"envelope,omitempty"`
	ContentHash    string                 `json:"content_hash,omitempty"`
}

// ScoredMemoryItem pairs a MemoryItem with its multi-dimensional score breakdown.
type ScoredMemoryItem struct {
	Memory         MemoryItem         `json:"memory"`
	Score          float64            `json:"score"`
	ScoreBreakdown map[string]float64 `json:"score_breakdown,omitempty"`
	StrategyUsed   string             `json:"strategy_used,omitempty"`
	ManifoldArm    string             `json:"manifold_arm,omitempty"`
}

// KnowledgeConflictRequest represents a Sycon Knowledge Graph conflict resolution request.
type KnowledgeConflictRequest struct {
	Project     string   `json:"project"`
	EntityID    string   `json:"entity_id"`
	Claims      []string `json:"claims"`
	Strategy    string   `json:"strategy,omitempty"` // trust_weighted, latest, consensus
	TrustScores []float64 `json:"trust_scores,omitempty"`
}

// ============================================================================
// RAE-CRL (Cognitive Research Layer) Canonical Models
// ============================================================================

// ArtifactType represents the 7 closed epistemic artifact types in RAE-CRL.
type ArtifactType string

const (
	ArtifactTrace       ArtifactType = "epistemic_trace"
	ArtifactHypothesis  ArtifactType = "hypothesis"
	ArtifactAssumption  ArtifactType = "assumption"
	ArtifactExperiment  ArtifactType = "experiment"
	ArtifactObservation ArtifactType = "observation"
	ArtifactDecision    ArtifactType = "decision"
	ArtifactReflection  ArtifactType = "reflection"
)

// ArtifactStatus represents the lifecycle state of a research artifact.
type ArtifactStatus string

const (
	StatusDraft                ArtifactStatus = "draft"
	StatusActive               ArtifactStatus = "active"
	StatusCompleted            ArtifactStatus = "completed"
	StatusFailedButInformative ArtifactStatus = "failed_but_informative"
	StatusDeprecated           ArtifactStatus = "deprecated"
	StatusSealed               ArtifactStatus = "sealed"
)

// ArtifactVisibility controls scope and Mesh transfer eligibility.
type ArtifactVisibility string

const (
	VisibilityPrivate ArtifactVisibility = "private" // Protected by 24h Grace Period
	VisibilityTeam    ArtifactVisibility = "team"
	VisibilityPublic  ArtifactVisibility = "public"
)

// EpistemicArtifact is the universal schema for RAE-CRL research artifacts and traces.
type EpistemicArtifact struct {
	ID             string                 `json:"id"`        // UUIDv7
	TenantID       string                 `json:"tenant_id"` // ULID
	Project        string                 `json:"project"`
	Type           ArtifactType           `json:"type"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	ResearchPhase  string                 `json:"research_phase,omitempty"`
	Status         ArtifactStatus         `json:"status"`
	Visibility     ArtifactVisibility     `json:"visibility"`
	InfoClass      InformationClass       `json:"info_class"`
	GracePeriodEnd *time.Time             `json:"grace_period_end,omitempty"`
	Author         string                 `json:"author"`
	ProposedBy     string                 `json:"proposed_by,omitempty"`
	ApprovedBy     string                 `json:"approved_by,omitempty"`
	MetadataBlob   map[string]interface{} `json:"metadata_blob,omitempty"`
	VersionCID     string                 `json:"version_cid,omitempty"` // Multihash of deterministic CBOR
	ProvenanceHash string                 `json:"provenance_hash,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	LastSyncedAt   *time.Time             `json:"last_synced_at,omitempty"`
}

// IsInGracePeriod checks whether a trace is still under private quarantine.
func (a *EpistemicArtifact) IsInGracePeriod(now time.Time) bool {
	if a.Visibility == VisibilityPrivate && a.GracePeriodEnd != nil {
		return now.Before(*a.GracePeriodEnd)
	}
	return false
}

// ArtifactEdge represents a typed directed edge in the RAE-CRL Provenance DAG.
type ArtifactEdge struct {
	SourceID     string    `json:"source_id"`
	TargetID     string    `json:"target_id"`
	RelationType string    `json:"relation_type"` // tests, supports, contradicts, refines, has_alternative_interpretation, derived_from
	CreatedAt    time.Time `json:"created_at"`
}

// EpistemicJournalEvent is an append-only event record encoded with deterministic CBOR.
type EpistemicJournalEvent struct {
	EventID      string                 `json:"event_id"` // UUIDv7
	TenantID     string                 `json:"tenant_id"`
	Project      string                 `json:"project"`
	ArtifactID   string                 `json:"artifact_id"`
	Action       string                 `json:"action"`
	Actor        string                 `json:"actor"`
	PrevEventCID string                 `json:"prev_event_cid"`
	EventCID     string                 `json:"event_cid"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

// EvidencePackage represents a signed, offline-verifiable RAE-CRL publication package.
type EvidencePackage struct {
	PackageID     string              `json:"package_id"`
	Project       string              `json:"project"`
	VersionCID    string              `json:"version_cid"`
	Artifacts     []EpistemicArtifact `json:"artifacts"`
	Edges         []ArtifactEdge      `json:"edges"`
	SBOMReference string              `json:"sbom_reference"`
	VEXStatus     string              `json:"vex_status"`
	MLBOMModels   []string            `json:"mlbom_models,omitempty"`
	SignatureHMAC string              `json:"signature_hmac"`
	SealedAt      time.Time           `json:"sealed_at"`
}
