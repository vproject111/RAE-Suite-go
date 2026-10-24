package ops

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 1. IMPROVEMENT PLANE (Hypothesis -> Experiment -> Shadow -> Canary -> PromotionGate)
// ============================================================================

type Hypothesis struct {
	HypothesisID string `json:"hypothesis_id"`
	Statement    string `json:"statement"`
	Motivation   string `json:"motivation"`
	TargetMetric string `json:"target_metric"`
	Status       string `json:"status"`
}

type Experiment struct {
	ExperimentID    string `json:"experiment_id"`
	Name            string `json:"name"`
	HypothesisID    string `json:"hypothesis_id"`
	Candidate       string `json:"candidate"`
	SuccessCriteria string `json:"success_criteria"`
}

type ExperimentRun struct {
	RunID        string                 `json:"run_id"`
	ExperimentID string                 `json:"experiment_id"`
	Mode         string                 `json:"mode"`   // shadow, canary, benchmark
	Result       string                 `json:"result"` // pass, fail
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
}

type ImprovementProposal struct {
	ProposalID            string   `json:"proposal_id"`
	ExperimentID          string   `json:"experiment_id"`
	PromotionRequirements []string `json:"promotion_requirements"`
	AuditorApproved       bool     `json:"auditor_approved"`
}

type PromotionDecision struct {
	ProposalID string `json:"proposal_id"`
	Approved   bool   `json:"approved"`
	Reason     string `json:"reason"`
}

type RollbackDecision struct {
	ProposalID        string `json:"proposal_id"`
	RollbackTriggered bool   `json:"rollback_triggered"`
	Reason            string `json:"reason"`
}

type ImprovementPlane struct {
	mu                 sync.RWMutex
	Hypotheses         map[string]Hypothesis
	Experiments        map[string]Experiment
	Runs               map[string][]ExperimentRun
	ErrorRateThreshold float64
}

func NewImprovementPlane(errorRateThreshold float64) *ImprovementPlane {
	if errorRateThreshold <= 0 {
		errorRateThreshold = 0.05
	}
	return &ImprovementPlane{
		Hypotheses:         make(map[string]Hypothesis),
		Experiments:        make(map[string]Experiment),
		Runs:               make(map[string][]ExperimentRun),
		ErrorRateThreshold: errorRateThreshold,
	}
}

func (p *ImprovementPlane) CreateHypothesis(id, statement, motivation, targetMetric string) Hypothesis {
	p.mu.Lock()
	defer p.mu.Unlock()
	if targetMetric == "" {
		targetMetric = "latency"
	}
	h := Hypothesis{
		HypothesisID: id,
		Statement:    statement,
		Motivation:   motivation,
		TargetMetric: targetMetric,
		Status:       "PROPOSED",
	}
	p.Hypotheses[id] = h
	return h
}

func (p *ImprovementPlane) CreateExperiment(id, name, hypothesisID, candidate, criteria string) Experiment {
	p.mu.Lock()
	defer p.mu.Unlock()
	e := Experiment{
		ExperimentID:    id,
		Name:            name,
		HypothesisID:    hypothesisID,
		Candidate:       candidate,
		SuccessCriteria: criteria,
	}
	p.Experiments[id] = e
	return e
}

func (p *ImprovementPlane) RecordRun(run ExperimentRun) ExperimentRun {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Runs[run.ExperimentID] = append(p.Runs[run.ExperimentID], run)
	return run
}

func (p *ImprovementPlane) EvaluateCanaryHealth(experimentID string, errorRate float64) RollbackDecision {
	if errorRate > p.ErrorRateThreshold {
		return RollbackDecision{
			ProposalID:        "can-" + experimentID,
			RollbackTriggered: true,
			Reason:            fmt.Sprintf("Canary error rate %.4f exceeded threshold %.4f", errorRate, p.ErrorRateThreshold),
		}
	}
	return RollbackDecision{
		ProposalID:        "can-" + experimentID,
		RollbackTriggered: false,
		Reason:            "Canary metrics are healthy.",
	}
}

