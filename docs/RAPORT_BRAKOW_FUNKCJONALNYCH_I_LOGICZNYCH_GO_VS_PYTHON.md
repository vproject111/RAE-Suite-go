# Raport Audytu Braków Funkcjonalnych i Logicznych: `RAE-Suite` (Python) vs `RAE-Suite-go` (`v4.6.0`)

**Data audytu:** `2026-10-19`  
**Zakres inspekcji:** Głęboka analiza porównawcza AST oraz przepływów sterowania i danych (data-flow / control-flow) pomiędzy pełnym kodem źródłowym Pythona (`/home/grzegorz/cloud/RAE-Suite` + `/home/grzegorz/cloud/RAE-CRL`) a implementacją w języku Go (`/home/grzegorz/cloud/RAE-Suite-go` w wersji `v4.6.0`).

---

## 1. Cel i Metodologia Audytu

W wersji `RAE-Suite-go v4.6.0` osiągnięto 100% pokrycia kontraktów powierzchniowych HTTP i CLI (`216/216 PASS`). Celem niniejszego audytu było jednak zejście **poniżej poziomu samych kodów odpowiedzi HTTP 200 OK** i zbadanie:
1. **Braków logicznych w implementacji Go**: miejsc, w których endpointy lub metody w Go zwracają zahardkodowane/statyczne struktury JSON, nie są połączone z żywymi magazynami danych (`UnifiedStore`, `AgnosticStorageBroker`, `ExecutionAuditLedger`, `MathController`, `TemporalGraphService`), albo pomijają wieloetapową logikę obliczeniową obecną w Pythonie.
2. **Braków funkcjonalnych i algorytmicznych względem Pythona**: wewnętrznych klas i silników z `apps/memory_api/core/*`, `apps/memory_api/services/*`, `apps/memory_api/workers/*`, `packages/rae-core/rae_core/*`, `packages/rae-phoenix/feniks/*` oraz `packages/rae-lab/*`, które w Pythonie odpowiadają za pełne zachowanie poznawcze i zarządcze (MDP, topologia grafu PageRank/Betweenness/Modularity, automat konsolidacji FSM + DreamingWorker, RulesEngine, PolicyVersioning, Retention GDPR/ISO, 6-czynnikowy ImportanceScoring, QueryClassifier + StrategyRouter, ConsistencyService).

W wyniku szczegółowej inspekcji wykryto **10 istotnych braków funkcjonalnych i logicznych (`L1`–`L10`)**, opisanych poniżej wraz z dokładnymi lokalizacjami w plikach `.py` i `.go`.

---

## 2. Szczegółowy Wykaz Wykrytych Braków Logicznych i Funkcjonalnych (`L1`–`L10`)

### Część A: Braki Logiczne w Istniejących Modułach Go (Statyczne Odpowiedzi i Odłączone Przepływy Danych)

---

### Brak `L1`: Odłączone („Ślepe”) i Zahardkodowane Silniki `RAEGlobalSearchEngine`, `ProjectPhaseAnalyzer` oraz `GitOpsDaemon`
- **Lokalizacja w Go (`v4.6.0`)**: [`pkg/raecore/ops/gitops_phases_search.go`](../pkg/raecore/ops/gitops_phases_search.go) (linie `38–211`) oraz [`services/orchestrator/ops_routes.go`](../services/orchestrator/ops_routes.go) (linie `328–363`).
- **Lokalizacja w Pythonie**: `RAE-Suite/core/portal_global_search.py` (131 linii), `core/project_phase_analyzer.py` (133 linie), `core/gitops_daemon.py` (142 linie).
- **Na czym polega błąd logiczny w Go**:
  1. `ExecuteGlobalSearch(query, project)` w `pkg/raecore/ops/gitops_phases_search.go` (linie `166–211`) **nie wykonuje żadnego rzeczywistego wyszukiwania** — zwraca 4 zahardkodowane na sztywno rekordy (`mem-global-1`, `crl-global-1`, `audit-global-1`, `quality-global-1`) z wklejonym tekstem zapytania `query`. Nie odpytuje ani `RAE-Memory`, ani `RAE-CRL`, ani `ExecutionAuditLedger`, ani `RAE-Quality`.
  2. `AnalyzeProjectPhases(project, entries)` (linie `96–106`) w przypadku wywołania bez ręcznie przekazanej tablicy `entries` podstawia 3 sztuczne wpisy zamiast pobrać i zagregować rzeczywiste paragony z żywego rejestru `s.auditLedger.ListReceipts()` w `Orchestratorze`.
  3. `GitOpsDaemon.ExecuteGitOpsWorkflow` generuje jedynie szablon tekstowy bez odczytania prawdziwego stanu repozytorium Git (`HEAD` SHA, bieżąca gałąź, zmodyfikowane pliki).

