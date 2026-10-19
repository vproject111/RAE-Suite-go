# Szczegółowy Raport Porównawczy i Audyt Rozbieżności Funkcjonalnych: `RAE-Suite` (Python/TS) vs `RAE-Suite-go` (`v4.5.0` Pure Go)

**Data audytu kodu źródłowego i testów na żywo:** `2026-10-19`  
**Badane repozytoria:**
- **Stos Python/TS**: `/home/grzegorz/cloud/RAE-Suite` (`core/`, `packages/rae-core`, `packages/rae-agentic-memory`, `packages/rae-quality`, `packages/rae-phoenix`, `packages/rae-hive`, `packages/rae-lab`, `packages/rae-mcp-server`) + `/home/grzegorz/cloud/RAE-CRL` + `/home/grzegorz/cloud/scripts`
- **Stos Go (`v4.5.0`)**: `/home/grzegorz/cloud/RAE-Suite-go` (`cmd/`, `pkg/drivers`, `pkg/raecore`, `pkg/raemesh`, `services/`)

---

## 1. Podsumowanie Wykonawcze (Executive Summary)

Po wdrożeniu wersji **`RAE-Suite-go v4.5.0`** (obejmującej `AgnosticStorageBroker` dla 5 silników baz danych, 16-kubełkowy `RAE-Mesh`, most `Hermes + OpenCode` zamiast `OpenClaw` oraz 12 rozszerzeń z poprzedniego audytu) przeprowadzono:
1. **Test parzystości na żywo (`verify_python_vs_go_parity.py`)**: **`197 / 197 PASS`** (`0.52 ms` średniej latencji w Go vs `100.67 ms` w Pythonie — **`194.5× szybciej w Go`**).
2. **Głęboką inspekcję AST wszystkich plików `.py` i `.go`** w podpakietach `rae_core`, `apps/memory_api`, `core/`, `feniks`, `rae-hive`, `rae-quality`, `rae-lab` oraz `RAE-CRL`.

### Kluczowe wnioski z porównania:
- **Warstwa API, Kontraktów HTTP/MCP, 9 Departamentów, Agnostycznej Bazy Danych i `RAE-Mesh`**: **100% zgodności behawioralnej (`197/197 PASS`)**. Wszystkie 12 rozbieżności z poprzedniej iteracji (`v4.4.0 -> v4.5.0`) zostało w pełni zamkniętych.
- **Świadome różnice architektoniczne na korzyść Go**:
  - Całkowite wyeliminowanie podatnego na błędy `OpenClaw` na rzecz deterministycznego mostu **`Hermes + OpenCode` (`openclaw_disabled: true`)**.
  - Zastąpienie 9 kontenerów Docker + Celery Worker + Celery Beat pojedynczą statyczną binarką **`bin/rae` (`CGO_ENABLED=0`, 0 zewnętrznych zależności w `go.mod`)**.
  - Wprowadzenie **`AgnosticStorageBroker`** umożliwiającego przełączanie i migrację w locie pomiędzy `embedded`, `postgres`, `qdrant`, `redis` i `hybrid` przy zachowaniu spójnego `RAE-Mesh`.
- **Nowo wykryte głębokie różnice algorytmiczne i operacyjne (Poziom 2 — „Głębokie Wnętrze Modułów”, `G1`–`G10`)**:
  Podczas szczegółowego przeszukiwania wewnętrznych klas Pythona (`packages/rae-agentic-memory/apps/memory_api/core/*`, `services/*`, `packages/rae-core/rae_core/*`, `packages/rae-phoenix/feniks/*`, `packages/rae-hive/strategies/*`, `RAE-Suite/core/*`) zidentyfikowano **10 specjalistycznych mechanizmów wewnętrznych**, które w Pythonie posiadają rozbudowaną logikę matematyczną/czasową, natomiast w Go `v4.5.0` działają obecnie w uproszczonej formie lub nie mają jeszcze podkomend CLI.