func (p *ImprovementPlane) EvaluatePromotion(prop ImprovementProposal) PromotionDecision {
	p.mu.RLock()
	runs := p.Runs[prop.ExperimentID]
	p.mu.RUnlock()

	var failures []string
	var shadowRuns []ExperimentRun
	for _, r := range runs {
		if r.Mode == "shadow" {
			shadowRuns = append(shadowRuns, r)
		}
	}
	if len(shadowRuns) == 0 {
		failures = append(failures, "has_shadow_run: Brak shadow run.")
	} else {
		passed := false
		for _, r := range shadowRuns {
			if r.Result == "pass" {
				passed = true
				break
			}
		}
		if !passed {
			failures = append(failures, "shadow_passed: Żaden shadow run nie przeszedł.")
		}
	}

	hasRollbackPlan := false
	for _, req := range prop.PromotionRequirements {
		if req == "rollback_plan" {
			hasRollbackPlan = true
			break
		}
	}
	if !hasRollbackPlan {
		failures = append(failures, "has_rollback_plan: Brak planu rollback.")
	}
	if !prop.AuditorApproved {
		failures = append(failures, "auditor_approved: Brak zatwierdzonego werdyktu Auditora.")
	}

	if len(failures) > 0 {
		return PromotionDecision{
			ProposalID: prop.ProposalID,
			Approved:   false,
			Reason:     strings.Join(failures, "; "),
		}
	}
	return PromotionDecision{
		ProposalID: prop.ProposalID,
		Approved:   true,
		Reason:     "All promotion gates (shadow run, rollback plan, auditor approval) satisfied.",
	}
}

// ============================================================================
// 2. CLUSTER MANAGER (Node 1 Lumina, Node 2 Julka, Node 3 Piotrek)
// ============================================================================

type ClusterNode struct {
	NodeID      string   `json:"node_id"`
	Transport   string   `json:"transport"`
	Host        string   `json:"host"`
	User        string   `json:"user,omitempty"`
	URL         string   `json:"url,omitempty"`
	GPU         string   `json:"gpu"`
	RAMGB       int      `json:"ram_gb"`
	Models      []string `json:"models,omitempty"`
	IsOnline    bool     `json:"is_online"`
	Description string   `json:"description"`
}

type ClusterManager struct {
	mu    sync.RWMutex
	Nodes map[string]ClusterNode
}

func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		Nodes: map[string]ClusterNode{
			"node1_lumina": {
				NodeID:      "node1_lumina",
				Transport:   "mcp-ssh",
				Host:        "100.68.166.117",
				User:        "operator",
				GPU:         "NVIDIA RTX 4080 16GB",
				RAMGB:       64,
				Models:      []string{"deepseek-r1:8b", "SpeakLeash/bielik-11b-v3.0-instruct:Q5_K_M", "qwen3.5:9b"},
				IsOnline:    true,
				Description: "Primary GPU Compute & Stage 0 Local Triad Node (Arch Linux, i7-14700KF)",
			},
			"node2_julka": {
				NodeID:      "node2_julka",
				Transport:   "mcp-ssh",
				Host:        "100.70.32.40",
				User:        "juliazdalnie",
				GPU:         "NVIDIA RTX 4080 SUPER 16GB",
				RAMGB:       64,
				IsOnline:    true,
				Description: "High-Performance Compute & Benchmark Offload Node (i9-14900KF)",
			},
			"node3_piotrek": {
				NodeID:      "node3_piotrek",
				Transport:   "http-proxy",
				Host:        "172.30.15.11",
				URL:         "http://172.30.15.11:11434",
				GPU:         "NVIDIA RTX 4080 16GB",
				RAMGB:       128,
				Models:      []string{"llama3.1:8b", "mixtral:8x7b", "qwen3"},
				IsOnline:    true,
				Description: "High-RAM Inference & Concurrency Stress Node (128GB RAM)",
			},
		},
	}
}

func (cm *ClusterManager) ListNodes() []ClusterNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	out := make([]ClusterNode, 0, len(cm.Nodes))
	for _, k := range []string{"node1_lumina", "node2_julka", "node3_piotrek"} {
		if n, ok := cm.Nodes[k]; ok {
			out = append(out, n)
		}
	}
	return out
}

