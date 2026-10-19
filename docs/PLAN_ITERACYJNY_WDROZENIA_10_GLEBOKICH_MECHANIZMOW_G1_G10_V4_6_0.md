# Plan Iteracyjny Wdrożenia 10 Głębokich Mechanizmów Algorytmicznych i Operacyjnych (`G1`–`G10`) w `RAE-Suite-go` (`v4.6.0`)

**Data utworzenia planu:** `2026-10-19`  
**Status realizacji:** **ZREALIZOWANO W 100% (`216/216 PASS`, `0/10` otwartych luk w `v4.6.0`)**  
**Repozytorium docelowe:** `/home/grzegorz/cloud/RAE-Suite-go` (`https://github.com/vproject111/RAE-Suite-go`)  
**Cel nadrzędny:** Zamknięcie wszystkich 10 głębokich rozbieżności algorytmicznych i operacyjnych Poziomu 2 (`G1`–`G10`) wykrytych podczas audytu AST wewnętrznych klas Pythona (`packages/rae-agentic-memory/apps/memory_api`, `packages/rae-core/rae_core`, `packages/rae-phoenix/feniks`, `packages/rae-hive/strategies`, `RAE-Suite/core` oraz skryptów CLI z `GEMINI.md`), przy zachowaniu `CGO_ENABLED=0`, zera zewnętrznych zależności w `go.mod` oraz pełnej agnostyczności bazodanowej i `RAE-Mesh`.

---

## Podział na Iteracje (`Iteracja F1` – `Iteracja F4`)

### [x] Iteracja F1: `InformationBottleneckSelector` (`G1`), Czasowy Graf Wiedzy `TemporalGraphService` (`G2`), `EntityResolutionService` (`G3`) i `ContextProvenanceService` (`G4`) (`feature/f1-ib-selector-temporal-graph-entity-resolution-and-provenance`) — ZREALIZOWANO W 100%
1. **G1 — `InformationBottleneckSelector` (`pkg/raecore/math/ib_selector.go`)**:
   - Matematyczna optymalizacja Tishby'ego $J_{IB}(Z) = I(Z; Y) - \beta \cdot I(Z; X)$:
     - `EstimateIZY`: trafność semantyczna i ważność względem zapytania $Y$,
     - `EstimateIZX`: koszt kompresji tokenów + kara za redundancję Jaccard/MMR względem już wybranych wspomnień,
     - `AdaptiveBeta`: dynamiczne dostrajanie współczynnika $\beta$ na podstawie budżetu tokenów i entropii kandydatów,
     - Podpięcie pod `POST /v2/context/ib-select`.
2. **G2 — `TemporalGraphService` i `GraphUpdateOperator` (`pkg/raecore/graph/temporal.go`)**:
   - Dziennik zdarzeń grafowych `GraphChange` (`ADD_NODE`, `REMOVE_NODE`, `ADD_EDGE`, `REMOVE_EDGE`) ze znacznikami czasu.
   - Metody: `CreateSnapshot(project, label)`, `RestoreSnapshot(snapshotID)`, `ReconstructGraphAtTime(project, timestamp)`, `CompareGraphs(project, t1, t2)` (różnica węzłów, krawędzi i gęstości grafu `DensityDelta`), `FindEmergingPatterns(project)`.
   - Endpointy: `POST /v2/graph/temporal/change`, `GET /v2/graph/temporal/reconstruct`, `POST /v2/graph/temporal/compare`, `GET /v2/graph/temporal/patterns`, `POST /v2/graph/snapshots/restore`.
3. **G3 — `EntityResolutionService` (`pkg/raecore/graph/entity_resolution.go`)**:
   - Klastrowanie synonimicznych nazw węzłów grafu (`NormalizeEntityKey`), generowanie `MergeDecision` i automatyczne przepinanie krawędzi do węzła kanonicznego (`POST /v2/graph/entity-resolution`).
4. **G4 — `ContextProvenanceService` (`pkg/raecore/security/provenance.go`)**:
   - Dwustopniowy rejestr zgodny z ISO/IEC 42001: `CreateDecisionContext`, `RecordDecision`, `GetDecisionLineage`, `AuditContextQuality` (`POST /v2/compliance/context-provenance/decision`, `GET /v2/compliance/context-provenance/lineage`, `GET /v2/compliance/context-provenance/audit`).

---