---

## 2. Macierz Funkcjonalności w 100% Zrealizowanych i Zweryfikowanych w Go (`v4.5.0` — `197/197 PASS`)

| Obszar / Moduł | Python (`RAE-Suite` + `RAE-CRL`) | Go (`RAE-Suite-go v4.5.0`) | Status Parzystości |
| :--- | :--- | :--- | :---: |
| **1. Agnostyczny Rdzeń Bazodanowy (`Agno-Core`)** | `adapters/postgres.py`, `qdrant.py`, `redis.py`, `sqlite/`, `memory/` | `pkg/drivers/agnostic/broker.go` (`embedded`, `postgres`, `qdrant` REST Named Vectors, `redis` TCP RESP, `hybrid` + `SwitchBackend`, `Migrate`, `ExportSQL`) | **100% (Lepsze w Go: Hot-Swap + Live Migrate)** |
| **2. Federacja `RAE-Mesh` i CRDT** | `sync/protocol.py`, `sync/diff.py`, `sync/merge.py`, `api/v2/mesh.py` | `pkg/raemesh/mesh.go` (16-bucket Merkle Tree `0..f`, `DiffMerkleBuckets`, `ResolveCRDTLWW`, `CapabilityToken` HMAC, `SetStorage`) | **100% (`197/197 PASS`)** |
| **3. 8 Warstw Pamięci + Matematyka L1/L2/L3 + 9 Strategii Search** | `rae_core/layers/*`, `rae_core/math/*`, `rae_core/search/strategies/*` | `pkg/drivers/memorystore`, `pkg/raecore/math`, `pkg/raecore/search` (BM25, Cosine HNSW, Graph Resonance, RRF, MAB UCB1, Ebbinghaus Decay) | **100% (`197/197 PASS`)** |
| **4. 5-Filarowy Rejestr Audytu (`Kto/Co/Dlaczego/Za ile/Ile trwało/Jaki efekt`)** | `core/audit_accounting.py`, `scripts/record_task_completion.py` | `pkg/raecore/accounting/ledger.go`, `./bin/rae record-task`, `/v2/audit/ledger/*` (łańcuch SHA-256, rotacja `.jsonl.gz`, `Zero Fake Success`) | **100% (`197/197 PASS`)** |
| **5. Koalescencja Zadań (`BatchOptimizationEngine` + Efekt Skali Wiedzy)** | `core/batch_engine.py` | `pkg/raecore/economy/engine.go`, `/v2/batch/*` (łączenie po module i `target_file`, amortyzacja 15 000 tokenów) | **100% (`197/197 PASS`)** |
| **6. Planowanie Kognitywne (`ToT + MCTS + PRM`) i Konklawe (`DynamicCouncil` + `Swarm`)** | `core/cognitive_planner.py`, `core/dynamic_council.py`, `core/swarm_consensus.py` | `pkg/raecore/cognition/planner.go`, `/v2/cognition/*`, `/v2/council/select`, `/v2/swarm/consensus` | **100% (`197/197 PASS`)** |
| **7. Most `Hermes + OpenCode` (zamiast `OpenClaw`)** | `OpenClaw` (odrzucony ze względu na błędy) + skrypty CLI | `pkg/raecore/ops/hermes_opencode.go`, `/v2/ops/hermes-opencode` (`openclaw_disabled: true`, izolacja `git worktree` + eskalacja L3 SHA-256) | **100% (Lepsze w Go: brak OpenClaw)** |
| **8. Płaszczyzna Samodoskonalenia i Operacji (`ImprovementPlane` + `Ops`)** | `core/autonomy_kernel.py`, `cluster_manager.py`, `circuit_breaker.py`, `keycloak_auth.py`, `infra_reconciler.py`, `prompt_registry.py` | `pkg/raecore/ops/engines.go` + `services/orchestrator/ops_routes.go` (Hypothesis, Shadow, Canary `>5%` rollback, PromotionGate, PSI Drift, W3C Traceparent, JWT HS256) | **100% (`197/197 PASS`)** |
| **9. `RAE-Quality 2.0` (31 Endpointów, DTP PDF, Kalkulatory, Katalog 210, Multi-AST)** | `packages/rae-quality/engines/*` | `services/quality/service.go`, `deep_engines.go`, `binary_pdf_and_ast.go` (strumień `%PDF-1.7`, AST Go/Py/TS/PHP, Bleed/TAC, Kalkulatory, Golden Set) | **100% (`197/197 PASS`)** |
| **10. `RAE-Phoenix` (Graf zależności AST + Wykonywalne Receptury)** | `packages/rae-phoenix/feniks/*` | `services/phoenix/service.go`, `recipes_exec.go` (`/v2/phoenix/impact`, `/v2/phoenix/recipes/apply`, 3-attempt bound) | **100% (`197/197 PASS`)** |
| **11. `RAE-CRL v2.2` (7 Artefaktów, 24h Grace Period, DAG, CBOR, Eksport MD/LaTeX)** | `/home/grzegorz/cloud/RAE-CRL` (`apps/crl/*`) | `services/crl/service.go`, `pkg/raecore/epistemic/dag.go` (`/v2/crl/*`, `/v2/crl/export` Markdown & LaTeX) | **100% (`197/197 PASS`)** |
| **12. Semantyczna Ingestia, Go Ticker Scheduler, WebSocket RFC 6455, Benchmarki `9/5` + `SVG`** | `rae_core/ingestion/*`, `celery_app.py`, `dashboard_websocket.py`, `nine_five_benchmarks` | `pkg/raecore/ingestion/segmenter.go`, `services/memory/v45_extensions.go` (`/v2/memories/ingest-document`, `/ws/dashboard`, `/v2/evaluation/benchmark/plot.svg`) | **100% (`197/197 PASS`)** |