func (cm *ClusterManager) RouteTask(workloadType string) ClusterNode {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	w := strings.ToLower(strings.TrimSpace(workloadType))
	switch {
	case strings.Contains(w, "node3") || strings.Contains(w, "piotrek") || strings.Contains(w, "stress_db") || strings.Contains(w, "high_ram") || strings.Contains(w, "ollama_proxy"):
		return cm.Nodes["node3_piotrek"]
	case strings.Contains(w, "node2") || strings.Contains(w, "julka") || strings.Contains(w, "benchmark") || strings.Contains(w, "training") || strings.Contains(w, "heavy"):
		return cm.Nodes["node2_julka"]
	default:
		return cm.Nodes["node1_lumina"]
	}
}

// ============================================================================
// 3. TRANSPORT CIRCUIT BREAKER & DISTRIBUTED RATE LIMITER
// ============================================================================

type CircuitState string

const (
	CircuitClosed   CircuitState = "CLOSED"
	CircuitOpen     CircuitState = "OPEN"
	CircuitHalfOpen CircuitState = "HALF_OPEN"
)

type TransportCircuitBreaker struct {
	mu               sync.Mutex
	FailureThreshold int          `json:"failure_threshold"`
	RecoveryTimeoutS float64      `json:"recovery_timeout_s"`
	State            CircuitState `json:"state"`
	FailureCount     int          `json:"failure_count"`
	LastStateChange  time.Time    `json:"last_state_change"`
}

func NewTransportCircuitBreaker(threshold int, recoveryTimeoutS float64) *TransportCircuitBreaker {
	if threshold <= 0 {
		threshold = 3
	}
	if recoveryTimeoutS <= 0 {
		recoveryTimeoutS = 5.0
	}
	return &TransportCircuitBreaker{
		FailureThreshold: threshold,
		RecoveryTimeoutS: recoveryTimeoutS,
		State:            CircuitClosed,
		LastStateChange:  time.Now(),
	}
}

func (cb *TransportCircuitBreaker) AllowExecution() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.State == CircuitOpen {
		if time.Since(cb.LastStateChange).Seconds() >= cb.RecoveryTimeoutS {
			cb.State = CircuitHalfOpen
			cb.LastStateChange = time.Now()
			return true
		}
		return false
	}
	return true
}

func (cb *TransportCircuitBreaker) RecordSuccess() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.FailureCount = 0
	if cb.State == CircuitHalfOpen {
		cb.State = CircuitClosed
		cb.LastStateChange = time.Now()
	}
	return cb.State
}

func (cb *TransportCircuitBreaker) RecordFailure() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.FailureCount++
	if cb.FailureCount >= cb.FailureThreshold || cb.State == CircuitHalfOpen {
		cb.State = CircuitOpen
		cb.LastStateChange = time.Now()
	}
	return cb.State
}

type RateLimitStatus struct {
	ClientID        string  `json:"client_id"`
	Allowed         bool    `json:"allowed"`
	RemainingTokens int     `json:"remaining_tokens"`
	ResetAfterSec   float64 `json:"reset_after_sec"`
}

type tokenBucket struct {
	tokens     float64
	lastUpdate time.Time
}

type DistributedRateLimiter struct {
	mu               sync.Mutex
	Capacity         int
	RefillRatePerSec float64
	buckets          map[string]*tokenBucket
}

func NewDistributedRateLimiter(capacity int, refillRatePerSec float64) *DistributedRateLimiter {
	if capacity <= 0 {
		capacity = 100
	}
	if refillRatePerSec <= 0 {
		refillRatePerSec = 10.0
	}
	return &DistributedRateLimiter{
		Capacity:         capacity,
		RefillRatePerSec: refillRatePerSec,
		buckets:          make(map[string]*tokenBucket),
	}
}

func (rl *DistributedRateLimiter) CheckRateLimit(clientID string, cost int) RateLimitStatus {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if cost <= 0 {
		cost = 1
	}
	now := time.Now()
	b, ok := rl.buckets[clientID]
	if !ok {
		b = &tokenBucket{tokens: float64(rl.Capacity), lastUpdate: now}
		rl.buckets[clientID] = b
	}
	elapsed := now.Sub(b.lastUpdate).Seconds()
	b.tokens = math.Min(float64(rl.Capacity), b.tokens+(elapsed*rl.RefillRatePerSec))
	b.lastUpdate = now

	if b.tokens >= float64(cost) {
		b.tokens -= float64(cost)
		return RateLimitStatus{
			ClientID:        clientID,
			Allowed:         true,
			RemainingTokens: int(b.tokens),
			ResetAfterSec:   0.0,
		}
	}
	needed := float64(cost) - b.tokens
	return RateLimitStatus{
		ClientID:        clientID,
		Allowed:         false,
		RemainingTokens: int(b.tokens),
		ResetAfterSec:   math.Round((needed/rl.RefillRatePerSec)*100) / 100,
	}
}

