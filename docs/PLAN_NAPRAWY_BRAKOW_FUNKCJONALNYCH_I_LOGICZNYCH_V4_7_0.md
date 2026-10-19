# Plan Iteracyjny Naprawy Braków Funkcjonalnych i Logicznych w `RAE-Suite-go` (`v4.7.0`)

**Data utworzenia planu:** `2026-10-19`  
**Dokument źródłowy audytu:** [`RAPORT_BRAKOW_FUNKCJONALNYCH_I_LOGICZNYCH_GO_VS_PYTHON.md`](./RAPORT_BRAKOW_FUNKCJONALNYCH_I_LOGICZNYCH_GO_VS_PYTHON.md)  
**Repozytorium docelowe:** `/home/grzegorz/cloud/RAE-Suite-go` (`https://github.com/vproject111/RAE-Suite-go`)  
**Cel nadrzędny:** Całkowite wyeliminowanie wszystkich 10 braków logicznych i funkcjonalnych (`L1`–`L10`) wykrytych podczas głębokiego audytu kodu Go `v4.6.0` względem Pythona (`RAE-Suite` + `RAE-CRL`), zastąpienie wszystkich statycznych/wydmuszkowych handlerów prawdziwymi obliczeniami na żywych strukturach danych oraz wdrożenie brakujących silników matematycznych, grafowych, poznawczych (MDP, FSM, Dreaming) i zarządczych (Rules, PolicyVersioning, Retention GDPR/ISO, Budget, Consistency).

---

## Podział na Iteracje (`Iteracja G1` – `Iteracja G4`)

### Iteracja G1: Naprawa Braków Logicznych (`L1`, `L2`, `L3`) oraz Wdrożenie `QueryClassifier`, `StrategyRouter` i `ConfidenceWeightedFusion` (`L9`) (`feature/g1-live-wiring-search-classifier-ir-eval-drift-and-global-search`)

1. **`L9` — Klasyfikator Zapytań, Router Strategii i Fuzja Ważona Ufnością (`pkg/raecore/search/classifier_router_fusion.go`)**:
   - `ClassifyQuery(query)`: klasyfikacja intencji zapytania do 6 klas (`FACTUAL`, `TEMPORAL`, `RELATIONAL_GRAPH`, `PROCEDURAL`, `EXPLORATORY`, `CODE_SYMBOL`), wyznaczenie złożoności (`0.0..1.0`), słów kluczowych i encji.
   - `BuildRoutingPlan(query)`: dynamiczny dobór wag `Alpha` (podobieństwo semantyczne), `Beta` (ważność), `Gamma` (świeżość czasowa) oraz wag poszczególnych strategii (`sparse`, `vector`, `graph`, `resonance`, `fulltext`).
   - `ConfidenceWeightedFuse(strategyResults, confidences)` + `FilterNoiseBySNR(items, minSNR)`: fuzja wyników ważona ufnością strategii oraz odfiltrowanie szumu poniżej progu `SNR`.
   - `OptimizeRetrievalMaturity(corpusSize, historicalMRR)`: automatyczny dobór `MaturityMode` (`BOOTSTRAP`, `BALANCED`, `PRECISION`, `HIGH_THROUGHPUT`).
2. **`L2` — Zastąpienie Statycznych Handlerów w `services/memory/extended_routes.go` Żywymi Silnikami Obliczeniowymi (`pkg/raecore/search/evaluator_drift.go`)**:
   - `/v2/evaluation/search`: rzeczywiste obliczenie metryk IR (`MRR`, `NDCG@k`, `Precision@k`, `Recall@k`, `HitRate`) na wynikach `s.store.QueryMemories` względem `relevant_ids` lub automatycznych trafień referencyjnych.
   - `/v2/evaluation/drift/detect`: rzeczywiste obliczenie dywergencji Kullbacka-Leiblera (`KL Divergence`), dywergencji Jensena-Shannona (`JSD`) i `PSI` pomiędzy rozkładem bazowym a aktualnym rozkładem warstw i ważności wspomnień w `s.store`.
   - `/v2/evaluation/ab-test/`: rejestracja próbek nagród wariantów A i B oraz wyliczenie statystyki `z-score`, `p-value` i zwycięskiego wariantu.
   - `/v2/search/analyze`, `/v2/search/weights/calculate`, `/v2/search/compare`: podpięcie pod żywy `ClassifyQuery`, `BuildRoutingPlan` i rzeczywiste wykonanie porównania strategii `sparse`, `vector`, `graph`, `hybrid` na `s.store`.
   - `/v2/feedback` i `/v2/tuning/run`: rzeczywista aktualizacja Multi-Armed Bandita (`s.mathCtrl.UpdateReward`) i wybór najlepszego ramienia wg `UCB1` (`s.mathCtrl.SelectArm()`).
   - `/v2/reflections/relationships`, `/v2/reflections/batch`, `/v2/nodes/tasks_poll`, `/v2/dashboard/*`: operacje na żywym grafie `s.store.AddGraphEdge`, usuwanie z `s.store.DeleteMemory`, filtrowanie zadań `PENDING` z `ext.catalog` i agregacja żywych metryk.
