package hive

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHiveParallelAndWatchdog(t *testing.T) {
	svc := NewService()
	svc.RecordHeartbeat("worker-1")
	stale := svc.CheckStaleHeartbeats(120)
	if len(stale) != 0 {
		t.Fatalf("expected 0 stale heartbeats, got %v", stale)
	}
	broken, sim, status := svc.EvaluateEchoLoop([]string{"same error message", "same error message"})
	if !broken || sim < 0.9 || status != "BLOCKED_LOOP" {
		t.Fatalf("expected BLOCKED_LOOP, got broken=%v, sim=%f, status=%s", broken, sim, status)
	}
	res := svc.RunParallelCommands(context.Background(), []string{"echo branch-a", "echo branch-b"})
	if len(res) != 2 || res[0].ExitCode != 0 {
		t.Fatalf("unexpected parallel results: %+v", res)
	}
}

func TestHiveComprehensiveMethodsAndRoutes(t *testing.T) {
	svc := NewService()

	// CheckStaleHeartbeats with timeout <= 0 and stale entry
	svc.mu.Lock()
	svc.heartbeats["stale-agent"] = time.Now().UTC().Add(-300 * time.Second)
	svc.mu.Unlock()
	staleDefault := svc.CheckStaleHeartbeats(0)
	if len(staleDefault) == 0 {
		t.Fatalf("expected stale-agent to be detected")
	}

	// CheckAndHealSupervisor heals stale agents
	sup := svc.CheckAndHealSupervisor()
	if !sup.FactoryAlive || sup.StalledRestarted < 1 {
		t.Fatalf("expected supervisor to heal stalled agent, got %+v", sup)
	}

	// EvaluateEchoLoop edge cases
	b0, _, st0 := svc.EvaluateEchoLoop([]string{"only-one"})
	if b0 || st0 != "OK" {
		t.Fatalf("expected OK for single reason")
	}
	b1, _, st1 := svc.EvaluateEchoLoop([]string{"alpha beta", "completely different reason xyz"})
	if b1 || st1 != "OK" {
		t.Fatalf("expected OK for distinct reasons")
	}
	b3, _, st3 := svc.EvaluateEchoLoop([]string{"r1", "r2", "r3"})
	if !b3 || st3 != "BLOCKED_LOOP" {
		t.Fatalf("expected BLOCKED_LOOP when retries >= MaxRetriesThreshold")
	}

	// RunParallelCommands edge cases: empty string command and failing command
	cmds := []string{"", "   ", "false", "echo ok1", "echo ok2"}
	res := svc.RunParallelCommands(context.Background(), cmds)
	if len(res) != 5 || res[0].ExitCode != 1 || res[2].ExitCode != 1 || res[3].ExitCode != 0 {
		t.Fatalf("unexpected parallel edge results: %+v", res)
	}

	// StartWatchdogLoop with default and short interval
	ctx1, cancel1 := context.WithCancel(context.Background())
	svc.StartWatchdogLoop(ctx1, 0)
	cancel1()

	ctx2, cancel2 := context.WithCancel(context.Background())
	svc.StartWatchdogLoop(ctx2, 5*time.Millisecond)
	time.Sleep(15 * time.Millisecond)
	cancel2()
	time.Sleep(5 * time.Millisecond)

	// SliceCodeAtomically edge cases
	code := `package demo
import "fmt"
from math import sqrt

// pre-symbol comment
var x = 1

func Alpha(a int) {
	fmt.Println(a)
}

def beta():
	pass

class Gamma:
	pass

type Delta struct{}
`
	slices := SliceCodeAtomically("demo.go", code)
	if len(slices) < 5 {
		t.Fatalf("expected at least 5 slices, got %d", len(slices))
	}

	emptySlices := SliceCodeAtomically("empty.go", "package main\nimport \"os\"\n\n")
	if len(emptySlices) != 1 || emptySlices[0].SymbolName != "root" {
		t.Fatalf("expected root fallback slice for imports-only file, got %+v", emptySlices)
	}

	// Test all HTTP routes
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	endpoints := []struct {
		method string
		path   string
		body   string
		expect string
	}{
		{http.MethodGet, "/health", "", "RAE-Hive"},
		{http.MethodGet, "/mcp/sse", "", "/mcp/messages"},
		{http.MethodPost, "/mcp/messages", `{"id":1}`, "RAE-Hive"},
		{http.MethodGet, "/v2/hive/watchdog", "", "monitoring"},
		{http.MethodPost, "/v2/hive/parallel", `{"commands":["echo p1",""]}`, "results"},
		{http.MethodPost, "/execute", `{}`, "hive-ok"},
		{http.MethodPost, "/execute", `{"command":"echo custom-cmd"}`, "custom-cmd"},
		{http.MethodGet, "/metrics", "", "max_parallel_branches"},
		{http.MethodPost, "/v2/hive/atomic-slice", `{"code":"func Foo() {}\n"}`, "slices_count"},
		{http.MethodPost, "/v2/hive/atomic-slice", `{"filename":"a.py","code":"def bar():\n  return 1\n"}`, "a.py"},
		{http.MethodGet, "/v2/hive/supervisor", "", "v2.0-atomic"},
		{http.MethodGet, "/metrics/prometheus", "", "rae_hive_max_parallel_branches"},
	}

	for _, tc := range endpoints {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s returned status %d", tc.method, tc.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.expect) {
			t.Fatalf("%s %s body %q did not contain %q", tc.method, tc.path, rec.Body.String(), tc.expect)
		}
	}
}