---

### Brak `L2`: Statyczne („Wydmuszkowe”) Handlery w `services/memory/extended_routes.go` (Ewaluacja IR, Detekcja Dryfu KL/JSD, Testy A/B, Analiza Zapytań, Tuning MAB, Relacje Refleksji i Kolejka Zadań Węzłów)
- **Lokalizacja w Go (`v4.6.0`)**: [`services/memory/extended_routes.go`](../services/memory/extended_routes.go) (linie `203–213`, `594–682`, `704–726`, `761–813`).
- **Lokalizacja w Pythonie**: `apps/memory_api/services/evaluation_service.py`, `drift_detector.py`, `hybrid_search_service.py`, `tuning_service.py`, `feedback_service.py`, `analytics.py`.
- **Na czym polega błąd logiczny w Go**:
  1. **`/v2/evaluation/search` (linia 594)**: zwraca stały literał `{"mrr": 1.0, "ndcg": 0.98, "hit_rate": 1.0, "precision_at_5": 0.95}` zamiast faktycznie uruchomić wyszukiwanie w `s.store.QueryMemories` i wyliczyć prawdziwe `MRR`, `NDCG@k`, `Precision@k`, `Recall@k` oraz `HitRate` względem `relevant_ids` lub zbioru ewaluacyjnego.
  2. **`/v2/evaluation/drift/detect` (linia 600)**: zwraca stałe `{"drift_detected": false, "kl_divergence": 0.012, "severity": "NONE"}` zamiast policzyć rzeczywistą dywergencję Kullbacka-Leiblera (`KL Divergence`), dywergencję Jensena-Shannona (`JSD`) i `PSI` na podstawie rozkładu warstw, ważności i wektorów wspomnień w `s.store`.
  3. **`/v2/evaluation/ab-test/` (linia 613)**: zwraca stałe `{"winner": "fluid_100", "p_value": 0.004}` zamiast zarejestrować obserwacje wariantów A/B i wyliczyć statystykę testową `z-score` / `p-value` oraz przedział ufności.
  4. **`/v2/search/analyze` (linia 650), `/v2/search/weights/calculate` (linia 668), `/v2/search/compare` (linia 671)**: zwracają stałe wartości (`"recommended_strategy": "hybrid"`, `{"alpha": 0.40, "beta": 0.30, "gamma": 0.30}`, `"winner": "hybrid"`) zamiast klasyfikować zapytanie, wyliczać dynamiczne wagi i faktycznie porównywać wyniki oraz czasy wykonania strategii `sparse`, `vector`, `graph`, `hybrid` na żywych danych `s.store`.
  5. **`/v2/feedback` (linia 203) i `/v2/tuning/run` (linia 811)**: `/v2/feedback` nie przekazuje nagrody `Reward` do `MathController.UpdateReward(armID, reward)`, a `/v2/tuning/run` zwraca stałe `"fluid_100"` zamiast wybrać najlepsze ramię Bandita (`SelectArm()` UCB1) i zaktualizować profil wag.
  6. **`/v2/reflections/relationships` (linia 704) i `/v2/reflections/batch` (linia 707)**: zwracają jedynie `{"status": "linked"}` i `{"status": "deleted"}` bez faktycznego utworzenia krawędzi w grafie (`s.store.AddGraphEdge`) ani usunięcia wskazanych identyfikatorów refleksji (`s.store.DeleteMemory`).
  7. **`/v2/nodes/tasks_poll` (linia 724)**: zwraca zawsze pustą listę `{"tasks": []}` zamiast pobrać z `ext.catalog` zadania w stanie `PENDING` dla danego `node_id`.
  8. **`/v2/dashboard/activity`, `/v2/dashboard/metrics/timeseries/`, `/v2/dashboard/compliance/*` (linie 761–794)**: zwracają stałe liczby zamiast rzeczywistych liczników z `s.store`, `ext.executions`, `s.guard` i `s.provenanceSvc`.