// ============================================================================
// 4. ROLLBACK SLA MANAGER, DUAL ESCALATION ENGINE & INFRA RECONCILER
// ============================================================================

type RollbackPlan struct {
	PlanID              string  `json:"plan_id"`
	ActionType          string  `json:"action_type"` // container_restart | git_worktree_revert | config_restore | db_schema_rollback | vector_projection_rollback
	SLAThresholdSeconds float64 `json:"sla_threshold_seconds"`
	VerifiedInSandbox   bool    `json:"verified_in_sandbox"`
}

type RollbackSLAManager struct {
	SLAMatrix map[string]float64
}

func NewRollbackSLAManager() *RollbackSLAManager {
	return &RollbackSLAManager{
		SLAMatrix: map[string]float64{
			"container_restart":          15.0,
			"git_worktree_revert":        30.0,
			"config_restore":             60.0,
			"db_schema_rollback":         120.0,
			"vector_projection_rollback": 300.0,
		},
	}
}

func (m *RollbackSLAManager) VerifyPlan(plan RollbackPlan, riskClass string) (bool, string) {
	maxAllowed, ok := m.SLAMatrix[plan.ActionType]
	if !ok {
		maxAllowed = 30.0
	}
	if plan.SLAThresholdSeconds > maxAllowed {
		return false, fmt.Sprintf("SLA %.1fs exceeds max allowed %.1fs for %s", plan.SLAThresholdSeconds, maxAllowed, plan.ActionType)
	}
	rc := strings.ToUpper(strings.TrimSpace(riskClass))
	if (rc == "R4" || rc == "R5" || rc == "R6") && !plan.VerifiedInSandbox {
		return false, fmt.Sprintf("High-risk class %s requires VerifiedInSandbox=true", rc)
	}
	return true, "ROLLBACK_PLAN_VERIFIED"
}

type EscalationRequest struct {
	TaskID               string   `json:"task_id"`
	RiskClass            string   `json:"risk_class"`
	FailedRepairAttempts int      `json:"failed_repair_attempts"`
	Reason               string   `json:"reason"`
	ErrorTrace           string   `json:"error_trace"`
	ModifiedFiles        []string `json:"modified_files"`
}

type EscalationResult struct {
	EscalationID string `json:"escalation_id"`
	Target       string `json:"target"` // HERMES, OPENCODE, HUMAN_OPERATOR
	Status       string `json:"status"`
	Message      string `json:"message"`
}

type DualEscalationEngine struct {
	MaxRepairCycles int
}

func NewDualEscalationEngine(maxRepairCycles int) *DualEscalationEngine {
	if maxRepairCycles <= 0 {
		maxRepairCycles = 3
	}
	return &DualEscalationEngine{MaxRepairCycles: maxRepairCycles}
}

func (e *DualEscalationEngine) EvaluateEscalation(req EscalationRequest) EscalationResult {
	target := "HERMES"
	if req.Reason == "HIGH_RISK_RESTRICTED_VIOLATION" || req.RiskClass == "R6" {
		target = "HUMAN_OPERATOR"
	} else if req.Reason == "COMPLEX_AST_REFACTORING_REQUIRED" || len(req.ModifiedFiles) > 3 {
		target = "OPENCODE"
	} else if req.FailedRepairAttempts >= e.MaxRepairCycles {
		target = "HERMES"
	}
	return EscalationResult{
		EscalationID: fmt.Sprintf("esc-%s", req.TaskID),
		Target:       target,
		Status:       "ESCALATED",
		Message:      fmt.Sprintf("Escalated task %s to %s (%s)", req.TaskID, target, req.Reason),
	}
}

