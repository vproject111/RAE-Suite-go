# Iteracyjny Plan Wdrożenia 5 Grup Brakujących i Uproszczonych Modułów (`RAE-Suite-go` v4.9.0)

**Data przygotowania:** 2026-10-19  
**Gałąź robocza:** `feature/v49-deep-engines-workers-mcp-parity`  
**Dokument bazowy:** [`docs/PELNE_ZESTAWIENIE_SPISU_POWSZECHNEGO_PY_GO.md`](file:///home/grzegorz/cloud/RAE-Suite-go/docs/PELNE_ZESTAWIENIE_SPISU_POWSZECHNEGO_PY_GO.md)  
**Cel wydania `v4.9.0`:** Pełne pokrycie 1:1 dla wszystkich 5 grup zidentyfikowanych w Spisie Powszechnym AST (`exhaustive_python_to_go_census.py`): wszystkie narzędzia i aliasy MCP (`31` narzędzi MCP = 100% unii 3 serwerów MCP z Pythona), głębokie silniki i przepisy AST `packages/rae-phoenix` (`feniks`), workery tła i głębokie potoki `apps/memory_api`, adaptery `Named Vectors` / `SQLite RAE-Lite` / `Multi-Armed Bandit` / inteligencja `orchestrator/` + providerzy `apps/llm/`, oraz silniki aplikacyjne `rae-portal`, `rae-quality` V2, SDK i narzędzia CI.

---

## Architektura i Założenia Niezmienników
1. **Brak OpenClaw (Strict Exclusion):** Wszystkie 251 plików `.openclaw_config` i `packages/rae-open-claw` pozostają wykluczone; integracja opiera się wyłącznie na `Hermes` + `OpenCode`.
2. **Agnostyczność Bazodanowa i RAE-Mesh:** Każdy nowy silnik, worker, przepis czy repozytorium współpracuje z brokerem [`pkg/drivers/agnostic/broker.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/agnostic/broker.go) oraz obsługuje replikację [`pkg/raemesh/mesh.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raemesh/mesh.go).
3. **Prawdziwa Logika Algorytmiczna (Zero Stubs):** Każda wdrażana klasa posiada realne rozgałęzienia warunkowe, transformacje struktur danych, obliczenia matematyczne i stanowość weryfikowaną testami.

---

## Podział na 5 Iteracji Wdrożeniowych

### Iteracja 1: Grupa 1 — Pełna Parzystość Serwera MCP (100% narzędzi z 3 serwerów MCP Pythona + `PassiveMemoryInterceptor`)
- **Pliki docelowe:**
  - [`services/mcp/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/mcp/service.go)
- **Zakres wdrożenia:**
  1. Rozszerzenie `ListTools()` i `ExecuteTool()` o wszystkie **15 brakujących narzędzi i aliasów MCP** (łącznie **31 narzędzi MCP** pokrywających 100% unii `packages/rae-mcp-server/server.py`, `rae_mcp_server.py` oraz `integrations/mcp/src/rae_mcp/server.py`):
     - Z `packages/rae-mcp-server/server.py`:
       - `rae_kernel_execute` — wykonanie zadania przez `AutonomyKernelEngine` ze śladem audytowym SHA-256 i kontrolą budżetu,
       - `rae_knowledge_resolve` — rozwiązywanie encji, symboli i trójek w grafie epistemicznym oraz pamięci semantycznej,
       - `rae_lab_insights` — pobieranie wniosków z eksperymentów chaosu, wzorców awarii i rekomendacji `RAE-Lab`,
       - `rae_quality_rpc` — zdalne wywołanie bramki jakości `RAEQualityGatewayV2` / `SecurityGateway` / `Bumpers`.
     - Z `rae_mcp_server.py`:
       - `get_cloud_status` — diagnostyka stanu klastrów, brokera DB, węzłów Mesh i pamięci,
       - `run_diagnostic` — aktywny test zapisu/odczytu/wektorów/grafu/mesh z pomiarem latencji,
       - `get_service_logs` — pobieranie logów pierścieniowych i zdarzeń audytowych z filtrowaniem po serwisie i poziomie,
       - `create_rae_memory` — kanoniczny zapis wspomnienia z warstwą, tagami i projektem,
       - `search_rae_memory` — kanoniczne wyszukiwanie hybrydowe w pamięci RAE.
     - Z `integrations/mcp/src/rae_mcp/server.py`:
       - `check_approval_status` — sprawdzenie statusu akceptacji `HumanApprovalService`,
       - `get_circuit_breakers` — odczyt stanu bezpieczników (Circuit Breakers) dla DB, Qdrant, LLM i Mesh,
       - `get_related_context` — budowanie okna kontekstowego (`ContextBuilder`) z refleksjami dla pliku/zapytania,
       - `list_policies` — lista aktywnych polityk ISO 42001/27001, strażnika `info_class` i limitów zapytań,
       - `save_memory` — zapis wspomnienia z automatycznym czyszczeniem PII (`ScrubPII`, `MaskEmail`),
       - `search_memory` — wyszukiwanie wspomnień z filtrowaniem PII i rankingiem hybrydowym.
  2. Pełna struktura `PassiveMemoryInterceptor` (`SanitizeArguments`, `SummarizeResult`, `CaptureTrace`, `ConsolidateTraces`) oraz funkcje ochrony danych `ScrubPII` i `MaskEmail`.

---

### Iteracja 2: Grupa 2 — Głębokie Silniki `packages/rae-phoenix` (`feniks`)
- **Pliki docelowe:**
  - [`services/phoenix/deep_recipes_and_engines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/deep_recipes_and_engines.go) (nowy moduł)
  - [`services/phoenix/recipes_exec.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/recipes_exec.go) (integracja przepisów)
  - [`services/phoenix/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/phoenix/service.go) (rejestracja silników i tras)
- **Zakres wdrożenia (1:1 z `packages/rae-phoenix/feniks/`):**
  1. **6 głębokich przepisów migracji AngularJS -> React/Next.js (`feniks/core/refactor/recipes/angularjs/`)**:
     - `ControllerToComponentRecipe` (`Name`, `Description`, `RiskLevel`, `Analyze`, `Execute` — ekstrakcja pól `$scope.<field>`, wstrzykiwanych serwisów DI, metod kontrolera i generowanie komponentu funkcyjnego React z `useState` i `useCallback`),
     - `DirectiveToComponentRecipe` (`Analyze`, `Execute` — parsowanie `restrict`, `scope` (`=`, `@`, `&`), `transclude`, `link`/`controller` i generowanie komponentu React z typowanym interfejsem `Props`),
     - `RoutingToAppRouterRecipe` (`Analyze`, `Execute` — ekstrakcja ścieżek `$routeProvider.when(...)` oraz `$stateProvider.state(...)` z parametrami `:id` i generowanie mapy plików Next.js App Router `app/.../page.tsx`),
     - `ScopeToHooksRecipe` (`Analyze`, `Execute` — konwersja `$scope.$watch` na `useEffect`, `$scope.$broadcast`/`$emit`/`$on` na `CustomEvent` / `EventBus`, usuwanie `$scope.$apply`),
     - `TemplateToJsxRecipe` (`Analyze`, `Execute` — transformacja szablonów HTML AngularJS: `ng-repeat="item in items"` -> `{items.map((item, idx) => ...)}`, `ng-if`/`ng-show` -> `{cond && (...)}`, `ng-click` -> `onClick`, `ng-model` -> `value` + `onChange`, `class=` -> `className=`, filtry `| uppercase` / `| currency` -> wywołania funkcji pomocniczych),
     - `BehaviorGuardIntegration` (`CreateTestPlan`, `GenerateBehaviorScenarios`, `GenerateTestScript`, `GenerateIntegrationGuide`).
  2. **Silniki refleksji, ewaluacji, raportowania i scenariuszy (`feniks/core/`)**:
     - `ReflectionRuleSet` (`Evaluate`),
     - `CapabilityDetector` (`DetectFromChunks`, `DetectFromSystemModel`, `EnrichSystemModel`),
     - `ReportGenerator` (`GenerateSummary`, `GenerateModuleAnalysis`, `GenerateCapabilityReport`, `GetRecommendationsData`, `GenerateRecommendations`),
     - `AnalysisPipeline` (`Run`),
     - `ScenarioLibrary` (`PublishScenario`, `ListPublicScenarios`, `SearchScenarios`, `ImportScenario`, `ExportLibraryToYAML`),
     - `UIRunner` (`ExecuteScenario`),
     - `PostgresBackend`, `QdrantBackend`, `BehaviorStore`, `EnhancedRAEClient`, `FeniksMemoryRouter`, `RAEFormatter`, `PhoenixRefactorer`.

---

### Iteracja 3: Grupa 3 — Workery Tła i Głębokie Potoki `apps/memory_api`
- **Pliki docelowe:**
  - [`pkg/raecore/workers/maintenance_and_pipelines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/workers/maintenance_and_pipelines.go) (nowy moduł)
  - [`services/memory/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/service.go) oraz [`services/memory/v47_deep_routes.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/memory/v47_deep_routes.go)
- **Zakres wdrożenia (1:1 z `apps/memory_api/`):**
  1. **Workery tła (`apps/memory_api/workers/`)**:
     - `MemoryMaintenanceWorker` (`RunCycle`, `ApplyDecay`, `PruneExpiredTTL`, `RunDreamConsolidation`, `SummarizeEpisodicBatch`, `GetStats`),
     - `CommunityWorker` (`RunCommunityDetection`, `BuildTopicHierarchies`, `SummarizeCommunities`).
  2. **Głębokie serwisy, potoki i repozytoria (`apps/memory_api/services/`, `core/`, `repositories/`)**:
     - `MemoryConsolidationService` (`ConsolidateEpisodicToSemantic`, `FindConsolidationClusters`, `MergeClusterToSemantic`, `EvaluateConsolidationQuality`),
     - `ReflectionPipeline` (`GenerateReflections`, `ClusterEvents`, `ExtractInsights`, `ValidateAndStoreReflection`),
     - `ImportanceScoringService` (`CalculateScore`, `ScoreRecency`, `ScoreFrequency`, `ScoreCentrality`, `ScoreSemanticRichness`, `RecalculateBatch`),
     - `DashboardWebSocketService` (`Subscribe`, `Unsubscribe`, `BroadcastEvent`, `GetRecentEvents`, `GetActiveSubscribers`),
     - `TriggerRepository` & `TriggerService` (`CreateTrigger`, `GetTrigger`, `ListTriggers`, `UpdateTrigger`, `DeleteTrigger`, `EvaluateEventTriggers`),
     - `ContextBuilder` (`BuildContext`, `InjectReflectionsIntoPrompt`),
     - `SemanticExtractor` (`ExtractFromMemories`, `CanonicalizeTerm`),
     - `SemanticSearchPipeline` (`Search`),
     - `QueryAnalyzer` (`AnalyzeIntent`, `CalculateDynamicWeights`, `GetAvailableProfiles`, `ExplainAnalysis`),
     - `HybridSearchCache` (`Get`, `Set`, `Invalidate`, `ClearExpired`, `GetHitRate`),
     - `MetricsTracker` (`RecordTransition`, `RecordInformationBottleneck`, `RecordGraphState`, `GetCurrentMetrics`, `GetBestActions`),
     - `ActionExecutor` (`CanExecute`, `Execute`, `GetLastTransition`, `GetTransitionHistory`, `GetMetricsSummary`),
     - `GraphUpdateOperator` (`Apply`, `AnalyzeConvergence`),
     - `MemoryIngestionDelegate`, `MemoryLifecycleDelegate`, `HumanApprovalService`, `EnhancedGraphRepository`, `GraphRepository`, `MetricsRepository`, `BenchmarkRepository`.

---

### Iteracja 4: Grupa 4 — Adaptery `rae-core` / `rae_adapters` (`Named Vectors` + `SQLite RAE-Lite` + `MultiArmedBandit`) oraz `orchestrator/` + `apps/llm/`
- **Pliki docelowe:**
  - [`pkg/drivers/vectorstore/hnsw_qdrant.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/vectorstore/hnsw_qdrant.go)
  - [`pkg/drivers/sqlstore/persistent.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/sqlstore/persistent.go)
  - [`pkg/raecore/math/controller.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/math/controller.go)
  - [`pkg/raecore/orchestrator/intelligence.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/orchestrator/intelligence.go) (nowy moduł)
  - [`pkg/drivers/llm/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/drivers/llm/client.go)
- **Zakres wdrożenia:**
  1. **Wielowektorowa przestrzeń `Named Vectors` w `QdrantVectorStore` + `InMemoryVectorStore` + `InMemoryCache`**:
     - `QdrantVectorStore` z obsługą nazwanych przestrzeni wektorowych (`dense` 384d, `ollama` 768d, `openai` 1536d itd.) bez obcinania wymiarów, `EnsureVectorConfig`, `StoreVector`, `BatchStoreVectors`, `SearchSimilarInSpace`, `UpdateVector`, `DeleteVector`, `GetVector`.
     - `InMemoryVectorStore` oraz `InMemoryCache` (z TTL, `Get`, `Set`, `Delete`, `Exists`, `Clear`).
  2. **Adaptery `SQLite RAE-Lite` (`SQLiteStorage`, `SQLiteVectorStore`, `SQLiteGraphStore`, `PostgreSQLStorage`, `InMemoryStorage`)**:
     - Pełna obsługa operacji pamięci, wyszukiwania FTS, wektorów i grafu sąsiedztwa (`CreateNode`, `CreateEdge`, `GetNeighbors`, `DeleteNode`).
  3. **Rozbudowa `MultiArmedBandit`, `RAEEngine`, `RAERuntime`, `Reflector` (`rae_core`)**:
     - `MultiArmedBandit` z UCB1, Context-Aware UCB, Thompson Sampling (`SampleThompson`), `DecayExploration`, `SaveWeights`/`LoadWeights`.
     - `Reflector` (`EvaluateAnswerDraft`, `GenerateReflection`, `IdentifyReflectionCandidates`), `RAEEngine`, `RAERuntime`.
  4. **Inteligencja `orchestrator/` i Providerzy `apps/llm/`**:
     - `ModelRouterV2`, `AdaptiveModelRouter`, `PerformanceAnalytics`, `PerformanceTracker`, `OrchestratorTelemetry`, `QualityGate`, `RunLogger`, `Orchestrator`.
     - `LLMRouter` z metodami `Complete`, `Stream`, `Embed`, `EmbedBatch` oraz natywne struktury providerów `QwenProvider`, `GeminiProvider`, `AnthropicProvider`, `OllamaProvider`, `DeepSeekProvider`.

---

### Iteracja 5: Grupa 5 — Silniki Aplikacyjne `rae-portal`, `rae-quality` V2, SDK oraz Narzędzia CI (`scripts/ci/`) + Weryfikacja i Raport Końcowy
- **Pliki docelowe:**
  - [`services/portal/deep_apps.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/deep_apps.go) (nowy moduł)
  - [`services/portal/service.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/portal/service.go)
  - [`services/quality/deep_engines.go`](file:///home/grzegorz/cloud/RAE-Suite-go/services/quality/deep_engines.go)
  - [`pkg/raesdk/client.go`](file:///home/grzegorz/cloud/RAE-Suite-go/pkg/raesdk/client.go) (nowy moduł SDK + CI tools)
  - [`tests/parity/v49_deep_groups_test.go`](file:///home/grzegorz/cloud/RAE-Suite-go/tests/parity/v49_deep_groups_test.go) (nowy zestaw testów dla wszystkich 5 grup)
- **Zakres wdrożenia:**
  1. **Aplikacje `rae-portal`**:
     - `PDKnowledgeHubApp` (`SearchISAPAndWeb`, `ExtractDocumentText`, `ChunkText`, `HandleFileUpload`, `DeleteDocumentChunks`),
     - `RAESuitePortal` (`DetectSystem`, `SetPage`, `OpenInspector`, `StartWebSocketClient`, `Render`),
     - `CentralAuditorApp` (`VerifyEventSignature`, `DetectSecrets`, `TriggerSentinelAudit`, `SignEvent`, `Render`),
     - `PhoenixRepairApp` (`SimulateRepair`, `ApplyRepair`, `TriggerSimulatedIncident`, `TickIncidentTimer`, `Render`),
     - `MissionControlApp` (`FetchData`, `RenderContent`, `Render`).
  2. **Silniki `rae-quality` V2**:
     - `SecurityGateway` (`ComputeFindingFingerprint`, `GetCompatibleScanners`, `ScanCode`, `ToSARIFReport`),
     - `RAEQualityGatewayV2` (`GetPolicy`, `AnalyzeDiff`, `SecurityScan`, `GenerateTests`, `VerifyFix`),
     - `RiskCoverageEngine`.
  3. **Klienci SDK i Narzędzia CI/Infra**:
     - `MemoryClient` (`sdk/python/rae_memory_sdk`), `RAEAPIClient`, `RAEClient`, `MLServiceClient`,
     - `FailureAnalyzer` (`LoadArtifacts`, `ClassifyFailure`, `ExtractContext`), `CIFixAgent` (`GenerateFix`, `SaveFix`, `GetCostSummary`), `NodeAgent` (`Register`, `HeartbeatLoop`, `ProcessTask`, `TaskLoop`, `Run`).
  4. **Testy, Spis Powszechny AST, Audyt 5-Warstwowy i Raport Końcowy**:
     - Uruchomienie pełnej suity testowej (`go test ./...`), 5-warstwowego audytora (`make audit-5stage`) oraz spisu powszechnego AST (`exhaustive_python_to_go_census.py`),
     - Zapis raportu wykonania w [`docs/RAPORT_WDROZENIA_5_GRUP_PARZYSTOSCI_V4_9_0.md`](file:///home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_WDROZENIA_5_GRUP_PARZYSTOSCI_V4_9_0.md).
