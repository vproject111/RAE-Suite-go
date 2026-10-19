# Raport Diagnostyczny 5-Warstwowego Audytu RAE-Suite (Python vs Go `v4.7.0`)

**Data wykonania**: 2026-10-19  
**Mechanizm wielokrotnego użytku**: `scripts/reusable_5stage_parity_auditor.py` (`make audit-5stage`)  
**Plik surowych wyników JSON**: `docs/AUDYT_5_WARSTWOWY_WYNIKI.json`  
**Badana wersja bazowa**: `v4.7.0` (`241/241` testów powierzchniowych PASS, audyt głęboki 5-warstwowy wykrył **24 ukryte luki i zaślepki**)

---

## 1. Architektura 5-Warstwowego Mechanizmu Weryfikacji Wielokrotnego Użytku

Aby wyeliminować tzw. *Błąd Przeżywalności Testów* (w którym testy sprawdzają tylko to, o czym pomyślał autor podczas portowania), wdrożono deterministyczny, wykonywalny w każdej chwili skrypt `scripts/reusable_5stage_parity_auditor.py` (dostępny pod komendą `make audit-5stage`), który automatycznie uruchamia tymczasowy klaster 9 usług Go i przeprowadza 5 niezależnych warstw kontroli:

1. **Warstwa 1 — Automatyczna Macierz Symboli AST (Python AST $\leftrightarrow$ Go AST)**:
   - Parsuje drzewo AST kodu pierwszej strony (`first-party`) w Pythonie oraz wszystkie struktury (`struct`), funkcje (`func`) i trasy HTTP (`HandleFunc`) w Go, weryfikując 30 kanonicznych domen funkcjonalnych wraz z obecnością wymaganych symboli AST.
2. **Warstwa 2 — Statyczny Skaner Anty-Stubowy (`Anti-Stub & Dead-Logic Scanner`)**:
   - Analizuje kod źródłowy wszystkich **335 handlerów HTTP** w Go pod kątem:
     - ignorowania obiektu żądania (`func(w http.ResponseWriter, _ *http.Request)`), gdy handler zwraca stały literał JSON bez wywołania jakiegokolwiek silnika domenowego (`s.*`, `store.*`, `graph.*`, `mdp.*`, `search.*`),
     - martwego dekodowania (`json.NewDecoder(r.Body).Decode(&req)`), po którym zmienna `req` nie jest nigdzie używana w logice funkcji,
     - zakazanych artefaktów (`OpenClawEscalation` zamiast `HermesOpenCodeEscalation`).
3. **Warstwa 3 — Testy Wrażliwości Różnicowej (`Differential Input-Sensitivity Probing` $f(x_1) \neq f(x_2)$)**:
   - Wysyła do każdego badanego endpointu dwa skrajnie różne ładunki wejściowe $x_1$ (zdrowy/mały/niskiego ryzyka) oraz $x_2$ (złożony/błędny/wysokiego ryzyka). Jeżeli odpowiedź po odfiltrowaniu znaczników czasu i losowych UUID spełnia $f(x_1) == f(x_2)$, endpoint zostaje oznaczony jako **ślepa na wejście zaślepka (`INPUT_INSENSITIVE_STUB`)**.
4. **Warstwa 4 — Łańcuch Niezmienników Stanowych End-to-End (`Stateful Workflow Invariant Chain`)**:
   - Weryfikuje 6-krokowy przepływ przyczynowo-skutkowy, w którym wyjście kroku $N$ zasila i mutuje stan odczytywany w kroku $N+1$:
     1. Semantyczna ingestia dokumentu (`/v2/memories/ingest-document`) $\rightarrow$ zapis chunków i krawędzi grafu wiedzy $\rightarrow$ weryfikacja odczytu trójek przez `/v2/graph/triples`.
     2. Zapis wspomnienia z `importance=0` + PII (`anna.nowak@example.com`) $\rightarrow$ automatyczne wyliczenie 6-czynnikowego Importance (`> 0`).
     3. Zapis nowego wspomnienia $\rightarrow$ natychmiastowa mutacja korzenia kryptograficznego `root_hash` w 16-kubełkowym drzewie Merkle (`/v2/mesh/merkle/buckets`).
     4. Cykl `DreamingWorker` (`/v2/dreaming/run`) $\rightarrow$ synteza wniosków z warstwy `episodic` do `reflective`.
     5. Egzekucja prawa do bycia zapomnianym RODO Art. 17 (`/v2/governance/retention/gdpr-erase`) $\rightarrow$ fizyczne usunięcie PII i podpisanie dowodu `SHA-256`.
     6. Zapytanie wyszukujące po wymazaniu RODO $\rightarrow$ `0` wyników zawierających PII oraz zachowanie kontraktu typu JSON (`"results": []`, a nie `"results": null`).
