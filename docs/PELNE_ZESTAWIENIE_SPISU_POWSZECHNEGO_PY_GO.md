# Pełne Zestawienie Spisu Powszechnego AST: Python (`RAE-Suite`) vs Go (`RAE-Suite-go` v4.8.0)

**Data wygenerowania:** 2026-10-19  
**Narzędzie audytowe:** [`scripts/exhaustive_python_to_go_census.py`](file:///home/grzegorz/cloud/RAE-Suite-go/scripts/exhaustive_python_to_go_census.py)  
**Dane surowe JSON:** [`docs/SPIS_POWSZECHNY_PY_GO.json`](file:///home/grzegorz/cloud/RAE-Suite-go/docs/SPIS_POWSZECHNY_PY_GO.json)  
**Baza Python:** `/home/grzegorz/cloud/RAE-Suite`  
**Baza Go:** `/home/grzegorz/cloud/RAE-Suite-go` (`v4.8.0`)

---

## 1. Podsumowanie Wykonawcze (Twarde Liczby)

Po przeskanowaniu **100% plików źródłowych `.py`** (bez żadnej ręcznej białej listy, z wyłączeniem katalogów `.venv`, migracji `alembic`, testów i kopii tymczasowych) otrzymaliśmy pełny obraz skali obu repozytoriów:

| Metryka | Python (`RAE-Suite`) | Go (`RAE-Suite-go` v4.8.0) | Status / Pokrycie |
| :--- | :---: | :---: | :--- |
| **Pliki produkcyjne ogółem (z OpenClaw)** | `1 175` plików (`279 341` LOC) | `69` plików (`19 462` LOC) | — |
| **Pliki OpenClaw (celowo odrzucone)** | `251` plików (`92 433` LOC) | `0` plików | `INTENTIONALLY_EXCLUDED` (decyzja architektoniczna: Hermes + OpenCode) |
| **Pliki produkcyjne netto (bez OpenClaw)** | **`924` plików (`186 908` LOC)** | **`69` plików (`19 462` LOC)** | Silna konsolidacja w Go (~9.6x mniej kodu) |
| **Trasy HTTP REST / FastAPI** | `316` tras | `316` tras (`332` handlery) | **100.0% (`PORTED_FULL` na warstwie API HTTP)** |
| **Klasy DTO / Pydantic / Enum / Wyjątki** | `790` klas | `~115` struktur DTO / `map[string]any` | Skonsolidowane do lekkich struktur Go i dynamicznych payloadów |
| **Klasy Domenowe (Silniki, Serwisy, Adaptery, Workery)** | **`799` klas** | **`106` struktur silnikowych + `641` funkcji** | **`180` dokładnych/semantycznych, `91` częściowych, `528` bez bezpośredniego odpowiednika 1:1** |
| **Narzędzia MCP (`tools/list`)** | `24` unikalne nazwy narzędzi (w 3 serwerach MCP) | `16` narzędzi w [`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go) | **Brakuje 14 aliasów/narzędzi MCP z Pythona** (np. `rae_kernel_execute`, `rae_knowledge_resolve`, `rae_lab_insights`, `rae_quality_rpc`, `get_circuit_breakers`) |

---

## 2. Zestawienie Podsystem po Podsystemie (Wszystkie 924 pliki `.py` bez OpenClaw)

Poniższa tabela prezentuje każdy podsystem z `RAE-Suite` wraz z podziałem klas domenowych na:
- **`PORTED_FULL / SEMANTIC`** — klasy posiadające bezpośredni odpowiednik struktury/silnika w Go,
- **`PORTED_SIMPLIFIED`** — klasy posiadające odpowiednik w Go (lub zintegrowane w handlerze), ale o uproszczonej logice/mniejszej liczbie metod,
- **`MISSING / CONSOLIDATED`** — klasy nieposiadające odpowiednika 1:1 w Go (pominięte, zastąpione lżejszym mechanizmem lub będące narzędziami pomocniczymi/GUI).

| # | Podsystem w Pythonie (`RAE-Suite`) | Pliki `.py` | LOC (Py) | Klasy DTO | Klasy Domenowe | `PORTED_FULL` / Semantyczne | `PORTED_SIMPLIFIED` | `MISSING` / Skonsolidowane | Główne miejsce w Go (`RAE-Suite-go`) |
| :-: | :--- | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :--- |
| 1 | `packages/rae-agentic-memory/apps/memory_api` | 182 | 60 453 | 354 | **191** | 59 | 26 | **106** | [`services/memory/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/service.go), [`pkg/raecore/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore) |
| 2 | `packages/rae-agentic-memory/rae-core` | 172 | 27 305 | 153 | **190** | 24 | 28 | **138** | [`pkg/raecore/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore), [`pkg/drivers/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers) |
| 3 | `packages/rae-phoenix` (`feniks`) | 108 | 25 140 | 69 | **124** | 15 | 10 | **99** | [`services/phoenix/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/service.go) |
| 4 | `packages/rae-agentic-memory/scripts` | 100 | 11 142 | 0 | **15** | 1 | 0 | **14** | [`cmd/rae/main.go`](file:///home/grzegorz/cloud/RAE-Suite-go/cmd/rae/main.go) (tylko główne subkomendy) |
| 5 | `packages/rae-agentic-memory/orchestrator` | 29 | 8 317 | 20 | **42** | 2 | 3 | **37** | [`services/orchestrator/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/orchestrator/service.go), [`pkg/raecore/cognition/planner.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/cognition/planner.go) |
| 6 | `core/` (Kernel, Trybunał, Kontrakty, Swarm) | 70 | 8 277 | 62 | **76** | 44 | 6 | **26** | [`pkg/raecore/ops/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/ops/engines.go), [`pkg/raecore/workflows/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/workflows/engines.go) |
| 7 | `packages/rae-quality` | 31 | 6 310 | 34 | **28** | 11 | 1 | **16** | [`services/quality/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/quality/service.go) |
| 8 | `packages/rae-agentic-memory/rae-portal` | 17 | 5 153 | 0 | **16** | 1 | 0 | **15** | [`services/portal/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/service.go) |
| 9 | `packages/rae-agentic-memory/tools` (`memory-dashboard`) | 14 | 4 194 | 2 | **5** | 1 | 2 | **2** | [`services/portal/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/service.go) |
| 10 | `packages/rae-agentic-memory/rae_adapters` | 22 | 4 033 | 2 | **22** | 4 | 7 | **11** | [`pkg/drivers/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/agnostic/broker.go) |
| 11 | `scripts/` (Root CLI: Triad, Mesh, Bootstrap) | 14 | 3 234 | 1 | **5** | 1 | 0 | **4** | [`cmd/rae/main.go`](file:///home/grzegorz/cloud/RAE-Suite-go/cmd/rae/main.go) |
| 12 | `packages/rae-agentic-memory/apps/llm` | 16 | 3 034 | 11 | **16** | 0 | 2 | **14** | [`pkg/drivers/llm/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/llm/client.go) |
| 13 | `packages/rae-agentic-memory/integrations` | 15 | 3 016 | 7 | **11** | 6 | 1 | **4** | [`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go), [`services/memory/v46_deep_routes.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/v46_deep_routes.go) |
| 14 | `packages/rae-agentic-memory/sdk` (`rae_memory_sdk`) | 5 | 1 973 | 26 | **3** | 1 | 1 | **1** | Brak osobnego pakietu klienckiego SDK Go (używany `net/http`) |
| 15 | `packages/rae-hive` | 18 | 1 808 | 0 | **7** | 2 | 2 | **3** | [`services/hive/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/hive/service.go) |
| 16 | `packages/rae-agentic-memory/rae-windows` (`rae_lite`) | 15 | 1 776 | 3 | **13** | 1 | 0 | **12** | Brak aplikacji Windows Tray / GUI (serwer Go działa natywnie) |
| 17 | `rae_contracts/` | 6 | 1 511 | 67 | **2** | 0 | 0 | **2** | [`pkg/raecore/models/`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/models/models.go) |
| 18 | `packages/rae-lab` | 22 | 1 015 | 30 | **10** | 5 | 2 | **3** | [`services/lab/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/lab/service.go) |
| 19 | `packages/rae-agentic-memory/apps/ml_service` | 6 | 932 | 4 | **4** | 3 | 0 | **1** | [`services/memory/v46_deep_routes.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/v46_deep_routes.go) |
| 20 | `packages/rae-mcp-server` + `rae_mcp_server.py` | 4 | 1 400 | 0 | **2** | 1 | 1 | **0** | [`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go) |
| 21 | Pozostałe (`src/`, `services/`, `infra/`, `rae_auditor/`) | 37 | 2 527 | 2 | **17** | 1 | 0 | **16** | [`services/orchestrator/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/orchestrator/service.go) |

---

## 3. Szczegółowa Mapa Różnic: Co Dokładnie Jest Uproszczone (`PORTED_SIMPLIFIED`), a Czego Brakuje (`MISSING_IN_GO`)?

Analiza drzewa AST wyodrębniła **6 konkretnych obszarów funkcjonalnych**, w których kod Go jest albo znacząco uproszczony względem oryginału w Pythonie, albo nie zawiera wewnętrznych podmodułów:

### Obszar A: `packages/rae-phoenix` (`feniks`) — 108 plików (25 140 LOC w Pythonie vs 878 LOC w 4 plikach Go)
W Go serwis [`services/phoenix/`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/service.go) obsługuje wszystkie endpointy HTTP i podstawową analizę AST/post-mortem, ale w Pythonie pakiet `feniks` posiada głębokie silniki, których w Go nie przeniesiono 1:1 (lub zastąpiono 56-liniowym [`recipes_exec.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/recipes_exec.go)):
1. **Głębokie przepisy migracji AngularJS -> React/Next.js (`feniks/core/refactor/recipes/angularjs/`)**:
   - `ControllerToComponentRecipe` (`657 LOC`, 6 metod analizy `$scope`, DI, metod kontrolera i generowania komponentu React),
   - `DirectiveToComponentRecipe` (`601 LOC`, konwersja dyrektyw, `restrict`, `link`, `compile`, `transclude`),
   - `RoutingToAppRouterRecipe` (`597 LOC`, transformacja `$routeProvider` / `$stateProvider` do Next.js App Router),
   - `ScopeToHooksRecipe` (`482 LOC`, konwersja `$watch`, `$broadcast`, `$emit` na `useState`/`useEffect`),
   - `TemplateToJsxRecipe` (`470 LOC`, parser szablonów HTML z `ng-repeat`, `ng-if`, `ng-model`, filtrami `|` do JSX),
   - `BehaviorGuardIntegration` (`473 LOC`, generowanie planu testów regresji behawioralnej PlayWright/UI).
2. **Silnik reguł refleksji i detekcji zdolności (`feniks/core/reflection/` i `evaluation/`)**:
   - `ReflectionRuleSet` (`617 LOC`, `feniks/core/reflection/rules.py`),
   - `ReportGenerator` (`337 LOC`, `feniks/core/evaluation/reporting.py`),
   - `AnalysisPipeline` (`312 LOC`, `feniks/core/evaluation/pipeline.py`),
   - `CapabilityDetector` (`287 LOC`, `feniks/core/reflection/capabilities.py`),
   - `ScenarioLibrary` (`273 LOC`, `feniks/core/behavior/scenario_library.py` — eksport/import scenariuszy YAML),
   - `UIRunner` (`338 LOC`, `feniks/adapters/runners/ui_runner.py`).
3. **Dedykowane backendy przechowywania Feniksa (`feniks/adapters/storage/`)**:
   - `PostgresBackend` (`508 LOC`, 20 metod), `QdrantBackend` (`476 LOC`, 22 metody), `BehaviorStore` (`326 LOC`, 23 metody).

---

### Obszar B: `packages/rae-agentic-memory/apps/memory_api` — Workery Tła, Potoki i Serwisy Głębokie (60 453 LOC w Pythonie vs ~2 900 LOC w `services/memory/`)
Wszystkie 316 tras HTTP w `services/memory/` działa i przechodzi testy, ale w Pythonie pod spodem znajdują się rozbudowane serwisy i workery asynchroniczne:
1. **Workery tła (`apps/memory_api/workers/`)**:
   - `MemoryMaintenanceWorker` (`723 LOC`, `workers/memory_maintenance.py`) — cykliczna konserwacja pamięci, wygaszanie `ttl`, okresowa konsolidacja snu (`dreaming`), automatyczna archiwizacja i czyszczenie sierot,
   - `CommunityWorker` (`244 LOC`, `workers/community_worker.py`) — okresowa detekcja społeczności na grafie wiedzy i generowanie podsumowań tematycznych.
2. **Wieloetapowe potoki i delegaty w `apps/memory_api/services/` i `core/`**:
   - `MemoryConsolidationService` (`717 LOC`, 7 metod konsolidacji epizodycznej -> semantycznej),
   - `ReflectionPipeline` (`623 LOC`, `services/reflection_pipeline.py`),
   - `ImportanceScoringService` (`513 LOC`, 11 metod wieloczynnikowej oceny istotności wspomnień),
   - `DashboardWebSocketService` (`479 LOC`, pełny hub subskrypcji i powiadomień WebSocket w czasie rzeczywistym),
   - `TriggerRepository` + `TriggerService` (`473 LOC`, `repositories/trigger_repository.py` — trwałe reguły wyzwalaczy zdarzeniowych i webhooków),
   - `ContextBuilder` (`460 LOC`, `services/context_builder.py` — zaawansowane składanie okna kontekstowego z budżetem tokenów i wstrzykiwaniem refleksji),
   - `SemanticExtractor` (`428 LOC`, `services/semantic_extractor.py` — ekstrakcja trójek i kanonizacja pojęć),
   - `MemoryIngestionDelegate` (`407 LOC`) oraz `MemoryLifecycleDelegate` (`331 LOC`, 20 metod),
   - `SemanticSearchPipeline` (`403 LOC`, `services/semantic_search.py`),
   - `MetricsTracker` (`351 LOC`, `core/metrics.py`), `ActionExecutor` (`345 LOC`, `core/action_executor.py`), `GraphUpdateOperator` (`299 LOC`, `core/graph_operator.py`),
   - `QueryAnalyzer` (`273 LOC`, `services/query_analyzer.py` — analiza intencji zapytania i dynamiczny dobór wag),
   - `HybridSearchCache` (`264 LOC`, `services/hybrid_cache.py` — cache wyników wyszukiwania z TTL i statystykami `get_hit_rate`).

---

### Obszar C: `packages/rae-agentic-memory/rae-core` i `rae_adapters` — Adaptery Bazodanowe i Silniki Rdzenia (31 338 LOC w Pythonie)
W Go wprowadziliśmy zunifikowany broker [`pkg/drivers/agnostic/broker.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/agnostic/broker.go) (`416 LOC`), [`pkg/drivers/postgres/adapter.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/postgres/adapter.go) (`293 LOC`) i [`pkg/drivers/vectorstore/hnsw_qdrant.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/vectorstore/hnsw_qdrant.go) (`184 LOC`). W porównaniu z Pythonem:
1. **Brak natywnego adaptera SQLite dla trybu RAE-Lite (`SQLiteStorage` 511 LOC, `SQLiteVectorStore` 385 LOC, `SQLiteGraphStore` 275 LOC)**:
   - W Go [`pkg/drivers/sqlstore/persistent.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/sqlstore/persistent.go) używa trwałego zapisu JSONL/snapshot na dysku zamiast relacyjnych tabel SQLite z zapytaniami SQL i indeksami FTS5/wektorowymi.
2. **Uproszczone metody w adapterach pamięci i cache (`InMemoryCache` 326 LOC, `RedisCache`, `QdrantVectorStore` 598 LOC)**:
   - W Pythonie `QdrantVectorStore` obsługuje Named Vectors (wielowektorowe kolekcje dla różnych modeli embeddingów jednocześnie: np. `dense` 384d + `ollama` 768d) i automatyczną walidację schematu kolekcji (`ensure_vector_config`). W Go klient Qdrant obsługuje pojedynczą przestrzeń wektorową.
3. **Podmoduły `rae_core/reflection/reflector.py` (`335 LOC`), `rae_core/runtime.py` (`298 LOC`), `rae_core/math/bandit/bandit.py` (`351 LOC`)**:
   - W Go `MultiArmedBandit` w [`pkg/raecore/math/controller.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/math/controller.go) ma podstawową formułę UCB1 (`SelectArm` + `UpdateArm`), podczas gdy w Pythonie liczy `351 LOC` z obsługą Context-Aware UCB, Thompson Sampling, trwałością wag ramion i degradacją eksploracji.

---

### Obszar D: `orchestrator/` (29 plików, 8 317 LOC) oraz `apps/llm/` (16 plików, 3 034 LOC)
1. **Inteligencja Orkiestratora (`packages/rae-agentic-memory/orchestrator/`)**:
   - `OrchestratorTelemetry` (`411 LOC`, pełny tracing kroków zadań i wywołań LLM),
   - `ModelRouterV2` (`395 LOC`) i `AdaptiveModelRouter` (`319 LOC`, uczenie się na historii sukcesów/błędów modeli),
   - `PerformanceAnalytics` (`369 LOC`, ranking modeli dla typów zadań, identyfikacja wąskich gardeł kosztowych),
   - `QualityGate` (`366 LOC`, wieloetapowa walidacja wyniku zadania przed akceptacją),
   - `RunLogger` (`316 LOC`) i `PerformanceTracker` (`279 LOC`).
2. **Natywni Providerzy LLM (`packages/rae-agentic-memory/apps/llm/providers/`)**:
   - W Pythonie istnieją osobne klasy `QwenProvider` (`335 LOC`), `GeminiProvider` (`325 LOC`), `AnthropicProvider`, `OllamaProvider`, `OpenAIProvider`, `DeepSeekProvider` oraz `LLMRouter` (`261 LOC`) obsługujący streaming (`stream`) i batch embedding (`embed_batch`).
   - W Go [`pkg/drivers/llm/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/llm/client.go) (`208 LOC`) obsługuje `Complete` i `Embed`, ale nie posiada metod `Stream` (SSE token-by-token) ani `EmbedBatch` oraz natywnego formatu Anthropic Messages API / Gemini `generateContent` (zakłada endpoint kompatybilny z OpenAI/Ollama).

---

### Obszar E: Serwery i Narzędzia MCP (`packages/rae-mcp-server`, `rae_mcp_server.py`, `integrations/mcp`)
Porównanie 1:1 zarejestrowanych narzędzi MCP (`tools/list`) między Pythonem a [`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go):

| Narzędzie MCP | Python (`RAE-Suite`) | Go (`services/mcp/service.go`) | Status |
| :--- | :---: | :---: | :--- |
| `rae_memory_store` / `rae_memory_search` | ✅ | ✅ | `PORTED_FULL` |
| `rae_quality_audit` | ✅ | ✅ | `PORTED_FULL` |
| `rae_phoenix_impact` / `rae_phoenix_repair` | ✅ | ✅ | `PORTED_FULL` |
| `rae_sandbox_exec` / `rae_sandbox_run` | ✅ | ✅ | `PORTED_FULL` |
| `rae_cluster_dispatch` / `rae_consolidate_traces` | ✅ | ✅ | `PORTED_FULL` |
| `quick_note` / `refine_artifact` / `fork_artifact` / `approve_artifact` / `record_failure` / `list_my_traces` / `request_approval` | ✅ (CRL / Integrations) | ✅ | `PORTED_FULL` |
| **`rae_kernel_execute`** (`packages/rae-mcp-server/server.py`) | ✅ | ❌ | **`MISSING_IN_GO`** |
| **`rae_knowledge_resolve`** (`packages/rae-mcp-server/server.py`) | ✅ | ❌ | **`MISSING_IN_GO`** |
| **`rae_lab_insights`** (`packages/rae-mcp-server/server.py`) | ✅ | ❌ | **`MISSING_IN_GO`** |
| **`rae_quality_rpc`** (`packages/rae-mcp-server/server.py`) | ✅ | ❌ | **`MISSING_IN_GO`** |
| **`get_cloud_status` / `run_diagnostic` / `get_service_logs` / `create_rae_memory` / `search_rae_memory`** (`rae_mcp_server.py`) | ✅ | ❌ | **`MISSING_IN_GO`** |
| **`check_approval_status` / `get_circuit_breakers` / `get_related_context` / `list_policies` / `save_memory` / `search_memory`** (`integrations/mcp`) | ✅ | ❌ | **`MISSING_IN_GO`** |

---

### Obszar F: Aplikacje GUI / Portal / CLI (`rae-portal`, `rae-windows`, `scripts/`, `sdk/`)
1. **Aplikacje Portalowe (`packages/rae-agentic-memory/rae-portal/`, 17 plików, 5 153 LOC)**:
   - `PDKnowledgeHubApp` (`973 LOC` — wyszukiwarka aktów prawnych ISAP, ekstrakcja tekstu z PDF, dzielenie na chunki, upload plików),
   - `RAESuitePortal` (`743 LOC` — główna powłoka UI z inspektorem i klientem WebSocket),
   - `CentralAuditorApp` (`461 LOC` — weryfikacja podpisów zdarzeń, detekcja sekretów, wyzwalanie audytu Sentinel),
   - `PhoenixRepairApp` (`408 LOC` — symulacja incydentów i aplikowanie napraw),
   - `MissionControlApp` (`347 LOC`).
   - W Go [`services/portal/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/service.go) (`205 LOC`) udostępnia podstawowe endpointy REST dla portalu, ale nie zawiera logiki biznesowej `PDKnowledgeHubApp` (ISAP/PDF chunking) ani `CentralAuditorApp`.
2. **Skrypty Operacyjne CLI (`scripts/`, 14 plików, 3 234 LOC + `packages/rae-agentic-memory/scripts/`, 100 plików)**:
   - W [`cmd/rae/main.go`](file:///home/grzegorz/cloud/RAE-Suite-go/cmd/rae/main.go) (`242 LOC`) przeniesiono główne komendy (`triad`, `mesh-status`, `record-task`, `bootstrap`), natomiast w Pythonie znajdują się dodatkowo narzędzia CI/diagnostyczne: `FailureAnalyzer` (`314 LOC`), `CIFixAgent` (`269 LOC`), `emergency_recover_memory.py`, `connect_cluster.py`, `deep_data_audit.py`.

---

## 4. Wnioski Końcowe i Rekomendacja

1. **Co jest gotowe w 100%:** Warstwa kontraktów HTTP REST (`316/316` tras), brak statycznych zaślepek w handlerach (`332` handlery), główna matematyka `rae-core` (`InformationBottleneck`, `MDP`, `MathController`, `EpistemicDAG`, `GraphRAG`), `RAE-Mesh` oraz baza agnostyczna (`InMemory` + `SQLStore` + `Postgres` + `Qdrant`).
2. **Co różni Python od Go (gdzie są braki):**
   - **Brakujące 14 narzędzi/aliasów w serwerze MCP** ([`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go)),
   - **Głębokie przepisy refaktoryzacji AST w `Phoenix` (`feniks`)** (6 przepisów AngularJS -> React/Next.js, `ReflectionRuleSet`, `CapabilityDetector`, `ScenarioLibrary`, `ReportGenerator`),
   - **Workery tła i głębokie serwisy `memory_api`** (`MemoryMaintenanceWorker`, `CommunityWorker`, `MemoryConsolidationService`, `ReflectionPipeline`, `TriggerService`, `SemanticExtractor`, `QueryAnalyzer`, `HybridSearchCache`),
   - **Zaawansowana inteligencja `orchestrator/` i `apps/llm/`** (`AdaptiveModelRouter`, `PerformanceAnalytics`, `OrchestratorTelemetry`, `QualityGate`, `Stream`/`EmbedBatch` i natywni providerzy Anthropic/Gemini/Qwen oraz Multi-Vector Named Vectors w Qdrant),
   - **Silniki aplikacyjne `rae-portal`** (`PDKnowledgeHubApp` — ISAP/PDF chunking, `CentralAuditorApp` — weryfikacja podpisów i detekcja sekretów).
