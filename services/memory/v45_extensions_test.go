package memory

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
)

func TestV45ExtensionsAgnosticMeshIngestionSchedulerAndBenchmarks(t *testing.T) {
	store := memorystore.NewUnifiedStore(nil, nil)
	svc := NewService(store, nil, nil)
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	// 1. Test Document Ingestion & Graph Triple Extraction
	ingestBody, _ := json.Marshal(map[string]interface{}{
		"title":   "RAE Agnostic Spec",
		"text":    "# Core Architecture\nRAEMemory uses AgnosticStorageBroker and connects to Qdrant.\n## Federation\nRAEMesh implements MerkleTree.",
		"project": "v45-test",
		"layer":   "semantic",
	})
	reqIngest := httptest.NewRequest(http.MethodPost, "/v2/memories/ingest-document", bytes.NewReader(ingestBody))
	wIngest := httptest.NewRecorder()
	mux.ServeHTTP(wIngest, reqIngest)
	if wIngest.Code != http.StatusOK {
		t.Fatalf("expected 200 on ingest-document, got %d: %s", wIngest.Code, wIngest.Body.String())
	}

	// 2. Test Agnostic Storage Migration (embedded -> postgres) & SQL Export
	migBody, _ := json.Marshal(map[string]interface{}{
		"source":  "embedded",
		"target":  "postgres",
		"project": "v45-test",
	})
	reqMig := httptest.NewRequest(http.MethodPost, "/v2/system/storage/migrate", bytes.NewReader(migBody))
	wMig := httptest.NewRecorder()
	mux.ServeHTTP(wMig, reqMig)
	if wMig.Code != http.StatusOK {
		t.Fatalf("expected 200 on storage/migrate, got %d", wMig.Code)
	}

	swBody, _ := json.Marshal(map[string]interface{}{"backend": "postgres"})
	reqSw := httptest.NewRequest(http.MethodPost, "/v2/system/storage/switch", bytes.NewReader(swBody))
	wSw := httptest.NewRecorder()
	mux.ServeHTTP(wSw, reqSw)
	if wSw.Code != http.StatusOK {
		t.Fatalf("expected 200 on storage/switch, got %d", wSw.Code)
	}

	reqSQL := httptest.NewRequest(http.MethodGet, "/v2/system/storage/export-sql?project=v45-test&dialect=postgres", nil)
	wSQL := httptest.NewRecorder()
	mux.ServeHTTP(wSQL, reqSQL)
	if wSQL.Code != http.StatusOK || !strings.Contains(wSQL.Body.String(), "CREATE TABLE IF NOT EXISTS memories") {
		t.Fatalf("expected SQL DDL/DML export, got %d: %s", wSQL.Code, wSQL.Body.String())
	}

	// 3. Test 16-Bucket Merkle Tree & Diff
	diffBody, _ := json.Marshal(map[string]interface{}{
		"project":        "v45-test",
		"remote_buckets": map[string]string{"0": "deadbeef"},
	})
	reqDiff := httptest.NewRequest(http.MethodPost, "/v2/mesh/diff", bytes.NewReader(diffBody))
	wDiff := httptest.NewRecorder()
	mux.ServeHTTP(wDiff, reqDiff)
	if wDiff.Code != http.StatusOK {
		t.Fatalf("expected 200 on /v2/mesh/diff, got %d", wDiff.Code)
	}

	// 4. Test Go Ticker Scheduler Cycle
	reqSched := httptest.NewRequest(http.MethodPost, "/v2/automation/scheduler/run", bytes.NewReader([]byte(`{"project":"v45-test"}`)))
	wSched := httptest.NewRecorder()
	mux.ServeHTTP(wSched, reqSched)
	if wSched.Code != http.StatusOK {
		t.Fatalf("expected 200 on scheduler/run, got %d", wSched.Code)
	}

	// 5. Test LangChain & LlamaIndex Adapters
	reqLC := httptest.NewRequest(http.MethodPost, "/v2/integrations/langchain/retrieve", bytes.NewReader([]byte(`{"query":"AgnosticStorageBroker","project":"v45-test"}`)))
	wLC := httptest.NewRecorder()
	mux.ServeHTTP(wLC, reqLC)
	if wLC.Code != http.StatusOK || !strings.Contains(wLC.Body.String(), "langchain") {
		t.Fatalf("expected 200 on langchain/retrieve, got %d", wLC.Code)
	}

	// 6. Test Live 9/5 Benchmarks & SVG Plot
	reqBench := httptest.NewRequest(http.MethodGet, "/v2/evaluation/benchmark/nine-five?project=v45-test", nil)
	wBench := httptest.NewRecorder()
	mux.ServeHTTP(wBench, reqBench)
	if wBench.Code != http.StatusOK || !strings.Contains(wBench.Body.String(), "passed_all_targets") {
		t.Fatalf("expected 200 on benchmark/nine-five, got %d: %s", wBench.Code, wBench.Body.String())
	}

	reqSVG := httptest.NewRequest(http.MethodGet, "/v2/evaluation/benchmark/plot.svg?project=v45-test", nil)
	wSVG := httptest.NewRecorder()
	mux.ServeHTTP(wSVG, reqSVG)
	if wSVG.Code != http.StatusOK || !strings.Contains(wSVG.Body.String(), "<svg") {
		t.Fatalf("expected SVG plot, got %d: %s", wSVG.Code, wSVG.Body.String())
	}
}

