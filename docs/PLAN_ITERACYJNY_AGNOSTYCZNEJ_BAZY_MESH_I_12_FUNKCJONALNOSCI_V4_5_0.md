# Plan Iteracyjny Wdrożenia Agnostyczności Bazodanowej (`Agno-Core`), Niezawodnego `RAE-Mesh` oraz 12 Funkcjonalności Domenowych w `RAE-Suite-go` (`v4.5.0`)

**Data utworzenia planu:** `2026-10-19`  
**Repozytorium docelowe:** `/home/grzegorz/cloud/RAE-Suite-go` (`https://github.com/vproject111/RAE-Suite-go`)  
**Nadrzędne zasady architektoniczne:**
1. **Pełna Agnostyczność Bazodanowa (`Database-Agnostic Core`)**: Rdzeń `RAE` operuje wyłącznie na kontraktach portów (`MemoryStoragePort`) i pozwala w locie przełączać oraz migrować dane pomiędzy silnikami: `embedded` (RAM + JSONL + HNSW), `qdrant` (REST API Named Vectors z auto-migracją wymiarów), `redis` (natywny protokół TCP RESP), `postgres` (12 kanonicznych tabel SQL/JSONL z izolacją RLS) oraz `hybrid` (kompozyt z automatycznym failoverem do `embedded`).
2. **Niezawodny `RAE-Mesh` między heterogenicznymi węzłami**: Synchronizacja `Laptop <-> Cloud <-> Node 1 Lumina / Node 2 Julka / Node 3 Piotrek` działa niezależnie od tego, jakiej bazy używa dany węzeł (dzięki 16-koszykowemu drzewu Merkle `0..f`, delcie `CRDT LWW-Element-Set`, tokenom `CapabilityToken` i blokadzie `RESTRICTED` / `GracePeriod`).
3. **Most `Hermes + OpenCode` zamiast `OpenClaw`**: Całkowite odrzucenie ciężkiego `OpenClaw` na rzecz czystego, deterministycznego mostu do **Hermes** i **OpenCode** na odizolowanych `git worktree`.
4. **100% Pure Go (`CGO_ENABLED=0`, zero zewnętrznych bibliotek w `go.mod`)**.

---

## Podział na Iteracje (`Iteracja E1` – `Iteracja E4`)

### Iteracja E1: Agnostyczny Broker Baz Danych (`pkg/drivers/agnostic`) oraz Dwukierunkowy `RAE-Mesh` z Drzewem Merkle 16-Bucket (`feature/e1-agnostic-db-broker-and-merkle-mesh`)
1. **Agnostyczny Broker Storage (`pkg/drivers/agnostic/broker.go`)**:
   - Jednolity `AgnosticStorageBroker` implementujący `ports.MemoryStoragePort` i obsługujący 5 trybów pracy:
     - `embedded`: ultra-szybki silnik wbudowany (RAM + atomowy snapshot JSONL + natywny graf HNSW `384d`/`768d`/`1536d`),
     - `qdrant`: natywny klient HTTP REST Qdrant (`:6333/collections/...`) z obsługą przestrzeni nazwanych (`dense`, `ollama`, `openai`) i automatyczną walidacją/migracją wymiarów kolekcji,
     - `redis`: natywny klient protokołu drutowego TCP `RESP` (`:6379`) w czystym `net.Conn` bez CGO,
     - `postgres`: przenośny adapter 12 tabel kanonicznych RAE z generatorem i parserem transakcyjnych zrzutów SQL (`EXPORT SQL` / `IMPORT SQL`) oraz izolacją `Row-Level Security (RLS)`,
     - `hybrid`: kompozyt łączący dowolne skonfigurowane silniki z **automatycznym, bezprzerwaniowym failoverem** do silnika `embedded`, gdy zewnętrzna baza jest niedostępna.
   - Endpointy zarządzania agnostyczną bazą w `services/memory`:
     - `GET /v2/system/storage/backends` — status wszystkich sterowników (`embedded`, `qdrant`, `redis`, `postgres`, `hybrid`),
     - `POST /v2/system/storage/switch` — przełączenie aktywnego sterownika w locie bez utraty danych,
     - `POST /v2/system/storage/migrate` — gorąca migracja wszystkich warstw pamięci i krawędzi grafu pomiędzy dowolnymi backendami (np. `embedded -> postgres` lub `postgres -> qdrant`) + eksport SQL DDL/DML.