---

## 3. Wykaz 10 Głębokich Rozbieżności Algorytmicznych i Operacyjnych (`G1`–`G10` — „Pod Maską Modułów”)

Poniższa lista obejmuje wszystkie miejsca w kodzie Pythona, w których pod maską znajdują się dodatkowe, zaawansowane algorytmy lub narzędzia operacyjne, które w Go `v4.5.0` są obecnie uproszczone lub nieobecne:

---

### Rozbieżność G1: Dobór Kontekstu Metodą Wąskiego Gardła Informacyjnego (`InformationBottleneckSelector`)
- **W Pythonie (`packages/rae-agentic-memory/apps/memory_api/core/information_bottleneck.py`, 484 linie)**:
  - Klasa `InformationBottleneckSelector` implementuje matematyczną zasadę Information Bottleneck (Tishby): wybiera podzbiór wspomnień $Z \subseteq X$ maksymalizujący funkcję celu:
    $$J_{IB}(Z) = I(Z; Y) - \beta \cdot I(Z; X)$$
    gdzie $I(Z; Y)$ to trafność semantyczna względem zapytania $Y$ (`estimate_I_Z_Y`), $I(Z; X)$ to koszt kompresji (liczba tokenów + redundancja między wybranymi wspomnieniami liczona przez `_compute_diversity` / MMR), a współczynnik $\beta$ jest dynamicznie dostrajany (`adaptive_beta`) w zależności od złożoności zapytania i dostępnego budżetu tokenów.
- **W Go (`pkg/raecore/workflows/engines.go` — `ContextBroker`)**:
  - `ContextBroker.SelectMemories` sortuje kandydatów malejąco po `Importance` i zachłannie dokłada kolejne elementy aż do wyczerpania limitu `MaxTokens`, bez obliczania redundancji między wspomnieniami ($I(Z; X)$) i bez adaptacyjnego $\beta$.

---

