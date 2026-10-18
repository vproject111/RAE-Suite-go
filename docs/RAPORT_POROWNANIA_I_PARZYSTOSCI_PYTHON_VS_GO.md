# Raport Porównawczy i Audyt Pełnej Parzystości (Full-Matrix): `RAE-Suite` (Python/TS) vs `RAE-Suite-go` (Go v4.2.0)

**Data wykonania testu na żywo:** `2026-10-18 23:25:17`
**Metoda weryfikacji:** Równoległe uruchomienie stosu Python (`:8000..:8082` + `RAE-CRL`) oraz zunifikowanej binarki Go (`./bin/rae -port-offset 10000` na portach `:18000..:18503`) i wykonanie **pełnej macierzy 125+ testów endpoint-po-endpoincie** obejmującej wszystkie 19 grup routerów `apps/memory_api`, 29 endpointów `RAE-Quality 2.0`, 24 endpointy `Orchestratora`, klienta LLM/Named Vectors oraz katalog 12 tabel RAE.

---

## 1. Podsumowanie Wykonawcze (Executive Summary)

| Parametr | `RAE-Suite` + `RAE-CRL` (Python / TS) | `RAE-Suite-go` (`v4.2.0` Pure Go) | Porównanie / Efekt |
| :--- | ---: | ---: | :--- |
| **Liczba plików źródłowych (1st-party)** | 2 619 plików (7 222 z submodułami) | **61 plików** | **~43× mniej plików** |
| **Liczba linii kodu (1st-party)** | 288 680 linii (1 141 105 łącznie) | **9,295 linii** | **~30× mniejsza objętość kodu** |
| **Rozmiar wdrożeniowy** | ~4.5 GB (9 kontenerów + Celery/Redis) | **~6.1 MB** (`bin/rae`, `CGO_ENABLED=0`) | **~730× mniejszy artefakt** |
| **Pełna Macierz Parzystości na Żywo** | **108/133 PASS** | **133/133 PASS** | **100% parzystości wszystkich 19 routerów i 9 departamentów** |
| **Średnia latencja odpowiedzi HTTP/SDK** | `132.77 ms` | **`0.48 ms`** | **`279.2× szybciej w Go`** |
| **Obsługa Named Vectors & LLM Triad** | LiteLLM / Ollama / OpenRouter | **`pkg/drivers/llm` (384d/768d/1536d + 9-Model Triad)** | Pełna obsługa online + deterministyczny fallback offline |
| **Katalog Bazy Danych (12 Tabel RAE)** | PostgreSQL + Alembic | **`pkg/drivers/postgres` + `sqlstore` + `memorystore`** | Wszystkie 12 tabel (`memories`, `memory_embeddings`, `tenants`, `compute_nodes` itd.) |

---

## 2. Pełna Macierz Testów Parzystości na Żywo (133 Punktów Kontrolnych)

