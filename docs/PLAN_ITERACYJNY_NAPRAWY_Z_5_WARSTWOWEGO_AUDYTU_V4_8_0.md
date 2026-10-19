# Iteracyjny Plan Naprawy Luk Wykrytych przez 5-Warstwowy Audytor (`v4.8.0`)

**Data utworzenia planu**: 2026-10-19  
**Gałąź docelowa**: `feature/v48-five-stage-audit-remediation` $\rightarrow$ `develop` $\rightarrow$ `main` (Tag: `v4.8.0`)  
**Kryterium akceptacji**:
1. `make audit-5stage` (`python3 scripts/reusable_5stage_parity_auditor.py`) musi zakończyć się wynikiem **`5/5 STAGES PASSED`** (`all_5_stages_passed: true`, `0` zaślepek w Warstwie 2, `17/17` sond wrażliwości w Warstwie 3, `6/6` kroków niezmienników E2E w Warstwie 4, `100%` kontraktów AST/DDL/Env w Warstwach 1 i 5).
2. `make verify-parity` (`python3 scripts/verify_python_vs_go_parity.py`) musi utrzymać **`100% PASS`**.
3. `make test` (`go vet ./...`, `go test -v ./...`, `go test -race ./...`) oraz `make build-cross` (`CGO_ENABLED=0`) muszą przejść z zerową liczbą ostrzeżeń.

---

## Iteracja 1: Eliminacja `OpenClawEscalation` i Naprawa Global Search w `RAE-Portal` (`:8082`) oraz Routing w `RAE-Orchestrator` (`:8009`)
- **Zakres plików**:
  - `services/portal/service.go`
  - `services/portal/portal_test.go`
  - `services/orchestrator/service.go`
- **Zadania inżynieryjne**:
  1. W `services/portal/service.go` zastąpić `"OpenClawEscalation"` przez `"HermesOpenCodeEscalation"` w `AllPortalViews()` oraz w szablonie HTML Command Center (`/`), całkowicie eliminując ślad OpenClaw z UI.
  2. W `services/portal/service.go` podpiąć handler `/v2/portal/global-search` do dekodowania zarówno parametrów URL (`?query=...`), jak i ciała POST JSON (`{"query": "...", "project": "..."}`) oraz uruchomienia żywego silnika `ops.ExecuteLiveGlobalSearch(query, project, nil)`, dzięki czemu $f(x_1) \neq f(x_2)$.
  3. W `services/orchestrator/service.go` podpiąć `/v2/fabric/route` do dekodowania ciała JSON (`workload`, `preferred_node`, `tier`, `gpu_required`) i wyznaczenia węzła obliczeniowego przez `s.clusterMgr.RouteTask(...)` (`node1-lumina` dla GPU/ML, `node2-julka` dla heavy benchmark/sandbox, `node3-piotrek` dla 128GB RAM/LLM proxy).
  4. W `services/orchestrator/service.go` podpiąć `/v2/knowledge/resolve` do dynamicznego wyliczania `sufficiency_met`, `confidence` i `resolved_memories` na podstawie zapytania (`req.Query`), progu (`req.MinConfidence`) oraz artefaktów w `s.crlSvc`.

---

## Iteracja 2: Naprawa Wszystkich Zaślepek i Sond Wrażliwości w `RAE-Quality` (`:8010`) oraz `RAE-Phoenix` (`:8012`)
- **Zakres plików**:
  - `services/quality/service.go`
  - `services/quality/deep_engines.go`
  - `services/phoenix/service.go`
- **Zadania inżynieryjne**:
  1. **`/v2/quality/risk-coverage` i `/v2/quality/risk-coverage/verify`**:
     - Rozszerzyć `EvaluateRiskCoverageMatrixWithTests(executedTests []string)` w `services/quality/deep_engines.go`, aby wyliczała faktyczne pokrycie wagowe ścieżek krytycznych (`RC-01..RC-05`) na podstawie przekazanej tablicy `executed_tests` w ciele POST JSON (gdy `executed_tests` jest puste w GET — używa domyślnego zestawu kanonicznego; gdy przekazano konkretną listę testów — dopasowuje ścieżki po słowach kluczowych `calc`/`dtp`/`rbac`/`merkle`/`dto` i wylicza `risk_coverage`, `actual_coverage`, `risk_score`, `passed`).
  2. **`/v2/quality/defect-metrics`**:
     - Dekodować `code` i `findings` z żądania `*http.Request`, uruchamiać `s.AuditCode(req.Code, nil, "")` i wyliczać `defect_density`, `escaped_defects`, `mttr_minutes` oraz `zero_warning_status` proporcjonalnie do liczby wykrytych naruszeń AST/SAST.
  3. **`/v2/quality/predictive-plan`**:
     - Dekodować `files []string` i `code string` z `*http.Request`, analizować złożoność i nazwy modułów domenowych (`memory`, `math`, `security`, `mesh`, `quality`), zwracając dynamiczną listę `predicted_hotspots`, `recommended_focus` oraz `confidence`.
  4. **`/v2/quality/unit-test-inventory` oraz `/v2/quality/check-architecture`**:
     - Odczytywać parametry żądania (`module` / `code` / `imports`) i weryfikować architekturę heksagonalną (zakaz importowania `services/*` wewnątrz `pkg/raecore/*`) oraz statystyki katalogu `BuildCatalog198Summary()` + `s.AuditCode(...)`.
  5. **`RAE-Phoenix` (`/v2/phoenix/create`, `/sessions/analyze`, `/patterns/errors`, `/metrics`)**:
     - W `/v2/phoenix/create` uwzględniać `req.Spec` (analiza AST/sygnatury specyfikacji i generowanie kodu szkieletu oraz skrótu SHA-256).
     - W `/sessions/analyze` analizować przekazane w JSON zdarzenia (`events`, `error_traces`, `session_id`) przy pomocy `AnalyzePostMortem`, wyliczając rzeczywiste `anomalies_found`, `root_cause` i `recommended_patch`.
     - W `/patterns/errors` oraz `/metrics` używać żywych liczników atomowych i reguł `s.ListRecipes()` + `r.URL.Query()`.

