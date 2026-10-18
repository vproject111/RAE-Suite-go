package memory

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
)

func TestAll19MemoryRouterGroups(t *testing.T) {
	store := memorystore.NewUnifiedStore(nil, nil)
	svc := NewService(store, store, nil)
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	endpoints := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/ready", ""},
		{http.MethodGet, "/live", ""},
		{http.MethodGet, "/metrics", ""},
		{http.MethodGet, "/openapi.json", ""},
		{http.MethodPost, "/embeddings", `{"text":"test embedding"}`},
		{http.MethodPost, "/rerank", `{"query":"test","top_k":3}`},
		{http.MethodPost, "/v2/memories", `{"content":"no-slash memory","layer":"semantic"}`},
		{http.MethodGet, "/v2/memories", ""},
		{http.MethodPost, "/v2/memories/reflections", `{"answer_draft":"ok"}`},
		{http.MethodGet, "/v2/memories/sessions/sess-1", ""},
		{http.MethodPost, "/v2/agent/execute", `{"prompt":"hello","project":"p1"}`},
		{http.MethodPost, "/v2/bridge/interact", `{"content":"implicit event","project":"p1"}`},
		{http.MethodPost, "/v2/bridge/audit", `{}`},
		{http.MethodPost, "/v2/feedback", `{"arm_id":"fluid_100","reward":0.95}`},
		{http.MethodPost, "/v2/compliance/approvals", `{"action":"export"}`},
		{http.MethodGet, "/v2/compliance/circuit-breakers", ""},
		{http.MethodPost, "/v2/mesh/invite", `{}`},
		{http.MethodGet, "/v2/mesh/handshake/challenge", ""},
		{http.MethodPost, "/v2/mesh/join", `{"peer_id":"peer-test"}`},
		{http.MethodPost, "/v2/mesh/handshake", `{}`},
		{http.MethodPost, "/v2/mesh/sync/receive", `{"memories":[]}`},
		{http.MethodPost, "/v2/graph/nodes", `{"id":"gn1","content":"Node 1"}`},
		{http.MethodPost, "/v2/graph/edges", `{"source_id":"gn1","target_id":"gn2","relation":"LINKS"}`},
		{http.MethodPost, "/v2/graph/path/shortest", `{"source_id":"gn1","target_id":"gn2"}`},
		{http.MethodPost, "/v2/graph/cycles/detect", `{}`},
		{http.MethodPost, "/v2/graph/snapshots", `{}`},
		{http.MethodGet, "/v2/graph/snapshots", ""},
		{http.MethodPost, "/v2/graph/statistics", `{}`},
		{http.MethodGet, "/v2/graph/health", ""},
		{http.MethodPost, "/v2/automation/create", `{"name":"trig1"}`},
		{http.MethodGet, "/v2/automation/list", ""},
		{http.MethodPost, "/v2/automation/events/emit", `{"type":"MEMORY_CREATED"}`},
		{http.MethodGet, "/v2/automation/events/types", ""},
		{http.MethodPost, "/v2/automation/workflows/create", `{"name":"wf1"}`},
		{http.MethodGet, "/v2/automation/workflows", ""},
		{http.MethodGet, "/v2/automation/templates", ""},
		{http.MethodGet, "/v2/automation/health", ""},
		{http.MethodPost, "/v2/evaluation/search", `{}`},
		{http.MethodGet, "/v2/evaluation/metrics/supported", ""},
		{http.MethodPost, "/v2/evaluation/drift/detect", `{}`},
		{http.MethodGet, "/v2/evaluation/drift/severity-levels", ""},
		{http.MethodPost, "/v2/evaluation/ab-test/create", `{}`},
		{http.MethodGet, "/v2/evaluation/quality/thresholds", ""},
		{http.MethodPost, "/v2/evaluation/benchmark/run", `{}`},
		{http.MethodGet, "/v2/evaluation/benchmark/suites", ""},
		{http.MethodPost, "/v2/search/hybrid", `{"query":"test"}`},
		{http.MethodPost, "/v2/search/evidence", `{"query":"test"}`},
		{http.MethodPost, "/v2/search/analyze", `{"query":"test query"}`},
		{http.MethodGet, "/v2/search/weights/profiles", ""},
		{http.MethodPost, "/v2/search/weights/calculate", `{}`},
		{http.MethodGet, "/v2/search/info", ""},
		{http.MethodPost, "/v2/reflections/communities/synthesize", `{}`},
		{http.MethodPost, "/v2/reflections/generate", `{"answer_draft":"test"}`},
		{http.MethodPost, "/v2/reflections/query", `{}`},
		{http.MethodPost, "/v2/nodes/tasks", `{"node_id":"node1-lumina","type":"bench"}`},
		{http.MethodGet, "/v2/nodes/nodes", ""},
		{http.MethodGet, "/v2/dashboard/metrics", ""},
		{http.MethodGet, "/v2/dashboard/compliance/report", ""},
		{http.MethodGet, "/v2/dashboard/compliance/rls-status", ""},
		{http.MethodGet, "/v2/sync/topology", ""},
		{http.MethodGet, "/v2/sync/stats", ""},
		{http.MethodGet, "/v2/tuning/theories", ""},
		{http.MethodPost, "/v2/tuning/run", `{}`},
		{http.MethodGet, "/v2/token-savings/metrics/savings", ""},
		{http.MethodPost, "/v2/federation/query", `{"query":"*"}`},
		{http.MethodGet, "/system/tenants", ""},
		{http.MethodGet, "/system/projects", ""},
		{http.MethodGet, "/admin/overview", ""},
	}

	for _, ep := range endpoints {
		req := httptest.NewRequest(ep.method, ep.path, bytes.NewReader([]byte(ep.body)))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s returned %d (expected 200): %s", ep.method, ep.path, rec.Code, rec.Body.String())
		}
	}
}
