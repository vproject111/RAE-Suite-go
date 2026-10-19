# Plan Iteracyjny Wdrożenia 10 Głębokich Mechanizmów Algorytmicznych i Operacyjnych (`G1`–`G10`) w `RAE-Suite-go` (`v4.6.0`)

**Data utworzenia planu:** `2026-10-19`  
**Repozytorium docelowe:** `/home/grzegorz/cloud/RAE-Suite-go` (`https://github.com/vproject111/RAE-Suite-go`)  
**Cel nadrzędny:** Zamknięcie wszystkich 10 głębokich rozbieżności algorytmicznych i operacyjnych Poziomu 2 (`G1`–`G10`) wykrytych podczas audytu AST wewnętrznych klas Pythona (`packages/rae-agentic-memory/apps/memory_api`, `packages/rae-core/rae_core`, `packages/rae-phoenix/feniks`, `packages/rae-hive/strategies`, `RAE-Suite/core` oraz skryptów CLI z `GEMINI.md`), przy zachowaniu `CGO_ENABLED=0`, zera zewnętrznych zależności w `go.mod` oraz pełnej agnostyczności bazodanowej i `RAE-Mesh`.

---

## Podział na Iteracje (`Iteracja F1` – `Iteracja F4`)

### Iteracja F1: `InformationBottleneckSelector` (`G1`), Czasowy Graf Wiedzy `TemporalGraphService` (`G2`), `EntityResolutionService` (`G3`) i `ContextProvenanceService` (`G4`) (`feature/f1-ib-selector-temporal-graph-entity-resolution-and-provenance`)
1. **G1 — `InformationBottleneckSelector` (`pkg/raecore/math/ib_selector.go`)**:
   - Matematyczna optymalizacja Tishby'ego $J_{IB}(Z) = I(Z; Y) - \beta \cdot I(Z; X)$:
     - `EstimateIZY`: trafność semantyczna i ważność względem zapytania $Y$,
     - `EstimateIZX`: koszt kompresji tokenów + kara za redundancję cosinusową/leksykalną (`ComputeDiversity` / MMR) względem już wybranych wspomnień,
     - `AdaptiveBeta`: dynamiczne dostrajanie współczynnika $\beta$ na podstawie budżetu tokenów i entropii kandydatów,
     - Podpięcie pod `ContextBroker` oraz endpoint `POST /v2/context/ib-select`.
2. **G2 — `TemporalGraphService` i `GraphUpdateOperator` (`pkg/raecore/graph/temporal.go`)**:
   - Dziennik zdarzeń grafowych `GraphChange` (`NODE_ADDED`, `NODE_REMOVED`, `EDGE_ADDED`, `EDGE_MODIFIED`, `WEIGHT_DECAYED`) ze znacznikami czasu.
   - Metody:
     - `CreateSnapshot(project, label)` i `RestoreSnapshot(snapshotID)`,
     - `ReconstructGraphAtTime(timestamp)` — odtwarzanie topologii grafu na dowolny punkt w czasie,
     - `CompareGraphs(t1, t2)` — różnica węzłów, krawędzi i gęstości grafu (`CalculateDensity`),
     - `FindEmergingPatterns()` — wykrywanie najszybciej rosnących węzłów/klastrów.
   - Endpointy: `POST /v2/graph/temporal/change`, `GET /v2/graph/temporal/reconstruct`, `POST /v2/graph/temporal/compare`, `GET /v2/graph/temporal/patterns`, `POST /v2/graph/snapshots/restore`.
3. **G3 — `EntityResolutionService` (`pkg/raecore/graph/entity_resolution.go`)**:
   - Klastrowanie synonimicznych nazw węzłów grafu (normalizacja, podobieństwo tokenowe/prefiksowe, reguły kanoniczne), generowanie `MergeDecision` i automatyczne przepinanie krawędzi do węzła kanonicznego (`POST /v2/graph/entity-resolution`).
4. **G4 — `ContextProvenanceService` (`pkg/raecore/security/provenance.go`)**:
   - Dwustopniowy rejestr zgodny z ISO/IEC 42001:
     - `CreateDecisionContext(query, project, sources []ContextSource)` -> `DecisionContext`,
     - `RecordDecision(contextID, agentID, decisionType, output, citedIDs)` -> `DecisionRecord`,
     - `GetDecisionLineage(decisionID)` oraz `AuditContextQuality(contextID)` (wskaźnik pokrycia cytowań, nieużyte źródła, ryzyko halucynacji).
   - Endpointy: `POST /v2/compliance/context-provenance/decision`, `GET /v2/compliance/context-provenance/lineage`, `GET /v2/compliance/context-provenance/audit`.

---

### Iteracja F2: Szyfrowanie `E2EEncryption` w `RAE-Mesh` (`G5`) oraz `EvidenceConflictDetector` + Dwuprzebiegowy `QueryRewriter` (`G6`) (`feature/f2-mesh-e2e-encryption-conflict-detector-query-rewriter`)
1. **G5 — Szyfrowanie End-to-End `E2EEncryption` dla `RAE-Mesh` (`pkg/raemesh/encryption.go`)**:
   - Natywna implementacja w czystym Go (`crypto/aes`, `crypto/cipher`, `crypto/hmac`, `crypto/sha256`):
     - `DeriveKeyPBKDF2(password, salt, iterations)` (standard PBKDF2-HMAC-SHA256, 32 bajty dla `AES-256`),
     - `EncryptMemory` / `DecryptMemory` oraz `EncryptBatch` / `DecryptBatch` z użyciem uwierzytelnionego szyfrowania `AES-256-GCM` (losowy nonce 12B + ciphertext + tag GCM).
   - Endpointy: `POST /v2/mesh/encrypt-batch` oraz `POST /v2/mesh/decrypt-batch`.