### [x] Iteracja F2: Szyfrowanie `E2EEncryption` w `RAE-Mesh` (`G5`) oraz `EvidenceConflictDetector` + Dwuprzebiegowy `QueryRewriter` (`G6`) (`feature/f2-mesh-e2e-encryption-conflict-detector-query-rewriter`) — ZREALIZOWANO W 100%
1. **G5 — Szyfrowanie End-to-End `E2EEncryption` dla `RAE-Mesh` (`pkg/raemesh/encryption.go`)**:
   - Natywna implementacja w czystym Go (`crypto/aes`, `crypto/cipher`, `crypto/hmac`, `crypto/sha256`):
     - `DeriveKeyPBKDF2(password, salt, iterations)` (standard PBKDF2-HMAC-SHA256, 32 bajty dla `AES-256`),
     - `EncryptMemory` / `DecryptMemory` oraz `EncryptBatch` / `DecryptBatch` z użyciem uwierzytelnionego szyfrowania `AES-256-GCM`.
   - Endpointy: `POST /v2/mesh/encrypt-batch` oraz `POST /v2/mesh/decrypt-batch`.
2. **G6 — `EvidenceConflictDetector` i 2-Przebiegowy `QueryRewriter` (`pkg/raecore/search/conflict_and_rewriter.go`)**:
   - `DetectEvidenceConflicts`: deterministyczne wykrywanie sprzeczności (`<1ms`) pomiędzy parami pobranych wspomnień (rozbieżne wartości liczbowe oraz antonimy `enabled/disabled`, `true/false`, `allowed/forbidden`).
   - `CreateRewritePlan`: dwuprzebiegowe rozszerzenie zapytania (`BoostFactor = 1.4`) połączone z automatyczną detekcją konfliktów (`POST /v2/search/detect-conflicts` oraz `POST /v2/search/two-pass`).

---

### [x] Iteracja F3: 6-Wymiarowy `BehaviorComparisonEngine` + `PostMortemAnalyzer` (`G7`), `AtomicSlicer v76` + `Supervisor v2` (`G8`), `GitOpsDaemon` + `ProjectPhaseAnalyzer` + `RAEGlobalSearchEngine` (`G9`) oraz Podkomendy CLI `./bin/rae` (`G10`) (`feature/f3-phoenix-6d-behavior-hive-slicer-gitops-phase-global-search-cli`) — ZREALIZOWANO W 100%
1. **G7 — 6-Wymiarowy `BehaviorComparisonEngine` i `PostMortemAnalyzer` w `RAE-Phoenix` (`services/phoenix/behavior_and_postmortem.go`)**:
   - Sprawdzanie kontraktów w 6 wymiarach (`return_arity`, `branch_topology`, `side_effects`, `error_handling`, `complexity_delta`, `signature_compat`) + `RiskScore` oraz `AnalyzePostMortem` (`POST /v2/phoenix/behavior-check`, `POST /v2/phoenix/post-mortem`).
2. **G8 — `AtomicSlicer v76` i `Supervisor v2` w `RAE-Hive` (`services/hive/slicer_and_supervisor.go`)**:
   - `SliceIntoAtomicTasks` z sortowaniem topologicznym DAG oraz 6-stanowy automat `RunSupervisorV2` (`INIT -> SLICING -> DISPATCHING -> VERIFYING -> MERGING -> COMPLETED`) (`POST /v2/hive/atomic-slice`, `POST /v2/hive/supervisor`).
3. **G9 — `GitOpsDaemon`, `ProjectPhaseAnalyzer` i `RAEGlobalSearchEngine` (`pkg/raecore/ops/gitops_phases_search.go`)**:
   - `ExecuteGitOpsWorkflow` (`POST /v2/ops/gitops`), `AnalyzeProjectPhases` (`POST /v2/ops/project-phases`) oraz `ExecuteGlobalSearch` (`POST /v2/portal/global-search`).
4. **G10 — Zunifikowane Podkomendy Operacyjne CLI w `./bin/rae` (`cmd/rae/main.go`)**:
   - `./bin/rae bootstrap`, `./bin/rae mesh status`, `./bin/rae mesh sync`, `./bin/rae triad --cascade --text "..."` oraz `./bin/rae record-task ...`.

---

### [x] Iteracja F4: Weryfikacja Pełnej Parzystości na Żywo (`216/216 PASS`), Aktualizacja Raportu Rozbieżności (`0/10` luk) i Wydanie `v4.6.0` (`feature/f4-full-parity-verification-and-release-v4.6.0`) — ZREALIZOWANO W 100%
1. Przebudowano wszystkie binarki (`make test && make build && make build-cross`).
2. Rozbudowano `/home/grzegorz/cloud/RAE-Suite-go/scripts/verify_python_vs_go_parity.py` o **Sekcję 12** weryfikującą na żywo wszystkie 10 mechanizmów `G1`–`G10` (**`216/216 PASS`**).
3. Zaktualizowano raporty `/home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_ROZBIEZNOSCI_FUNKCJONALNYCH_PYTHON_VS_GO.md` oraz `RAPORT_POROWNANIA_I_PARZYSTOSCI_PYTHON_VS_GO.md`.