3. **`L1` — Podpięcie `RAEGlobalSearchEngine`, `ProjectPhaseAnalyzer` i `GitOpsDaemon` pod Żywe Dane (`pkg/raecore/ops/gitops_phases_search.go` + `services/orchestrator/ops_routes.go`)**:
   - `ExecuteGlobalSearch`: rzeczywiste przeszukiwanie żywych wspomnień `RAE-Memory`, artefaktów `RAE-CRL` (`crlSvc.ListArtifacts`), paragonów `ExecutionAuditLedger` (`auditLedger.ListReceipts`) oraz katalogu 210 tematów `RAE-Quality` (`qualitySvc.PolicyCatalog()`).
   - `AnalyzeProjectPhases`: automatyczna agregacja prawdziwych paragonów z `s.auditLedger.ListReceipts()` z klasyfikacją zadań do 5 faz SDLC (`discovery`, `architecture`, `implementation`, `verification`, `release`).
   - `GitOpsDaemon.ExecuteGitOpsWorkflow`: odczyt rzeczywistego `HEAD` SHA (`git rev-parse --short HEAD`), aktualnej gałęzi i statusu drzewa roboczego (`git status --porcelain`).
4. **`L3` — Spięcie End-to-End Mechanizmów `G1`–`G6` z Głównym Potokiem `memory.Service` i `raemesh.MeshEngine`**:
   - Przeniesienie `ibSelector`, `temporalGraph`, `provenanceSvc` i `meshCrypto` jako pełnoprawnych pól struktury `memory.Service`.
   - Automatyczne rejestrowanie zmian `GraphChange` w `s.temporalGraph` oraz kanoniczna normalizacja encji przez `EntityResolution` przy `/v2/graph/nodes`, `/v2/graph/edges` i `/v2/memories/ingest-document`.
   - Wbudowanie opcjonalnego szyfrowania `E2EEncryption` bezpośrednio w `MeshEngine.SyncWithPeer` i `MeshEngine.ReceiveMemories`.

---

### Iteracja G2: Model Decyzyjny `MDP` (`L4`), Algorytmy Topologii Grafu `PageRank / Betweenness / Modularity` (`L5`) oraz `ConsolidationFSM + DreamingWorker + CommunitySummarizer` (`L6`) (`feature/g2-mdp-engine-graph-algorithms-fsm-dreaming-and-community`)

1. **`L4` — Proces Decyzyjny Markowa `MDP` (`pkg/raecore/mdp/mdp.go` + `mdp_test.go`)**:
   - Struktury stanu `RAEState` (`WorkingContext`, `MemoryState` z entropią Shannona po 8 warstwach, `BudgetState`, `GraphState`).
   - Przestrzeń 11 akcji `ActionType` (`RETRIEVE_EPISODIC`, `RETRIEVE_WORKING`, `RETRIEVE_SEMANTIC`, `RETRIEVE_LTM`, `RETRIEVE_REFLECTIVE`, `CALL_LLM`, `PRUNE_CONTEXT`, `GENERATE_REFLECTION`, `UPDATE_GRAPH`, `CONSOLIDATE_EPISODIC_TO_WORKING`, `SUMMARIZE_CONTEXT`) z walidacją dopuszczalności i estymacją kosztu.
   - Funkcja nagrody $R(S_t, A_t, S_{t+1}) = \text{QualityGain} - \lambda \cdot \text{TokenCost} - \mu \cdot \text{LatencyCost} + \nu \cdot \text{SafetyBonus}$.
   - `ActionExecutor` oraz `MDPMetrics` (`POST /v2/mdp/step`, `GET /v2/mdp/state`).
2. **`L5` — Algorytmy Topologii Grafu Wiedzy (`pkg/raecore/graph/algorithms.go` + `algorithms_test.go`)**:
   - `ComputePageRank(adj, damping, iterations)`: iteracyjny PageRank z obsługą wiszących węzłów (dangling nodes).
   - `ComputeBetweennessCentrality(adj)`: centralność pośrednictwa Brandesa identyfikująca węzły-mosty.
   - `ComputeClusteringCoefficients(adj)`: lokalne i globalne współczynniki gronowania.
   - `ComputeModularityQ(adj, communities)`: modularność Newmana-Girvana $Q$.
   - `ExtractSubgraph(adj, centerNodes, maxDepth)`: ekstrakcja k-hopowego podgrafu (`POST /v2/graph/algorithms/analyze`, `POST /v2/graph/subgraph`).
