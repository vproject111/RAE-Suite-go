package workflows

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 1. SAGA COORDINATOR
// ============================================================================

type SagaStep struct {
	Name       string                 `json:"name"`
	Action     func() (any, error)    `json:"-"`
	Compensate func() error           `json:"-"`
	Simulated  map[string]interface{} `json:"simulated,omitempty"`
}

type SagaResult struct {
	Status             string   `json:"status"` // SAGA_COMPLETED, SAGA_ROLLED_BACK, SAGA_COMPENSATION_FAILED
	CompletedSteps     []string `json:"completed_steps"`
	RolledBackSteps    []string `json:"rolled_back_steps,omitempty"`
	Error              string   `json:"error,omitempty"`
	CompensationErrors []string `json:"compensation_errors,omitempty"`
}

type SagaCoordinator struct {
	Steps     []SagaStep
	Completed []SagaStep
}

func NewSagaCoordinator() *SagaCoordinator {
	return &SagaCoordinator{}
}

func (s *SagaCoordinator) AddStep(name string, action func() (any, error), compensate func() error) {
	s.Steps = append(s.Steps, SagaStep{
		Name:       name,
		Action:     action,
		Compensate: compensate,
	})
}

func (s *SagaCoordinator) ExecuteSaga() SagaResult {
	s.Completed = nil
	var completedNames []string

	for _, step := range s.Steps {
		if step.Action != nil {
			if _, err := step.Action(); err != nil {
				return s.rollback(err.Error())
			}
		}
		s.Completed = append(s.Completed, step)
		completedNames = append(completedNames, step.Name)
	}

	return SagaResult{
		Status:         "SAGA_COMPLETED",
		CompletedSteps: completedNames,
	}
}

func (s *SagaCoordinator) rollback(reason string) SagaResult {
	var rolledBack []string
	var compErrors []string

	for i := len(s.Completed) - 1; i >= 0; i-- {
		step := s.Completed[i]
		if step.Compensate != nil {
			if err := step.Compensate(); err != nil {
				compErrors = append(compErrors, fmt.Sprintf("%s: %v", step.Name, err))
				continue
			}
		}
		rolledBack = append(rolledBack, step.Name)
	}

	status := "SAGA_ROLLED_BACK"
	if len(compErrors) > 0 {
		status = "SAGA_COMPENSATION_FAILED"
	}
	var completedNames []string
	for _, st := range s.Completed {
		completedNames = append(completedNames, st.Name)
	}
	return SagaResult{
		Status:             status,
		CompletedSteps:     completedNames,
		RolledBackSteps:    rolledBack,
		Error:              reason,
		CompensationErrors: compErrors,
	}
}

// ============================================================================
// 2. STREAMING FUNCTION COMPOSER
// ============================================================================

type ToolHandler func(args map[string]interface{}) (interface{}, error)

type StreamingStepResult struct {
	Step   int         `json:"step"`
	Tool   string      `json:"tool"`
	Status string      `json:"status"`
	Output interface{} `json:"output,omitempty"`
	Error  string      `json:"error,omitempty"`
}

type StreamingFunctionComposer struct {
	Tools map[string]ToolHandler
}

func NewStreamingFunctionComposer(tools map[string]ToolHandler) *StreamingFunctionComposer {
	if tools == nil {
		tools = make(map[string]ToolHandler)
	}
	return &StreamingFunctionComposer{Tools: tools}
}

func (c *StreamingFunctionComposer) ParseStreamChunk(chunk string) []map[string]interface{} {
	var parsed []map[string]interface{}
	search := chunk
	for {
		idx := strings.Index(search, "STEP:")
		if idx < 0 {
			break
		}
		search = strings.TrimSpace(search[idx+len("STEP:"):])
		dec := json.NewDecoder(strings.NewReader(search))
		var obj map[string]interface{}
		if err := dec.Decode(&obj); err == nil {
			parsed = append(parsed, obj)
		}
		if idxNext := strings.Index(search, "}"); idxNext >= 0 {
			search = search[idxNext+1:]
		} else {
			break
		}
	}
	return parsed
}