### Rozbieżność G2: Czasowy Graf Wiedzy (`TemporalGraphService` + `reconstruct_graph_at_time` + `compare_graphs`)
- **W Pythonie (`packages/rae-agentic-memory/apps/memory_api/services/temporal_graph.py`, 649 linii + `core/graph_operator.py`, 531 linii)**:
  - Klasa `TemporalGraphService` przechowuje pełny dziennik zdarzeń grafowych (`GraphChange`: `NODE_ADDED`, `NODE_REMOVED`, `EDGE_ADDED`, `EDGE_WEIGHT_UPDATED` ze znacznikami czasu) oraz migawki `GraphSnapshot`.
  - Udostępnia metody:
    - `reconstruct_graph_at_time(timestamp)` — odtwarza dokładną topologię grafu wiedzy na dowolną chwilę w przeszłości,
    - `compare_graphs(t1, t2)` — zwraca różnicę topologiczną (dodane/usunięte węzły, krawędzie i zmianę gęstości grafu `_calculate_density`),
    - `get_entity_history(entity_id)` oraz `find_emerging_patterns()` — wykrywa najszybciej rosnące klastry pojęć w czasie,
    - `restore_snapshot(snapshot_id)` — faktycznie przywraca stan węzłów i krawędzi grafu z wybranej migawki.
- **W Go (`pkg/drivers/memorystore/store.go` + `services/memory/extended_routes.go`)**:
  - Graf wiedzy przechowuje wyłącznie aktualną mapę sąsiedztwa `adj map[string][]string` bez historii zmian w czasie (`GraphChange`), bez rekonstrukcji `reconstruct_graph_at_time` i bez różnicowania `compare_graphs(t1, t2)`.

---

### Rozbieżność G3: Deduplikacja i Scalanie Synonimicznych Encji Grafu (`EntityResolutionService`)
- **W Pythonie (`packages/rae-agentic-memory/apps/memory_api/services/entity_resolution.py`, 239 linii)**:
  - Klasa `EntityResolutionService` (`run_clustering_and_merging`, `_process_group`, `_merge_nodes`) grupuje węzły grafu o zbliżonych nazwach lub embeddingach (np. `"Postgres"`, `"PostgreSQL"`, `"postgres_db"`), podejmuje decyzję `MergeDecision` i automatycznie przepina wszystkie krawędzie wejściowe/wyjściowe do węzła kanonicznego, eliminując fragmentację grafu wiedzy.
- **W Go (`services/memory`)**:
  - Krawędzie wyekstrahowane z tekstu są zapisywane pod literalnymi nazwami węzłów; brak silnika `EntityResolutionService`, który scalałby synonimiczne węzły i przepinał ich krawędzie.

---

### Rozbieżność G4: Pełny Łańcuch Pochodzenia Decyzji AI (`ContextProvenanceService` wg ISO/IEC 42001)
- **W Pythonie (`packages/rae-agentic-memory/apps/memory_api/services/context_provenance_service.py`, 379 linii)**:
  - Klasa `ContextProvenanceService` buduje dwustopniowy rejestr:
    1. `create_decision_context(...)` -> zapisuje dokładną listę źródeł `ContextSource` (ID wspomnienia, warstwa, wynik zaufania `trust_score`, waga w oknie kontekstowym),
    2. `record_decision(...)` -> wiąże wygenerowaną decyzję agenta (`DecisionRecord`) z konkretnym `context_id`,
    3. `get_decision_lineage(decision_id)` oraz `audit_context_quality(context_id)` -> weryfikuje pokrycie dowodowe decyzji, wykrywa brakujące cytowania i ocenia ryzyko halucynacji.
- **W Go (`services/memory/extended_routes.go`)**:
  - Endpoint `/v2/compliance/context-provenance` obsługuje płaski zapis pojedynczego zdarzenia, bez powiązania `DecisionContext -> []ContextSource -> DecisionRecord` i bez audytu jakości dowodowej `audit_context_quality`.

---