---

### Brak `L3`: Brak Spięcia End-to-End Mechanizmów `G1`–`G6` z Głównym Potokiem `StoreMemory`, `QueryMemories`, `Graph` i `RAE-Mesh`
- **Lokalizacja w Go (`v4.6.0`)**: [`services/memory/service.go`](../services/memory/service.go), [`services/memory/v46_deep_routes.go`](../services/memory/v46_deep_routes.go), [`pkg/raemesh/mesh.go`](../pkg/raemesh/mesh.go).
- **Na czym polega błąd logiczny w Go**:
  1. W `v4.6.0` instancje `ibSelector`, `temporalGraph`, `provenanceSvc` i `meshCrypto` są tworzone jako zmienne lokalne wewnątrz funkcji `registerV46Routes(mux)` w `v46_deep_routes.go`, przez co główny serwis `memory.Service` nie ma do nich dostępu!
  2. Dodawanie węzłów i krawędzi przez `/v2/graph/nodes`, `/v2/graph/edges` oraz `/v2/memories/ingest-document` nie rejestruje zmian w `TemporalGraphService` (`RecordChange`) ani nie wykonuje kanonicznej normalizacji synonimów przez `EntityResolutionService`.
  3. Synchronizacja `RAE-Mesh` (`SyncWithPeer` / `ReceiveMemories`) nie korzysta z warstwy `E2EEncryption` (`EncryptBatch` / `DecryptBatch`), gdy włączony jest klucz szyfrowania Mesh.
  4. Główne wyszukiwanie `/v2/memories/query` oraz `/v2/search/hybrid` nie udostępnia wbudowanej integracji z `InformationBottleneckSelector` (`use_ib_selector`), `QueryRewriter` (`two_pass`) i `EvidenceConflictDetector` (`detect_conflicts`).

---

### Część B: Braki Funkcjonalne i Algorytmiczne względem Pythona

---

### Brak `L4`: Brak Sformalizowanego Modelu Decyzyjnego `MDP` (`RAEState`, 11 Akcji `ActionType`, `RewardFunction` i `ActionExecutor`)
- **Lokalizacja w Pythonie**: `apps/memory_api/core/state.py` (483 linie), `core/actions.py` (744 linie), `core/reward.py` (405 linii), `core/action_executor.py` (390 linii), `core/metrics.py` (471 linii) — łącznie **2 493 linie kodu**.
- **Czego brakuje w Go (`v4.6.0`)**:
  - Modelowania operacji pamięciowych jako Procesu Decyzyjnego Markowa (`MDP`):
    - **Stan `RAEState`**: `WorkingContext` (wykorzystanie budżetu tokenów, gęstość informacyjna), `MemoryState` (liczebność, średnia ważność i entropia Shannona w każdej z 8 warstw), `BudgetState` (pozostałe tokeny, USD i budżet latencji ms), `GraphState` (węzły, krawędzie, gęstość, spójne składowe).
    - **11 Akcji `ActionType`**: `RETRIEVE_EPISODIC`, `RETRIEVE_WORKING`, `RETRIEVE_SEMANTIC`, `RETRIEVE_LTM`, `RETRIEVE_REFLECTIVE`, `CALL_LLM`, `PRUNE_CONTEXT`, `GENERATE_REFLECTION`, `UPDATE_GRAPH`, `CONSOLIDATE_EPISODIC_TO_WORKING`, `SUMMARIZE_CONTEXT` — wraz z predykcją kosztu i walidacją warunków wstępnych (`CanExecute(state)`).
    - **Funkcja Nagrody `RewardFunction`**:
      $$R(S_t, A_t, S_{t+1}) = \text{QualityGain}(S_t, S_{t+1}) - \lambda \cdot \text{TokenCost}(A_t) - \mu \cdot \text{LatencyCost}(A_t) + \nu \cdot \text{SafetyBonus}(S_{t+1})$$
    - **`ActionExecutor` i `MDPMetrics`**: wybór optymalnej akcji maksymalizującej oczekiwaną użyteczność, wykonanie jej na żywym `memory.Service`, transformacja $S_t \to S_{t+1}$ i rejestracja metryk przejścia.