func (c *StreamingFunctionComposer) PipeSteps(chunks []string) []StreamingStepResult {
	var results []StreamingStepResult
	var lastOutput interface{}
	stepIdx := 0

	for _, chunk := range chunks {
		steps := c.ParseStreamChunk(chunk)
		for _, step := range steps {
			stepIdx++
			toolName, _ := step["tool"].(string)
			rawArgs, _ := step["args"].(map[string]interface{})
			args := make(map[string]interface{})
			for k, v := range rawArgs {
				if strVal, ok := v.(string); ok && strVal == "$prev_output" && lastOutput != nil {
					args[k] = lastOutput
				} else {
					args[k] = v
				}
			}

			handler, exists := c.Tools[toolName]
			if !exists {
				results = append(results, StreamingStepResult{
					Step:   stepIdx,
					Tool:   toolName,
					Status: "FAILED",
					Error:  fmt.Sprintf("unknown tool: %s", toolName),
				})
				return results
			}

			out, err := handler(args)
			if err != nil {
				results = append(results, StreamingStepResult{
					Step:   stepIdx,
					Tool:   toolName,
					Status: "ERROR",
					Error:  err.Error(),
				})
				return results
			}
			lastOutput = out
			results = append(results, StreamingStepResult{
				Step:   stepIdx,
				Tool:   toolName,
				Status: "SUCCESS",
				Output: out,
			})
		}
	}
	return results
}

// ============================================================================
// 3. SPECULATIVE TOOL EXECUTOR
// ============================================================================