5. **Warstwa 5 — Audyt Kontraktu Schematu Bazy Danych, SQL DDL i Zmiennych Środowiskowych (`Env & Schema Contract`)**:
   - Sprawdza obecność 12 kanonicznych tabel PostgreSQL, eksport wykonywalnego schematu DDL SQL (`/v2/db/schema/sql` oraz `/v2/system/storage/export-sql`) oraz obsługę 16 kanonicznych zmiennych środowiskowych w kodzie Go (`os.Getenv`).

---

## 2. Zestawienie Wykrytych Niezgodności i Zaślepek w `v4.7.0`

### 2.1. Znaleziska Warstwy 2 (`Static Anti-Stub & Dead-Logic Scanner` — 13 problemów w 335 handlerach)

| # | Plik Go | Trasa / Symbol | Typ problemu | Opis techniczny |
|---|---|---|---|---|
| 1 | `services/portal/service.go` | `AllPortalViews` / HTML `/` | `FORBIDDEN_OPENCLAW_VIEW_NAME` | Widok nr 7 w `AllPortalViews()` oraz w HTML Command Center nadal nosi nazwę `"OpenClawEscalation"` zamiast `"HermesOpenCodeEscalation"` (naruszenie reguły `openclaw_disabled=true`). |
| 2 | `services/memory/extended_routes.go` | `/v2/bridge/audit` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca stały literał `{"status": "audited", "iso_42001_compliant": true}` bez weryfikacji ładunku przez `s.guard` lub `s.provenanceSvc`. |
| 3 | `services/memory/extended_routes.go` | `/v2/compliance/circuit-breakers` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca statyczną tablicę wyłączników nadprądowych bez odpytania żywego stanu `s.guard` / `s.broker`. |
| 4 | `services/memory/extended_routes.go` | `/v2/dashboard/compliance/report` | `STATIC_LITERAL_STUB` | Zwraca statyczny słownik bez wyliczenia rzeczywistych statystyk z `s.store.GetMemoryStats` i `ext.approvals`. |
| 5 | `services/memory/extended_routes.go` | `/v2/dashboard/compliance/metrics` | `STATIC_LITERAL_STUB` | Zwraca stałe zera bez odczytu statystyk warstw (`working` vs pozostałe) z `s.store`. |
| 6 | `services/memory/extended_routes.go` | `/v2/dashboard/compliance/risks` | `STATIC_LITERAL_STUB` | Zwraca stałe `{"high_risks": 0}` bez sprawdzenia oczekujących wniosków HITL (`ext.approvals`) i wspomnień `RESTRICTED`. |
| 7 | `services/memory/extended_routes.go` | `/v2/dashboard/compliance/audit-trail` | `STATIC_LITERAL_STUB` | Zwraca statyczny string `"VERIFIED_SHA256"` bez wyliczenia skrótu łańcucha audytowego z wykonanych operacji. |
| 8 | `services/quality/service.go` | `/v2/quality/defect-metrics` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca na sztywno `"defect_density": 0.01` zamiast wyliczyć gęstość defektów z przekazanego kodu/znalezisk przez `s.AuditCode`. |
| 9 | `services/quality/service.go` | `/v2/quality/predictive-plan` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca pustą tablicę `"predicted_hotspots": []` zamiast wyznaczyć hotspoty na podstawie przekazanej listy plików (`files`) i kodu. |
| 10 | `services/quality/service.go` | `/v2/quality/unit-test-inventory` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca stałą liczbę bez uwzględnienia filtra modułu lub katalogu reguł `BuildCatalog198Summary()`. |
| 11 | `services/quality/service.go` | `/v2/quality/check-architecture` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca stałe `"passed": true` bez sprawdzenia przekazanego kodu/importów przez `s.AuditCode`. |
| 12 | `services/phoenix/service.go` | `/sessions/analyze` | `UNUSED_DECODED_REQUEST` | Dekoduje `var req map[string]interface{}`, ale nigdy nie używa zmiennej `req`, zwracając stałe `"anomalies_found": 0, "recommended_patch": "none"`. |
| 13 | `services/phoenix/service.go` | `/patterns/errors` | `STATIC_LITERAL_STUB` | Ignoruje `*http.Request` i zwraca 3 zahardkodowane nazwy błędów zamiast dynamicznego wzorca z `s.ListRecipes()` i analizatora post-mortem. |

---

### 2.2. Znaleziska Warstwy 3 (`Differential Input-Sensitivity Probing` $f(x_1) \neq f(x_2)$ — 6 ślepych endpointów)

