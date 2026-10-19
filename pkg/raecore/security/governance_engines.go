package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ports"
)

// ECARule defines an Event-Condition-Action rule in RulesEngine (apps/memory_api/services/rules_engine.py).
type ECARule struct {
	RuleID        string  `json:"rule_id"`
	Name          string  `json:"name"`
	EventType     string  `json:"event_type"` // on_memory_store, on_cost_threshold, on_drift_detected
	MinImportance float64 `json:"min_importance,omitempty"`
	RequiredClass string  `json:"required_info_class,omitempty"`
	Action        string  `json:"action"` // trigger_reflection, promote_semantic, quarantine_working, emit_alert
	Enabled       bool    `json:"enabled"`
}

// RuleExecutionResult records a fired ECA rule and its dispatched action.
type RuleExecutionResult struct {
	RuleID    string    `json:"rule_id"`
	RuleName  string    `json:"rule_name"`
	EventType string    `json:"event_type"`
	Action    string    `json:"action"`
	TargetID  string    `json:"target_id"`
	Timestamp time.Time `json:"timestamp"`
}

// RulesEngine evaluates Event-Condition-Action governance and automation triggers.
type RulesEngine struct {
	mu      sync.RWMutex
	rules   []ECARule
	history []RuleExecutionResult
}

// NewRulesEngine creates a RulesEngine pre-populated with canonical RAE ECA rules.
func NewRulesEngine() *RulesEngine {
	return &RulesEngine{
		rules: []ECARule{
			{
				RuleID:        "rule-auto-reflect-high-imp",
				Name:          "Auto-Reflect High Importance Episodic Memory",
				EventType:     "on_memory_store",
				MinImportance: 0.80,
				Action:        "trigger_reflection",
				Enabled:       true,
			},
			{
				RuleID:        "rule-quarantine-restricted",
				Name:          "Quarantine Restricted Data to Encrypted Working Layer",
				EventType:     "on_memory_store",
				RequiredClass: string(models.InfoRestricted),
				Action:        "quarantine_working",
				Enabled:       true,
			},
			{
				RuleID:    "rule-drift-recalibrate",
				Name:      "Trigger MAB Recalibration on Search Drift",
				EventType: "on_drift_detected",
				Action:    "recalibrate_mab_weights",
				Enabled:   true,
			},
		},
	}
}

// AddRule registers a new ECA rule.
func (re *RulesEngine) AddRule(r ECARule) ECARule {
	re.mu.Lock()
	defer re.mu.Unlock()
	if r.RuleID == "" {
		r.RuleID = "rule-" + models.GenerateUUIDv7()[:8]
	}
	r.Enabled = true
	re.rules = append(re.rules, r)
	return r
}

// EvaluateEvent evaluates all enabled ECA rules matching eventType against the provided payload.
func (re *RulesEngine) EvaluateEvent(eventType string, targetID string, importance float64, infoClass string) []RuleExecutionResult {
	re.mu.Lock()
	defer re.mu.Unlock()

	fired := make([]RuleExecutionResult, 0)
	now := time.Now().UTC()
	for _, r := range re.rules {
		if !r.Enabled || r.EventType != eventType {
			continue
		}
		if r.MinImportance > 0 && importance < r.MinImportance {
			continue
		}
		if r.RequiredClass != "" && !strings.EqualFold(r.RequiredClass, infoClass) {
			continue
		}
		res := RuleExecutionResult{
			RuleID:    r.RuleID,
			RuleName:  r.Name,
			EventType: eventType,
			Action:    r.Action,
			TargetID:  targetID,
			Timestamp: now,
		}
		fired = append(fired, res)
		re.history = append(re.history, res)
	}
	return fired
}

// ListRules returns all registered ECA rules and firing history count.
func (re *RulesEngine) ListRules() ([]ECARule, []RuleExecutionResult) {
	re.mu.RLock()
	defer re.mu.RUnlock()
	rOut := make([]ECARule, len(re.rules))
	copy(rOut, re.rules)
	hOut := make([]RuleExecutionResult, len(re.history))
	copy(hOut, re.history)
	return rOut, hOut
}

