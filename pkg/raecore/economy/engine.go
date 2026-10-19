package economy

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

// BatchTask represents an individual engineering task queued for batch coalescing.
type BatchTask struct {
	TaskID     string  `json:"task_id"`
	Module     string  `json:"module"`
	TargetFile string  `json:"target_file"`
	ActionType string  `json:"action_type"` // refactor, test, lint, analyse
	Complexity float64 `json:"complexity"`
	TokenCost  int     `json:"token_cost"`
	CreatedAt  float64 `json:"created_at"`
}

// Batch represents a coalesced group of tasks sharing module/file context.
type Batch struct {
	BatchID   string      `json:"batch_id"`
	Module    string      `json:"module"`
	Tasks     []BatchTask `json:"tasks"`
	IsWarm    bool        `json:"is_warm"`
	SetupCost int         `json:"setup_cost"`
}

// BatchOptimizationEngine groups tasks to minimize Context Switch Cost (CSC),
// amortize setup token cost (15,000 tokens), and enforce the Knowledge Scale Effect (Efekt Skali Wiedzy).
type BatchOptimizationEngine struct {
	mu                     sync.Mutex
	pendingTasks           []BatchTask
	maxPending             int
	warmAgents             map[string]float64
	totalSwitchesPrevented int
	totalTokensSaved       int
	totalTasksProcessed    int
}

// NewBatchOptimizationEngine creates a new BatchOptimizationEngine.
func NewBatchOptimizationEngine(maxPending int) *BatchOptimizationEngine {
	if maxPending <= 0 {
		maxPending = 1000
	}
	return &BatchOptimizationEngine{
		pendingTasks: make([]BatchTask, 0),
		maxPending:   maxPending,
		warmAgents:   make(map[string]float64),
	}
}

// AddTask validates and appends a BatchTask to the pending queue.
func (e *BatchOptimizationEngine) AddTask(task BatchTask) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.pendingTasks) >= e.maxPending {
		return fmt.Errorf("batch queue overflow: reached maximum pending tasks capacity (%d)", e.maxPending)
	}
	switch task.ActionType {
	case "refactor", "test", "lint", "analyse":
	default:
		return fmt.Errorf("invalid task action type: %s", task.ActionType)
	}
	if task.TaskID == "" {
		task.TaskID = "tsk-" + models.GenerateUUIDv7()[:6]
	}
	if task.Complexity <= 0 {
		task.Complexity = 0.5
	}
	if task.TokenCost <= 0 {
		task.TokenCost = 2000
	}
	if task.CreatedAt <= 0 {
		task.CreatedAt = float64(time.Now().Unix())
	}
	e.pendingTasks = append(e.pendingTasks, task)
	return nil
}

// OptimizeAndGroup groups pending tasks by target module and applies the Knowledge Scale Effect
// (pulling extra tasks touching the same TargetFile into the batch).
func (e *BatchOptimizationEngine) OptimizeAndGroup() []Batch {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.pendingTasks) == 0 {
		return []Batch{}
	}

	now := float64(time.Now().Unix())
	batchesDict := make(map[string]*Batch)
	var moduleOrder []string

	// 1. Similarity Check & Task Merging (Group by target module)
	for _, task := range e.pendingTasks {
		mod := task.Module
		b, exists := batchesDict[mod]
		if !exists {
			lastActive, warm := e.warmAgents[mod]
			isWarm := warm && (now-lastActive < 600.0)
			b = &Batch{
				BatchID:   "bat-" + models.GenerateUUIDv7()[:6],
				Module:    mod,
				Tasks:     make([]BatchTask, 0),
				IsWarm:    isWarm,
				SetupCost: 15000,
			}
			batchesDict[mod] = b
			moduleOrder = append(moduleOrder, mod)
		}
		b.Tasks = append(b.Tasks, task)
	}

	// 2. Efekt Skali Wiedzy (Knowledge Scale Effect):
	// Pull extra related pending tasks touching the same target files into active groups.
	for _, mod := range moduleOrder {
		b := batchesDict[mod]
		activeFiles := make(map[string]bool)
		inBatch := make(map[string]bool)
		for _, t := range b.Tasks {
			activeFiles[t.TargetFile] = true
			inBatch[t.TaskID] = true
		}
		for _, cand := range e.pendingTasks {
			if !inBatch[cand.TaskID] && cand.TargetFile != "" && activeFiles[cand.TargetFile] {
				b.Tasks = append(b.Tasks, cand)
				inBatch[cand.TaskID] = true
			}
		}
	}

	e.pendingTasks = make([]BatchTask, 0)
	out := make([]Batch, 0, len(moduleOrder))
	for _, mod := range moduleOrder {
		out = append(out, *batchesDict[mod])
	}
	return out
}