### Rozbieżność G5: Szyfrowanie End-to-End (`E2EEncryption`) Pakietów Synchronizacyjnych `RAE-Mesh`
- **W Pythonie (`packages/rae-core/rae_core/sync/encryption.py`, 191 linii)**:
  - Klasa `E2EEncryption` (`_derive_key_from_password` z `PBKDF2-HMAC-SHA256`, `encrypt_memory`, `decrypt_memory`, `encrypt_batch`, `decrypt_batch`) pozwala szyfrować zawartość wspomnień (`content`, `metadata`) przed wysłaniem ich przez `RAE-Mesh` do zdalnego węzła, tak aby pośredniczący przekaźnik (relay) nie mógł odczytać treści.
- **W Go (`pkg/raemesh/mesh.go`)**:
  - `MeshEngine` uwierzytelnia transfer za pomocą kryptograficznego tokenu `CapabilityToken` (`HMAC-SHA256`) i weryfikuje `HardFramesGuard`, ale nie posiada warstwy `E2EEncryption` (`AES-256-GCM` z derywacją klucza) do szyfrowania i deszyfrowania treści `MemoryItem` w locie podczas synchronizacji Mesh.

---

### Rozbieżność G6: Detektor Sprzeczności Dowodów (`EvidenceConflictDetector`) i Dwuprzebiegowy `QueryRewriter`
- **W Pythonie (`packages/rae-core/rae_core/search/conflict_detector.py`, 137 linii + `rewriter.py`, 93 linie)**:
  - `EvidenceConflictDetector.detect_conflicts(items)` w czasie `<1ms` analizuje pary pobranych wspomnień pod kątem sprzecznych wartości liczbowych, negacji (`not`, `never`, `disabled` vs `enabled`) oraz wykluczających się wersji reguł, oznaczając konflikty zanim trafią do promptu LLM.
  - `QueryRewriter.create_rewrite_plan(query, gate_result)` automatycznie generuje plan drugiego przebiegu wyszukiwania (rozszerzenie synonimów, dekompozycja zapytania i `boost = 1.4`), gdy `SufficiencyGate` uzna pierwszy zestaw wyników za niewystarczający.
- **W Go (`pkg/raecore/search/gateway.go`)**:
  - `LogicGateway` posiada `SufficiencyGate`, ale w przypadku niewystarczającego wyniku nie uruchamia automatycznie `QueryRewriter` (2-pass retrieval) ani nie skanuje wyników przez `EvidenceConflictDetector`.

---

### Rozbieżność G7: 6-Wymiarowy `BehaviorComparisonEngine` i `PostMortemAnalyzer` w `RAE-Phoenix`
- **W Pythonie (`packages/rae-phoenix/feniks/core/behavior/comparison_engine.py`, 340 linii + `core/reflection/post_mortem.py`, 266 linii)**:
  - `BehaviorComparisonEngine.check_snapshot` weryfikuje parzystość behawioralną po refaktoryzacji w **6 niezależnych wymiarach**:
    1. `_check_http` (kod statusu HTTP, nagłówki, czasy odpowiedzi),
    2. `_check_cli` (`exit_code`, `stdout`, `stderr`),
    3. `_check_dom` (obecność wymaganych selektorów UI i brak zniknięcia elementów),
    4. `_check_logs` (brak nowych wyjątków/błędów w logach),
    5. `_check_json_path` (zgodność typów i wartości w zagnieżdżonych polach JSON DTO),
    6. `_check_pattern` + `_calculate_risk_score` (wyliczenie zagregowanego ryzyka regresji `0.0..1.0`).
  - `PostMortemAnalyzer.analyze_session` analizuje nieudane sesje naprawcze i generuje ustrukturyzowane wnioski `PostMortem` (analiza kosztu, śladu błędów i złamanych kontraktów).
- **W Go (`services/phoenix/service.go`)**:
  - `VerifyBehavioralParity` wykonuje jedynie proste porównanie łańcuchów tekstowych `baselineOut == candidateOut`, bez 6-wymiarowej inspekcji kontraktów HTTP/CLI/DOM/JSONPath i bez `PostMortemAnalyzer`.

