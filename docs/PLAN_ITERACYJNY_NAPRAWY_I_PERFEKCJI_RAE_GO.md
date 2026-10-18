# 🛠️ Iteracyjny Plan Naprawy i Doprowadzenia do Perfekcji `RAE-Suite-go` (`v4.1.0`)

**Data audytu:** 2026-10-18  
**Repozytorium docelowe:** `/home/grzegorz/cloud/RAE-Suite-go` (`https://github.com/vproject111/RAE-Suite-go`)  
**Cel:** Eliminacja wszystkich luk implementacyjnych, pustych pakietów, uproszczeń i brakujących powiązań między modułami wykrytych podczas rygorystycznego audytu wersji `v4.0.0`.

---

## 🔍 1. Wyniki Audytu Wersji `v4.0.0` (Zidentyfikowane Luki i Niedoróbki)

Szczegółowa inspekcja drzewa kodu oraz raportu pokrycia (`go test -cover ./...`) wykazała następujące braki względem pełnej specyfikacji `RAE-Suite` i `RAE-CRL 2.2`:

| # | Obszar / Pakiet | Zidentyfikowana luka w `v4.0.0` | Wymagany stan docelowy (100% Perfekcji w `v4.1.0`) |
| :--- | :--- | :--- | :--- |
| **1** | **Puste pakiety w `pkg/raecore/*` i `pkg/drivers/*`** | Katalogi `pkg/raecore/search`, `pkg/raecore/epistemic`, `pkg/drivers/sqlstore`, `pkg/drivers/vectorstore` oraz `pkg/drivers/outbox` zostały utworzone jako puste foldery (0 plików `.go`). | Pełna implementacja wszystkich 5 pakietów: `LogicGateway` + 9 strategii wyszukiwania w `pkg/raecore/search`, pełny silnik DAG (DFS cycle detection dowolnej głębokości) i weryfikator offline `EvidencePackage` w `pkg/raecore/epistemic`, trwały sterownik dyskowy WAL/SQL + adapter HTTP Postgresa w `pkg/drivers/sqlstore`, natywny wieloprzestrzenny indeks `HNSW` + klient `Qdrant` w `pkg/drivers/vectorstore` oraz trwały `Outbox` z obsługą Dead-Letter Queue w `pkg/drivers/outbox`. |
| **2** | **Cykl życia 8 warstw w `services/memory`** | Brak jawnych metod przejść międzypoziomowych z Pythonowego `rae-core`: konsolidacji `working` $\rightarrow$ `episodic`/`semantic`, awansu `UpgradeEpisodicToSemantic`, archiwizacji `PromoteToLTM`, `CommunityDetector` na warstwie `graph`, propagacji `SemanticResonance` oraz endpointów `DELETE /v2/memories/{id}` i `GET /v2/memories/stats`. | Pełna implementacja wszystkich przejść międzypoziomowych między 8 warstwami (`sensory`, `working`, `episodic`, `semantic`, `procedural`, `reflective`, `ltm`, `graph`), algorytmu `CommunityDetector`, `SemanticResonance`, `MemoryStats` oraz persystencji dyskowej (`RAE_DATA_DIR`). |
| **3** | **Niepełne CLI i PI Governance w `RAE-CRL` (`services/crl` & `cmd/rae-crl`)** | W `cmd/rae-crl/main.go` zaimplementowano tylko komendy `init` i `add` (brak `query`, `fork`, `seal`, `verify`). Brak dwuetapowego `RequestApproval` przed `ApproveArtifact` oraz brak pętli tła `SyncEngine` synchronizującej artefakty po wygaśnięciu 24h `Grace Period`. | Wszystkie 6 komend CLI (`init`, `add`, `query`, `fork`, `seal`, `verify`), pełny przepływ `request_approval` $\rightarrow$ `approve_artifact`, weryfikacja offline podpisów `HMAC-SHA256` i `VersionCID` paczek `SBOM/VEX` oraz pętla `StartSyncEngine` w tle. |
| **4** | **Narzędzia `rae-mcp` (`services/mcp`)** | 6 z 16 narzędzi MCP (`rae_quality_audit`, `rae_phoenix_impact`, `rae_phoenix_repair`, `rae_sandbox_run`, `rae_cluster_dispatch`, `rae_consolidate_traces`) wpadało w domyślny `default:` zamiast wywoływać rzeczywiste serwisy `quality`, `phoenix`, `hive`, `orchestrator` i `memory`. | Podpięcie wszystkich 16 narzędzi MCP do żywych instancji serwisów (`quality`, `phoenix`, `hive`, `orchestrator`, `memory`, `crl`) oraz obsługa `POST /mcp/messages` (JSON-RPC over HTTP SSE). |
| **5** | **Natywny parser `go/ast` i `MutationRunner` w `services/quality`** | Skaner opierał się wyłącznie na dopasowaniu podciągów zamiast użyć natywnego `go/ast` i wielowątkowego `ScanDirectory`, brakowało też endpointów `/v2/quality/mutation` i `/v2/quality/coverage`. | Prawdziwa analiza drzewa AST (`go/ast`, `go/parser`), wielowątkowy worker pool skanujący katalogi, `MutationRunner` oraz `TestAggregator`. |
| **6** | **Domknięcie pętli naprawczej `Phoenix` $\leftrightarrow$ `CRL` $\leftrightarrow$ `Lab`** | Po nieudanej naprawie (3 próby) w `services/phoenix` brakowało automatycznego przekazania porażki do `RAE-CRL` (`FAILED_BUT_INFORMATIVE`) oraz do `RAE-Lab` (`RegisterScan` $\rightarrow$ propozycja reguły `CANDIDATE`). W `services/lab` brakowało endpointów `/lab/experiment/run` i `/lab/analyze/failures`. | Pełne sprzężenie zwrotne między `Phoenix`, `CRL` i `Lab` oraz komplet endpointów kompatybilnych z Pythonowym `rae-lab`. |
| **7** | **Prawdziwa synchronizacja sieciowa `RAE-Mesh` i `Consensual Memory Transfer (CMT)`** | `pkg/raemesh/mesh.go` liczył tylko lokalny `MerkleRoot`, bez wysyłania zapytań synchronizacyjnych HTTP, weryfikacji tokenów uprawnień `CMT Capability Token` i odbierania delt (`POST /v2/mesh/receive`, `POST /v2/mesh/peers/{peer_id}/sync`). | Pełna implementacja protokołu `CMT` z tokenami uprawnień HMAC, porównywaniem drzew Merkle i dwukierunkową wymianą delt po HTTP z poszanowaniem blokady `RESTRICTED` i 24h `Grace Period`. |
| **8** | **Pokrycie testami jednostkowymi we wszystkich pakietach (`go test -cover ./...`)** | Testy znajdowały się tylko w 4 pakietach, przez co pozostałe pakiety wykazywały `0.0%` bezpośredniego pokrycia w `go test -cover ./...`. | Dedykowane testy `*_test.go` w każdym pakiecie (`pkg/*` oraz `services/*`) zapewniające $>80\%$ pokrycia instrukcji i 0 wyścigów (`go test -race ./...`). |

