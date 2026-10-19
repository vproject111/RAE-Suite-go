package memory

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
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