// PolicyStatus enumerates the lifecycle states of an ISO 42001 governance policy version.
type PolicyStatus string

const (
	PolicyDraft      PolicyStatus = "DRAFT"
	PolicyActive     PolicyStatus = "ACTIVE"
	PolicyDeprecated PolicyStatus = "DEPRECATED"
)

// GovernancePolicyVersion models a versioned policy in PolicyVersioningService (apps/memory_api/services/policy_versioning_service.py).
type GovernancePolicyVersion struct {
	PolicyID                string       `json:"policy_id"`
	Version                 int          `json:"version"`
	Status                  PolicyStatus `json:"status"`
	AllowRestrictedOutsideW bool         `json:"allow_restricted_outside_working"`
	RequireHumanApprovalRisk float64     `json:"require_human_approval_risk"`
	MaxDailyBudgetUSD       float64      `json:"max_daily_budget_usd"`
	CreatedAt               time.Time    `json:"created_at"`
}

// PolicyVersioningService manages versioned governance policies and rollback.
type PolicyVersioningService struct {
	mu       sync.RWMutex
	versions map[string][]GovernancePolicyVersion
}

// NewPolicyVersioningService creates a PolicyVersioningService with default active v1 policy.
func NewPolicyVersioningService() *PolicyVersioningService {
	svc := &PolicyVersioningService{
		versions: make(map[string][]GovernancePolicyVersion),
	}
	svc.CreateVersion("iso-42001-default", false, 0.80, 25.0, true)
	return svc
}

// CreateVersion creates a new policy version and optionally activates it (deprecating prior active versions).
func (p *PolicyVersioningService) CreateVersion(policyID string, allowRestricted bool, approvalRisk float64, maxBudgetUSD float64, activate bool) GovernancePolicyVersion {
	p.mu.Lock()
	defer p.mu.Unlock()
	if policyID == "" {
		policyID = "iso-42001-default"
	}
	list := p.versions[policyID]
	nextVer := len(list) + 1
	status := PolicyDraft
	if activate {
		status = PolicyActive
		for i := range list {
			if list[i].Status == PolicyActive {
				list[i].Status = PolicyDeprecated
			}
		}
	}
	ver := GovernancePolicyVersion{
		PolicyID:                 policyID,
		Version:                  nextVer,
		Status:                   status,
		AllowRestrictedOutsideW:  allowRestricted,
		RequireHumanApprovalRisk: approvalRisk,
		MaxDailyBudgetUSD:        maxBudgetUSD,
		CreatedAt:                time.Now().UTC(),
	}
	list = append(list, ver)
	p.versions[policyID] = list
	return ver
}

// Rollback activates a specific prior version of policyID.
func (p *PolicyVersioningService) Rollback(policyID string, targetVersion int) (GovernancePolicyVersion, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	list, ok := p.versions[policyID]
	if !ok || len(list) == 0 {
		return GovernancePolicyVersion{}, false
	}
	var activated GovernancePolicyVersion
	found := false
	for i := range list {
		if list[i].Version == targetVersion {
			list[i].Status = PolicyActive
			activated = list[i]
			found = true
		} else if list[i].Status == PolicyActive {
			list[i].Status = PolicyDeprecated
		}
	}
	p.versions[policyID] = list
	return activated, found
}

// GetActive returns the currently active GovernancePolicyVersion for policyID.
func (p *PolicyVersioningService) GetActive(policyID string) GovernancePolicyVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if policyID == "" {
		policyID = "iso-42001-default"
	}
	list := p.versions[policyID]
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].Status == PolicyActive {
			return list[i]
		}
	}
	if len(list) > 0 {
		return list[len(list)-1]
	}
	return GovernancePolicyVersion{PolicyID: policyID, Version: 1, Status: PolicyActive, RequireHumanApprovalRisk: 0.80, MaxDailyBudgetUSD: 25.0}
}

// ListVersions returns all versions for policyID.
func (p *PolicyVersioningService) ListVersions(policyID string) []GovernancePolicyVersion {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if policyID == "" {
		policyID = "iso-42001-default"
	}
	out := make([]GovernancePolicyVersion, len(p.versions[policyID]))
	copy(out, p.versions[policyID])
	return out
}

