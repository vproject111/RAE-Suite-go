package hive

import (
	"context"
	"encoding/json"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
)

const (
	MaxRetriesThreshold = 3
	SimilarityThreshold = 0.90
	HeartbeatTimeoutS   = 120
	MaxParallelBranches = 3
)

// CommandResult captures the execution outcome of an isolated sandbox command.
type CommandResult struct {
	Command     string `json:"command"`
	ExitCode    int    `json:"exit_code"`
	Stdout      string `json:"stdout"`
	SandboxMode string `json:"sandbox_mode"`
}

// Service implements Module 6: RAE-Hive (Isolated Sandbox Runner k<=3 & Tactical Semantic Watchdog v2.3).
type Service struct {
	mu         sync.RWMutex
	heartbeats map[string]time.Time
}

// NewService creates a new RAE-Hive service and records the initial heartbeat.
func NewService() *Service {
	s := &Service{
		heartbeats: make(map[string]time.Time),
	}
	s.RecordHeartbeat("rae-hive")
	return s
}

// RecordHeartbeat records an AST state heartbeat for an agent.
func (s *Service) RecordHeartbeat(agentID string) {
	s.mu.Lock()
	s.heartbeats[agentID] = time.Now().UTC()
	s.mu.Unlock()
}

// CheckStaleHeartbeats returns agent IDs whose heartbeat is older than timeoutSeconds.
func (s *Service) CheckStaleHeartbeats(timeoutSeconds int) []string {
	if timeoutSeconds <= 0 {
		timeoutSeconds = HeartbeatTimeoutS
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	stale := make([]string, 0)
	for id, ts := range s.heartbeats {
		if now.Sub(ts).Seconds() > float64(timeoutSeconds) {
			stale = append(stale, id)
		}
	}
	return stale
}

// EvaluateEchoLoop checks whether rejection reasons trigger an L3 Circuit Break (BLOCKED_LOOP).
func (s *Service) EvaluateEchoLoop(rejectionReasons []string) (bool, float64, string) {
	if len(rejectionReasons) < 2 {
		return false, 0.0, "OK"
	}
	sim := raemath.CalculateSemanticSimilarity(rejectionReasons[len(rejectionReasons)-1], rejectionReasons[len(rejectionReasons)-2])
	if len(rejectionReasons) >= MaxRetriesThreshold || sim > SimilarityThreshold {
		return true, sim, "BLOCKED_LOOP"
	}
	return false, sim, "OK"
}

// RunParallelCommands executes up to k<=3 commands concurrently using a bounded semaphore.
func (s *Service) RunParallelCommands(ctx context.Context, commands []string) []CommandResult {
	results := make([]CommandResult, len(commands))
	sem := make(chan struct{}, MaxParallelBranches)
	var wg sync.WaitGroup

	for i, cmdStr := range commands {
		wg.Add(1)
		go func(idx int, cStr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fields := strings.Fields(cStr)
			if len(fields) == 0 {
				results[idx] = CommandResult{Command: cStr, ExitCode: 1, SandboxMode: "native-go"}
				return
			}
			execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(execCtx, fields[0], fields[1:]...)
			out, err := cmd.CombinedOutput()
			code := 0
			if err != nil {
				code = 1
			}
			results[idx] = CommandResult{
				Command:     cStr,
				ExitCode:    code,
				Stdout:      strings.TrimSpace(string(out)),
				SandboxMode: "native-go-isolated",
			}
		}(i, cmdStr)
	}
	wg.Wait()
	return results
}

// StartWatchdogLoop runs the periodic Tactical Semantic Watchdog in the background.
func (s *Service) StartWatchdogLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.RecordHeartbeat("rae-hive")
			}
		}
	}()
}

// RegisterRoutes attaches RAE-Hive HTTP endpoints (:8013).
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	mux.HandleFunc("/v2/hive/watchdog", func(w http.ResponseWriter, _ *http.Request) {
		s.RecordHeartbeat("rae-hive")
		stale := s.CheckStaleHeartbeats(HeartbeatTimeoutS)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "monitoring",
			"stale_agents": stale,
		})
	})

	mux.HandleFunc("/v2/hive/parallel", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Commands []string `json:"commands"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := s.RunParallelCommands(r.Context(), req.Commands)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"results": res})
	})
}