// DispatchBatch dispatches a coalesced batch and computes tokens saved and duration saved (seconds).
func (e *BatchOptimizationEngine) DispatchBatch(batch Batch) (int, float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	numTasks := len(batch.Tasks)
	if numTasks == 0 {
		return 0, 0.0
	}
	setupCost := batch.SetupCost
	if setupCost <= 0 {
		setupCost = 15000
	}
	setupRuns := numTasks
	if batch.IsWarm {
		setupRuns = numTasks + 1
	}
	tokensSaved := setupCost * (setupRuns - 1)
	e.warmAgents[batch.Module] = float64(time.Now().Unix())

	e.totalSwitchesPrevented += numTasks - 1
	e.totalTokensSaved += tokensSaved
	e.totalTasksProcessed += numTasks

	return tokensSaved, float64(numTasks-1) * 1.5
}

// GetSavingsReport returns aggregated outcome metrics for the BatchOptimizationEngine.
func (e *BatchOptimizationEngine) GetSavingsReport() map[string]interface{} {
	e.mu.Lock()
	defer e.mu.Unlock()

	amortizationRate := 0.0
	if e.totalTasksProcessed > 0 {
		amortizationRate = float64(e.totalTokensSaved) / float64(e.totalTasksProcessed*15000)
	}
	return map[string]interface{}{
		"switches_prevented": e.totalSwitchesPrevented,
		"tokens_saved":       e.totalTokensSaved,
		"tasks_processed":    e.totalTasksProcessed,
		"amortization_rate":  math.Round(amortizationRate*10000) / 10000,
		"batch_gain":         math.Round(float64(e.totalTokensSaved)*0.000015*1e6) / 1e6,
	}
}

// ContextEnvelopeItem represents a retrieved memory envelope for hierarchical pruning.
type ContextEnvelopeItem struct {
	ContextID        string  `json:"context_id"`
	MemoryLayer      string  `json:"memory_layer"`
	TrustScore       float64 `json:"trust_score"`
	SourceHash       string  `json:"source_hash"`
	RetrievedContent string  `json:"retrieved_content"`
	InformationClass string  `json:"information_class"`
}

// ContextBroker implements Adaptive Retrieval Depth and Hierarchical Context Pruning.
type ContextBroker struct {
	TokenLimit      int
	RejectThreshold float64
}

// NewContextBroker creates a new ContextBroker.
func NewContextBroker(tokenLimit int, rejectThreshold float64) *ContextBroker {
	if tokenLimit <= 0 {
		tokenLimit = 1500
	}
	if rejectThreshold <= 0 {
		rejectThreshold = 0.4
	}
	return &ContextBroker{
		TokenLimit:      tokenLimit,
		RejectThreshold: rejectThreshold,
	}
}

// DetermineRetrievalDepth analyzes candidate scores and returns (kDepth, runReranker).
func (cb *ContextBroker) DetermineRetrievalDepth(scores []float64) (int, bool, error) {
	if len(scores) == 0 {
		return 0, false, nil
	}
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })

	if len(sorted) == 1 {
		return 1, false, nil
	}
	gap := sorted[0] - sorted[1]
	if gap > 0.15 {
		return 3, false, nil
	} else if gap < 0.05 {
		return 30, true, nil
	}
	return 10, true, nil
}

// PruneContext compresses envelopes into Constitution -> Project Profile -> Retrieved Memory Leaves within TokenLimit.
func (cb *ContextBroker) PruneContext(envelopes []ContextEnvelopeItem, constitution string) string {
	var valid []ContextEnvelopeItem
	for _, e := range envelopes {
		if e.TrustScore >= cb.RejectThreshold {
			valid = append(valid, e)
		}
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].TrustScore > valid[j].TrustScore })

	if constitution == "" {
		constitution = "Standard safety invariants"
	}
	constSec := "=== CONSTITUTION ===\n" + constitution
	profSec := "=== PROJECT PROFILE ===\nProject: RAE-Suite v4.3\nConfig: STRICT_SEMVER=true"
	fixedStr := constSec + "\n\n" + profSec
	fixedWords := strings.Fields(fixedStr)
	if len(fixedWords) > cb.TokenLimit {
		return strings.Join(fixedWords[:cb.TokenLimit], " ")
	}

	var leafParts []string
	currCount := len(fixedWords)
	for _, env := range valid {
		sh := env.SourceHash
		if len(sh) > 8 {
			sh = sh[:8]
		}
		if sh == "" {
			sh = "n/a"
		}
		formatted := fmt.Sprintf("[%s] (Trust: %.2f) (Hash: %s):\n%s",
			strings.ToUpper(env.MemoryLayer), env.TrustScore, sh, strings.TrimSpace(env.RetrievedContent))
		wc := len(strings.Fields(formatted))
		if currCount+wc <= cb.TokenLimit {
			leafParts = append(leafParts, formatted)
			currCount += wc
		}
	}
	leavesSec := "=== RETRIEVED MEMORY LEAVES ===\n" + strings.Join(leafParts, "\n\n")
	return fixedStr + "\n\n" + leavesSec
}