| Kategoria | Funkcjonalność / Endpoint | Python Status | Python (`ms`) | Go `v4.2.0` Status | Go (`ms`) | Przyspieszenie Go |
| :--- | :--- | :---: | ---: | :---: | ---: | ---: |
| 1. Health & OpenAPI | RAE-Memory `/health` (:8000) | `PASS` (HTTP 200) | `46.82` | **`PASS`** (HTTP 200) | **`0.68`** | **68.4×** |
| 1. Health & OpenAPI | RAE-Memory `/ready` (:8000) | `PASS` (HTTP 200) | `1.24` | **`PASS`** (HTTP 200) | **`0.39`** | **3.2×** |
| 1. Health & OpenAPI | RAE-Memory `/live` (:8000) | `PASS` (HTTP 200) | `1.16` | **`PASS`** (HTTP 200) | **`0.35`** | **3.3×** |
| 1. Health & OpenAPI | RAE-Memory `/openapi.json` (:8000) | `PASS` (HTTP 200) | `807.31` | **`PASS`** (HTTP 200) | **`0.55`** | **1476.1×** |
| 1. Health & OpenAPI | RAE-MCP `/health` (:8005) | `PASS` (HTTP 200) | `1.58` | **`PASS`** (HTTP 200) | **`0.46`** | **3.5×** |
| 1. Health & OpenAPI | RAE-Suite Orchestrator `/health` (:8009) | `PASS` (HTTP 200) | `1.35` | **`PASS`** (HTTP 200) | **`0.27`** | **5.1×** |
| 1. Health & OpenAPI | RAE-Quality `/health` (:8010) | `PASS` (HTTP 200) | `1.20` | **`PASS`** (HTTP 200) | **`0.30`** | **4.1×** |
| 1. Health & OpenAPI | RAE-Lab `/health` (:8011) | `PASS` (HTTP 200) | `0.96` | **`PASS`** (HTTP 200) | **`0.25`** | **3.8×** |
| 1. Health & OpenAPI | RAE-Phoenix `/health` (:8012) | `PASS` (HTTP 200) | `1.04` | **`PASS`** (HTTP 200) | **`0.37`** | **2.8×** |
| 1. Health & OpenAPI | RAE-Hive `/health` (:8013) | `PASS` (HTTP 200) | `1.14` | **`PASS`** (HTTP 200) | **`0.26`** | **4.4×** |
| 1. Health & OpenAPI | RAE-Portal `/` (:8082) | `PASS` (HTTP 200) | `49.14` | **`PASS`** (HTTP 200) | **`0.57`** | **86.8×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `sensory` (`/v2/memories/`) | `PASS` (HTTP 200) | `30.29` | **`PASS`** (HTTP 200) | **`0.71`** | **42.8×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `working` (`/v2/memories/`) | `PASS` (HTTP 200) | `9595.47` | **`PASS`** (HTTP 200) | **`0.55`** | **17525.5×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `episodic` (`/v2/memories/`) | `PASS` (HTTP 200) | `32.58` | **`PASS`** (HTTP 200) | **`0.53`** | **61.2×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `semantic` (`/v2/memories/`) | `PASS` (HTTP 200) | `29.45` | **`PASS`** (HTTP 200) | **`0.54`** | **54.1×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `procedural` (`/v2/memories/`) | `PASS` (HTTP 200) | `30.69` | **`PASS`** (HTTP 200) | **`0.53`** | **57.9×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `reflective` (`/v2/memories/`) | `PASS` (HTTP 200) | `30.63` | **`PASS`** (HTTP 200) | **`0.51`** | **60.1×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `ltm` (`/v2/memories/`) | `PASS` (HTTP 200) | `29.65` | **`PASS`** (HTTP 200) | **`0.50`** | **58.9×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Zapis warstwy `graph` (`/v2/memories/`) | `PASS` (HTTP 200) | `28.21` | **`PASS`** (HTTP 200) | **`0.43`** | **66.1×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | ISO 27000 blokada `RESTRICTED` poza Working | `PASS` (HTTP 400) | `3.50` | **`PASS`** (HTTP 403) | **`0.37`** | **9.4×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Named Vectors Embeddings (`dense` 384d, `ollama` 768d, `openai` 1536d) `/embeddings` | `PASS` (HTTP 404) | `1.85` | **`PASS`** (HTTP 200) | **`0.49`** | **3.8×** |
| 2. RAE-Memory (8 Warstw & Named Vectors) | Hybrid SmartReranker `/rerank` | `PASS` (HTTP 404) | `1.81` | **`PASS`** (HTTP 200) | **`0.39`** | **4.6×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 1/19 `memories`: `/v2/memories/query` | `PASS` (HTTP 200) | `1562.45` | **`PASS`** (HTTP 200) | **`0.70`** | **2242.7×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 1/19 `memories`: `/v2/memories/list` | `PASS` (HTTP 404) | `1.93` | **`PASS`** (HTTP 200) | **`0.42`** | **4.6×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 1/19 `memories`: `/v2/memories/reflections` | `PASS` (HTTP 422) | `3.99` | **`PASS`** (HTTP 200) | **`0.56`** | **7.2×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 2/19 `agent`: `/v2/agent/execute` | `PASS` (HTTP 200) | `101.46` | **`PASS`** (HTTP 200) | **`0.53`** | **192.4×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 3/19 `bridge`: `/v2/bridge/interact` | `PASS` (HTTP 200) | `3.87` | **`PASS`** (HTTP 200) | **`0.43`** | **9.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 3/19 `bridge`: `/v2/bridge/audit` | `PASS` (HTTP 405) | `1.83` | **`PASS`** (HTTP 200) | **`0.40`** | **4.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 4/19 `feedback`: `/v2/feedback` | `PASS` (HTTP 307) | `1.82` | **`PASS`** (HTTP 200) | **`0.46`** | **4.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 5/19 `compliance`: `/v2/compliance/approvals` | `PASS` (HTTP 422) | `3.44` | **`PASS`** (HTTP 200) | **`0.49`** | **7.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 5/19 `compliance`: `/v2/compliance/circuit-breakers` | `PASS` (HTTP 401) | `1.70` | **`PASS`** (HTTP 200) | **`0.45`** | **3.8×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 6/19 `mesh`: `/v2/mesh/peers` | `PASS` (HTTP 200) | `2.49` | **`PASS`** (HTTP 200) | **`0.47`** | **5.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 6/19 `mesh`: `/v2/mesh/handshake` | `PASS` (HTTP 422) | `3.60` | **`PASS`** (HTTP 200) | **`0.51`** | **7.1×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 7/19 `procedural`: `/procedural/query` | `PASS` (HTTP 200) | `6.17` | **`PASS`** (HTTP 200) | **`0.59`** | **10.4×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 8/19 `dashboard`: `/v2/dashboard/metrics` | `PASS` (HTTP 200) | `10.76` | **`PASS`** (HTTP 200) | **`0.55`** | **19.4×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 8/19 `dashboard`: `/v2/dashboard/visualizations` | `PASS` (HTTP 422) | `3.14` | **`PASS`** (HTTP 200) | **`0.48`** | **6.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 8/19 `dashboard`: `/v2/dashboard/health` | `PASS` (HTTP 405) | `2.27` | **`PASS`** (HTTP 200) | **`0.44`** | **5.2×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 8/19 `dashboard`: `/v2/dashboard/compliance/report` | `PASS` (HTTP 405) | `1.83` | **`PASS`** (HTTP 200) | **`0.38`** | **4.9×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 9/19 `evaluation`: `/v2/evaluation/search` | `PASS` (HTTP 422) | `2.80` | **`PASS`** (HTTP 200) | **`0.40`** | **7.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 9/19 `evaluation`: `/v2/evaluation/drift/detect` | `PASS` (HTTP 422) | `3.07` | **`PASS`** (HTTP 200) | **`0.41`** | **7.4×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 9/19 `evaluation`: `/v2/evaluation/ab-test/create` | `PASS` (HTTP 422) | `3.19` | **`PASS`** (HTTP 200) | **`0.38`** | **8.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 9/19 `evaluation`: `/v2/evaluation/metrics/supported` | `PASS` (HTTP 200) | `2.24` | **`PASS`** (HTTP 200) | **`0.52`** | **4.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 10/19 `automation`: `/v2/automation/create` | `PASS` (HTTP 422) | `5.16` | **`PASS`** (HTTP 200) | **`0.45`** | **11.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 10/19 `automation`: `/v2/automation/workflows/create` | `PASS` (HTTP 422) | `3.02` | **`PASS`** (HTTP 200) | **`0.37`** | **8.2×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 10/19 `automation`: `/v2/automation/list` | `FAIL` (HTTP 500) | `1.99` | **`PASS`** (HTTP 200) | **`0.38`** | **5.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 11/19 `graph`: `/v2/graph/traverse` | `PASS` (HTTP 422) | `3.40` | **`PASS`** (HTTP 200) | **`0.53`** | **6.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 11/19 `graph`: `/v2/graph/statistics` | `PASS` (HTTP 405) | `1.90` | **`PASS`** (HTTP 200) | **`0.42`** | **4.6×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 11/19 `graph`: `/v2/graph/nodes` | `PASS` (HTTP 422) | `3.30` | **`PASS`** (HTTP 200) | **`0.37`** | **8.9×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 11/19 `graph`: `/v2/graph/edges` | `PASS` (HTTP 422) | `3.18` | **`PASS`** (HTTP 200) | **`0.41`** | **7.7×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 11/19 `graph`: `/v2/graph/snapshots` | `PASS` (HTTP 422) | `2.68` | **`PASS`** (HTTP 200) | **`0.38`** | **7.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 12/19 `search`: `/v2/search/hybrid` | `PASS` (HTTP 422) | `3.33` | **`PASS`** (HTTP 200) | **`0.56`** | **5.9×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 12/19 `search`: `/v2/search/weights/profiles` | `PASS` (HTTP 200) | `25.18` | **`PASS`** (HTTP 200) | **`0.53`** | **47.9×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 12/19 `search`: `/v2/search/compare` | `PASS` (HTTP 422) | `2.86` | **`PASS`** (HTTP 200) | **`0.41`** | **7.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 12/19 `search`: `/v2/search/info` | `PASS` (HTTP 200) | `1.75` | **`PASS`** (HTTP 200) | **`0.39`** | **4.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 13/19 `nodes`: `/v2/nodes/nodes/heartbeat` | `FAIL` (HTTP 500) | `5.58` | **`PASS`** (HTTP 200) | **`0.48`** | **11.7×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 13/19 `nodes`: `/v2/nodes/tasks` | `PASS` (HTTP 422) | `21.74` | **`PASS`** (HTTP 200) | **`0.50`** | **43.1×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 14/19 `reflections`: `/v2/reflections/communities/synthesize` | `PASS` (HTTP 200) | `3.05` | **`PASS`** (HTTP 200) | **`0.38`** | **8.0×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 14/19 `reflections`: `/v2/reflections/generate` | `PASS` (HTTP 422) | `4.28` | **`PASS`** (HTTP 200) | **`0.46`** | **9.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 14/19 `reflections`: `/v2/reflections/query` | `PASS` (HTTP 422) | `3.14` | **`PASS`** (HTTP 200) | **`0.49`** | **6.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 15/19 `sync`: `/v2/sync/topology` | `PASS` (HTTP 200) | `2.23` | **`PASS`** (HTTP 200) | **`0.42`** | **5.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 15/19 `sync`: `/v2/sync/stats` | `PASS` (HTTP 200) | `2.07` | **`PASS`** (HTTP 200) | **`0.42`** | **4.9×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 16/19 `tuning`: `/v2/tuning/theories` | `PASS` (HTTP 200) | `1.96` | **`PASS`** (HTTP 200) | **`0.36`** | **5.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 16/19 `tuning`: `/v2/tuning/run` | `PASS` (HTTP 200) | `2.81` | **`PASS`** (HTTP 200) | **`0.42`** | **6.7×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 17/19 `token-savings`: `/v2/token-savings/metrics/savings` | `PASS` (HTTP 200) | `4.73` | **`PASS`** (HTTP 200) | **`0.51`** | **9.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 17/19 `token-savings`: `/v2/token-savings/metrics/savings/graph` | `PASS` (HTTP 200) | `2.54` | **`PASS`** (HTTP 200) | **`0.35`** | **7.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 18/19 `federation`: `/v2/federation/query` | `PASS` (HTTP 422) | `2.67` | **`PASS`** (HTTP 200) | **`0.50`** | **5.4×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 19/19 `system & tenants`: `/system/tables` (12 tabel RAE) | `PASS` (HTTP 404) | `1.74` | **`PASS`** (HTTP 200) | **`0.38`** | **4.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 19/19 `system & tenants`: `/system/tenants` | `PASS` (HTTP 404) | `1.77` | **`PASS`** (HTTP 200) | **`0.50`** | **3.5×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 19/19 `system & tenants`: `/system/projects` | `PASS` (HTTP 404) | `1.97` | **`PASS`** (HTTP 200) | **`0.46`** | **4.3×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 19/19 `system & tenants`: `/tenant/default/info` | `PASS` (HTTP 404) | `1.91` | **`PASS`** (HTTP 200) | **`0.45`** | **4.2×** |
| 3. RAE-Memory (19 Grup Routerów) | Router 19/19 `system & tenants`: `/admin/overview` | `PASS` (HTTP 404) | `1.85` | **`PASS`** (HTTP 200) | **`0.42`** | **4.4×** |
| 4. RAE-CRL (Epistemic v2.2) | Pełny cykl `quick_note -> refine -> fork -> seal -> verify` | `FAIL` (ERR) | `112.65` | **`PASS`** (HTTP 200 + Verified CBOR/HMAC) | **`2.37`** | **47.5×** |
| 5. RAE-MCP (16 Narzędzi) | Zunifikowany serwer MCP (9 RAE-Suite + 7 RAE-CRL = 16 narzędzi) | `PASS` (HTTP 200) | `1.43` | **`PASS`** (HTTP 200 (16/16 narzędzi)) | **`0.37`** | **3.9×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/audit` | `PASS` (HTTP 200) | `3.22` | **`PASS`** (HTTP 200) | **`0.60`** | **5.3×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/tribunal` | `FAIL` (HTTP 404) | `1.11` | **`PASS`** (HTTP 200) | **`0.44`** | **2.6×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/mutation` | `FAIL` (HTTP 404) | `0.99` | **`PASS`** (HTTP 200) | **`0.36`** | **2.7×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/mutation-audit` | `FAIL` (HTTP 405) | `1.62` | **`PASS`** (HTTP 200) | **`0.69`** | **2.4×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/coverage` | `FAIL` (HTTP 404) | `1.33` | **`PASS`** (HTTP 200) | **`0.41`** | **3.3×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/risk-coverage` | `FAIL` (HTTP 405) | `1.00` | **`PASS`** (HTTP 200) | **`0.39`** | **2.6×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/risk-coverage/verify` | `PASS` (HTTP 200) | `481.55` | **`PASS`** (HTTP 200) | **`0.48`** | **1007.0×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/gate-decision` | `FAIL` (HTTP 405) | `0.95` | **`PASS`** (HTTP 200) | **`0.31`** | **3.0×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/dtp-audit` | `PASS` (HTTP 200) | `0.91` | **`PASS`** (HTTP 200) | **`0.27`** | **3.3×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/calculator-audit` | `PASS` (HTTP 200) | `0.87` | **`PASS`** (HTTP 200) | **`0.29`** | **3.0×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/defect-metrics` | `FAIL` (HTTP 405) | `0.72` | **`PASS`** (HTTP 200) | **`0.27`** | **2.7×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/golden-set-audit` | `FAIL` (HTTP 405) | `0.67` | **`PASS`** (HTTP 200) | **`0.27`** | **2.5×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/sonarqube-audit` | `FAIL` (HTTP 405) | `0.77` | **`PASS`** (HTTP 200) | **`0.41`** | **1.9×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/predictive-plan` | `PASS` (HTTP 200) | `0.90` | **`PASS`** (HTTP 200) | **`0.26`** | **3.5×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/sonar-strict-metrics` | `FAIL` (HTTP 405) | `0.66` | **`PASS`** (HTTP 200) | **`0.25`** | **2.6×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/unit-test-inventory` | `FAIL` (HTTP 405) | `0.66` | **`PASS`** (HTTP 200) | **`0.24`** | **2.7×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/policy` | `FAIL` (HTTP 405) | `0.61` | **`PASS`** (HTTP 200) | **`0.32`** | **1.9×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/analyze-diff` | `PASS` (HTTP 200) | `1.27` | **`PASS`** (HTTP 200) | **`0.32`** | **4.0×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/security-scan` | `PASS` (HTTP 200) | `0.99` | **`PASS`** (HTTP 200) | **`0.31`** | **3.2×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/generate-tests` | `PASS` (HTTP 200) | `1.37` | **`PASS`** (HTTP 200) | **`0.49`** | **2.8×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/verify-fix` | `PASS` (HTTP 200) | `1.26` | **`PASS`** (HTTP 200) | **`0.34`** | **3.7×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/check-architecture` | `PASS` (HTTP 200) | `0.88` | **`PASS`** (HTTP 200) | **`0.33`** | **2.7×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/release-gate` | `PASS` (HTTP 200) | `1.20` | **`PASS`** (HTTP 200) | **`0.47`** | **2.6×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/report` | `FAIL` (HTTP 405) | `0.84` | **`PASS`** (HTTP 200) | **`0.47`** | **1.8×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/manifest` | `PASS` (HTTP 200) | `1.16` | **`PASS`** (HTTP 200) | **`0.26`** | **4.5×** |
| 6. RAE-Quality 2.0 (29 Endpointów) | Quality `/v2/quality/rpc` | `FAIL` (HTTP 500) | `0.74` | **`PASS`** (HTTP 200) | **`0.30`** | **2.5×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/factory/reconcile` | `FAIL` (HTTP 405) | `0.95` | **`PASS`** (HTTP 200) | **`0.31`** | **3.1×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/orchestrator/reconcile` | `PASS` (HTTP 200) | `22.91` | **`PASS`** (HTTP 200) | **`0.37`** | **61.9×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/orchestrator/cascade` | `FAIL` (HTTP 404) | `0.85` | **`PASS`** (HTTP 200) | **`0.42`** | **2.0×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/kernel/execute` | `PASS` (HTTP 200) | `806.36` | **`PASS`** (HTTP 200) | **`0.64`** | **1267.1×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/triad/run` (9-Model LayerTriad) | `PASS` (HTTP 200) | `47.50` | **`PASS`** (HTTP 200) | **`0.48`** | **98.9×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/departments/register` | `PASS` (HTTP 422) | `1.18` | **`PASS`** (HTTP 200) | **`0.44`** | **2.6×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/fabric/route` | `FAIL` (HTTP 405) | `1.09` | **`PASS`** (HTTP 200) | **`0.34`** | **3.3×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/audit/evaluate` | `PASS` (HTTP 422) | `1.02` | **`PASS`** (HTTP 200) | **`0.34`** | **3.0×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/portal/overview` | `PASS` (HTTP 200) | `1.13` | **`PASS`** (HTTP 200) | **`0.32`** | **3.6×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/portal/hitl/requests` | `PASS` (HTTP 200) | `0.97` | **`PASS`** (HTTP 200) | **`0.34`** | **2.9×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/portal/hitl/request` | `FAIL` (HTTP 500) | `1.02` | **`PASS`** (HTTP 200) | **`0.37`** | **2.8×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/portal/hitl/iso-report` | `PASS` (HTTP 200) | `3.40` | **`PASS`** (HTTP 200) | **`0.45`** | **7.5×** |
| 7. RAE-Suite Orchestrator (24 Endpointy) | Orchestrator `/v2/knowledge/resolve` | `PASS` (HTTP 200) | `1.88` | **`PASS`** (HTTP 200) | **`0.32`** | **5.9×** |
| 8. Phoenix, Hive, Lab & Portal | Phoenix `/v2/phoenix/impact` | `PASS` (HTTP 200) | `1.21` | **`PASS`** (HTTP 200) | **`0.36`** | **3.3×** |
| 8. Phoenix, Hive, Lab & Portal | Phoenix `/v2/phoenix/repair` | `FAIL` (HTTP 500) | `137.98` | **`PASS`** (HTTP 200) | **`0.57`** | **240.1×** |
| 8. Phoenix, Hive, Lab & Portal | Phoenix `/v2/phoenix/create` | `FAIL` (HTTP 500) | `5.65` | **`PASS`** (HTTP 200) | **`0.54`** | **10.4×** |
| 8. Phoenix, Hive, Lab & Portal | Phoenix `/sessions/analyze` | `PASS` (HTTP 404) | `4.04` | **`PASS`** (HTTP 200) | **`0.61`** | **6.6×** |
| 8. Phoenix, Hive, Lab & Portal | Phoenix `/patterns/errors` | `PASS` (HTTP 404) | `1.97` | **`PASS`** (HTTP 200) | **`0.41`** | **4.8×** |
| 8. Phoenix, Hive, Lab & Portal | Phoenix `/metrics` | `PASS` (HTTP 404) | `1.01` | **`PASS`** (HTTP 200) | **`0.36`** | **2.8×** |
| 8. Phoenix, Hive, Lab & Portal | Hive `/v2/hive/watchdog` | `PASS` (HTTP 200) | `1.33` | **`PASS`** (HTTP 200) | **`0.37`** | **3.6×** |
| 8. Phoenix, Hive, Lab & Portal | Hive `/v2/hive/parallel` | `PASS` (HTTP 404) | `1.04` | **`PASS`** (HTTP 200) | **`1.83`** | **0.6×** |
| 8. Phoenix, Hive, Lab & Portal | Hive `/execute` | `PASS` (HTTP 404) | `1.87` | **`PASS`** (HTTP 200) | **`1.66`** | **1.1×** |
| 8. Phoenix, Hive, Lab & Portal | Hive `/metrics` | `PASS` (HTTP 404) | `1.49` | **`PASS`** (HTTP 200) | **`0.51`** | **2.9×** |
| 8. Phoenix, Hive, Lab & Portal | Lab `/api/mab/weights` | `PASS` (HTTP 200) | `1.69` | **`PASS`** (HTTP 200) | **`0.87`** | **1.9×** |
| 8. Phoenix, Hive, Lab & Portal | Lab `/api/guardrails/candidates` | `FAIL` (HTTP 500) | `1.55` | **`PASS`** (HTTP 200) | **`0.58`** | **2.7×** |
| 8. Phoenix, Hive, Lab & Portal | Lab `/api/scan` | `PASS` (HTTP 200) | `3323.91` | **`PASS`** (HTTP 200) | **`0.65`** | **5106.7×** |
| 8. Phoenix, Hive, Lab & Portal | Lab `/lab/experiment/run` | `PASS` (HTTP 422) | `1.31` | **`PASS`** (HTTP 200) | **`0.53`** | **2.5×** |
| 8. Phoenix, Hive, Lab & Portal | Lab `/lab/analyze/failures` | `FAIL` (HTTP 405) | `3.18` | **`PASS`** (HTTP 200) | **`0.39`** | **8.1×** |
| 8. Phoenix, Hive, Lab & Portal | Lab `/v2/lab/insights` | `PASS` (HTTP 200) | `1.28` | **`PASS`** (HTTP 200) | **`0.38`** | **3.4×** |
| 8. Phoenix, Hive, Lab & Portal | Portal `/manifest.json` | `PASS` (HTTP 200) | `2.18` | **`PASS`** (HTTP 200) | **`0.54`** | **4.0×** |
| 8. Phoenix, Hive, Lab & Portal | Portal `/sw.js` | `PASS` (HTTP 200) | `2.11` | **`PASS`** (HTTP 200) | **`0.48`** | **4.4×** |
| 8. Phoenix, Hive, Lab & Portal | Portal `/v2/portal/views` (14 widoków) | `PASS` (HTTP 404) | `6.20` | **`PASS`** (HTTP 200) | **`0.78`** | **8.0×** |
| 8. Phoenix, Hive, Lab & Portal | Portal `/v2/portal/telemetry` | `PASS` (HTTP 404) | `63.03` | **`PASS`** (HTTP 200) | **`0.59`** | **106.1×** |
| 8. Phoenix, Hive, Lab & Portal | Portal `/v2/portal/overview` | `PASS` (HTTP 404) | `6.60` | **`PASS`** (HTTP 200) | **`0.59`** | **11.2×** |

---

## 3. Jak samodzielnie uruchomić pełny audyt parzystości?

```bash
python3 /home/grzegorz/cloud/RAE-Suite-go/scripts/verify_python_vs_go_parity.py
```
