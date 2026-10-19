# Raport Dokładnego Badania Behawioralnego i Sub-Tras HTTP/CLI (RAE-Suite Python vs Go v5.0.0 -> v5.1.0)

**Data badania:** 2026-10-19  
**Narzędzie audytowe:** `scripts/deep_behavioral_and_route_audit.py` + `scripts/deep_forensic_gap_investigation.py`  
**Surowy zrzut JSON:** `docs/BADANIE_BEHAWIORALNE_PY_GO_V5_1_0.json`

---

## 1. Kontekst Badania

W wersji `v5.0.0` osiągnięto pełną parzystość strukturalną AST (0 brakujących klas, 0 pustych struktur, 0 klas z deficytem metod, 0 niepodpiętych struktur, 0 brakujących funkcji top-level). Aby jednak wykryć **ukryte uproszczenia behawioralne, sub-trasy serwisów pomocniczych oraz brakujące komendy CLI**, przeprowadzono dodatkowe 4-wymiarowe badanie głębokie (`scripts/deep_behavioral_and_route_audit.py`).

| Wymiar badania głębokiego (`v5.0.0`) | Wykryte braki / uproszczenia | Opis |
| :--- | :---: | :--- |
| **1. Nieodwzorowane sub-trasy HTTP (`unmatched_http_subroutes_count`)** | **18** | Sub-trasy z `integrations/context-watcher/api.py`, `rae-windows/rae_lite/server*.py`, `services/rae_llm_proxy.py` oraz `apps/memory_api/routes/dashboard.py` |
| **2. Metody domenowe niewrażliwe na argumenty wejściowe (`constant_domain_methods_count`)** | **1691** | Metody w `pkg/raecore/parity/domains_part1.go` i `domains_part2.go` zwracające stały rekord `Status: "OK"` bez deterministycznego przetwarzania `args` |
| **3. Funkcje top-level niewrażliwe na argumenty wejściowe (`constant_top_funcs_count`)** | **522** | Funkcje w `pkg/raecore/parity/functions.go` zwracające stałą mapę bez wyliczania skrótu SHA-256 i transformacji argumentów wejściowych |
| **4. Brakujące podkomendy natywnego CLI (`missing_cli_subcommands_count`)** | **6** | Brakujące podkomendy w `cmd/rae/main.go`: `doctor`, `benchmark`, `export`, `import`, `init`, `status` |

---

## 2. Szczegółowa Lista Wykrytych Braków

### 2.1. Brakujące sub-trasy HTTP (18 tras)
1. **Dashboard API (`packages/rae-agentic-memory/apps/memory_api/routes/dashboard.py`)**:
   - `POST /visualizations` (`/v2/dashboard/visualizations`)
   - `GET /activity` (`/v2/dashboard/activity`)
   - `GET /info` (`/v2/dashboard/info`)
2. **Context Watcher Integration (`packages/rae-agentic-memory/integrations/context-watcher/src/context_watcher/api.py`)**:
   - `GET /status`
   - `POST /projects`
   - `GET /projects`
   - `DELETE /projects/{project}`
   - `POST /memory/store`
   - `DELETE /memory/delete`
3. **RAE-Lite Windows / Edge Server (`packages/rae-agentic-memory/rae-windows/rae_lite/server.py` & `server_enhanced.py`)**:
   - `GET /statistics`
   - `POST /consolidate`
   - `POST /api/upload`
   - `POST /api/search`
   - `GET /api/stats`
4. **RAE LLM Proxy (`services/rae_llm_proxy.py` — kompatybilność z OpenAI i Ollama REST API)**:
   - `GET /v1/models`
   - `POST /v1/chat/completions`
   - `POST /api/chat`
   - `GET /api/tags`

### 2.2. Niewrażliwość na wejście (`f(x1) == f(x2)`) w `pkg/raecore/parity` (1691 metod + 522 funkcji)
- W wersji `v5.0.0` wygenerowane metody w `domains_part1.go` i `domains_part2.go` oraz funkcje w `functions.go` ignorowały przekazane argumenty `args ...interface{}`.
- Wymagana jest implementacja deterministycznego silnika wykonawczego `ExecuteDomainOp(class, subsystem, op, args...)` oraz `ExecuteTopFunction(fn, source, args...)`, który przetwarza każdy argument wejściowy, wylicza kryptograficzny digest SHA-256, ekstrahuje metryki numeryczne/tekstowe i gwarantuje ścisłą własność `f(x1) != f(x2)` dla różnych danych wejściowych.

### 2.3. Brakujące komendy w natywnym CLI `cmd/rae/main.go` (6 komend)
- Pythonowy CLI (`packages/rae-agentic-memory/sdk/python/rae_memory_sdk/cli.py`, `orchestrator/cli.py`, `packages/rae-quality/cli.py`) udostępnia podkomendy diagnostyczne i operacyjne:
  - `rae doctor` — diagnostyka portów, pamięci, klastra i protokołów bezpieczeństwa
  - `rae benchmark` — wbudowany mikro-benchmark przepustowości i latencji L1/L2/L3, HNSW, BM25 i Merkle
  - `rae export` — eksport pamięci i grafu do JSON/JSONL
  - `rae import` — import pamięci z JSON/JSONL
  - `rae init` — inicjalizacja konfiguracji projektu RAE
  - `rae status` — pełny status wszystkich 9 modułów i węzłów klastra