// DeletionAuditEntry records an immutable cryptographic proof of GDPR Right-to-be-Forgotten or TTL erasure.
type DeletionAuditEntry struct {
	AuditID            string    `json:"audit_id"`
	TenantID           string    `json:"tenant_id"`
	SubjectHashSHA256  string    `json:"subject_hash_sha256"`
	DeletedMemoryCount int       `json:"deleted_memory_count"`
	Reason             string    `json:"reason"` // GDPR_ART_17_ERASURE, TTL_RETENTION_SWEEP
	ProofHashSHA256    string    `json:"proof_hash_sha256"`
	ExecutedAt         time.Time `json:"executed_at"`
}

// RetentionService implements GDPR Right-to-be-Forgotten and TTL expiration sweeps
// matching apps/memory_api/services/retention_service.py.
type RetentionService struct {
	mu        sync.RWMutex
	auditLogs []DeletionAuditEntry
}

// NewRetentionService initializes a new RetentionService.
func NewRetentionService() *RetentionService {
	return &RetentionService{}
}

// ExecuteRightToBeForgotten erases all memories matching tenantID and subject/session/query and records a SHA-256 DeletionAuditEntry.
func (rs *RetentionService) ExecuteRightToBeForgotten(ctx context.Context, store ports.MemoryStoragePort, tenantID, subjectIdentifier string) DeletionAuditEntry {
	items, _ := store.QueryMemories(ctx, ports.QueryFilter{TopK: 1000})
	deleted := 0
	deletedIDs := make([]string, 0)
	subLower := strings.ToLower(strings.TrimSpace(subjectIdentifier))

	for _, sc := range items {
		it := sc.Memory
		match := false
		if tenantID != "" && it.TenantID == tenantID && subLower == "" {
			match = true
		}
		if subLower != "" && (strings.Contains(strings.ToLower(it.Content), subLower) || strings.EqualFold(it.SessionID, subjectIdentifier) || strings.EqualFold(it.ID, subjectIdentifier)) {
			match = true
		}
		if match {
			if err := store.DeleteMemory(ctx, it.ID); err == nil {
				deleted++
				deletedIDs = append(deletedIDs, it.ID)
			}
		}
	}

	subHash := sha256.Sum256([]byte(subjectIdentifier))
	subHex := hex.EncodeToString(subHash[:])
	now := time.Now().UTC()
	proofPayload := fmt.Sprintf("%s|%s|%d|%s|%s", tenantID, subHex, deleted, strings.Join(deletedIDs, ","), now.Format(time.RFC3339Nano))
	proofHash := sha256.Sum256([]byte(proofPayload))

	entry := DeletionAuditEntry{
		AuditID:            "gdpr-" + models.GenerateUUIDv7()[:8],
		TenantID:           tenantID,
		SubjectHashSHA256:  subHex,
		DeletedMemoryCount: deleted,
		Reason:             "GDPR_ART_17_ERASURE",
		ProofHashSHA256:    hex.EncodeToString(proofHash[:]),
		ExecutedAt:         now,
	}

	rs.mu.Lock()
	rs.auditLogs = append(rs.auditLogs, entry)
	rs.mu.Unlock()
	return entry
}

// ListDeletionAudits returns all cryptographic deletion audit entries.
func (rs *RetentionService) ListDeletionAudits() []DeletionAuditEntry {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	out := make([]DeletionAuditEntry, len(rs.auditLogs))
	copy(out, rs.auditLogs)
	return out
}

// BudgetAlertStatus enumerates the 3 threshold levels of BudgetService (apps/memory_api/services/budget_service.py).
type BudgetAlertStatus string

const (
	BudgetStatusOK        BudgetAlertStatus = "OK"
	BudgetStatusWarn80    BudgetAlertStatus = "WARN_80_PERCENT"
	BudgetStatusCrit95    BudgetAlertStatus = "CRITICAL_95_PERCENT"
	BudgetStatusHardBlock BudgetAlertStatus = "HARD_BLOCK_100_PERCENT"
)