---

### Brak `L5`: Brak Zaawansowanych Algorytmów Topologii Grafu Wiedzy (`PageRank`, `BetweennessCentrality`, `ClusteringCoefficient`, `Modularity Q`, `ExtractSubgraph`)
- **Lokalizacja w Pythonie**: `apps/memory_api/services/graph_algorithms.py` (742 linie) oraz `services/graph_topology_engine.py` (188 linii) — łącznie **930 linii kodu**.
- **Czego brakuje w Go (`v4.6.0`)**:
  - Iteracyjnego algorytmu **`PageRank`** ($d = 0.85$) wyznaczającego centralność i autorytet węzłów w grafie wiedzy,
  - Algorytmu Brandesa **`BetweennessCentrality`** wykrywającego węzły-mosty łączące odległe klastry tematyczne,
  - Współczynnika gronowania **`ClusteringCoefficient`** (lokalnego dla węzła i średniego globalnego),
  - Miary modularności Newmana-Girvana **`Modularity Q`** oceniającej jakość podziału grafu na społeczności,
  - Ekstrakcji k-hopowego podgrafu **`ExtractSubgraph(centerNodes, maxDepth)`** wraz z metrykami gęstości i centralności.

---

### Brak `L6`: Brak 7-Stanowego Automatu Konsolidacji (`ConsolidationFSM`), `DreamingWorker` (Synteza Analogii Offline) i `CommunitySummarizer`
- **Lokalizacja w Pythonie**: `packages/rae-core/rae_core/reflection/consolidation_fsm.py` (127 linii), `reflection/community_summarizer.py` (112 linii), `apps/memory_api/workers/memory_maintenance.py` (723 linie), `workers/community_worker.py` (244 linie) — łącznie **1 206 linii kodu**.
- **Czego brakuje w Go (`v4.6.0`)**:
  - Formalnego automatu skończonego **`ConsolidationFSM`** z 7 stanami cyklu życia wspomnienia: `NEW -> ACTIVE -> CONSOLIDATING -> CONSOLIDATED -> DECAYING -> ARCHIVED -> PRUNED`,
  - **`DreamingWorker`**: procesu cyklicznej syntezy offline („śnienia”), który analizuje wspomnienia o wysokiej ważności z różnych sesji/epizodów, znajduje wspólne motywy międzydomenowe i generuje nowe artefakty `dream_insight` w warstwie `reflective`,
  - **`SummarizationWorker` & `CommunitySummarizer`**: automatycznej syntezy podsumowań sesji (`SessionSummary`) oraz podsumowań społeczności grafowych (`CommunitySummary`).

---

### Brak `L7`: Brak 4 Silników Zarządzania i Zgodności: `RulesEngine`, `PolicyVersioningService`, `RetentionService` (RODO/GDPR + `DeletionAuditEntry`) i `BudgetService`
- **Lokalizacja w Pythonie**: `apps/memory_api/services/rules_engine.py` (673 linie), `policy_versioning_service.py` (379 linii), `retention_service.py` (567 linii), `budget_service.py` (334 linie) — łącznie **1 953 linie kodu**.
- **Czego brakuje w Go (`v4.6.0`)**:
  1. **`RulesEngine`**: deklaratywnego silnika reguł warunkowych (`WHEN field OP value THEN action`, np. `SET_IMPORTANCE`, `ADD_TAG`, `PROMOTE_LAYER`, `BLOCK_INGESTION`, `EMIT_ALERT`) wykonywanego automatycznie podczas ingestii wspomnień,
  2. **`PolicyVersioningService`**: wersjonowania polityk ISO/RAE (`DRAFT -> ACTIVE -> DEPRECATED -> ARCHIVED`) z sumą SHA-256, historią zmian, walidacją zgodności `EnforcePolicy` i natychmiastowym rollbackiem do poprzedniej wersji,
  3. **`RetentionService`**: egzekwowania okresów retencji wg klasyfikacji danych (`PUBLIC`, `INTERNAL`, `CONFIDENTIAL`, `RESTRICTED`), obsługi żądania RODO / GDPR „Right to be Forgotten” (całkowite wymazanie encji i jej śladów z pamięci, wektorów i grafu) oraz generowania niezmiennego pokwitowania `DeletionAuditEntry` z podpisem SHA-256,
  4. **`BudgetService`**: śledzenia i twardego egzekwowania limitów budżetowych per tenant/projekt (dzienny/miesięczny limit USD, tokenów i zapytań, alerty progowe `80%` / `95%` oraz blokada `100%`).