type mockHijackResponseWriter struct {
	*httptest.ResponseRecorder
	conn net.Conn
}

func (m *mockHijackResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return m.conn, bufio.NewReadWriter(bufio.NewReader(m.conn), bufio.NewWriter(m.conn)), nil
}

func makeMaskedClientFrame(opcode byte, payload []byte) []byte {
	mask := [4]byte{0x12, 0x34, 0x56, 0x78}
	var hdr []byte
	hdr = append(hdr, 0x80|(opcode&0x0f))
	l := len(payload)
	if l <= 125 {
		hdr = append(hdr, 0x80|byte(l))
	} else if l <= 65535 {
		hdr = append(hdr, 0x80|126, byte(l>>8), byte(l&0xff))
	} else {
		hdr = append(hdr, 0x80|127, 0, 0, 0, 0, byte(l>>24), byte(l>>16), byte(l>>8), byte(l&0xff))
	}
	hdr = append(hdr, mask[:]...)
	masked := make([]byte, l)
	for i := 0; i < l; i++ {
		masked[i] = payload[i] ^ mask[i%4]
	}
	return append(hdr, masked...)
}

func TestFullMemoryV46V47ExtendedAndWebSocketCoverage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(RegisteredPackageTypes()) == 0 || len(AllLayerNames()) != 8 {
		t.Fatal("expected registered package types and 8 layer names")
	}

	store := memorystore.NewUnifiedStore(nil, nil)
	svc := NewService(store, store, nil)
	svcNoOutbox := NewService(store, nil, nil)
	if c, err := svcNoOutbox.ProcessGovernanceOutboxOnce(ctx, 10); err != nil || c != 0 {
		t.Fatalf("expected 0 from nil outbox, got %d, %v", c, err)
	}

	// StartBackgroundWorkers default & fast tick
	svc.StartBackgroundWorkers(ctx, 0)
	svc.StartBackgroundWorkers(ctx, 5*time.Millisecond)
	time.Sleep(15 * time.Millisecond)

	// UpgradeToSemantic, PromoteToLTM, ProceduralChunk, ResolveKnowledgeConflict defaults
	_, _ = svc.UpgradeToSemantic(ctx, "p-def", 0)
	_, _ = svc.PromoteToLTM(ctx, "p-def", 0)
	_ = ProceduralChunk("Step 1: A\nKrok 2: B", 0)
	_, _ = svc.ResolveKnowledgeConflict(ctx, models.KnowledgeConflictRequest{EntityID: "e-empty"})

	// BFS shortest path edge cases
	adj := map[string][]string{"A": {"B"}, "B": {"C"}}
	if p := findShortestPathBFS(adj, "A", "A"); len(p) != 1 {
		t.Fatalf("expected self path, got %v", p)
	}
	if p := findShortestPathBFS(adj, "A", "C"); len(p) != 3 {
		t.Fatalf("expected A->B->C, got %v", p)
	}
	if p := findShortestPathBFS(adj, "C", "A"); p != nil {
		t.Fatalf("expected nil path, got %v", p)
	}

	// RFC6455 frame encode/decode for small, medium (126), and large (127) frames + masked
	for _, sz := range []int{10, 300, 66000} {
		data := bytes.Repeat([]byte("a"), sz)
		enc := EncodeRFC6455Frame(0x1, data)
		op, dec, err := DecodeRFC6455Frame(bytes.NewReader(enc))
		if err != nil || op != 0x1 || len(dec) != sz {
			t.Fatalf("RFC6455 unmasked roundtrip failed for size=%d: op=%d, len=%d, err=%v", sz, op, len(dec), err)
		}
		mFrame := makeMaskedClientFrame(0x1, data)
		opM, decM, errM := DecodeRFC6455Frame(bytes.NewReader(mFrame))
		if errM != nil || opM != 0x1 || len(decM) != sz {
			t.Fatalf("RFC6455 masked roundtrip failed for size=%d: err=%v", sz, errM)
		}
	}
	_ = encodeWebSocketTextFrame([]byte("hello"))

	// WebSocket Upgrade: HTTP fallback, non-Hijacker 101, and full Hijacker Ping/Close/Text
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)

	doReq := func(method, path, body string) *httptest.ResponseRecorder {
		var r *http.Request
		if body != "" {
			r = httptest.NewRequest(method, path, strings.NewReader(body))
		} else {
			r = httptest.NewRequest(method, path, nil)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	// WS fallback & non-Hijacker
	if w := doReq(http.MethodGet, "/ws/dashboard", ""); w.Code != http.StatusOK {
		t.Fatalf("expected 200 on ws fallback, got %d", w.Code)
	}
	rWS := httptest.NewRequest(http.MethodGet, "/ws/dashboard", nil)
	rWS.Header.Set("Upgrade", "websocket")
	rWS.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	wWS := httptest.NewRecorder()
	mux.ServeHTTP(wWS, rWS)
	if wWS.Code != http.StatusSwitchingProtocols {
		t.Fatalf("expected 101 on non-hijacker WS, got %d", wWS.Code)
	}

	// WS Hijacker with Ping (0x9), Close (0x8), and Text (0x1)
	for _, op := range []byte{0x9, 0x8, 0x1} {
		c1, c2 := net.Pipe()
		go func(opcode byte) {
			defer c2.Close()
			buf := make([]byte, 4096)
			_, _ = c2.Read(buf) // HTTP 101 headers
			_, _ = c2.Read(buf) // initial frame
			_, _ = c2.Write(makeMaskedClientFrame(opcode, []byte("ping-or-msg")))
			_, _ = c2.Read(buf) // reply frame
		}(op)
		hjW := &mockHijackResponseWriter{ResponseRecorder: httptest.NewRecorder(), conn: c1}
		UpgradeRFC6455WebSocket(hjW, rWS, map[string]interface{}{"event": "test"})
	}

	// Store 10 memories via POST /v2/memories/ to trigger OnMemoryStoredHook auto-scheduler (count%10 == 0) + PII scrub
	for i := 0; i < 10; i++ {
		_ = doReq(http.MethodPost, "/v2/memories/", `{"content":"RAEMemory connects to Qdrant email test@example.com","layer":"working","project":"p1","metadata":{"scrub_pii":true}}`)
	}
	_ = doReq(http.MethodPost, "/v2/memories/", `{bad-json`)
	_ = doReq(http.MethodPost, "/v2/memories/", `{"content":"secret","layer":"semantic","info_class":"restricted"}`)
	_ = doReq(http.MethodDelete, "/v2/memories/?id=missing-id", "")
	_ = doReq(http.MethodGet, "/v2/memories/?query=*&project=p1&layer=working", "")
	_ = doReq(http.MethodGet, "/health", "")
	_ = doReq(http.MethodPost, "/v2/memories/scrub", `{"text":"email user@test.com"}`)
	_ = doReq(http.MethodPost, "/v2/memories/consolidate", `{"project":"p1"}`)
	_ = doReq(http.MethodPost, "/v2/memories/upgrade", `{"project":"p1","promote_to_ltm":true}`)
	_ = doReq(http.MethodPost, "/v2/memories/query", `{bad`)
	_ = doReq(http.MethodPost, "/v2/memories/query", `{"query":"*","project":"p1"}`)
	_ = doReq(http.MethodPost, "/v2/knowledge/resolve", `{bad`)
	_ = doReq(http.MethodPost, "/v2/reflection/evaluate", `{"answer_draft":"good"}`)
	_ = doReq(http.MethodPost, "/procedural/query", `{"query":"Step 1: Do A\nStep 2: Do B","project":"p1"}`)
	_ = doReq(http.MethodGet, "/v2/mesh/peers", "")
	_ = doReq(http.MethodGet, "/v2/mesh/peers?format=object", "")
	_ = doReq(http.MethodGet, "/v2/mesh/sync?project=p1", "")
	_ = doReq(http.MethodPost, "/v2/mesh/peers/sync", `{"peer_id":"node1-lumina","project":"p1"}`)
	_ = doReq(http.MethodPost, "/v2/mesh/receive", `{bad`)
	_ = doReq(http.MethodPost, "/v2/mesh/receive", `{"token":"bad-token"}`)

	// Consistency repair with LayerGraph memory and orphan graph node
	_ = store.StoreMemory(ctx, &models.MemoryItem{ID: "graph-mem-1", Content: "Graph entity", Layer: models.LayerGraph, Project: "p1"})
	_ = store.AddGraphEdge(ctx, "orphan-node-1", "orphan-node-2", "LINKS")
	_ = doReq(http.MethodPost, "/v2/consistency/verify", `{"auto_repair":true}`)

	// Test remaining Extended, V45, V46, and V47 routes
	routes := []struct {
		method string
		path   string
		body   string
	}{
		// Extended routes
		{http.MethodGet, "/docs", ""},
		{http.MethodPost, "/v2/memories", `{bad`},
		{http.MethodPost, "/v2/memories", `{"content":"secret","layer":"semantic","info_class":"restricted"}`},
		{http.MethodPut, "/v2/compliance/approvals/ap-1/approve", `{"status":"APPROVED","approved_by":"admin"}`},
		{http.MethodGet, "/v2/compliance/approvals/ap-1", ""},
		{http.MethodGet, "/v2/compliance/approvals/ap-missing", ""},
		{http.MethodPost, "/v2/mesh/join", `{"my_peer_id":"p2","my_name":"Peer2","my_public_url":"http://localhost:8001"}`},
		{http.MethodPost, "/v2/mesh/join", `{}`},
		{http.MethodGet, "/v2/mesh/peers/node1-lumina", ""},
		{http.MethodPost, "/v2/mesh/peers/node1-lumina/sync", `{}`},
		{http.MethodPost, "/v2/graph/nodes/batch", `{"nodes":[{"id":"bn1","content":"B1"},{"id":"bn2","content":"B2"}]}`},
		{http.MethodPost, "/v2/graph/nodes/connected", `{"node_id":"bn1"}`},
		{http.MethodGet, "/v2/graph/nodes/bn1", ""},
		{http.MethodPost, "/v2/graph/edges/batch", `{"edges":[{"source_id":"bn1","target_id":"bn2","relation_type":"LINKS"},{"source_id":"bn2","target_id":"bn1","relation_type":"LINKS"}]}`},
		{http.MethodPut, "/v2/graph/edges/e1", `{}`},
		{http.MethodPost, "/v2/graph/traverse", `{"start_id":"bn1","max_hops":2}`},
		{http.MethodPost, "/v2/graph/cycles/detect", `{}`},
		{http.MethodPost, "/v2/graph/snapshots", `{}`},
		{http.MethodGet, "/v2/graph/snapshots/snap-1", ""},
		{http.MethodPost, "/v2/graph/snapshots/snap-1/restore", `{}`},
		{http.MethodGet, "/v2/automation/executions", ""},
		{http.MethodGet, "/v2/automation/workflows/wf-1", ""},
		{http.MethodPost, "/v2/automation/templates/tpl-decay", `{}`},
		{http.MethodGet, "/v2/automation/info", ""},
		{http.MethodGet, "/v2/automation/custom-route", ""},
		{http.MethodPost, "/v2/evaluation/ab-test/evaluate", `{"arm_a":"a","rewards_a":[0.9,0.8],"arm_b":"b","rewards_b":[0.5,0.4]}`},
		{http.MethodGet, "/v2/evaluation/quality/metrics", ""},
		{http.MethodGet, "/v2/evaluation/health", ""},
		{http.MethodGet, "/v2/evaluation/info", ""},
		{http.MethodGet, "/v2/search/analyze/explain?query=how+to+deploy", ""},
		{http.MethodGet, "/v2/search/weights/profiles/research", ""},
		{http.MethodPost, "/v2/search/compare", `{"query":"*"}`},
		{http.MethodPost, "/v2/search/test/weights", `{}`},
		{http.MethodGet, "/v2/search/health", ""},
		{http.MethodGet, "/v2/reflections/graph", ""},
		{http.MethodPost, "/v2/reflections/relationships", `{"source_id":"bn1","target_id":"bn2"}`},
		{http.MethodPost, "/v2/reflections/batch", `{"ids":["bn1"]}`},
		{http.MethodGet, "/v2/reflections/custom", ""},
		{http.MethodGet, "/v2/nodes/tasks_poll?node_id=node1-lumina", ""},
		{http.MethodGet, "/v2/nodes/tasks/t-1", ""},
		{http.MethodPost, "/v2/nodes/tasks/t-1/result", `{"ok":true}`},
		{http.MethodPost, "/v2/nodes/nodes/register", `{"node_id":"n1"}`},
		{http.MethodPost, "/v2/nodes/nodes/heartbeat", `{"node_id":"n1"}`},
		{http.MethodGet, "/v2/dashboard/metrics/timeseries/mrr", ""},
		{http.MethodGet, "/v2/dashboard/visualizations", ""},
		{http.MethodGet, "/v2/dashboard/health", ""},
		{http.MethodGet, "/v2/dashboard/health/simple", ""},
		{http.MethodGet, "/v2/dashboard/activity", ""},
		{http.MethodGet, "/v2/dashboard/info", ""},
		{http.MethodGet, "/v2/dashboard/compliance/metrics", ""},
		{http.MethodGet, "/v2/dashboard/compliance/risks", ""},
		{http.MethodGet, "/v2/dashboard/compliance/audit-trail", ""},
		{http.MethodGet, "/v2/tuning/theories/cheap", ""},
		{http.MethodGet, "/v2/token-savings/metrics/savings/graph", ""},
		{http.MethodGet, "/system/tables", ""},
		{http.MethodPost, "/system/tenants/t1", `{"name":"Tenant 1"}`},
		{http.MethodPost, "/system/projects/p1", `{"env":"prod"}`},
		{http.MethodGet, "/tenant/t1", ""},

		// V45 routes
		{http.MethodGet, "/v2/system/storage/backends", ""},
		{http.MethodPost, "/v2/system/storage/switch", `{"backend":"invalid"}`},
		{http.MethodPost, "/v2/system/storage/migrate", `{"source":"invalid","target":"postgres"}`},
		{http.MethodGet, "/v2/mesh/merkle-tree", ""},
		{http.MethodGet, "/v2/mesh/merkle/buckets", ""},
		{http.MethodPost, "/v2/ingest/document", `{"title":"Doc2","content":"RAEMemory uses Qdrant."}`},
		{http.MethodGet, "/v2/automation/scheduler/status", ""},
		{http.MethodPost, "/v2/integrations/llamaindex/query", `{"query_str":"*"}`},

		// V46 routes
		{http.MethodPost, "/v2/context/ib-select", `{"query":"*","max_tokens":500}`},
		{http.MethodPost, "/v2/graph/temporal/change", `{"source_node":"n1","target_node":"n2","relation":"LINKS","project":"p1"}`},
		{http.MethodGet, "/v2/graph/temporal/reconstruct?project=p1", ""},
		{http.MethodPost, "/v2/graph/temporal/compare", `{"project":"p1","changes":[{"source_node":"n1"},{"source_node":"n2"}]}`},
		{http.MethodGet, "/v2/graph/temporal/patterns?project=p1", ""},
		{http.MethodPost, "/v2/graph/snapshots/restore", `{"project":"p1"}`},
		{http.MethodPost, "/v2/graph/snapshots/restore", `{"snapshot_id":"missing-snap","project":"p1"}`},
		{http.MethodPost, "/v2/graph/entity-resolution", `{}`},
		{http.MethodPost, "/v2/compliance/context-provenance/decision", `{"project":"p1","query":"q","agent_id":"a1","decision_type":"approve","output":"ok"}`},
		{http.MethodGet, "/v2/compliance/context-provenance/lineage?decision_id=d1", ""},
		{http.MethodGet, "/v2/compliance/context-provenance/audit?context_id=c1", ""},
		{http.MethodPost, "/v2/mesh/encrypt-batch", `{"password":"pw1","memories":[{"id":"m1","content":"secret"}]}`},
		{http.MethodPost, "/v2/mesh/decrypt-batch", `{"password":"pw1","memories":[]}`},
		{http.MethodPost, "/v2/search/detect-conflicts", `{"items":[]}`},
		{http.MethodPost, "/v2/search/two-pass", `{"query":"test query","project":"p1","top_k":5}`},

		// V47 routes
		{http.MethodGet, "/v2/mdp/state", ""},
		{http.MethodPost, "/v2/mdp/policy/select", `{}`},
		{http.MethodPost, "/v2/mdp/step", `{"quality_gain":0.2,"tokens_consumed":100,"latency_ms":12}`},
		{http.MethodGet, "/v2/mdp/metrics", ""},
		{http.MethodPost, "/v2/graph/algorithms/pagerank", `{}`},
		{http.MethodPost, "/v2/graph/algorithms/betweenness", `{}`},
		{http.MethodPost, "/v2/graph/algorithms/clustering", `{}`},
		{http.MethodPost, "/v2/graph/algorithms/modularity", `{}`},
		{http.MethodPost, "/v2/graph/algorithms/subgraph", `{"seeds":["bn1"],"max_hops":2,"max_nodes":10}`},
		{http.MethodPost, "/v2/graph/algorithms/topology", `{}`},
		{http.MethodPost, "/v2/consolidation/fsm/evaluate", `{}`},
		{http.MethodPost, "/v2/dreaming/run", `{"min_importance":0.1}`},
		{http.MethodPost, "/v2/communities/summarize", `{}`},
		{http.MethodPost, "/v2/sessions/summarize", `{"session_id":"s1","project":"p1"}`},
		{http.MethodPost, "/v2/governance/rules", `{"name":"R1","event":"on_memory_store","action":"tag"}`},
		{http.MethodGet, "/v2/governance/rules", ""},
		{http.MethodPost, "/v2/governance/rules/evaluate", `{"importance":0.9,"info_class":"internal"}`},
		{http.MethodPost, "/v2/governance/policies/versions", `{"policy_id":"iso-42001-default","max_daily_budget_usd":20,"activate":true}`},
		{http.MethodGet, "/v2/governance/policies/versions?policy_id=iso-42001-default", ""},
		{http.MethodPost, "/v2/governance/policies/rollback", `{"target_version":1}`},
		{http.MethodPost, "/v2/governance/retention/gdpr-erase", `{"tenant_id":"t1","subject_identifier":"user@test.com"}`},
		{http.MethodPost, "/v2/governance/budget/check", `{"tenant_id":"t1","daily_limit_usd":10,"daily_token_limit":10000,"cost_usd":0.5,"tokens":200}`},
		{http.MethodPost, "/v2/math/importance/six-factor", `{"item":{"content":"important memory"},"graph_degree":3}`},
		{http.MethodPost, "/v2/math/cost/calculate", `{"model_id":"z-ai/glm-5.3-flash","prompt_tokens":1000,"completion_tokens":500,"cached_tokens":200}`},
		{http.MethodGet, "/visualizations", ""},
		{http.MethodGet, "/activity", ""},
		{http.MethodGet, "/info", ""},
		{http.MethodGet, "/status", ""},
		{http.MethodGet, "/projects", ""},
		{http.MethodDelete, "/projects/p1", ""},
		{http.MethodPost, "/memory/store", `{}`},
		{http.MethodPost, "/memory/delete?id=m1", `{}`},
		{http.MethodGet, "/statistics", ""},
		{http.MethodPost, "/consolidate", `{}`},
		{http.MethodPost, "/api/upload", `{"content":"uploaded","project":"p1"}`},
		{http.MethodPost, "/api/search", `{"query":"*"}`},
		{http.MethodGet, "/api/stats", ""},
		{http.MethodGet, "/v1/models", ""},
		{http.MethodPost, "/v1/chat/completions", `{}`},
		{http.MethodPost, "/api/chat", `{"model":"qwen3.5:9b"}`},
		{http.MethodGet, "/api/tags", ""},
	}

	for _, rt := range routes {
		_ = doReq(rt.method, rt.path, rt.body)
	}
}