// ContextTrustEvaluator evaluates retrieved context against memory poisoning and ISO 27000 isolation.
type ContextTrustEvaluator struct {
	RejectThreshold   float64
	AdvisoryThreshold float64
}

// NewContextTrustEvaluator creates a new ContextTrustEvaluator.
func NewContextTrustEvaluator() *ContextTrustEvaluator {
	return &ContextTrustEvaluator{
		RejectThreshold:   0.4,
		AdvisoryThreshold: 0.7,
	}
}

// EvaluateTrust computes (trustScore, classification) for a memory metadata map.
func (cte *ContextTrustEvaluator) EvaluateTrust(meta map[string]interface{}) (float64, string) {
	infoClass, _ := meta["information_class"].(string)
	if infoClass == "" {
		infoClass = "internal"
	}
	layer, _ := meta["layer"].(string)
	if layer == "" {
		layer = "episodic"
	}
	if strings.EqualFold(infoClass, "restricted") && !strings.EqualFold(layer, "working") {
		return 0.0, "REJECTED"
	}

	score := 0.5
	switch strings.ToLower(layer) {
	case "semantic":
		score += 0.2
	case "reflective":
		score += 0.1
	case "working":
		score -= 0.1
	}

	succRate := 0.5
	if v, ok := meta["historical_success_rate"].(float64); ok {
		succRate = v
	}
	score += (succRate - 0.5) * 0.4

	riskClass, _ := meta["risk_class"].(string)
	if riskClass == "R5" || riskClass == "R6" {
		score -= 0.3
	}

	quarantined, _ := meta["quarantined"].(bool)
	if !quarantined {
		quarantined, _ = meta["quarantine_status"].(bool)
	}
	if quarantined {
		score = 0.5
	}
	score = math.Max(0.0, math.Min(1.0, score))

	classification := "REJECTED"
	if quarantined {
		classification = "ADVISORY_ONLY"
	} else if score >= cte.AdvisoryThreshold {
		classification = "USABLE_PLANNING_INPUT"
	} else if score >= cte.RejectThreshold {
		classification = "ADVISORY_ONLY"
	}
	return math.Round(score*100) / 100, classification
}

// HardenedSemanticCache implements Singleflight request coalescing, TTL Jitter (±15%), and Negative Caching.
type HardenedSemanticCache struct {
	mu          sync.Mutex
	baseTTLS    float64
	jitterRatio float64
	entries     map[string]cacheItem
	inflight    map[string]*sync.Mutex
}

type cacheItem struct {
	kind       string
	value      interface{}
	freshUntil time.Time
}

// NewHardenedSemanticCache constructs a new HardenedSemanticCache.
func NewHardenedSemanticCache(baseTTLS, jitterRatio float64) *HardenedSemanticCache {
	if baseTTLS <= 0 {
		baseTTLS = 60.0
	}
	if jitterRatio <= 0 {
		jitterRatio = 0.15
	}
	return &HardenedSemanticCache{
		baseTTLS:    baseTTLS,
		jitterRatio: jitterRatio,
		entries:     make(map[string]cacheItem),
		inflight:    make(map[string]*sync.Mutex),
	}
}

// GetOrCompute retrieves a cached value or computes it under a per-key Singleflight lock.
func (c *HardenedSemanticCache) GetOrCompute(key string, computeFn func() interface{}, isNegative bool) (interface{}, error) {
	if computeFn == nil {
		return nil, errors.New("computeFn cannot be nil")
	}
	now := time.Now()
	c.mu.Lock()
	if item, ok := c.entries[key]; ok && now.Before(item.freshUntil) {
		c.mu.Unlock()
		return item.value, nil
	}
	keyLock, ok := c.inflight[key]
	if !ok {
		keyLock = &sync.Mutex{}
		c.inflight[key] = keyLock
	}
	c.mu.Unlock()

	keyLock.Lock()
	defer keyLock.Unlock()

	c.mu.Lock()
	if item, ok := c.entries[key]; ok && time.Now().Before(item.freshUntil) {
		c.mu.Unlock()
		return item.value, nil
	}
	c.mu.Unlock()

	val := computeFn()
	mult := (1.0 - c.jitterRatio) + rand.Float64()*(2.0*c.jitterRatio)
	ttl := time.Duration(c.baseTTLS*mult*1000) * time.Millisecond
	kind := "VALUE"
	if isNegative {
		kind = "NEGATIVE"
	}

	c.mu.Lock()
	c.entries[key] = cacheItem{
		kind:       kind,
		value:      val,
		freshUntil: time.Now().Add(ttl),
	}
	delete(c.inflight, key)
	c.mu.Unlock()
	return val, nil
}