---

### Brak `L8`: Brak 6-Czynnikowego `ImportanceScoringService` oraz Tabeli Cenowej `CostController`
- **Lokalizacja w Pythonie**: `apps/memory_api/services/importance_scoring.py` (631 linii) oraz `services/cost_controller.py` (361 linii) — łącznie **992 linie kodu**.
- **Czego brakuje w Go (`v4.6.0`)**:
  - **`ImportanceScoringService`**: dynamicznego obliczania ważności wspomnienia (`0.0..1.0`) oraz klasy ważności (`CRITICAL`, `HIGH`, `MEDIUM`, `LOW`, `TRIVIAL`) na podstawie **6 czynników (`ScoringFactors`)**:
    1. `Novelty` (unikalność leksykalna/wektorowa względem istniejących wspomnień),
    2. `Relevance` (gęstość pojęć technicznych i architektonicznych),
    3. `RiskIntensity` (obecność słów kluczowych bezpieczeństwa, błędów krytycznych, SLA, ISO),
    4. `Frequency` (liczba odwołań / użyć),
    5. `Recency` (czynnik zaniku czasowego Ebbinghausa),
    6. `GraphCentrality` (stopień powiązania encji w grafie wiedzy),
    wraz z automatycznym wywołaniem przy `StoreMemory`, gdy użytkownik nie narzuci własnej wagi `Importance`.
  - **`CostController`**: precyzyjnej kalkulacji kosztu wejścia/wyjścia (`$/1M prompt tokens`, `$/1M completion tokens`) dla wszystkich 9 modeli Triady + modeli lokalnych `Tier 0` oraz automatycznej rekomendacji najtańszego modelu spełniającego wymagany próg jakości.

---

### Brak `L9`: Brak `QueryClassifier`, `StrategyRouter`, `ConfidenceWeightedFusion`, `NoiseAwareSearchEngine` i `RetrievalOptimizer`
- **Lokalizacja w Pythonie**: `packages/rae-core/rae_core/search/classifier.py` (166 linii), `router.py` (142 linie), `fusion.py` (70 linii), `engine.py` (620 linii), `optimizer.py` (186 linii), `apps/memory_api/services/query_analyzer.py` (407 linii) — łącznie **1 591 linii kodu**.
- **Czego brakuje w Go (`v4.6.0`)**:
  - **`QueryClassifier` & `QueryAnalyzer`**: klasyfikacji zapytania do jednej z 6 klas intencji (`FACTUAL`, `TEMPORAL`, `RELATIONAL_GRAPH`, `PROCEDURAL`, `EXPLORATORY`, `CODE_SYMBOL`), ekstrakcji słów kluczowych, encji i ograniczeń czasowych,
  - **`StrategyRouter`**: automatycznego budowania `RoutingPlan` (dobór wag `alpha`, `beta`, `gamma` oraz wag strategii `bm25`, `vector`, `graph`, `resonance` pod wykrytą intencję zapytania),
  - **`ConfidenceWeightedFusion` & `NoiseAwareSearchEngine`**: fuzji wyników z wielu strategii ważonej współczynnikiem ufności każdej strategii oraz odcinania zaszumionych wyników poniżej dynamicznego progu stosunku sygnału do szumu (`SNR`),
  - **`RetrievalOptimizer`**: dynamicznego wyznaczania trybu dojrzałości (`MaturityMode`: `BOOTSTRAP`, `BALANCED`, `PRECISION`, `HIGH_THROUGHPUT`) na podstawie wielkości korpusu i historycznego MRR.