2. **Rozbudowa `RAE-Mesh` (`pkg/raemesh/mesh.go`)**:
   - **16-koszykowe Drzewo Merkle (`ComputeBucketHashes`, `DiffBuckets`)**: podział przestrzeni pamięci na 16 koszyków szesnastkowych (`0..f`), pozwalający przesyłać przez sieć wyłącznie brakujące lub zmienione koszyki (*Delta Sync*).
   - **Dwukierunkowa synchronizacja `SyncBidirectional` (`Pull + Push`)**: automatyczne pobieranie i wysyłanie zmian między węzłami Go i Python niezależnie od używanej pod spodem bazy danych.
   - **Aktywne sondowanie węzłów (`ProbePeers`)**: sprawdzanie opóźnień i zgodności `MerkleRoot` pomiędzy węzłami (`rae-host`, `node1-lumina`, `node2-julka`, `node3-piotrek`) pod `/v2/mesh/status` i `/v2/mesh/diff`.

---

### Iteracja E2: Most `Hermes + OpenCode` (bez `OpenClaw`), Wykonywalne Receptury `RAE-Phoenix` oraz Binarny Czytnik PDF i Wielojęzyczny AST w `RAE-Quality` (`feature/e2-hermes-opencode-bridge-and-phoenix-recipes`)
1. **Natywny Most `Hermes + OpenCode` (`pkg/raecore/ops/hermes_opencode.go` + `/v2/ops/hermes-opencode`)**:
   - Bezpośredni dyspozytor zadań dla `OPENCODE` (uruchamianie w odizolowanym `git worktree`, przechwytywanie `git diff`, weryfikacja w `RAE-Quality`) oraz `HERMES` (strukturalny pakiet eskalacyjny L3 z historią 3 prób naprawy `RAE-Phoenix`, śladem błędu i podpisem SHA-256).
   - Multi-provider router embeddingów w `pkg/drivers/llm` (automatyczny failover `Node 1 Lumina Ollama -> Node 3 Piotrek Ollama -> OpenRouter -> FNV-1a Int8 Quantized`).
2. **Wykonywalny Silnik Receptur Refaktoryzacyjnych w `RAE-Phoenix` (`services/phoenix/recipes_exec.go` + `/v2/phoenix/recipes/apply`)**:
   - `RecipeExecutor` wykonujący rzeczywistą transformację kodu dla 4 receptur (`feniks-01-extract-port`, `feniks-02-angularjs-component`, `feniks-03-zero-hardcode-path`, `feniks-04-controller-split`), zwracający zmodyfikowany kod, `unified diff` oraz wynik `VerifyBehavioralParity`.
3. **Binarny Parser PDF i Wielojęzyczny Analizator AST w `RAE-Quality` (`services/quality/binary_pdf_and_ast.go`)**:
   - `ParseBinaryPDF`: natywny skaner surowego strumienia PDF w czystym Go wyciągający `/MediaBox`, `/TrimBox`, `/BleedBox`, `/DeviceCMYK`, `/FontDescriptor` bez bibliotek C (`POST /v2/quality/pdf-binary-audit`).
   - `AnalyzeMultiLangAST`: natywny analizator drzewa funkcji, klas, głębokości zagnieżdżenia i złożoności cyklomatycznej dla **Go, Pythona, TypeScriptu, JavaScriptu i PHP** (`POST /v2/quality/multi-ast-audit`).

---

### Iteracja E3: Semantyczny Chunking & Ekstrakcja Grafu, Harmonogram Tła (`Go Ticker`), Natywny `WebSocket RFC 6455`, Benchmarki `9/5` + `SVG` oraz Eksport `LaTeX` w `RAE-CRL` (`feature/e3-ingestion-scheduler-websocket-and-crl-export`)
1. **Semantyczny Chunking i Automatyczna Ekstrakcja Grafu Wiedzy (`pkg/raecore/ingestion` + `POST /v2/memories/ingest-document`)**:
   - `SemanticSegmenter`: podział długich dokumentów z zachowaniem granic nagłówków Markdown, akapitów i zdań z konfigurowalnym oknem `chunk_size` i `overlap` oraz wzbogaceniem kontekstu (`ContextEnricher`).
   - `GraphTripleExtractor`: automatyczna ekstrakcja encji i relacji (`Subject -> Predicate -> Object`) podczas ingestii dokumentu i automatyczny zapis węzłów oraz krawędzi w warstwie `graph`.
