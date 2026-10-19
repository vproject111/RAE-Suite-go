# Plan Iteracyjny Domknięcia 100% Głębokiej Logiki Domenowej `RAE-Suite` (Python) w `RAE-Suite-go` (`v4.4.0`)

## 1. Cel Planu

Po wdrożeniu `v4.3.0` (`159/159 PASS` na żywo, `0.44 ms` średniej latencji) wersja Go posiadała już pełną 8-warstwową pamięć `RAE-Memory`, `RAE-CRL`, 16 narzędzi MCP, 9 departamentów oraz 16 silników rdzeniowych (`ExecutionAuditLedger` Kto/Co/Dlaczego/Za ile/Ile trwało/Jaki efekt, `BatchOptimizationEngine` z Efektem Skali Wiedzy, `CognitivePlanner` ToT+MCTS+PRM, `DynamicCouncilSelector`, `SwarmConsensusEngine`, `SagaCoordinator`, `StreamingFunctionComposer`, `SpeculativeToolExecutor`, `ClaimCheckManager`, `SafeReplayEngine`, `A2AProtocolAdapter`, `IngestionPIIScrubber`, `ShadowEvaluator`, `CuriosityEngine`, `ToolForge`, `ThinDeviceRuntime`).

Szczegółowy audyt porównawczy wszystkich `72` plików produkcyjnych w `RAE-Suite/core/`, `26` silników w `packages/rae-quality/engines/`, `130` plików w `packages/rae-phoenix/feniks/` oraz `19` routerów w `apps/memory_api/` wykazał jednak **4 obszary głębokiej logiki domenowej**, które w Go działały jeszcze w wersji uproszczonej (*fast-path*) lub nie były przeniesione. Niniejszy plan domyka te 4 obszary w 100%.

---

## 2. Podział na Iteracje (`Iteracja D1` – `Iteracja D4`)

### Iteracja D1: Płaszczyzna Samodoskonalenia (`Improvement Plane`), Zarządzanie Klastrem GPU, Odporność Sieciowa, Telemetria W3C/PSI i Bezpieczeństwo Keycloak (`feature/d1-improvement-plane-and-ops-resilience`)
Utworzenie pakietu `/home/grzegorz/cloud/RAE-Suite-go/pkg/raecore/ops/engines.go` oraz `engines_test.go` wdrażającego pozostałe 22 silniki z `RAE-Suite/core/`:
1. **Self-Improvement Plane (`core/improvement_plane/*`)**:
   - `HypothesisManager`, `ExperimentManager`, `ShadowRunner`, `CanaryManager` (automatyczny rollback gdy `error_rate > 0.05`) oraz `PromotionGate` (wymagający zaliczenia `has_shadow_run`, `shadow_passed`, `has_rollback_plan` oraz `auditor_approved`).
2. **Orkiestracja Klastra Obliczeniowego (`ClusterManager` — `core/cluster_manager.py`)**:
   - Rejestr węzłów `Node 1 Lumina` (`100.68.166.117`, RTX 4080, 64GB RAM), `Node 2 Julka` (`100.70.32.40`, RTX 4080 SUPER, 64GB RAM) i `Node 3 Piotrek` (`172.30.15.11:11434`, 128GB RAM, Ollama), aktywne sondowanie zdrowia i routowanie zadań obliczeniowych.
3. **Odporność i Samonaprawa Infrastruktury (`core/circuit_breaker.py`, `redis_rate_limiter.py`, `rollback_manager.py`, `escalation_engine.py`, `infra_reconciler.py`, `gitops_daemon.py`)**:
   - `TransportCircuitBreaker`: maszyna stanów `CLOSED -> OPEN -> HALF_OPEN` z konfigurowalnym `failure_threshold` i `recovery_timeout_s`.
   - `DistributedRateLimiter`: algorytm Token Bucket (`capacity`, `refill_rate_per_sec`) z dokładnym wyliczaniem `remaining_tokens` i `reset_after_sec`.
   - `RollbackSLAManager`: macierz SLA (`container_restart: 15s`, `git_worktree_revert: 30s`, `config_restore: 60s`, `db_schema_rollback: 120s`, `vector_projection_rollback: 300s`) oraz blokada akcji `R4`/`R5` bez zweryfikowanego w sandboxie planu rollbacku.
   - `DualEscalationEngine`: automatyczna eskalacja do `HERMES`, `OPENCODE` lub `HUMAN_OPERATOR`.
   - `InfraReconciler`: monitorowanie portów TCP i maksymalnie 3 mikro-restarty przed eskalacją przy 4. awarii.
   - `GitOpsDaemon`: zarządzanie gałęziami `agent/task-<id>`, kryptograficzne podpisywanie commitów i generowanie artefaktów PR z dowodami ISO.
4. **Rejestry, Telemetria W3C/PSI, Keycloak JWT i Global Search (`core/prompt_registry.py`, `tool_gateway.py`, `telemetry_monitor.py`, `telemetry_propagation.py`, `keycloak_auth.py`, `autonomy_evaluation_harness.py`, `portal_global_search.py`)**:
   - `FederatedPromptRegistry`: warstwowe składanie szablonów `Base -> Org -> Team -> Feature` z zamrożeniem szablonu bazowego i skrótem SHA-256.
   - `ToolGateway`: logowanie trajektorii wywołań narzędzi i pamięć podręczna `Empty Run Prevention` (`context_hash -> cached_result`).
   - `EmbeddingDriftDetector`: obliczanie wskaźnika **PSI (Population Stability Index)** na koszykach rozkładu z progiem alarmowym `0.25`.
   - `TraceContextPropagator`: wstrzykiwanie i ekstrakcja nagłówków W3C `traceparent` (`00-<32hex>-<16hex>-<01|00>`).
   - `KeycloakAuthValidator`: tworzenie i weryfikacja tokenów JWT `HS256` z walidacją `iss`, `aud`, `roles` oraz `capability_scopes`.
   - `AutonomyEvaluationHarness`: obrona przed *Prompt Injection*, weryfikacja twardych limitów budżetowych (*Global Stop Condition*) oraz macierz retencji danych ISO (`R0/R1=90 dni`, `R2/R3=365 dni`, `R4..R6=2555 dni`).
   - `RAEGlobalSearchEngine`: globalna wyszukiwarka krzyżowa w Portalu przeszukująca rejestr audytu i pamięć z automatyczną redakcją PII.

