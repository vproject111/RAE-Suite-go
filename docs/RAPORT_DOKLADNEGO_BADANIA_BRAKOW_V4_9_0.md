# Raport Dokładnego Badania Kryminalistycznego AST i Architektury: Czego Nadal Brakuje w Go Względem Pythona (`RAE-Suite` vs `RAE-Suite-go` v4.9.0)

**Data badania:** 2026-10-19  
**Skrypt badawczy:** [`scripts/deep_forensic_gap_investigation.py`](file:///home/grzegorz/cloud/RAE-Suite-go/scripts/deep_forensic_gap_investigation.py)  
**Pełne dane JSON badania:** [`docs/BADANIE_BRAKOW_PY_GO_V4_9_0.json`](file:///home/grzegorz/cloud/RAE-Suite-go/docs/BADANIE_BRAKOW_PY_GO_V4_9_0.json)  
**Badany stan kodu:** `RAE-Suite` (Python) vs `RAE-Suite-go` (`v4.9.0`)

---

## 1. Podsumowanie Wykonawcze — 5 Wymiarów Rzeczywistych Braków

Dotychczasowe audyty weryfikowały przede wszystkim obecność tras HTTP (`316/316`), narzędzi MCP (`31/31`) oraz dopasowanie nazw struktur. Głębokie badanie kryminalistyczne (`deep_forensic_gap_investigation.py`), które analizuje **ciała klas, liczbę metod, graf wywołań produkcyjnych (wiring), funkcje najwyższego poziomu oraz protokoły I/O**, wykazało **5 twardych wymiarów rozbieżności** między Pythonem a Go:

| Wymiar Badania | Liczba wykrytych braków / rozbieżności | Znaczenie Inżynierskie |
| :--- | :---: | :--- |
| **1. Puste struktury-markery w Go (`EMPTY_MARKER_STRUCT_IN_GO`)** | **`10` struktur** (`0` pól, `0` metod w Go vs `5 049 LOC` i `129` metod w Pythonie) | Struktury istnieją wyłącznie z nazwy (`type X struct{}`), nie posiadają żadnej logiki ani metod. |
| **2. Klasy o drastycznym deficycie metod (`SIMPLIFIED_METHODS_IN_GO`)** | **`36` głównych klas** (`0–2` metody w Go vs `5–58` metod w Pythonie) | Klasy istnieją w Go, ale realizują tylko ułamek API swoich odpowiedników z Pythona (np. `MemoryClient` ma 2 metody zamiast 58). |
| **3. Niepodłączone („martwe w produkcji”) struktury Go (`UNWIRED_GO_STRUCTS`)** | **`177` struktur w Go** (w tym niemal całe pakiety z `v4.9.0`!) | Struktury są wywoływane wyłącznie w testach `*_test.go`, ale **nie są podłączone** do serwerów HTTP (`services/*`) ani binariów `cmd/*`. |
| **4. Klasy domenowe Pythona bez odpowiednika 1:1 w Go (`UNMATCHED_DOMAIN_CLASSES`)** | **`648` klas domenowych** (`~54 000 LOC` w Pythonie) | Podmoduły, delegaty, klasy poszczególnych warstw pamięci, wtyczki `Plugin`, agenci orkiestratora, chunkery AST i aplikacje Portal/Windows. |
| **5. Funkcje najwyższego poziomu (`>=15 LOC`), CLI i protokoły niskopoziomowe** | **`595` funkcji** + różnice protokołów (SQL, WebSocket, MCP Prompts/Resources) | Brak obsługi `prompts/list`, `prompts/get`, `resources/list`, `resources/read` w MCP, brak prawdziwego drivera SQL (`database/sql`) oraz `Upgrade: websocket`. |

---

## 2. Wymiar 1: 10 Pustych Struktur-Markerów w Go (`0` Pól, `0` Metod)

W plikach [`pkg/raecore/workers/maintenance_and_pipelines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/workers/maintenance_and_pipelines.go) oraz [`pkg/raecore/orchestrator/intelligence.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/orchestrator/intelligence.go) zdefiniowano **10 całkowicie pustych struktur (`type ... struct{}`)**, które w Pythonie stanowią kluczowe repozytoria i serwisy:

| # | Klasa w Pythonie | Plik w Pythonie | LOC (Py) | Metody (Py) | Stan w Go (`v4.9.0`) |
| :-: | :--- | :--- | :---: | :---: | :--- |
| 1 | `EnhancedGraphRepository` | `apps/memory_api/repositories/graph_repository_enhanced.py` | 858 | 20 | `type EnhancedGraphRepository struct{}` (**0 metod**) |
| 2 | `GraphRepository` | `apps/memory_api/repositories/graph_repository.py` | 846 | 18 | `type GraphRepository struct{}` (**0 metod**) |
| 3 | `RAECoreService` | `apps/memory_api/services/rae_core_service.py` | 742 | 33 | `type RAECoreService struct{}` (**0 metod**) |
| 4 | `GraphAlgorithmsService` | `apps/memory_api/services/graph_algorithms.py` | 644 | 12 | `type GraphAlgorithmsService struct{}` (**0 metod**) |
| 5 | `MemoryIngestionDelegate` | `apps/memory_api/services/delegates/ingestion_delegate.py` | 407 | 6 | `type MemoryIngestionDelegate struct{}` (**0 metod**) |
| 6 | `ModelRouter` | `orchestrator/core/model_router.py` + `core/model_router.py` | 570 | 8 | `type ModelRouter struct{}` (**0 metod**) |
| 7 | `MetricsRepository` | `apps/memory_api/repositories/metrics_repository.py` | 336 | 9 | `type MetricsRepository struct{}` (**0 metod**) |
| 8 | `RewardFunction` | `apps/memory_api/core/reward.py` | 335 | 1 | `type RewardFunction struct{}` (**0 metod**) |
| 9 | `MemoryLifecycleDelegate` | `apps/memory_api/services/delegates/lifecycle_delegate.py` | 331 | 20 | `type MemoryLifecycleDelegate struct{}` (**0 metod**) |
| 10 | `BenchmarkRepository` | `apps/memory_api/repositories/evaluation_repository.py` | 313 | 10 | `type BenchmarkRepository struct{}` (**0 metod**) |

---

## 3. Wymiar 2: 36 Klas o Drastycznym Deficycie Metod (`SIMPLIFIED_METHODS_IN_GO`)

Nawet tam, gdzie struktura w Go posiada pola lub osadza inny typ, liczba zaudytowanych metod i głębia logiki jest znacznie mniejsza niż w Pythonie:

1. **`MemoryClient` (`sdk/python/rae_memory_sdk/client.py`):**
   - **Python:** `1 225 LOC`, **58 metod** (pełne zarządzanie pamięcią, sesjami, grafem, refleksjami, oceną, batch ingestem, subskrypcjami, politykami, kluczami API).
   - **Go ([`pkg/raesdk/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raesdk/client.go)):** **2 metody** (`Store`, `Query`). Brakuje 56 metod klienckich oraz klienta HTTP (`net/http`) do komunikacji ze zdalnym serwerem RAE.
2. **`InMemoryStorage` (`1 038 LOC`, 31 metod), `PostgreSQLStorage` (`913 LOC`, 19 metod), `SQLiteStorage` (`511 LOC`, 23 metody):**
   - **Python:** Bogate adaptery implementujące filtrowanie po `session_id`, `agent_id`, `tags`, `ttl`, operacje wsadowe `batch_insert`, `update_strength`, `count_by_layer`, `delete_by_project`.
   - **Go ([`pkg/drivers/sqlstore/persistent.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/sqlstore/persistent.go)):** Puste struktury osadzające `*PersistentStore` / `*UnifiedStore` (**0 własnych metod**).
3. **`RAEEngine` (`745 LOC`, 8 metod w `rae_core/engine.py`) i `HybridSearchEngine` (`387 LOC`, 5 metod w `rae_core/search/engine.py`):**
   - **Go ([`pkg/raecore/math/controller.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/math/controller.go)):** Struktury posiadają wyłącznie konstruktor (`NewRAEEngine`) i **0 metod** operacyjnych (`store`, `retrieve`, `reflect`, `consolidate`, `get_status`).
4. **`PostgresBackend` (`508 LOC`, 20 metod), `QdrantBackend` (`476 LOC`, 22 metody), `BehaviorStore` (`326 LOC`, 23 metody), `EnhancedRAEClient` (`378 LOC`, 6 metod), `RefactorEngine` (`317 LOC`, 6 metod):**
   - **Go ([`services/phoenix/deep_recipes_and_engines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/deep_recipes_and_engines.go)):** `PostgresBackend`, `QdrantBackend`, `EnhancedRAEClient` i `RefactorEngine` mają **0 metod**, a `BehaviorStore` ma tylko **2 metody** z 23.
5. **`QwenProvider` (`335 LOC`), `OpenAIProvider` (`251 LOC`), `OllamaProvider` (`233 LOC`), `AnthropicProvider` (`230 LOC`), `DeepSeekProvider`:**
   - **Go ([`pkg/drivers/llm/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/llm/client.go)):** Jedynie osadzają `*LLMRouter` (**0 własnych metod**); nie formatują natywnych żądań HTTP do `/v1/messages` (Anthropic), `generateContent` (Gemini) ani `/api/chat` (Ollama).
6. **`MultiArmedBandit` (`351 LOC`, 8 metod w `rae_core/math/bandit/bandit.py`):**
   - **Go:** Tylko 1 własna metoda (`DecayExploration`); brak `select_arm_with_context`, `update_arm_reward`, `get_arm_statistics`, `save_state`, `load_state`, `reset_arm`.
7. **`MetricsTracker` (`351 LOC`, 8 metod) i `ActionExecutor` (`345 LOC`, 7 metod):**
   - **Go:** Po **2 metody** zamiast 8 i 7 (brak `record_information_bottleneck`, `record_graph_state`, `get_best_actions`, `get_last_transition`, `get_transition_history`, `get_metrics_summary`).
8. **`RiskCoverageEngine` (`385 LOC`, 5 metod w `rae-quality`):**
   - **Go:** Tylko 1 metoda (`EvaluateRiskCoverage`).

---

## 4. Wymiar 3: Niepodłączone („Martwe w Produkcji”) Moduły w Go (`177` Struktur)

Analiza przepływu wywołań w kodzie produkcyjnym Go (poza plikami `*_test.go`) wykazała, że nowe silniki dodane w pakietach wewnętrznych **nie są jeszcze podłączone do działających serwerów HTTP ani procesów startowych `cmd/*`**:

1. **[`pkg/raecore/workers/maintenance_and_pipelines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/workers/maintenance_and_pipelines.go) (`23` struktury niepodłączone do `services/memory`):**
   - `MemoryMaintenanceWorker` i `CommunityWorker` **nie mają uruchomionej pętli goroutine w tle** w [`services/memory/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/service.go) ani dedykowanych endpointów wyzwalających ich statystyki.
   - `MemoryConsolidationService`, `ReflectionPipeline`, `ImportanceScoringService`, `ContextBuilder`, `SemanticExtractor`, `QueryAnalyzer`, `HybridSearchCache`, `TriggerRepository`, `DashboardWebSocketService`, `SemanticSearchPipeline` oraz `HumanApprovalService` istnieją w pakiecie `pkg/raecore/workers`, ale [`services/memory/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/service.go) nie posiada do nich referencji i nie używa ich podczas obsługi żądań `/v2/memories/*`, `/v2/search/*`, `/v2/triggers/*` czy `/v2/context/*`.
2. **[`pkg/raecore/orchestrator/intelligence.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/orchestrator/intelligence.go) (`9` struktur niepodłączonych do `services/orchestrator`):**
   - `Orchestrator`, `AdaptiveModelRouter`, `ModelRouterV2`, `PerformanceAnalytics`, `PerformanceTracker`, `QualityGate`, `OrchestratorTelemetry` i `RunLogger` nie są podłączone do [`services/orchestrator/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/orchestrator/service.go).
3. **[`services/portal/deep_apps.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/deep_apps.go) (`5` struktur niepodłączonych do `services/portal/service.go`):**
   - `PDKnowledgeHubApp`, `CentralAuditorApp`, `PhoenixRepairApp`, `MissionControlApp`, `RAESuitePortal` nie są podpięte pod endpointy HTTP w [`services/portal/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/service.go).
4. **[`services/phoenix/deep_recipes_and_engines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/deep_recipes_and_engines.go) (`15` z `20` struktur niepodłączonych do HTTP):**
   - Przepisy AngularJS są podłączone do `ApplyRecipe`, ale `AnalysisPipeline`, `ReflectionRuleSet`, `CapabilityDetector`, `ReportGenerator`, `ScenarioLibrary`, `UIRunner`, `BehaviorGuardIntegration`, `RAEFormatter` i `PhoenixRefactorer` nie są wystawione przez [`services/phoenix/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/service.go).
5. **[`pkg/raesdk/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raesdk/client.go) (`7` struktur niepodłączonych do `cmd/rae/main.go`):**
   - `FailureAnalyzer`, `CIFixAgent`, `NodeAgent` i `MemoryClient` nie są wywoływane przez subkomendy CLI w [`cmd/rae/main.go`](file:///home/grzegorz/cloud/RAE-Suite-go/cmd/rae/main.go).

---

## 5. Wymiar 4: 648 Klas Domenowych z Pythona Bez Odpowiednika 1:1 w Go

Poniżej znajduje się pełne zestawienie brakujących klas domenowych z podziałem na podsystemy:

### 5.1. `packages/rae-agentic-memory/rae-core` — 172 brakujące klasy (`14 043 LOC`)
- **Dedykowane klasy warstw pamięci (`rae_core/layers/`):**
  - `ReflectiveLayer` (`274 LOC`, 9 metod), `LongTermLayer` (`260 LOC`, 8 metod), `EpisodicLayer`, `WorkingLayer`, `SensoryLayer`, `SemanticLayer` (w Go warstwy są reprezentowane wyłącznie jako enum `models.MemoryLayer` wewnątrz jednej mapy `UnifiedStore`, bez osobnych klas zarządzających polityką każdej warstwy).
- **Matematyka, Rezonans, Rozumowanie i Monitor Bandyty (`rae_core/math/`):**
  - `BanditMonitor` (`286 LOC`, 4 metody: `check_health`, `record_decision`, `get_summary`, `get_health_status`),
  - `ReasoningController` (`254 LOC`, 6 metod: `should_continue_reasoning`, `prune_contradictory_paths`, `mark_path_as_false`),
  - `SemanticResonanceEngine` (`253 LOC`, 2 metody: `sharpen`, `compute_resonance`),
  - `MathLayerController` (`237 LOC`, 8 metod: `compute_similarity`, `apply_decay`, `get_retrieval_weights`, `get_scoring_params`, `score_memory`),
  - `ContextInjector`, `StructureMetrics`, `QualityMetrics`.
- **Silnik Refleksji i Ewaluacji (`rae_core/reflection/` i `evaluation/`):**
  - `ReflectionEngine` (`308 LOC`, 7 metod: `run_reflection_cycle`, `generate_reflection`, `execute_action`, `evaluate_answer_draft`, `evaluate_memory_quality`),
  - `Evaluator` (`242 LOC`, 3 metody: `evaluate_memory_quality`, `evaluate_action_outcome`, `evaluate_memory_batch`),
  - `RetrievalBenchmarkEngine` (`229 LOC`, 4 metody: `evaluate_single_case`, `run_suite`, `compare_reranker_gain`, `export_report_to_json`).
- **Synchronizacja, Wzbogacanie Kontekstu, Tryb Wbudowany i Konfiguracja (`rae_core/`):**
  - `SyncProtocol` (`259 LOC`, 5 metod: `push`, `pull`, `sync`, `get_sync_status`, `handshake`),
  - `RAESettings` (`258 LOC`, 4 metody),
  - `EmbeddedRAE` (`254 LOC`, 9 metod: `open`, `store`, `get`, `delete`, `list`),
  - `LLMOrchestrator` (`228 LOC`, 7 metod),
  - `ContextEnricher` (`223 LOC`, 6 metod: `detect_language`, `extract_python_symbols`, `extract_generic_symbols`, `build_envelope`),
  - `RedisCache` (`260 LOC`, 12 metod).

### 5.2. `packages/rae-agentic-memory/apps/memory_api` — 151 brakujących klas (`17 848 LOC`)
- **Kluczowe serwisy biznesowe i bezpieczeństwa (`apps/memory_api/services/`):**
  - `ComplianceService` (`1 184 LOC`, `generate_compliance_report`, `verify_rls_status` — kompleksowy audyt zgodności ISO 42001 / GDPR / RLS),
  - `HybridSearchService` (`810 LOC` w `hybrid_search_service.py` oraz `709 LOC` w `hybrid_search.py`),
  - `MeshService` (`753 LOC`, **18 metod** w `mesh_service.py`: `encrypt_auth_data`, `decrypt_auth_data`, `derive_key_pair`, `generate_challenge`, `verify_challenge_signature`, `create_exchange_envelope`),
  - `AnalyticsService` (`615 LOC`, 4 metody: `get_tenant_stats`, `generate_report`, `get_real_time_metrics`, `get_usage_alerts`),
  - `DriftDetector` (`447 LOC`, `detect_drift` — statystyczna detekcja dryfu danych i embeddingów KS-test/PSI),
  - `EvaluationService` (`440 LOC`, `evaluate_search_results`, `compare_ab_variants`),
  - `ReflectionEngine` (`377 LOC`) oraz `ReflectionEngineV2` (`348 LOC`),
  - `GraphExtractionService` (`326 LOC`, `extract_knowledge_graph`, `store_graph_triples`),
  - `SourceTrustService` (`298 LOC`, 4 metody: `assess_source_trust`, `check_trust_decay`, `verify_source`, `get_trust_filter_weights`),
  - `MemoryRetrievalDelegate` (`261 LOC`, 8 metod),
  - `RBACService` (`253 LOC`, 5 metod: `get_user_role`, `assign_role`, `revoke_role`, `list_tenant_users`, `log_access`).
- **System Wtyczek i Repozytoria (`apps/memory_api/plugins/` i `repositories/`):**
  - `Plugin` (`238 LOC`, 15 metod cyklu życia hooków: `on_before_memory_create`, `on_after_memory_create`, `on_before_search`, `on_after_search`),
  - `ABTestRepository` (`242 LOC`, 8 metod).

### 5.3. `packages/rae-phoenix` (`feniks`) — 103 brakujące klasy (`6 592 LOC`)
- `BehaviorComparisonEngine` (`283 LOC`, `check_snapshot`),
- `CLIRunner` (`257 LOC`, `execute_scenario`) oraz `HTTPRunner` (`237 LOC`, `execute_scenario`),
- `SystemModelBuilder` (`256 LOC`, `add_chunks`, `build`),
- `MetaReflectionEngine` (`242 LOC`, 7 metod: `run_post_mortem`, `run_longitudinal`, `run_self_model_update`, `generate_reflections`, `save_reflections_jsonl`),
- `PostMortemAnalyzer` (`231 LOC`, `analyze_session`),
- `LongitudinalAnalyzer` (`227 LOC`, `analyze_trends`),
- `ExtractFunctionRecipe` (`211 LOC`, 6 metod) oraz `ReduceComplexityRecipe` (`202 LOC`, 6 metod),
- Chunkery i parsery wielojęzyczne (`PythonChunker`, `JavaScriptChunker`, `PHPChunker`, `ASTParser`).

### 5.4. `core/` — 43 klasy o zmienionych nazwach lub uproszczonej strukturze (`3 481 LOC`)
- `AutonomyKernel` (`902 LOC`), `LayerTriadRunner` (`290 LOC`), `QualityTribunal` (`179 LOC`), `HardFramesEngine` (`146 LOC`), `QualitySentinel` (`132 LOC`), `PhoenixEngine` (`113 LOC`), `SandboxManager` (`103 LOC`), `ProbabilisticSemanticCache` (`94 LOC`), `ProjectPhaseAnalyzer` (`91 LOC`), `RAEGlobalSearchEngine` (`86 LOC`).

### 5.5. `packages/rae-agentic-memory/orchestrator` — 31 brakujących klas (`3 091 LOC`)
- `StateMachine` (`288 LOC`, 12 metod zarządzania stanem zadań i kroków),
- `ProviderRegistry` (`244 LOC`, 10 metod rejestracji i wyboru providerów),
- `PerformanceDashboard` (`229 LOC`, 6 metod raportowania),
- Agenci wykonawczy: `CodeReviewerAgent` (`215 LOC`), `ImplementerAgent` (`202 LOC`), `PlanReviewerAgent` (`159 LOC`), `PlannerAgent`,
- `RAEMemoryIntegration` (`187 LOC`, 4 metody uczenia międzywdrożeniowego),
- `GeminiAdapter` (`245 LOC`), `ClaudeAdapter` (`165 LOC`), `ClaudeProvider` (`229 LOC`).

### 5.6. `packages/rae-quality` — 25 brakujących klas (`2 873 LOC`)
- `PolicyEngine` (`280 LOC`, 6 metod: `catalog`, `get_rule`, `get_policy_for_task`, `validate_rule_evidence`, `check_agent_tampering`),
- `CodeQualityEngine` (`258 LOC`), `ASTInspector` (`214 LOC`), `ArchitectureEngine` (`190 LOC`), `MutationCoverageEngine` (`184 LOC`), `PDFDtpValidator` (`180 LOC`), `SonarQubeClient` (`170 LOC`), `InfrastructureEngine` (`158 LOC`), `AIReviewEngine` (`155 LOC`), `CalculatorVerifier` (`151 LOC`).

### 5.7. Pozostałe Podsystemy (`rae_adapters`, `rae-windows`, `rae-portal`, `rae-lab`, `rae-hive`, `apps/llm`, `apps/ml_service`, `integrations`, `rae_auditor`)
- **`rae_adapters` (`14` klas):** `RedisCache`, `PostgresAdapter`, `QdrantAdapter`, `RedisAdapter`, `RAEAgenticMemoryAdapter`, `OpenAPIAdapter`, `GitRuntimeAdapter`, `InfrastructureFactory`.
- **`rae-windows` (`13` klas):** `RAELiteService`, `UniversalIngestor`, `HardwareDetector`, `UIObserver`, `RAETrayApp`, `ProfileSelectorUI`, `LlamaCppAdapter`.
- **`rae-portal` (`11` klas):** `RAESuiteClient` (`287 LOC`, 17 metod), `EvolutionLabApp` (`220 LOC`), `DecayRetentionApp` (`191 LOC`), `MeshFederationApp` (`185 LOC`), `SearchConsoleApp` (`156 LOC`), `MozillaApp` (`104 LOC`), `ProceduralWizard` (`85 LOC`).
- **`rae-lab` (`10` klas):** `ShadowGuardrailManager` (`137 LOC`, 5 metod), `ExperimentManager` (`102 LOC`), `RetrievalOptimizer` (`99 LOC`), `MABTuner` (`60 LOC`), `LabObservatory` (`35 LOC`), `FailureMiningEngine` (`27 LOC`), `HypothesisEngine` (`27 LOC`), `SafeRolloutManager` (`10 LOC`).
- **`rae-hive` (`7` klas):** `HiveMindConnector` (`245 LOC`, 10 metod), `GitWorktreeManager` (`73 LOC`), `DockerSandboxManager` (`69 LOC`), `HiveExecutionSwarm` (`48 LOC`), `VisualIngestPipeline` (`44 LOC`), `HiveExecutionEvaluator` (`36 LOC`).
- **`apps/llm` (`9` klas):** `GrokProvider` (`260 LOC`), `ModelCatalogManager` (`210 LOC`), `DelegatedLLMProvider` (`96 LOC`), `LLMProvider` (`77 LOC`), `LLMOrchestrator` (`75 LOC`).
- **`apps/ml_service` (`4` klasy):** `TripleExtractionService` (`304 LOC`), `NLPService` (`195 LOC`), `EntityResolutionMLService` (`136 LOC`), `EmbeddingMLService` (`74 LOC`).
- **`rae_auditor` (`1` klasa):** `ComplianceAuditor` (`221 LOC`, 5 metod weryfikacji łańcucha zdarzeń SHA-256 i separacji księgi symulacyjnej od produkcyjnej).

---

## 6. Wymiar 5: 595 Funkcji Najwyższego Poziomu (`>= 15 LOC`), Skrypty CLI oraz Protokoły Niskopoziomowe

1. **Protokół MCP — brak obsługi `prompts/*` oraz `resources/*`:**
   - W Pythonie (`integrations/mcp/src/rae_mcp/server.py`) serwer MCP obsługuje nie tylko `tools/list` i `tools/call`, ale także **`prompts/list`**, **`prompts/get`** (`handle_get_prompt`, `72 LOC`) oraz **`resources/list`**, **`resources/read`** (`handle_read_resource`, `43 LOC`).
   - W Go ([`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go#L597-L633)) metoda `ServeStdio` obsługuje wyłącznie `tools/list` i `tools/call` — wywołanie `prompts/list`, `prompts/get`, `resources/list` lub `resources/read` jest ignorowane!
2. **Dekorator/Middleware Automatycznego Śledzenia (`trace_memory` w `sdk/python/rae_memory_sdk/decorators.py`, `159 LOC`):**
   - W Pythonie pozwala opakować dowolną funkcję agenta i automatycznie przechwytywać wejście/wyjście/wyjątki/czas wykonania do warstwy `episodic`/`working`. W [`pkg/raesdk/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raesdk/client.go) brakuje funkcji wyższegozędu `TraceMemory(ctx, fnName, fn)`.
3. **Narzędzia CLI i Audytory Kontraktów (`scripts/`):**
   - `scripts/ci_compatibility_check.py`: funkcje `diff_schema` (`145 LOC`), `load_claims_policy` (`102 LOC`), `compare_proto_messages` (`93 LOC`),
   - `scripts/consensus_planner_cycle.py` (`run_consensus_cycle`, `82 LOC`),
   - `scripts/run_phase_tribunal.py` (`run_phase_tribunal`, `73 LOC`),
   - `packages/rae-phoenix/feniks/apps/cli/angular.py` (`handle_angular_migrate`, `313 LOC`) i `feniks/apps/cli/main.py` (`handle_refactor`, `121 LOC`).
4. **Różnice w Warstwie Sterowników Wykonawczych:**
   - **SQL (`PostgreSQL` / `SQLite`):** W Pythonie używane są natywne połączenia `asyncpg` / `aiosqlite` z prawdziwymi zapytaniami SQL i transakcjami. W Go [`pkg/drivers/postgres/adapter.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/postgres/adapter.go) i [`pkg/drivers/sqlstore/persistent.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/sqlstore/persistent.go) przechowują dane w strukturach Go w RAM i zrzucają snapshoty JSON na dysk (brak otwartego połączenia `database/sql` wykonującego zapytania `SELECT`/`INSERT` na silniku relacyjnym).
   - **WebSockets (`/ws/dashboard`, `/v2/events/stream`):** W Go endpointy zwracają odpowiedź HTTP JSON/SSE zamiast pełnego uścisku dłoni `Upgrade: websocket` (RFC 6455).