// TenantBudget tracks daily/monthly USD and token limits and live consumption for a tenant/project.
type TenantBudget struct {
	TenantID         string            `json:"tenant_id"`
	DailyLimitUSD    float64           `json:"daily_limit_usd"`
	SpentTodayUSD    float64           `json:"spent_today_usd"`
	DailyTokenLimit  int               `json:"daily_token_limit"`
	TokensUsedToday  int               `json:"tokens_used_today"`
	UtilizationRatio float64           `json:"utilization_ratio"`
	Status           BudgetAlertStatus `json:"status"`
	Allowed          bool              `json:"allowed"`
}

// BudgetService enforces 80% WARN / 95% CRITICAL / 100% HARD_BLOCK budget governance.
type BudgetService struct {
	mu      sync.RWMutex
	budgets map[string]*TenantBudget
}

// NewBudgetService initializes a new BudgetService.
func NewBudgetService() *BudgetService {
	bs := &BudgetService{
		budgets: make(map[string]*TenantBudget),
	}
	bs.SetBudget("default", 10.0, 500000)
	return bs
}

// SetBudget sets or resets the daily limits for a tenant.
func (bs *BudgetService) SetBudget(tenantID string, dailyLimitUSD float64, dailyTokenLimit int) TenantBudget {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if tenantID == "" {
		tenantID = "default"
	}
	if dailyLimitUSD <= 0 {
		dailyLimitUSD = 10.0
	}
	if dailyTokenLimit <= 0 {
		dailyTokenLimit = 500000
	}
	b, ok := bs.budgets[tenantID]
	if !ok {
		b = &TenantBudget{TenantID: tenantID}
		bs.budgets[tenantID] = b
	}
	b.DailyLimitUSD = dailyLimitUSD
	b.DailyTokenLimit = dailyTokenLimit
	bs.recomputeStatus(b)
	return *b
}

// CheckAndRecordSpend checks whether estimatedCostUSD and estimatedTokens are within the 100% hard cap,
// records the spend if allowed, and returns the updated TenantBudget with 80%/95%/100% alert status.
func (bs *BudgetService) CheckAndRecordSpend(tenantID string, costUSD float64, tokens int) TenantBudget {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if tenantID == "" {
		tenantID = "default"
	}
	b, ok := bs.budgets[tenantID]
	if !ok {
		b = &TenantBudget{
			TenantID:        tenantID,
			DailyLimitUSD:   10.0,
			DailyTokenLimit: 500000,
		}
		bs.budgets[tenantID] = b
	}

	projCost := b.SpentTodayUSD + costUSD
	projTok := b.TokensUsedToday + tokens
	costRatio := projCost / b.DailyLimitUSD
	tokRatio := float64(projTok) / float64(b.DailyTokenLimit)
	maxRatio := math.Max(costRatio, tokRatio)

	if maxRatio >= 1.0 {
		b.UtilizationRatio = math.Round(maxRatio*10000) / 10000
		b.Status = BudgetStatusHardBlock
		b.Allowed = false
		return *b
	}

	b.SpentTodayUSD = math.Round(projCost*1000000) / 1000000
	b.TokensUsedToday = projTok
	bs.recomputeStatus(b)
	return *b
}

func (bs *BudgetService) recomputeStatus(b *TenantBudget) {
	costRatio := b.SpentTodayUSD / b.DailyLimitUSD
	tokRatio := float64(b.TokensUsedToday) / float64(b.DailyTokenLimit)
	maxRatio := math.Max(costRatio, tokRatio)
	b.UtilizationRatio = math.Round(maxRatio*10000) / 10000
	switch {
	case maxRatio >= 1.0:
		b.Status = BudgetStatusHardBlock
		b.Allowed = false
	case maxRatio >= 0.95:
		b.Status = BudgetStatusCrit95
		b.Allowed = true
	case maxRatio >= 0.80:
		b.Status = BudgetStatusWarn80
		b.Allowed = true
	default:
		b.Status = BudgetStatusOK
		b.Allowed = true
	}
}