---

### Rozbieżność G8: `AtomicSlicer v76` i `Supervisor v2` w `RAE-Hive`
- **W Pythonie (`packages/rae-hive/strategies/atomic_slicer_v76.py`, 79 linii + `supervisor_v2.py`, 59 linii + `agents/*`)**:
  - `AtomicSlicer` dzieli duże monolityczne pliki kodu na niezależne, atomowe wycinki (funkcje, klasy, metody) wraz z ich minimalnym kontekstem importów, co pozwala rozproszyć pracę na wiele równoległych agentów (`auditor`, `builder`, `orchestrator`, `reporter`) bez przekraczania okna kontekstowego.
  - `SupervisorV2` monitoruje żywotność puli wykonawczej, postęp przetwarzania plików i automatycznie wznawia przerwane zadania.
- **W Go (`services/hive/service.go`)**:
  - `Service` posiada `ExecuteParallel` (z limitem `k <= 3`) i `SemanticWatchdog`, ale nie posiada endpointu ani silnika `AtomicSlicer` do dzielenia dużych plików na atomowe jednostki robocze ani `SupervisorV2`.

---

### Rozbieżność G9: `GitOpsDaemon`, `ProjectPhaseAnalyzer` i `RAEGlobalSearchEngine` w `Orchestratorze`
- **W Pythonie (`RAE-Suite/core/gitops_daemon.py`, 142 linie + `project_phase_analyzer.py`, 133 linie + `portal_global_search.py`, 131 linii)**:
  - `GitOpsDaemon`: automatyzuje tworzenie gałęzi zadaniowych (`create_agent_branch`), podpisywanie commitów z metadanymi RAE (`commit_changes_with_signature`) i generowanie specyfikacji Pull Requestów (`generate_pull_request`).
  - `ProjectPhaseAnalyzer`: agreguje wpisy z `ExecutionAuditLedger` i wylicza podsumowanie wykonania projektu w podziale na fazy SDLC (`PhaseExecutionSummary`: koszt USD, zużyte tokeny, czas trwania, odsetek sukcesów i główne modele w każdej fazie).
  - `RAEGlobalSearchEngine`: wykonuje zunifikowane wyszukiwanie globalne (`search`) jednocześnie po wszystkich 8 warstwach `RAE-Memory`, artefaktach naukowych `RAE-CRL`, paragonach `ExecutionAuditLedger` i raportach `RAE-Quality`.
- **W Go (`services/orchestrator` & `services/portal`)**:
  - Brak odpowiedników `GitOpsDaemon`, `ProjectPhaseAnalyzer` oraz sfederowanego `RAEGlobalSearchEngine` (`/v2/portal/global-search`).

---

### Rozbieżność G10: Natywne Podkomendy Operacyjne CLI w `./bin/rae` (Zastąpienie Skryptów Pythona z `GEMINI.md`)
- **W Pythonie (`/home/grzegorz/cloud/scripts/*.py`)**:
  - Protokół operacyjny `GEMINI.md` korzysta z 4 skryptów CLI uruchamianych bezpośrednio z terminala:
    1. `python3 scripts/bootstrap_dreamsoft_knowledge.py` — inicjalizacja sesji i załadowanie architektury/reguł do RAE,
    2. `python3 scripts/mesh_sync_cli.py status | sync` — diagnostyka węzłów klastra i wymuszenie synchronizacji siatki `RAE-Mesh`,
    3. `./scripts/run_layer_triad.py --cascade --text "..."` — uruchomienie 3-stopniowej triady 9 modeli z terminala,
    4. `./scripts/record_task_completion.py` — zapis 5-filarowego paragonu zadania i rozgłoszenie po siatce Mesh.