2. **G6 — `EvidenceConflictDetector` i 2-Przebiegowy `QueryRewriter` (`pkg/raecore/search/conflict_and_rewriter.go`)**:
   - `DetectEvidenceConflicts`: deterministyczne wykrywanie sprzeczności (`<1ms`) pomiędzy parami pobranych wspomnień (rozbieżne wartości liczbowe dla tego samego podmiotu, sprzeczne negacje `enabled/disabled`, `true/false`, `must/forbidden`).
   - `CreateRewritePlan` i `ExecuteTwoPassRetrieval`: gdy pierwszy przebieg nie spełnia progu `SufficiencyGate`, automatycznie przepisuje zapytanie (ekspansja synonimów architektonicznych, dekompozycja podzapytań, `BoostFactor = 1.4`) i wykonuje drugi przebieg wyszukiwania połączony z detekcją konfliktów.
   - Endpointy: `POST /v2/search/detect-conflicts` oraz `POST /v2/search/two-pass`.

---

### Iteracja F3: 6-Wymiarowy `BehaviorComparisonEngine` + `PostMortemAnalyzer` (`G7`), `AtomicSlicer v76` + `Supervisor v2` (`G8`), `GitOpsDaemon` + `ProjectPhaseAnalyzer` + `RAEGlobalSearchEngine` (`G9`) oraz Podkomendy CLI `./bin/rae` (`G10`) (`feature/f3-phoenix-6d-behavior-hive-slicer-gitops-phase-global-search-cli`)
1. **G7 — 6-Wymiarowy `BehaviorComparisonEngine` i `PostMortemAnalyzer` w `RAE-Phoenix` (`services/phoenix/behavior_and_postmortem.go`)**:
   - Sprawdzanie kontraktów w 6 wymiarach: `CheckHTTP`, `CheckCLI`, `CheckDOM`, `CheckLogs`, `CheckJSONPath`, `CheckPattern` + `CalculateRiskScore`.
   - `AnalyzePostMortem`: strukturalna analiza nieudanej sesji naprawczej (przyczyna źródłowa, koszt tokenów, złamane kontrakty, rekomendacja dla warstwy `reflective`).
   - Endpointy: `POST /v2/phoenix/behavior-check` oraz `POST /v2/phoenix/post-mortem`.
2. **G8 — `AtomicSlicer v76` i `Supervisor v2` w `RAE-Hive` (`services/hive/slicer_and_supervisor.go`)**:
   - `SliceCodeAtomically`: dekompozycja dużego pliku źródłowego na atomowe wycinki (funkcje/struktury/klasy) wraz z nagłówkiem importów i przypisaniem roli agenta (`auditor`, `builder`, `orchestrator`, `reporter`).
   - `SupervisorStatusAndHeal`: kontrola żywotności potoków i automatyczne wznawianie zatrzymanych zadań.
   - Endpointy: `POST /v2/hive/atomic-slice` oraz `POST /v2/hive/supervisor`.
3. **G9 — `GitOpsDaemon`, `ProjectPhaseAnalyzer` i `RAEGlobalSearchEngine` (`pkg/raecore/ops/gitops_phases_search.go`)**:
   - `GitOpsDaemon`: `CreateAgentBranch`, `CommitWithSignature`, `GeneratePullRequestSpec` (`POST /v2/ops/gitops`).
   - `ProjectPhaseAnalyzer`: agregacja wpisów `ExecutionAuditLedger` po fazach SDLC projektu (`POST /v2/ops/project-phases`).
   - `RAEGlobalSearchEngine`: sfederowane wyszukiwanie po 8 warstwach `RAE-Memory`, artefaktach `RAE-CRL`, paragonach `ExecutionAuditLedger` i regułach `RAE-Quality` (`POST /v2/portal/global-search`).
4. **G10 — Zunifikowane Podkomendy Operacyjne CLI w `./bin/rae` (`cmd/rae/main.go`)**:
   - `./bin/rae bootstrap` — inicjalizacja wiedzy architektonicznej i protokołu sesji (odpowiednik `bootstrap_dreamsoft_knowledge.py`),
   - `./bin/rae mesh status` oraz `./bin/rae mesh sync` — diagnostyka węzłów i synchronizacja `RAE-Mesh` z CLI (odpowiednik `mesh_sync_cli.py`),
   - `./bin/rae triad --cascade --text "..."` oraz `./bin/rae triad --tier ...` — wykonywanie 3-stopniowej triady 9 modeli z CLI (odpowiednik `run_layer_triad.py`),
   - `./bin/rae record-task ...` — już istniejący zapis 5-filarowego paragonu SHA-256.

---

### Iteracja F4: Weryfikacja Pełnej Parzystości na Żywo (`213/213 PASS`), Aktualizacja Raportu Rozbieżności (`0/10` luk) i Wydanie `v4.6.0` (`feature/f4-full-parity-verification-and-release-v4.6.0`)
1. Przebudowanie wszystkich binarek (`make test && make build && make build-cross`).
2. Rozbudowa `/home/grzegorz/cloud/RAE-Suite-go/scripts/verify_python_vs_go_parity.py` o **Sekcję 12** weryfikującą na żywo wszystkie 10 mechanizmów `G1`–`G10` (w tym wywołania podkomend CLI `./bin/rae bootstrap`, `./bin/rae mesh status`, `./bin/rae triad`).
3. Aktualizacja raportów `/home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_ROZBIEZNOSCI_FUNKCJONALNYCH_PYTHON_VS_GO.md` oraz `RAPORT_POROWNANIA_I_PARZYSTOSCI_PYTHON_VS_GO.md`.
4. Merge zgodnie z Git Flow (`feature/* -> develop -> main`), tag `v4.6.0`, push na GitHub i zapis paragonu w RAE.