3. **`L6` — 7-Stanowy `ConsolidationFSM`, `DreamingWorker` i `CommunitySummarizer` (`pkg/raecore/reflection/lifecycle_dreaming.go` + `lifecycle_dreaming_test.go`)**:
   - `ConsolidationFSM`: deterministyczne przejścia `NEW -> ACTIVE -> CONSOLIDATING -> CONSOLIDATED -> DECAYING -> ARCHIVED -> PRUNED`.
   - `RunDreamingCycle`: wykrywanie powtarzających się motywów pomiędzy różnymi epizodami/sesjami i synteza artefaktów `dream_insight` w warstwie `reflective`.
   - `SummarizeCommunities` i `SummarizeSession`: generowanie ustrukturyzowanych podsumowań dla klastrów grafowych i sesji (`POST /v2/memories/lifecycle-fsm`, `POST /v2/reflections/dream`, `POST /v2/reflections/summarize-session`).

---

### Iteracja G3: Silniki Governance (`RulesEngine`, `PolicyVersioning`, `Retention GDPR/ISO`, `BudgetService` — `L7`), 6-Czynnikowy `ImportanceScoring + CostController` (`L8`) oraz `ConsistencyService + Phoenix MetaReflection + Lab Mining/Rollout` (`L10`) (`feature/g3-governance-rules-policy-retention-budget-importance-consistency`)

1. **`L7` — 4 Silniki Governance i Zgodności (`pkg/raecore/security/governance_engines.go` + `governance_engines_test.go`)**:
   - `RulesEngine`: deklaratywne reguły `WHEN <field> <op> <value> THEN <action>` wpięte w potok zapisu wspomnień (`POST /v2/governance/rules`).
   - `PolicyVersioningService`: cykl życia polityk `DRAFT -> ACTIVE -> DEPRECATED -> ARCHIVED`, suma SHA-256 i rollback (`POST /v2/governance/policies`).
   - `RetentionService`: polityki TTL wg `InfoClass` + RODO/GDPR `ExecuteRightToBeForgotten` (kaskadowe wymazanie wspomnienia z magazynu, wektorów i grafu + kryptograficzny `DeletionAuditEntry` SHA-256) (`POST /v2/governance/retention/forget`).
   - `BudgetService`: egzekwowanie limitów USD/tokenów per tenant/projekt z progami `80% WARN`, `95% CRITICAL`, `100% HARD_BLOCK` (`POST /v2/governance/budgets/check`).
2. **`L8` — 6-Czynnikowy `ImportanceScoringService` i `CostController` (`pkg/raecore/math/importance_cost.go` + `importance_cost_test.go`)**:
   - `ComputeSixFactorImportance`: wyliczanie ważności (`0.0..1.0` + `CRITICAL..TRIVIAL`) z 6 czynników (`Novelty`, `Relevance`, `RiskIntensity`, `Frequency`, `Recency`, `GraphCentrality`) oraz automatyczne wywołanie w `StoreMemory` dla wspomnień bez zadanej ważności (`POST /v2/math/importance-score`).
   - `CostController`: kalkulator kosztów tokenów wejścia/wyjścia dla wszystkich 9 modeli Triady + Tier 0 i dobór optymalnego modelu do budżetu (`POST /v2/math/cost-estimate`).
3. **`L10` — `ConsistencyService`, `Phoenix Meta-Reflection` i `Lab FailureMining + SafeRollout` (`services/memory/v47_deep_routes.go`, `services/phoenix/meta_reflection.go`, `services/lab/mining_rollout.go`)**:
   - `ConsistencyService`: weryfikacja i naprawa spójności między magazynem wspomnień, wektorami `NamedVectors` i grafem wiedzy (`POST /v2/system/consistency/repair`).
   - `RAE-Phoenix`: `BuildSystemModel`, `AnalyzeLongitudinalTrends` oraz `DetectCapabilities` (`POST /v2/phoenix/system-model`, `POST /v2/phoenix/longitudinal`).
   - `RAE-Lab`: `MineFailurePatterns` oraz `ManageSafeRollout` (`10% -> 25% -> 50% -> 100%`) (`POST /v2/lab/failure-mining`, `POST /v2/lab/safe-rollout`).

---

### Iteracja G4: Pełna Weryfikacja na Żywo (`232/232 PASS`), Raport Wykonania Planu i Wydanie `v4.7.0` (`feature/g4-verification-232-checks-and-execution-report-v4.7.0`)

1. Przebudowanie wszystkich 10 binarek (`make test && make build && make build-cross`) z weryfikacją `CGO_ENABLED=0` i `go test -race ./...`.
2. Rozbudowa [`scripts/verify_python_vs_go_parity.py`](../scripts/verify_python_vs_go_parity.py) o **Sekcję 13** testującą na żywo wszystkie naprawione mechanizmy logiczne i funkcjonalne `L1`–`L10` (**`232/232 PASS`**).
3. Zapisanie raportu końcowego z wykonania planu w `/home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_WYKONANIA_PLANU_NAPRAWY_V4_7_0.md` oraz aktualizacja raportów porównawczych.
4. Merge zgodnie z Git Flow (`feature/* -> develop -> main`), tag `v4.7.0`, push na GitHub i zapis paragonu SHA-256 w RAE.