---

## 🚀 2. Plan Wykonania Naprawy (Iteracje Naprawcze R1 – R5)

### 🔹 Iteracja Naprawcza R1: Wypełnienie Wszystkich Pakietów `pkg/raecore/*` i `pkg/drivers/*`
1. **`pkg/raecore/epistemic/dag.go`**:
   - Pełny algorytm DFS wykrywania cykli dowolnej głębokości w `Typed Provenance DAG`,
   - Operator detekcji sprzeczności logicznych (`DetectContradictions`),
   - Weryfikator offline paczek dowodowych `VerifyEvidencePackageOffline(pkg, guard)` sprawdzający integralność `VersionCID` (`CBOR`), obecność `SBOM`/`VEX` i podpis `HMAC-SHA256`.
2. **`pkg/raecore/search/gateway.go`**:
   - Implementacja **`LogicGateway`** łączącego **3 Silniki Matematyczne** (*Engine 1: Heuristic Fast Path*, *Engine 2: Probabilistic Semantic Resonance*, *Engine 3: Neural Multi-Vector Deep Path*),
   - Wszystkie **9 strategii wyszukiwania** (`anchor`, `fulltext`, `sparse` z punktacją BM25, `vector`, `multi_vector`, `graph`, `graph_lite`, `visual`, `hybrid`),
   - `SmartReranker`, `SufficiencyGate` oraz `ConflictDetector`.