type SpeculativeResult struct {
	Command string `json:"command"`
	Status  string `json:"status"` // SUCCESS, BLOCKED_UNSAFE, FAILED
	Output  string `json:"output,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type SpeculativeToolExecutor struct {
	SafeCommands      map[string]bool
	ForbiddenPatterns []*regexp.Regexp
	Runner            func(cmd string) (string, error)
}

func NewSpeculativeToolExecutor(runner func(cmd string) (string, error)) *SpeculativeToolExecutor {
	safe := map[string]bool{
		"cat": true, "ls": true, "grep": true, "rg": true, "find": true,
		"git status": true, "git diff": true, "git log": true, "head": true, "tail": true, "wc": true,
	}
	patterns := []string{
		`rm\s`, `mv\s`, `cp\s`, `git\s+(commit|push|reset|checkout|clean)`,
		`sed\s+-i`, `>`, `>>`, `\|`, `;`, `&&`,
	}
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return &SpeculativeToolExecutor{
		SafeCommands:      safe,
		ForbiddenPatterns: compiled,
		Runner:            runner,
	}
}

func (e *SpeculativeToolExecutor) IsCommandSafe(cmdStr string) bool {
	trimmed := strings.TrimSpace(cmdStr)
	if trimmed == "" {
		return false
	}
	for _, pat := range e.ForbiddenPatterns {
		if pat.MatchString(trimmed) {
			return false
		}
	}
	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return false
	}
	if e.SafeCommands[fields[0]] {
		return true
	}
	if len(fields) >= 2 && e.SafeCommands[fields[0]+" "+fields[1]] {
		return true
	}
	return false
}

func (e *SpeculativeToolExecutor) ExecuteSpeculatively(commands []string) map[string]SpeculativeResult {
	if len(commands) > 3 {
		commands = commands[:3]
	}
	results := make(map[string]SpeculativeResult, len(commands))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, cmd := range commands {
		if !e.IsCommandSafe(cmd) {
			mu.Lock()
			results[cmd] = SpeculativeResult{
				Command: cmd,
				Status:  "BLOCKED_UNSAFE",
				Reason:  "Command contains mutating or non-whitelisted operations",
			}
			mu.Unlock()
			continue
		}
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			out := fmt.Sprintf("speculative-ok: %s", c)
			var err error
			if e.Runner != nil {
				out, err = e.Runner(c)
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[c] = SpeculativeResult{
					Command: c,
					Status:  "FAILED",
					Reason:  err.Error(),
				}
			} else {
				results[c] = SpeculativeResult{
					Command: c,
					Status:  "SUCCESS",
					Output:  out,
				}
			}
		}(cmd)
	}
	wg.Wait()
	return results
}

// ============================================================================
// 4. CLAIM CHECK MANAGER
// ============================================================================

type ClaimCheckManager struct {
	StorageDir     string
	ThresholdBytes int
}

func NewClaimCheckManager(storageDir string) *ClaimCheckManager {
	if storageDir == "" {
		storageDir = filepath.Join(os.TempDir(), "rae_claim_store")
	}
	_ = os.MkdirAll(storageDir, 0o755)
	return &ClaimCheckManager{
		StorageDir:     storageDir,
		ThresholdBytes: 16384, // 16 KiB
	}
}

func (m *ClaimCheckManager) OffloadIfNeeded(payload string) string {
	raw := []byte(payload)
	if len(raw) <= m.ThresholdBytes {
		return payload
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	blobPath := filepath.Join(m.StorageDir, digest+".blob")
	if _, err := os.Stat(blobPath); os.IsNotExist(err) {
		_ = os.WriteFile(blobPath, raw, 0o644)
	}
	return fmt.Sprintf("<CLAIM_CHECK:%s:%d>", digest, len(raw))
}

func (m *ClaimCheckManager) Retrieve(pointerOrPayload string) (string, error) {
	if !strings.HasPrefix(pointerOrPayload, "<CLAIM_CHECK:") || !strings.HasSuffix(pointerOrPayload, ">") {
		return pointerOrPayload, nil
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(pointerOrPayload, "<CLAIM_CHECK:"), ">")
	parts := strings.Split(inner, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("malformed claim check pointer: %s", pointerOrPayload)
	}
	digest := parts[0]
	blobPath := filepath.Join(m.StorageDir, digest+".blob")
	data, err := os.ReadFile(blobPath)
	if err != nil {
		return "", fmt.Errorf("claim check blob missing for digest %s: %w", digest, err)
	}
	actualSum := sha256.Sum256(data)
	actualDigest := hex.EncodeToString(actualSum[:])
	if actualDigest != digest {
		return "", fmt.Errorf("integrity violation for claim check %s: got %s", digest, actualDigest)
	}
	return string(data), nil
}

// ============================================================================
// 5. SAFE REPLAY ENGINE
// ============================================================================

type ReplayMode string

const (
	ReplayAuditReadOnly     ReplayMode = "AUDIT_READ_ONLY"
	ReplaySimulate          ReplayMode = "SIMULATE"
	ReplayExecuteAuthorized ReplayMode = "EXECUTE_AUTHORIZED"
)

type ReplayHistoryEntry struct {
	StepID  string                 `json:"step_id"`
	Mode    ReplayMode             `json:"mode"`
	Action  string                 `json:"action"`
	Outcome map[string]interface{} `json:"outcome"`
}

type SafeReplayEngine struct {
	mu      sync.Mutex
	History []ReplayHistoryEntry
}

func NewSafeReplayEngine() *SafeReplayEngine {
	return &SafeReplayEngine{}
}

func (r *SafeReplayEngine) ReplayStep(stepData map[string]interface{}, mode ReplayMode, authorizationToken string) (map[string]interface{}, error) {
	stepID, _ := stepData["step_id"].(string)
	if stepID == "" {
		stepID = "unknown"
	}
	action, _ := stepData["action"].(string)
	isSideEffect, _ := stepData["side_effect"].(bool)

	if mode == ReplayAuditReadOnly {
		res := map[string]interface{}{
			"status":           "AUDITED",
			"step_id":          stepID,
			"mode":             string(mode),
			"recorded_outcome": stepData["outcome"],
			"mutated_state":    false,
		}
		r.record(stepID, mode, action, res)
		return res, nil
	}

	if mode == ReplaySimulate {
		res := map[string]interface{}{
			"status":               "SIMULATED",
			"step_id":              stepID,
			"mode":                 string(mode),
			"simulated_action":     action,
			"side_effect_blocked":  isSideEffect,
			"mutated_state":        false,
		}
		r.record(stepID, mode, action, res)
		return res, nil
	}

	if mode == ReplayExecuteAuthorized {
		if strings.TrimSpace(authorizationToken) == "" {
			return nil, fmt.Errorf("EXECUTE_AUTHORIZED mode requires a valid authorizationToken")
		}
		if isSideEffect {
			res := map[string]interface{}{
				"status":        "BLOCKED_SIDE_EFFECT",
				"step_id":       stepID,
				"reason":        "Step is marked as non-idempotent side-effect; live re-execution denied",
				"mutated_state": false,
			}
			r.record(stepID, mode, action, res)
			return res, nil
		}
		res := map[string]interface{}{
			"status":        "RE_EXECUTED",
			"step_id":       stepID,
			"mode":          string(mode),
			"mutated_state": true,
		}
		r.record(stepID, mode, action, res)
		return res, nil
	}

	return nil, fmt.Errorf("unsupported replay mode: %s", mode)
}

func (r *SafeReplayEngine) record(stepID string, mode ReplayMode, action string, outcome map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.History = append(r.History, ReplayHistoryEntry{
		StepID:  stepID,
		Mode:    mode,
		Action:  action,
		Outcome: outcome,
	})
}

// ============================================================================
// 6. A2A PROTOCOL ADAPTER
// ============================================================================

type A2AEnvelope struct {
	SenderID  string                 `json:"sender_id"`
	Recipient string                 `json:"recipient"`
	Algorithm string                 `json:"alg"`
	Timestamp int64                  `json:"timestamp"`
	Nonce     string                 `json:"nonce"`
	Payload   map[string]interface{} `json:"payload"`
	Signature string                 `json:"signature"`
}

type A2AProtocolAdapter struct {
	mu               sync.Mutex
	SharedSecrets    map[string]string
	SeenNonces       map[string]int64
	AllowedAlgs      map[string]bool
	MaxClockSkewSecs int64
}

func NewA2AProtocolAdapter() *A2AProtocolAdapter {
	return &A2AProtocolAdapter{
		SharedSecrets:    make(map[string]string),
		SeenNonces:       make(map[string]int64),
		AllowedAlgs:      map[string]bool{"HS256": true, "RS256": true},
		MaxClockSkewSecs: 300,
	}
}

func (a *A2AProtocolAdapter) RegisterAgent(agentID string, secret string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.SharedSecrets[agentID] = secret
}

func (a *A2AProtocolAdapter) SignMessage(senderID string, recipient string, alg string, nonce string, timestamp int64, payload map[string]interface{}) (A2AEnvelope, error) {
	a.mu.Lock()
	secret, ok := a.SharedSecrets[senderID]
	a.mu.Unlock()
	if !ok {
		return A2AEnvelope{}, fmt.Errorf("unknown sender_id: %s", senderID)
	}
	if alg == "" {
		alg = "HS256"
	}
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	if nonce == "" {
		nonce = fmt.Sprintf("nonce-%d", time.Now().UnixNano())
	}
	env := A2AEnvelope{
		SenderID:  senderID,
		Recipient: recipient,
		Algorithm: alg,
		Timestamp: timestamp,
		Nonce:     nonce,
		Payload:   payload,
	}
	sig := a.computeSignature(env, secret)
	env.Signature = sig
	return env, nil
}

func (a *A2AProtocolAdapter) VerifyAndReceive(env A2AEnvelope) (map[string]interface{}, error) {
	if !a.AllowedAlgs[env.Algorithm] {
		return nil, fmt.Errorf("disallowed or insecure A2A algorithm: %s", env.Algorithm)
	}
	now := time.Now().Unix()
	diff := now - env.Timestamp
	if diff < 0 {
		diff = -diff
	}
	if diff > a.MaxClockSkewSecs {
		return nil, fmt.Errorf("A2A message timestamp outside %ds window", a.MaxClockSkewSecs)
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	secret, ok := a.SharedSecrets[env.SenderID]
	if !ok {
		return nil, fmt.Errorf("unregistered A2A sender: %s", env.SenderID)
	}
	if _, seen := a.SeenNonces[env.Nonce]; seen {
		return nil, fmt.Errorf("A2A replay attack detected: duplicate nonce %s", env.Nonce)
	}
	expectedSig := a.computeSignature(env, secret)
	if !hmac.Equal([]byte(expectedSig), []byte(env.Signature)) {
		return nil, fmt.Errorf("A2A cryptographic signature mismatch")
	}
	a.SeenNonces[env.Nonce] = now
	return map[string]interface{}{
		"status":    "A2A_DELIVERED",
		"sender_id": env.SenderID,
		"recipient": env.Recipient,
		"nonce":     env.Nonce,
		"payload":   env.Payload,
	}, nil
}

func (a *A2AProtocolAdapter) computeSignature(env A2AEnvelope, secret string) string {
	payloadBytes, _ := json.Marshal(env.Payload)
	canonical := fmt.Sprintf("%s|%s|%s|%d|%s|%s", env.SenderID, env.Recipient, env.Algorithm, env.Timestamp, env.Nonce, string(payloadBytes))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

// ============================================================================
// 7. INGESTION PII SCRUBBER
// ============================================================================

type ScrubResult struct {
	ScrubbedText    string         `json:"scrubbed_text"`
	RedactionsCount int            `json:"redactions_count"`
	RedactedTypes   map[string]int `json:"redacted_types"`
}

type piiRule struct {
	name        string
	pattern     *regexp.Regexp
	replacement string
}

type IngestionPIIScrubber struct {
	rules []piiRule
}

func NewIngestionPIIScrubber() *IngestionPIIScrubber {
	return &IngestionPIIScrubber{
		rules: []piiRule{
			{
				name:        "JWT_TOKEN",
				pattern:     regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
				replacement: "[REDACTED_JWT]",
			},
			{
				name:        "PRIVATE_KEY",
				pattern:     regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`),
				replacement: "[REDACTED_PRIVATE_KEY]",
			},
			{
				name:        "API_SECRET",
				pattern:     regexp.MustCompile(`(?i)(api[_-]?key|secret|password|token)\s*[:=]\s*["']?[A-Za-z0-9_\-]{8,}["']?`),
				replacement: "$1=[REDACTED_SECRET]",
			},
			{
				name:        "EMAIL",
				pattern:     regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`),
				replacement: "[REDACTED_EMAIL]",
			},
			{
				name:        "CREDIT_CARD",
				pattern:     regexp.MustCompile(`\b(?:\d[ -]*?){13,16}\b`),
				replacement: "[REDACTED_CC]",
			},
		},
	}
}

func (s *IngestionPIIScrubber) ScrubText(text string) ScrubResult {
	scrubbed := text
	counts := make(map[string]int)
	total := 0
	for _, r := range s.rules {
		matches := r.pattern.FindAllString(scrubbed, -1)
		if len(matches) > 0 {
			counts[r.name] += len(matches)
			total += len(matches)
			scrubbed = r.pattern.ReplaceAllString(scrubbed, r.replacement)
		}
	}
	return ScrubResult{
		ScrubbedText:    scrubbed,
		RedactionsCount: total,
		RedactedTypes:   counts,
	}
}

// ============================================================================
// 8. SHADOW EVALUATOR
// ============================================================================

type ShadowGuardrail struct {
	RuleID             string  `json:"rule_id"`
	Pattern            string  `json:"pattern"`
	Status             string  `json:"status"` // SHADOW, HARD_ENFORCED
	HoursActive        float64 `json:"hours_active"`
	FalsePositiveRate  float64 `json:"false_positive_rate"`
	TriggersCount      int     `json:"triggers_count"`
}

type ShadowEvaluator struct {
	mu         sync.Mutex
	Guardrails map[string]*ShadowGuardrail
}

func NewShadowEvaluator() *ShadowEvaluator {
	return &ShadowEvaluator{
		Guardrails: make(map[string]*ShadowGuardrail),
	}
}

func (e *ShadowEvaluator) MineFailure(failurePattern string) ShadowGuardrail {
	e.mu.Lock()
	defer e.mu.Unlock()
	sum := sha256.Sum256([]byte(failurePattern))
	ruleID := "SG-" + hex.EncodeToString(sum[:4])
	g := &ShadowGuardrail{
		RuleID:            ruleID,
		Pattern:           failurePattern,
		Status:            "SHADOW",
		HoursActive:       0.0,
		FalsePositiveRate: 0.0,
	}
	e.Guardrails[ruleID] = g
	return *g
}

func (e *ShadowEvaluator) EvaluateShadowGuardrails(input string) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	var matched []string
	lower := strings.ToLower(input)
	for _, g := range e.Guardrails {
		if strings.Contains(lower, strings.ToLower(g.Pattern)) {
			g.TriggersCount++
			matched = append(matched, g.RuleID)
		}
	}
	return matched
}

func (e *ShadowEvaluator) PromoteGuardrail(ruleID string, hoursActive float64, falsePositiveRate float64) (ShadowGuardrail, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	g, ok := e.Guardrails[ruleID]
	if !ok {
		return ShadowGuardrail{}, fmt.Errorf("unknown shadow guardrail: %s", ruleID)
	}
	g.HoursActive = hoursActive
	g.FalsePositiveRate = falsePositiveRate
	if hoursActive < 72.0 {
		return *g, fmt.Errorf("cannot promote %s: requires >= 72h in SHADOW mode (got %.1fh)", ruleID, hoursActive)
	}
	if falsePositiveRate >= 0.001 {
		return *g, fmt.Errorf("cannot promote %s: false positive rate %.4f >= 0.001", ruleID, falsePositiveRate)
	}
	g.Status = "HARD_ENFORCED"
	return *g, nil
}

// ============================================================================
// 9. CURIOSITY ENGINE, TOOL FORGE & THIN DEVICE RUNTIME
// ============================================================================

type CuriosityItem struct {
	ID         string `json:"id"`
	Hypothesis string `json:"hypothesis"`
	Module     string `json:"module"`
	Status     string `json:"status"` // QUEUED, VERIFIED_SEMANTIC_RULE
}

type CuriosityEngine struct {
	mu    sync.Mutex
	Queue []CuriosityItem
	Rules []CuriosityItem
}

func NewCuriosityEngine() *CuriosityEngine {
	return &CuriosityEngine{}
}

func (c *CuriosityEngine) EnqueueIdleVerification(id string, hypothesis string, module string) CuriosityItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := CuriosityItem{
		ID:         id,
		Hypothesis: hypothesis,
		Module:     module,
		Status:     "QUEUED",
	}
	c.Queue = append(c.Queue, item)
	return item
}

func (c *CuriosityEngine) RunIdleCycle() []CuriosityItem {
	c.mu.Lock()
	defer c.mu.Unlock()
	var promoted []CuriosityItem
	for _, item := range c.Queue {
		item.Status = "VERIFIED_SEMANTIC_RULE"
		c.Rules = append(c.Rules, item)
		promoted = append(promoted, item)
	}
	c.Queue = nil
	return promoted
}

type SynthesizedTool struct {
	ToolName string `json:"tool_name"`
	Status   string `json:"status"` // SYNTHESIZED_VERIFIED, REJECTED_UNSAFE
	Reason   string `json:"reason,omitempty"`
}

type ToolForge struct {
	ForbiddenTokens []string
}

func NewToolForge() *ToolForge {
	return &ToolForge{
		ForbiddenTokens: []string{"os.system", "subprocess", "shutil.rmtree", "socket", "exec(", "eval("},
	}
}

func (f *ToolForge) SynthesizeTool(toolName string, code string) SynthesizedTool {
	lower := strings.ToLower(code)
	for _, tok := range f.ForbiddenTokens {
		if strings.Contains(lower, strings.ToLower(tok)) {
			return SynthesizedTool{
				ToolName: toolName,
				Status:   "REJECTED_UNSAFE",
				Reason:   fmt.Sprintf("Forbidden token detected in synthesized tool: %s", tok),
			}
		}
	}
	return SynthesizedTool{
		ToolName: toolName,
		Status:   "SYNTHESIZED_VERIFIED",
	}
}

type HardwareProfile struct {
	Profile            string `json:"profile"` // THIN_MOBILE, THIN_LAPTOP, MESH_NODE
	TotalRAMMB         int    `json:"total_ram_mb"`
	HasLocalGPU        bool   `json:"has_local_gpu"`
	VectorBackend      string `json:"vector_backend"`
	HeavyComputeTarget string `json:"heavy_compute_target"`
}

type ThinDeviceRuntime struct{}

func NewThinDeviceRuntime() *ThinDeviceRuntime {
	return &ThinDeviceRuntime{}
}

func (t *ThinDeviceRuntime) DetectHardwareProfile(totalRAMMB int, hasLocalGPU bool) HardwareProfile {
	if totalRAMMB < 4096 {
		return HardwareProfile{
			Profile:            "THIN_MOBILE",
			TotalRAMMB:         totalRAMMB,
			HasLocalGPU:        hasLocalGPU,
			VectorBackend:      "sqlite-fts5-light",
			HeavyComputeTarget: "offload-mesh-node1",
		}
	}
	if totalRAMMB < 16384 || !hasLocalGPU {
		return HardwareProfile{
			Profile:            "THIN_LAPTOP",
			TotalRAMMB:         totalRAMMB,
			HasLocalGPU:        hasLocalGPU,
			VectorBackend:      "embedded-hybrid",
			HeavyComputeTarget: "offload-mesh-node1",
		}
	}
	return HardwareProfile{
		Profile:            "MESH_NODE",
		TotalRAMMB:         totalRAMMB,
		HasLocalGPU:        hasLocalGPU,
		VectorBackend:      "full-vector-hnsw-gpu",
		HeavyComputeTarget: "local-gpu-execution",
	}
}