---

### Iteracja D2: Pełne Silniki Domenowe `RAE-Quality 2.0` — Katalog 198 Reguł, Walidator PDF/DTP, Weryfikator Kalkulatorów Cenowych, Pokrycie Ryzyka i Golden Set (`feature/d2-quality-198-dtp-and-calculators`)
Rozbudowa `/home/grzegorz/cloud/RAE-Suite-go/services/quality/` (`deep_engines.go` oraz `deep_engines_test.go`) o pełną logikę z `packages/rae-quality/engines/`:
1. **Katalog 198 Reguł Jakości i Bezpieczeństwa (`Catalog198Engine` — `catalog_198.py`)**:
   - Pełna lista wszystkich **198 tematów inżynieryjnych i bezpieczeństwa** + **12 obszarów AI/SaaS** + **9 reguł `RAE-EDITOR-SECURITY`** dostępna i weryfikowalna przez `/v2/quality/policy` oraz `/v2/quality/security-scan`.
2. **Walidator Geometrii Druku i PDF (`PDFDtpValidator` — `pdf_dtp_validator.py`)**:
   - Weryfikacja `TrimBox` vs `BleedBox` (wyliczanie spadów poziomych i pionowych `(bleed - trim)/2 >= min_bleed_mm`), limitu nafarbienia CMYK (`TAC = C + M + Y + K <= 300%`), zakazu *rich-black* dla drobnego tekstu (`< 12 pt`), osadzenia czcionek (`is_embedded == true`), warstw dodatkowych (`CutContour`, `Varnish_UV` z `overprint == true`) oraz rozdzielczości `DPI >= 300`.
3. **Weryfikator Silników Kalkulacji Cenowej (`CalculatorVerifier` — `calculator_verifier.py`)**:
   - `VerifyPriceMonotonicity`: sprawdza, czy wraz ze wzrostem nakładu (`volume`) cena całkowita (`total_price`) nigdy nie maleje, a cena jednostkowa (`unit_price = total_price / volume`) nigdy nie rośnie (z tolerancją zaokrągleń `0.1%`).
   - `VerifyFormatProportionality`: sprawdza spójność cenową formatów (np. A4 vs A5).
4. **Silnik Pokrycia Ryzyka (`RiskCoverageEngine`) oraz Zbiór Referencyjny (`GoldenSetRunner`)**:
   - Macierz krytycznych ścieżek ryzyka `P0/P1` (`RC-01` kalkulacja cen, `RC-02` geometria PDF Trim/Bleed/CMYK, `RC-03` izolacja JWT/RBAC, `RC-04` synchronizacja Merkle Mesh, `RC-05` twarde kontrakty DTO) oraz automatyczny przebieg audytu DTP na 5 kanonicznych produktach poligraficznych (*Golden Set*).

---

### Iteracja D3: Głęboki Graf Zależności AST w `RAE-Phoenix` oraz Pełna Stanowość 19 Routerów `RAE-Memory` (`feature/d3-phoenix-ast-graph-and-stateful-memory-routers`)
1. **Głęboki Graf Zależności i Receptury Refaktoryzacji w `RAE-Phoenix` (`services/phoenix/service.go`)**:
   - Parser importów dla plików Go, Python i TypeScript/JavaScript wyliczający przechodni promień rażenia (*Transitive Blast Radius*) na podstawie rzeczywistego grafu zależności, migawki zachowania (*Behavior Snapshots*) oraz katalog receptur refaktoryzacyjnych.
2. **Pełna Stanowość 19 Grup Routerów `RAE-Memory` (`services/memory/extended_routes.go`)**:
   - Reguły wyzwalaczy (`event-triggers`) rzeczywiście przechwytujące zdarzenia zapisu pamięci, stanowy rejestr budżetów i limitów najemców (`governance`), stanowy dziennik zgodności ISO (`compliance`) oraz integracja operacji grafowych (`/v2/graph`) bezpośrednio z `MemoryStoragePort`.

---

### Iteracja D4: Podłączenie Endpointów HTTP, Głęboka Weryfikacja Behawioralna Python vs Go (`180+` testów) i Wydanie `v4.4.0` (`feature/d4-wiring-verification-and-release-v4.4.0`)
1. Podłączenie wszystkich silników z Iteracji D1–D3 do `services/orchestrator`, `services/quality`, `services/phoenix`, `services/memory` i `services/portal`.
2. Rozbudowa `/home/grzegorz/cloud/RAE-Suite-go/scripts/verify_python_vs_go_parity.py` do **ponad 180 głębokich testów behawioralnych** sprawdzających nie tylko kod HTTP 200, ale też dokładne wykrywanie błędów spadów DTP, inwersji cen w kalkulatorach, auto-rollbacku Canary, przejść stanów Circuit Breakera, wyliczeń PSI, walidacji JWT Keycloak oraz pełnego katalogu 198 reguł.
3. Wygenerowanie zaktualizowanego raportu w `/home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_POROWNANIA_I_PARZYSTOSCI_PYTHON_VS_GO.md`, merge do `develop` i `main`, tag `v4.4.0` i publikacja na GitHubie.