2. **Wbudowany Harmonogram Tła (`BackgroundScheduler`) i Automatyczne Haki Pamięci (`services/memory`)**:
   - Gorutyna `BackgroundScheduler` (`time.Ticker`) wykonująca cyklicznie wygaszanie Ebbinghausa (`Decay`), konsolidację refleksyjną, wykrywanie społeczności i opróżnianie `TransactionalOutbox` (`/v2/automation/scheduler/status`, `/v2/automation/scheduler/run`).
   - Automatyczne wyzwalanie reguł `/v2/automation/*` oraz webhooków HTTP przy każdym wywołaniu `StoreMemory` (*PostStore Hook*).
3. **Natywny Serwer `WebSocket (RFC 6455)` w Bibliotece Standardowej Go (`services/portal` & `services/memory`)**:
   - Pełny handshake `Upgrade: websocket` (`Sec-WebSocket-Accept` z `crypto/sha1` + `http.Hijacker`) i ramkowanie RFC 6455 pod `/ws/dashboard` oraz `/v2/portal/ws` (zero zewnętrznych zależności).
4. **Adaptery Kontraktowe `LangChain` i `LlamaIndex` (`/v2/integrations/langchain/retrieve`, `/v2/integrations/llamaindex/query`)**:
   - Zwracanie wyników w natywnym schemacie `Document(page_content, metadata, score)` oraz `NodeWithScore`.
5. **Natywny Runner 6 Benchmarków Naukowych `9/5` (`LECT`, `MMIT`, `GRDT`, `RST`, `MPEB`, `ORB`) + Generator Wykresów `SVG`**:
   - Wykonywanie pełnego zestawu `9/5` w Go (`/v2/evaluation/benchmark/nine-five`) oraz generowanie wektorowych wykresów porównawczych `SVG` (`/v2/evaluation/benchmark/plot.svg`).
6. **Generator Publikacji Naukowych `Markdown` i `LaTeX (.tex)` w `RAE-CRL` (`POST /v2/crl/export`)**:
   - Natywny odpowiednik `ExportService.to_markdown` i `ExportService.to_latex` składający artefakty projektu badawczego w gotowy artykuł `.md` lub `.tex`.

---

### Iteracja E4: Weryfikacja 100% Parzystości na Żywo (`200+` testów), Aktualizacja Raportu Rozbieżności (0 luk) i Wydanie `v4.5.0` (`feature/e4-full-parity-verification-and-release-v4.5.0`)
1. Rozbudowa `/home/grzegorz/cloud/RAE-Suite-go/scripts/verify_python_vs_go_parity.py` o **Sekcję 11** testującą na żywo agnostyczny przełącznik i migrację baz danych, 16-koszykowy `RAE-Mesh`, most `Hermes + OpenCode`, wykonywalne receptury `RAE-Phoenix`, binarny audyt PDF i wielojęzyczny AST, semantyczny chunking z ekstrakcją grafu, harmonogram tła i haki pamięci, handshake `WebSocket RFC 6455`, adaptery `LangChain`/`LlamaIndex`, benchmarki `9/5` z wykresem `SVG` oraz eksport `LaTeX`/`Markdown` w `RAE-CRL`.
2. Aktualizacja `/home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_ROZBIEZNOSCI_FUNKCJONALNYCH_PYTHON_VS_GO.md` oraz `RAPORT_POROWNANIA_I_PARZYSTOSCI_PYTHON_VS_GO.md`.
3. Merge zgodnie z Git Flow (`feature/* -> develop -> main`), tag `v4.5.0`, publikacja na GitHubie oraz zapis paragonu audytowego SHA-256 w RAE i siatce Mesh.