| # | Usługa | Endpoint | Wynik sondy różnicowej | Przyczyna źródłowa |
|---|---|---|---|---|
| 1 | `rae-suite` (`:8009`) | `/v2/fabric/route` | `FAIL` ($f(x_1) == f(x_2)$) | Ignoruje ciało żądania (`workload`, `preferred_node`, `tier`) i zawsze zwraca `"selected_node": "node1-lumina"`, mimo że `s.clusterMgr` (`ops.ClusterComputeManager`) jest zainicjalizowany w serwisie. |
| 2 | `rae-quality` (`:8010`) | `/v2/quality/risk-coverage` | `FAIL` ($f(x_1) == f(x_2)$) | Ignoruje przekazaną w POST listę wykonanych testów (`executed_tests`) i zawsze zwraca `risk_coverage: 1.0` (`passed: true`), nawet gdy `executed_tests: ["test_unrelated"]`. |
| 3 | `rae-quality` (`:8010`) | `/v2/quality/defect-metrics` | `FAIL` ($f(x_1) == f(x_2)$) | Dla czystego kodu i kodu z `eval(user_input)` zwraca identyczne `defect_density: 0.01, zero_warning_status: true`. |
| 4 | `rae-quality` (`:8010`) | `/v2/quality/predictive-plan` | `FAIL` ($f(x_1) == f(x_2)$) | Dla `files: ["README.md"]` oraz `files: ["services/memory/service.go", "pkg/raecore/math/controller.go"]` zwraca identyczne `predicted_hotspots: []`. |
| 5 | `rae-phoenix` (`:8012`) | `/sessions/analyze` | `FAIL` ($f(x_1) == f(x_2)$) | Dla sesji zdrowej i sesji z 3 błędami (`panic: nil pointer`, `timeout`) zwraca identyczne `anomalies_found: 0`. |
| 6 | `rae-portal` (`:8082`) | `/v2/portal/global-search` | `FAIL` ($f(x_1) == f(x_2)$) | Handler w `services/portal/service.go` czyta wyłącznie `r.URL.Query().Get("query")` i ignoruje ciało POST JSON `{"query": "..."}`, zwracając stałe `query: "*", total_hits: 4`. |

*(Dodatkowo w trakcie inspekcji zidentyfikowano, że `/v2/knowledge/resolve` w `services/orchestrator/service.go` oraz `/v2/phoenix/create` w `services/phoenix/service.go` również zwracały stałe pola niezależnie od treści `query`/`spec` — zostaną naprawione razem z powyższą szóstką).*

---

### 2.3. Znaleziska Warstwy 4 (`Stateful End-to-End Workflow Invariant Chain` — 3 przerwane ogniwa stanowe)

1. **Ogniwo 1 (`Semantic Document Ingestion -> KnowledgeGraph Triples`)**:
   - Endpoint ingestii dokumentów był zamontowany wyłącznie pod `/v2/memories/ingest-document` (brak aliasu `/v2/ingest/document`), nie zapisywał zmian do `s.temporalGraph.RecordChange(...)` oraz brakowało endpointu `/v2/graph/triples` zwracającego zagregowane trójki `(subject, relation, object)` z grafu wiedzy i logu temporalnego.
2. **Ogniwo 3 (`16-Bucket Merkle Tree root_hash mutation on write`)**:
   - Endpoint drzewa Merkle w `v45_extensions.go` był zamontowany pod `/v2/mesh/merkle-tree` i zagnieżdżał skrót pod `tree.merkle_root`, podczas gdy klienci oczekują również ścieżki `/v2/mesh/merkle/buckets` z kluczem głównym `root_hash` oraz `bucket_hashes`.
3. **Ogniwo 6 (`Post-erasure query non-null JSON array contract`)**:
   - Gdy po wymazaniu danych RODO (`gdpr-erase`) zapytanie `GET /v2/memories/?query=anna.nowak@example.com` zwraca 0 wyników, sterowniki `memorystore` i `sqlstore` zwracały pusty wskaźnik wycinka `nil` (`var results []models.ScoredMemoryItem`), co biblioteka `encoding/json` w Go serializuje jako `"results": null` zamiast pustej tablicy JSON `"results": []` wymaganej przez kontrakt Python FastAPI/Pydantic.

---

### 2.4. Znaleziska Warstwy 5 (`Database Schema, SQL DDL & Environment Variable Contract Audit`)

1. **Brakujący alias eksportu DDL SQL (`/v2/db/schema/sql`)**:
   - Eksport SQL istniał pod `/v2/system/storage/export-sql` (klucz `"sql"`), natomiast brakowało kanonicznej ścieżki `/v2/db/schema/sql` zwracającej pełny schemat DDL 12 tabel PostgreSQL (w tym `CREATE TABLE IF NOT EXISTS memories` oraz `CREATE TABLE IF NOT EXISTS governance_outbox`) pod kluczem `"sql_ddl"` oraz `"ddl"`.
2. **5 nieobsłużonych zmiennych środowiskowych w sterownikach Go**:
   - W `pkg/drivers/agnostic/broker.go` oraz `pkg/drivers/llm/client.go` brakowało jawnego odczytu `os.Getenv` dla 5 kanonicznych zmiennych z wersji Python:
     - `RAE_DB_MODE` (`migrate` / `validate` / `ignore`),
     - `POSTGRES_URL` (DSN połączenia PostgreSQL obok `RAE_POSTGRES_DSN`),
     - `ANTHROPIC_API_KEY`,
     - `OPENAI_API_KEY`,
     - `GEMINI_API_KEY`.