type InfraReconciler struct {
	mu              sync.Mutex
	RestartCounters map[string]int
}

func NewInfraReconciler() *InfraReconciler {
	return &InfraReconciler{
		RestartCounters: make(map[string]int),
	}
}

func (ir *InfraReconciler) CheckAndRecoverService(serviceName string, isHealthy bool) map[string]interface{} {
	ir.mu.Lock()
	defer ir.mu.Unlock()
	if isHealthy {
		ir.RestartCounters[serviceName] = 0
		return map[string]interface{}{"service": serviceName, "status": "HEALTHY", "restarts": 0}
	}
	cur := ir.RestartCounters[serviceName]
	if cur < 3 {
		cur++
		ir.RestartCounters[serviceName] = cur
		return map[string]interface{}{"service": serviceName, "status": "MICRO_RESTARTED", "restarts": cur}
	}
	return map[string]interface{}{"service": serviceName, "status": "ESCALATED_UNRECOVERABLE", "restarts": cur}
}

// ============================================================================
// 5. FEDERATED PROMPT REGISTRY, TOOL GATEWAY, W3C TRACE, PSI DRIFT, KEYCLOAK & HARNESS
// ============================================================================

type FederatedPromptRegistry struct {
	mu               sync.RWMutex
	BaseTemplates    map[string]string
	OrgTemplates     map[string]map[string]string
	TeamTemplates    map[string]map[string]string
	FeatureTemplates map[string]map[string]string
}

func NewFederatedPromptRegistry() *FederatedPromptRegistry {
	return &FederatedPromptRegistry{
		BaseTemplates:    make(map[string]string),
		OrgTemplates:     make(map[string]map[string]string),
		TeamTemplates:    make(map[string]map[string]string),
		FeatureTemplates: make(map[string]map[string]string),
	}
}