---

### Brak `L10`: Brak `ConsistencyService` (Spójność Magazyn-Wektory-Graf), Meta-Refleksji `RAE-Phoenix` (`SystemModelBuilder`, `LongitudinalAnalyzer`, `CapabilityDetector`) oraz `FailureMiningEngine` / `SafeRolloutManager` w `RAE-Lab`
- **Lokalizacja w Pythonie**:
  - `apps/memory_api/services/consistency_service.py` (136 linii) + `core/contract.py` (68 linii),
  - `packages/rae-phoenix/work_dir/feniks/feniks/core/reflection/{system_model,longitudinal,capabilities}.py` (876 linii),
  - `packages/rae-lab/src/rae_lab/{failure_mining_engine,safe_rollout_manager}.py` (44 linie) + `core/shadow_guardrail_manager.py` (164 linie).
- **Czego brakuje w Go (`v4.6.0`)**:
  1. **`ConsistencyService` & `SchemaContractValidator`**: audytu i automatycznej naprawy (`VerifyAndRepairConsistency`) niespójności pomiędzy głównym magazynem wspomnień, indeksem wektorowym `NamedVectors` (brakujące/osierocone punkty wektorowe) a grafem wiedzy (krawędzie wskazujące na nieistniejące węzły),
  2. **W `RAE-Phoenix`**: `SystemModelBuilder` (budowa mapy komponentów, zależności i gorących punktów ryzyka z kodu projektu), `LongitudinalAnalyzer` (analiza trendów złożoności, pokrycia i awaryjności w kolejnych migawkach czasu) oraz `CapabilityDetector` (wykrywanie możliwości architektonicznych i wzorców w kodzie),
  3. **W `RAE-Lab`**: `FailureMiningEngine` (automatyczna ekstrakcja wzorców regex/AST z serii awarii i promocja reguł `CANDIDATE -> SHADOW -> ACTIVE`) oraz `SafeRolloutManager` (wieloetapowy rollout `10% -> 25% -> 50% -> 100%` z bramką regresji).

---

## 3. Podsumowanie Tabelaryczne Braków (`L1`–`L10`) — Status po Wdrożeniu `RAE-Suite-go v4.7.0`

> **Aktualizacja po realizacji planu (`v4.7.0`):** Wszystkie 10 braków funkcjonalnych i logicznych (`L1`–`L10`) zostało w 100% wyeliminowanych i zweryfikowanych na żywo (`241/241 PASS`). Szczegóły wykonania znajdują się w [`RAPORT_WYKONANIA_PLANU_NAPRAWY_V4_7_0.md`](./RAPORT_WYKONANIA_PLANU_NAPRAWY_V4_7_0.md).