3. **`pkg/drivers/vectorstore/hnsw_qdrant.go`**:
   - Natywny wieloprzestrzenny indeks `HNSWIndex` w czystym Go obsługujący *Named Vectors* (`dense`, `ollama`, `openai`) z zapisem/odczytem z pliku `.hnsw.json`,
   - Adapter klienta HTTP `QdrantClient` dla profilu `cluster`/`hybrid`.
4. **`pkg/drivers/outbox/outbox.go` & `pkg/drivers/sqlstore/persistent.go`**:
   - Trwały `PersistentOutbox` (`PENDING` $\rightarrow$ `PROCESSING` $\rightarrow$ `DELIVERED` / `DEAD_LETTER` po przekroczeniu `MaxAttempts`) z zapisem WAL na dysku,
   - `PersistentStore` automatycznie zapisujący i odtwarzający wszystkie 8 warstw pamięci, krawędzie grafu, artefakty `RAE-CRL` i dziennik `CBOR` z katalogu `RAE_DATA_DIR`.

---

### 🔹 Iteracja Naprawcza R2: Kompletny Silnik 8 Warstw `RAE-Memory` i `RAE-Mesh CMT` (`services/memory` & `pkg/raemesh`)
1. Dodanie pełnego cyklu życia 8 warstw w `services/memory/service.go`:
   - `ConsolidateFromWorking(ctx, memoryID, asSemantic)` ($+0.2$ importance, blokada jeśli `InfoClass == RESTRICTED` bez jawnej deklasyfikacji),
   - `UpgradeToSemantic(ctx, episodicID, generalizedContent)` ($+0.1$ importance, ślad `derived_from_episodic`),
   - `PromoteToLTM(ctx, memoryID)` (archiwizacja w warstwie `ltm`),
   - `DetectGraphCommunities(ctx)` (klasteryzacja spójnych składowych w warstwie `graph`) oraz `PropagateSemanticResonance(ctx, seedIDs, decayFactor)`,
   - Endpointy `DELETE /v2/memories/{id}`, `GET /v2/memories/stats`, `POST /v2/memories/consolidate`, `POST /v2/memories/upgrade`.
2. Rozbudowa `pkg/raemesh/mesh.go`:
   - Protokół **CMT (*Consensual Memory Transfer*)** z generowaniem i weryfikacją `CapabilityToken`,
   - `ReceiveMemories(ctx, items, capabilityToken)` (weryfikacja tokenu, odrzucenie danych `RESTRICTED`/`working`, deduplikacja po `ContentHash`),
   - `SyncWithPeer(ctx, peerID, project)` oraz endpointy `POST /v2/mesh/peers/{peer_id}/sync` i `POST /v2/mesh/receive`.

---