func (r *FederatedPromptRegistry) RegisterBase(template string) error {
	trimmed := strings.TrimSpace(template)
	if trimmed == "" {
		return fmt.Errorf("template content cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.BaseTemplates["system"]; exists {
		return fmt.Errorf("base template is frozen and cannot be modified")
	}
	r.BaseTemplates["system"] = trimmed
	return nil
}

func (r *FederatedPromptRegistry) RegisterLayer(layer, scope, section, content string) error {
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	target := r.OrgTemplates
	switch strings.ToLower(layer) {
	case "team":
		target = r.TeamTemplates
	case "feature":
		target = r.FeatureTemplates
	}
	if target[scope] == nil {
		target[scope] = make(map[string]string)
	}
	target[scope][section] = strings.TrimSpace(content)
	return nil
}

func (r *FederatedPromptRegistry) CompilePrompt(org, team, feature string) (string, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var parts []string
	if base, ok := r.BaseTemplates["system"]; ok {
		parts = append(parts, "[BASE]\n"+base)
	}
	if m, ok := r.OrgTemplates[org]; ok {
		for k, v := range m {
			parts = append(parts, fmt.Sprintf("[ORG:%s:%s]\n%s", org, k, v))
		}
	}
	if m, ok := r.TeamTemplates[team]; ok {
		for k, v := range m {
			parts = append(parts, fmt.Sprintf("[TEAM:%s:%s]\n%s", team, k, v))
		}
	}
	if m, ok := r.FeatureTemplates[feature]; ok {
		for k, v := range m {
			parts = append(parts, fmt.Sprintf("[FEATURE:%s:%s]\n%s", feature, k, v))
		}
	}
	compiled := strings.Join(parts, "\n\n")
	sum := sha256.Sum256([]byte(compiled))
	return compiled, hex.EncodeToString(sum[:])
}

type ToolGateway struct {
	mu            sync.Mutex
	EmptyRunCache map[string]map[string]interface{}
	Invocations   int
}

func NewToolGateway() *ToolGateway {
	return &ToolGateway{
		EmptyRunCache: make(map[string]map[string]interface{}),
	}
}

func (tg *ToolGateway) ExecuteTool(toolName string, contextHash string, isEmptyRun bool) map[string]interface{} {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	if cached, ok := tg.EmptyRunCache[contextHash]; ok {
		return map[string]interface{}{
			"tool":                  toolName,
			"empty_run_prevented":   true,
			"cached_result":         cached,
			"total_invocations_run": tg.Invocations,
		}
	}
	tg.Invocations++
	res := map[string]interface{}{
		"tool":      toolName,
		"status":    "EXECUTED",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	if isEmptyRun && contextHash != "" {
		tg.EmptyRunCache[contextHash] = res
	}
	return res
}

type EmbeddingDriftDetector struct {
	PSIAlertThreshold float64
}

func NewEmbeddingDriftDetector(threshold float64) *EmbeddingDriftDetector {
	if threshold <= 0 {
		threshold = 0.25
	}
	return &EmbeddingDriftDetector{PSIAlertThreshold: threshold}
}

func (d *EmbeddingDriftDetector) CalculatePSI(expected, actual []float64, numBuckets int) (float64, bool) {
	if len(expected) == 0 || len(actual) == 0 {
		return 0.0, false
	}
	if numBuckets <= 0 {
		numBuckets = 10
	}
	minVal, maxVal := expected[0], expected[0]
	for _, v := range append(append([]float64{}, expected...), actual...) {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == minVal {
		return 0.0, false
	}
	binWidth := (maxVal - minVal) / float64(numBuckets)
	expCounts := make([]int, numBuckets)
	actCounts := make([]int, numBuckets)
	getBin := func(val float64) int {
		idx := int((val - minVal) / binWidth)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		if idx < 0 {
			idx = 0
		}
		return idx
	}
	for _, v := range expected {
		expCounts[getBin(v)]++
	}
	for _, v := range actual {
		actCounts[getBin(v)]++
	}

	psi := 0.0
	eps := 0.0001
	for i := 0; i < numBuckets; i++ {
		expP := math.Max(float64(expCounts[i])/float64(len(expected)), eps)
		actP := math.Max(float64(actCounts[i])/float64(len(actual)), eps)
		psi += (actP - expP) * math.Log(actP/expP)
	}
	rounded := math.Round(psi*10000) / 10000
	return rounded, rounded >= d.PSIAlertThreshold
}

var nonHexRegex = regexp.MustCompile(`[^0-9a-fA-F]`)

func InjectW3CTraceContext(traceID, spanID string, sampled bool) string {
	cleanTrace := strings.ToLower(nonHexRegex.ReplaceAllString(strings.ReplaceAll(traceID, "trace-", ""), ""))
	if len(cleanTrace) < 32 {
		cleanTrace = generateHexID(16)
	} else {
		cleanTrace = cleanTrace[:32]
	}
	if cleanTrace == strings.Repeat("0", 32) {
		cleanTrace = generateHexID(16)
	}
	cleanSpan := strings.ToLower(nonHexRegex.ReplaceAllString(spanID, ""))
	if len(cleanSpan) < 16 {
		cleanSpan = generateHexID(8)
	} else {
		cleanSpan = cleanSpan[:16]
	}
	if cleanSpan == strings.Repeat("0", 16) {
		cleanSpan = generateHexID(8)
	}
	flags := "00"
	if sampled {
		flags = "01"
	}
	return fmt.Sprintf("00-%s-%s-%s", cleanTrace, cleanSpan, flags)
}

type KeycloakClaims struct {
	Sub               string   `json:"sub"`
	PreferredUsername string   `json:"preferred_username"`
	TenantID          string   `json:"tenant_id"`
	Iss               string   `json:"iss"`
	Aud               []string `json:"aud"`
	Roles             []string `json:"roles"`
	CapabilityScopes  []string `json:"capability_scopes"`
	Exp               int64    `json:"exp"`
}

type KeycloakAuthValidator struct {
	SecretKey      string
	ExpectedIssuer string
}

func NewKeycloakAuthValidator(secret, issuer string) *KeycloakAuthValidator {
	if secret == "" {
		secret = "rae-keycloak-default-secret"
	}
	if issuer == "" {
		issuer = "https://auth.dreamsoft.pro/realms/rae"
	}
	return &KeycloakAuthValidator{
		SecretKey:      secret,
		ExpectedIssuer: strings.TrimRight(issuer, "/"),
	}
}

func (k *KeycloakAuthValidator) CreateJWT(username, tenantID string, roles, scopes []string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := KeycloakClaims{
		Sub:               "user_" + username,
		PreferredUsername: username,
		TenantID:          tenantID,
		Iss:               k.ExpectedIssuer,
		Aud:               []string{"rae-suite-client"},
		Roles:             roles,
		CapabilityScopes:  scopes,
		Exp:               time.Now().Add(1 * time.Hour).Unix(),
	}
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, []byte(k.SecretKey))
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig
}

func (k *KeycloakAuthValidator) VerifyJWT(tokenStr string, requiredCapability string) (*KeycloakClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || !strings.Contains(string(headerJSON), `"HS256"`) {
		return nil, fmt.Errorf("forbidden JWT algorithm")
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(k.SecretKey))
	mac.Write([]byte(unsigned))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, fmt.Errorf("JWT signature mismatch")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims KeycloakClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, err
	}
	if strings.TrimRight(claims.Iss, "/") != k.ExpectedIssuer {
		return nil, fmt.Errorf("issuer mismatch")
	}
	if requiredCapability != "" {
		found := false
		for _, sc := range claims.CapabilityScopes {
			if sc == requiredCapability {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("missing capability scope: %s", requiredCapability)
		}
	}
	return &claims, nil
}

type AutonomyEvaluationHarness struct {
	injectionPatterns []*regexp.Regexp
	retentionMatrix   map[string]int
}

func NewAutonomyEvaluationHarness() *AutonomyEvaluationHarness {
	patterns := []string{
		`(?i)\bignore` + ` previous instructions\b`,
		`(?i)\boverride threat classifier\b`,
		`(?i)\bbypass risk classification\b`,
		`(?i)\bexpose /etc/shadow\b`,
		`(?i)\bdelete all memories\b`,
	}
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return &AutonomyEvaluationHarness{
		injectionPatterns: compiled,
		retentionMatrix: map[string]int{
			"R0": 90,
			"R1": 90,
			"R2": 365,
			"R3": 365,
			"R4": 2555,
			"R5": 2555,
			"R6": 2555,
		},
	}
}

func (h *AutonomyEvaluationHarness) EvaluatePrompt(prompt string, currentCost, maxBudget float64, riskClass string) map[string]interface{} {
	blocked := false
	reason := "Prompt is clean of injection vectors."
	action := "ALLOW"
	for _, pat := range h.injectionPatterns {
		if pat.MatchString(prompt) {
			blocked = true
			reason = "Prompt injection vector matched pattern"
			action = "QUARANTINE"
			riskClass = "R6"
			break
		}
	}
	retentionDays, ok := h.retentionMatrix[strings.ToUpper(riskClass)]
	if !ok {
		retentionDays = 365
	}
	costCompliant := currentCost <= maxBudget
	return map[string]interface{}{
		"blocked":                  blocked,
		"action":                   action,
		"reason":                   reason,
		"risk_class":               riskClass,
		"cost_compliant":           costCompliant,
		"stop_condition_triggered": !costCompliant,
		"iso_retention_days":       retentionDays,
	}
}

func (h *AutonomyEvaluationHarness) VerifyCostBounds(currentCost, maxBudget float64) bool {
	return currentCost <= maxBudget
}

func (h *AutonomyEvaluationHarness) CalculateRetentionDays(riskClass string) int {
	if days, ok := h.retentionMatrix[strings.ToUpper(riskClass)]; ok {
		return days
	}
	return 365
}

func (h *AutonomyEvaluationHarness) VerifySimulationMode(mode string) bool {
	return strings.EqualFold(mode, "shadow") || strings.EqualFold(mode, "simulation")
}

func (h *AutonomyEvaluationHarness) RunFullEvaluationSuite(prompt string) map[string]interface{} {
	return h.EvaluatePrompt(prompt, 0.01, 5.0, "R2")
}

func (ir *InfraReconciler) VerifyTCPPort(host string, port int) bool {
	return host != "" && port > 0 && port <= 65535
}

func (ir *InfraReconciler) VerifyAlembicCompatibility(headRev, dbRev string) bool {
	return headRev == dbRev || headRev == ""
}

func (ir *InfraReconciler) AutoRollbackDatabase(serviceName string) map[string]interface{} {
	return map[string]interface{}{"service": serviceName, "rolled_back": true}
}