---

## Iteracja 3: Eliminacja Zaślepek Compliance/Bridge w `RAE-Memory` (`extended_routes.go`)
- **Zakres plików**:
  - `services/memory/extended_routes.go`
- **Zadania inżynieryjne**:
  1. Podpiąć `/v2/bridge/audit` do weryfikacji przekazanego ładunku przez `s.guard` (`InspectContent` / `ValidateLayerTransition`) oraz `s.provenanceSvc`.
  2. Podpiąć `/v2/compliance/circuit-breakers` do odczytu żywego stanu wyłączników z `s.guard`, `s.broker` oraz opcjonalnego parametru `?breaker=...`.
  3. Podpiąć `/v2/dashboard/compliance/report`, `/v2/dashboard/compliance/metrics`, `/v2/dashboard/compliance/risks` oraz `/v2/dashboard/compliance/audit-trail` do żywych danych z `s.store.GetMemoryStats(r.Context(), project)`, `s.provenanceSvc`, `ext.approvals` i kryptograficznego skrótu stanu pamięci `s.mesh.ComputeMerkleRoot`.

---

## Iteracja 4: Naprawa Łańcucha Niezmienników Stanowych E2E (Warstwa 4)
- **Zakres plików**:
  - `services/memory/v45_extensions.go`
  - `services/memory/extended_routes.go`
  - `services/memory/service.go`
  - `pkg/drivers/memorystore/store.go`
  - `pkg/drivers/sqlstore/persistent.go`
- **Zadania inżynieryjne**:
  1. **Ogniwo 1 (`IngestDocument -> KnowledgeGraph Triples`)**:
     - W `s.IngestDocument` zapisywać wyekstrahowane trójki zarówno do `s.store.AddGraphEdge`, jak i do `s.temporalGraph.RecordChange(...)`.
     - Dodać alias HTTP `/v2/ingest/document` obok `/v2/memories/ingest-document`.
     - Dodać endpoint `/v2/graph/triples` zwracający pełną listę trójek `(subject, relation, object)` z `s.store.GetGraphAdjacency` oraz `s.temporalGraph`.
  2. **Ogniwo 3 (`16-Bucket Merkle Tree root_hash mutation`)**:
     - Dodać alias HTTP `/v2/mesh/merkle/buckets` obok `/v2/mesh/merkle-tree` i zwracać na najwyższym poziomie obiektu JSON klucze `"root_hash": tree.MerkleRoot`, `"merkle_root": tree.MerkleRoot` oraz `"bucket_hashes": tree.BucketHashes`.
  3. **Ogniwo 6 (`Non-null JSON array contract on empty search`)**:
     - W `pkg/drivers/memorystore/store.go`, `pkg/drivers/sqlstore/persistent.go` oraz w handlerach `/v2/memories/` i `/v2/memories` inicjalizować pusty wynik jako `make([]models.ScoredMemoryItem, 0)`, tak aby `encoding/json` zawsze serializował `"results": []` (nigdy `"results": null`).

---

## Iteracja 5: Naprawa Kontraktu Schematu SQL DDL i Zmiennych Środowiskowych (Warstwa 5) oraz Walidacja Końcowa
- **Zakres plików**:
  - `pkg/drivers/agnostic/broker.go`
  - `pkg/drivers/llm/client.go`
  - `pkg/drivers/postgres/adapter.go`
  - `services/memory/v45_extensions.go`
- **Zadania inżynieryjne**:
  1. Dodać w `pkg/drivers/postgres/adapter.go` funkcję `GenerateCanonicalDDL() string` generującą kompletny schemat SQL DDL wszystkich 12 kanonicznych tabel (w tym `CREATE TABLE IF NOT EXISTS memories` oraz `CREATE TABLE IF NOT EXISTS governance_outbox`).
  2. W `services/memory/v45_extensions.go` zamontować `/v2/db/schema/sql` (oraz wzbogacić `/v2/system/storage/export-sql`), zwracając klucze `"sql_ddl"`, `"ddl"` oraz `"sql"`.
  3. W `pkg/drivers/agnostic/broker.go` obsłużyć zmienne środowiskowe `RAE_DB_MODE` (`migrate` / `validate` / `ignore`) oraz `POSTGRES_URL` (obok `RAE_POSTGRES_DSN`).
  4. W `pkg/drivers/llm/client.go` obsłużyć klucze dostawców `ANTHROPIC_API_KEY`, `OPENAI_API_KEY` oraz `GEMINI_API_KEY` (obok `OPENROUTER_API_KEY` i `LITELLM_API_KEY`).
  5. Uruchomić `make test`, `make build`, `make build-cross`, `make audit-5stage` (`5/5 PASS`) i `make verify-parity` (`100% PASS`), zapisać raport wykonania i wdrożyć przez Git Flow (`v4.8.0`).