### 🔹 Iteracja Naprawcza R3: Pełne `RAE-CRL` (6 Komend CLI, PI Governance, SyncEngine) oraz Pełne Podpięcie 16 Narzędzi `rae-mcp`
1. Rozbudowa `services/crl/service.go` i `cmd/rae-crl/main.go`:
   - Dwuetapowy nadzór PI: `RequestApproval(ctx, artifactID, approver)` (`StatusDraft/Active` + `proposed_by` + `approver_target`) $\rightarrow$ `ApproveArtifact(ctx, artifactID, approver)`,
   - Tłowy `StartSyncEngine(ctx, interval)` synchronizujący zatwierdzone artefakty po wygaśnięciu `Grace Period`,
   - Wszystkie 6 komend CLI w `cmd/rae-crl`: `init`, `add`, `query`, `fork`, `seal`, `verify`.
2. Pełne podpięcie wszystkich 16 narzędzi w `services/mcp/service.go`:
   - Połączenie `rae_quality_audit` $\rightarrow$ `quality.Service`, `rae_phoenix_impact` i `rae_phoenix_repair` $\rightarrow$ `phoenix.Service`, `rae_sandbox_run` $\rightarrow$ `hive.Service.RunParallelCommands`, `rae_cluster_dispatch` $\rightarrow$ `orchestrator.Service`, `rae_consolidate_traces` $\rightarrow$ `memory.Service.ConsolidateFromWorking`,
   - Dodanie obsługi `POST /mcp/messages` (JSON-RPC 2.0 over HTTP).

---

### 🔹 Iteracja Naprawcza R4: Natywny AST (`go/ast`) i `MutationRunner` w `rae-quality` oraz Domknięcie Pętli `Phoenix` $\leftrightarrow$ `CRL` $\leftrightarrow$ `Lab`
1. Rozbudowa `services/quality/service.go`:
   - Natywna analiza składniowa kodu Go za pomocą standardowych pakietów `go/parser` i `go/ast` (dokładne liczenie funkcji, instrukcji rozgałęziających i złożoności cyklomatycznej),
   - Wielowątkowy `ScanDirectory(root string, workers int)` na puli gorutyn,
   - `RunMutationTesting` (`POST /v2/quality/mutation`) wyliczający `Mutation Score Indicator (MSI)` oraz `AggregateCoverage` (`POST /v2/quality/coverage`).
2. Domknięcie pętli samonaprawy w `services/phoenix/service.go` i `services/lab/service.go`:
   - Automatyczne rejestrowanie nieudanych napraw Phoenix w `RAE-CRL` (`RecordFailure` $\rightarrow$ `FAILED_BUT_INFORMATIVE`) i `RAE-Lab` (`RegisterScan(false, ...)` $\rightarrow$ `CANDIDATE` guardrail),
   - Dodanie w `services/lab` endpointów `POST /lab/experiment/run` (`RunOfflineReplay`) i `POST /lab/analyze/failures` (`AnalyzeFailures`).

---

### 🔹 Iteracja Naprawcza R5: Pełna Maszyna Stanów `AutonomyKernel`, Live Telemetry w `rae-portal` i Kompleksowe Testy Pokrycia (`>85%`)
1. Rozbudowa `services/orchestrator/service.go` i `services/portal/service.go`:
   - Pełna maszyna stanów zadania w `AutonomyKernel` (`PENDING` $\rightarrow$ `RESEARCHING` $\rightarrow$ `PLANNING` $\rightarrow$ `CODING` $\rightarrow$ `AUDITING` $\rightarrow$ `NEEDS_APPROVAL` $\rightarrow$ `APPROVED` $\rightarrow$ `COMPLETED`),
   - Integracja z `Portal` (`GET /v2/portal/telemetry` agregujący na żywo statystyki 8 warstw pamięci, wag MAB, kandydatów Guardrails, statusu Watchdoga i artefaktów `RAE-CRL` + pełny interaktywny widok HTML/JS na `GET /`).
2. Dodanie testów jednostkowych `*_test.go` we wszystkich pakietach (`pkg/*` i `services/*`), przebudowa wszystkich 10 binarek + 4 binarek skrośnych oraz wypchnięcie wydania `v4.1.0` na GitHub (`develop` i `main`).