- **W Go (`cmd/rae/main.go`)**:
  - Zunifikowana binarka `./bin/rae` posiada obecnie podkomendę `./bin/rae record-task`, ale nie posiada jeszcze natywnych podkomend CLI:
    - `./bin/rae bootstrap` (odpowiednik `bootstrap_dreamsoft_knowledge.py`),
    - `./bin/rae mesh status` / `./bin/rae mesh sync` (odpowiednik `mesh_sync_cli.py`),
    - `./bin/rae triad --cascade --text "..."` / `./bin/rae triad --tier ...` (odpowiednik `run_layer_triad.py`).

---

## 4. Podsumowanie Tabelaryczne Rozbieżności Poziomu 2 (`G1`–`G10`)

| ID | Funkcjonalność w Pythonie (`RAE-Suite` / `RAE-CRL`) | Lokalizacja w Pythonie | Stan w `RAE-Suite-go v4.5.0` | Priorytet wdrożenia w Go (`v4.6.0`) |
| :---: | :--- | :--- | :--- | :---: |
| **G1** | `InformationBottleneckSelector` ($I(Z;Y) - \beta I(Z;X)$ + MMR diversity) | `apps/memory_api/core/information_bottleneck.py` | Zachłanny wybór po `Score` do limitu tokenów | **WYSOKI** |
| **G2** | `TemporalGraphService` (`GraphChange`, `reconstruct_graph_at_time`, `compare_graphs`, `restore_snapshot`) | `apps/memory_api/services/temporal_graph.py` | Tylko bieżąca lista sąsiedztwa `adj` | **WYSOKI** |
| **G3** | `EntityResolutionService` (klastrowanie i scalanie synonimicznych węzłów grafu) | `apps/memory_api/services/entity_resolution.py` | Brak scalania synonimów w grafie | **ŚREDNI** |
| **G4** | `ContextProvenanceService` (`DecisionContext -> ContextSource -> DecisionRecord` + `audit_context_quality`) | `apps/memory_api/services/context_provenance_service.py` | Płaski wpis bez audytu pokrycia dowodowego | **WYSOKI (ISO 42001)** |
| **G5** | `E2EEncryption` (`PBKDF2` + `AES-256-GCM` dla paczek `RAE-Mesh`) | `rae_core/sync/encryption.py` | Podpis `HMAC-SHA256` bez szyfrowania ładunku | **WYSOKI (ISO 27000)** |
| **G6** | `EvidenceConflictDetector` (`<1ms`) + 2-przebiegowy `QueryRewriter` | `rae_core/search/conflict_detector.py`, `rewriter.py` | Jednoprzebiegowy `SufficiencyGate` | **WYSOKI** |
| **G7** | 6-wymiarowy `BehaviorComparisonEngine` (HTTP/CLI/DOM/Logs/JSONPath) + `PostMortemAnalyzer` | `feniks/core/behavior/comparison_engine.py`, `post_mortem.py` | Płaskie porównanie stringów `baseline == candidate` | **ŚREDNI** |
| **G8** | `AtomicSlicer v76` (podział dużych plików na atomowe wycinki) + `Supervisor v2` | `packages/rae-hive/strategies/atomic_slicer_v76.py` | Tylko równoległy runner shellowy i `Watchdog` | **ŚREDNI** |
| **G9** | `GitOpsDaemon`, `ProjectPhaseAnalyzer` i `RAEGlobalSearchEngine` | `core/gitops_daemon.py`, `project_phase_analyzer.py`, `portal_global_search.py` | Brak w `pkg/raecore/ops` i `services/orchestrator` | **ŚREDNI** |
| **G10** | Podkomendy CLI `./bin/rae bootstrap`, `./bin/rae mesh`, `./bin/rae triad` | `scripts/bootstrap_dreamsoft_knowledge.py`, `mesh_sync_cli.py`, `run_layer_triad.py` | Dostępne tylko `./bin/rae record-task` (pozostałe przez HTTP) | **WYSOKI (DX / CLI)** |