| ID | Kategoria | Nazwa Braku / Rozbieżności | Pliki w Pythonie (`RAE-Suite`) | Wdrożone Pliki w Go (`RAE-Suite-go v4.7.0`) | Status w `v4.7.0` |
| :---: | :---: | :--- | :--- | :--- | :---: |
| **`L1`** | **Logiczny** | Zahardkodowane `RAEGlobalSearchEngine`, niepodpięty `ProjectPhaseAnalyzer` i statyczny `GitOpsDaemon` | `core/portal_global_search.py`, `project_phase_analyzer.py`, `gitops_daemon.py` | `pkg/raecore/ops/gitops_phases_search.go`, `services/orchestrator/ops_routes.go` | **NAPRAWIONE (`PASS`)** |
| **`L2`** | **Logiczny** | Statyczne handlery w `extended_routes.go` (Ewaluacja IR, Dryf KL/JSD, A/B Test, Search Analyze/Compare, Tuning MAB, Relacje Refleksji, Tasks Poll) | `services/evaluation_service.py`, `drift_detector.py`, `tuning_service.py`, `feedback_service.py` | `services/memory/extended_routes.go`, `pkg/raecore/search/evaluator_drift.go` | **NAPRAWIONE (`PASS`)** |
| **`L3`** | **Logiczny** | Brak spięcia end-to-end `G1`–`G6` (`IBSelector`, `TemporalGraph`, `EntityResolution`, `Provenance`, `E2EEncryption`, `ConflictDetector`/`QueryRewriter`) z głównym potokiem `memory.Service` i `MeshEngine` | `rae_core_service.py`, `mesh_service.py`, `hybrid_search.py` | `services/memory/service.go`, `services/memory/v46_deep_routes.go`, `pkg/raemesh/mesh.go` | **NAPRAWIONE (`PASS`)** |
| **`L4`** | **Funkcjonalny** | Brak modelu decyzyjnego `MDP` (`RAEState`, 11 akcji `ActionType`, `RewardFunction`, `ActionExecutor`, `MDPMetrics`) | `apps/memory_api/core/{state,actions,reward,action_executor,metrics}.py` | `pkg/raecore/mdp/mdp.go`, `pkg/raecore/mdp/mdp_test.go` | **NAPRAWIONE (`PASS`)** |
| **`L5`** | **Funkcjonalny** | Brak algorytmów topologii grafu wiedzy (`PageRank`, `BetweennessCentrality`, `ClusteringCoefficient`, `Modularity Q`, `ExtractSubgraph`) | `services/graph_algorithms.py`, `graph_topology_engine.py` | `pkg/raecore/graph/algorithms.go`, `pkg/raecore/graph/algorithms_test.go` | **NAPRAWIONE (`PASS`)** |
| **`L6`** | **Funkcjonalny** | Brak 7-stanowej maszyny `ConsolidationFSM`, `DreamingWorker` i `CommunitySummarizer` / `SummarizationWorker` | `rae_core/reflection/consolidation_fsm.py`, `community_summarizer.py`, `workers/memory_maintenance.py` | `pkg/raecore/reflection/lifecycle_dreaming.go`, `pkg/raecore/reflection/lifecycle_dreaming_test.go` | **NAPRAWIONE (`PASS`)** |
| **`L7`** | **Funkcjonalny** | Brak `RulesEngine`, `PolicyVersioningService`, `RetentionService` (RODO/GDPR + `DeletionAuditEntry`) i `BudgetService` | `services/{rules_engine,policy_versioning_service,retention_service,budget_service}.py` | `pkg/raecore/security/governance_engines.go`, `pkg/raecore/security/governance_engines_test.go` | **NAPRAWIONE (`PASS`)** |
| **`L8`** | **Funkcjonalny** | Brak 6-czynnikowego `ImportanceScoringService` (z auto-scoringiem przy `StoreMemory`) i tabeli cenowej `CostController` | `services/importance_scoring.py`, `cost_controller.py` | `pkg/raecore/math/importance_cost.go`, `pkg/raecore/math/importance_cost_test.go` | **NAPRAWIONE (`PASS`)** |
| **`L9`** | **Funkcjonalny** | Brak `QueryClassifier`, `StrategyRouter`, `ConfidenceWeightedFusion`, `NoiseAwareSearchEngine` i `RetrievalOptimizer` | `rae_core/search/{classifier,router,fusion,engine,optimizer}.py`, `services/query_analyzer.py` | `pkg/raecore/search/classifier_router_fusion.go`, `pkg/raecore/search/classifier_router_fusion_test.go` | **NAPRAWIONE (`PASS`)** |
| **`L10`** | **Funkcjonalny** | Brak `ConsistencyService` (Magazyn-Wektory-Graf), `SystemModelBuilder` / `LongitudinalAnalyzer` / `CapabilityDetector` w `Phoenix` oraz `FailureMiningEngine` / `SafeRolloutManager` w `Lab` | `services/consistency_service.py`, `feniks/core/reflection/*`, `rae_lab/*` | `services/memory/v47_deep_routes.go`, `services/phoenix/meta_reflection.go`, `services/lab/mining_rollout.go` | **NAPRAWIONE (`PASS`)** |

