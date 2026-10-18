# Plan Iteracyjny Pełnej Parzystości Funkcjonalnej, Kontraktowej i Integracyjnej (`v4.2.0`)
## 100% Pokrycia wszystkich Routerów API V2, Klientów LLM/Embeddingów (Ollama/OpenRouter), Adaptera PostgreSQL i UI

**Data przygotowania:** 2026-10-18  
**Repozytorium docelowe:** `/home/grzegorz/cloud/RAE-Suite-go` (`https://github.com/vproject111/RAE-Suite-go`)  
**Cel nadrzędny:** Doprowadzenie `RAE-Suite-go` do 100% parzystości funkcjonalnej i kontraktowej ze wszystkimi aktywnymi routerami `RAE-Suite` (Python/TS) oraz `RAE-CRL`, obejmującej:
1. Prawdziwego klienta HTTP do modeli LLM i Embeddingów (**Ollama Node 1/Node 3**, **LiteLLM / OpenRouter**, **9 modeli w 3-stopniowej kaskadzie**) z deterministycznym fallbackiem offline.
2. Adapter bazy danych **PostgreSQL / SQLite** obsługujący schemat 12 tabel RAE (`memories`, `memory_embeddings`, `tenants`, `user_tenant_roles`, `access_logs`, `budgets`, `token_savings_log`, `knowledge_graph_*`, `compute_nodes`, `delegated_tasks`).
3. Wszystkie **19 grup routerów `RAE-Memory` (`:8000`)** (~130 endpointów REST API v2: `/v2/memories/*`, `/v2/graph/*`, `/v2/automation/*`, `/v2/evaluation/*`, `/v2/search/*`, `/v2/mesh/*`, `/v2/reflections/*`, `/v2/dashboard/*`, `/v2/nodes/*`, `/v2/compliance/*`, `/v2/bridge/*`, `/v2/agent/*`, `/v2/feedback`, `/v2/sync/*`, `/v2/tuning/*`, `/v2/token-savings/*`, `/v2/federation/*`, `/system/*`, `/tenant/*`, `/ready`, `/live`, `/metrics`, `/openapi.json`, `/docs`).
4. Wszystkie **23 endpointy `RAE-Suite Orchestrator` (`:8009`)** oraz **26 endpointów `RAE-Quality 2.0` (`:8010`)** (w tym `dtp-audit`, `calculator-audit`, `sonarqube-audit`, `golden-set-audit`, `risk-coverage`, `gate-decision`, `predictive-plan`, `analyze-diff`, `security-scan`, `generate-tests`, `verify-fix`, `check-architecture`, `release-gate`, `rpc`).
5. Wszystkie endpointy **`RAE-Phoenix` (`:8012`)**, **`RAE-Hive` (`:8013`)**, **`RAE-Lab` (`:8011`)**, **`RAE-Portal` (`:8082`)** oraz **`RAE-CRL` (`:8503`)**.
6. Rozbudowany skrypt weryfikacyjny `scripts/verify_full_parity_matrix.py` testujący **ponad 150 żywych endpointów i integracji** jednocześnie.

---

## Plan Realizacji (Iteracje P1 – P6)

### Iteracja P1: Natywny Klient LLM / Embeddingów (Ollama + LiteLLM/OpenRouter) oraz Adapter PostgreSQL (`pkg/drivers/llm` & `pkg/drivers/postgres`)
1. **`pkg/drivers/llm/client.go` + `client_test.go`**:
   - Natywny klient HTTP w Go obsługujący:
     - **Ollama API** (`/api/embeddings`, `/api/chat`, `/api/tags` — kompatybilny z Node 1 Lumina i Node 3 Piotrek),
     - **OpenAI / LiteLLM / OpenRouter API** (`/v1/embeddings`, `/v1/chat/completions`, `/v1/models` — obsługujący wszystkie 9 modeli z 3-stopniowej hierarchii RAE),
     - Bezstanowy **Local Deterministic Fallback** (gdy serwer LLM jest niedostępny lub w trybie `edge`/`math-only`, generuje deterministyczne wektory znormalizowane 384d/768d/1536d oraz ustrukturyzowane odpowiedzi bez blokowania wywołania).
2. **`pkg/drivers/postgres/adapter.go` + `adapter_test.go`**:
   - Adapter obsługujący schemat 12 tabel RAE (`DDLSchema12Tables`), migracje SQL, zapytania parametryzowane, budżety tokenów (`budgets`, `token_savings_log`), logi dostępu (`access_logs`), węzły obliczeniowe (`compute_nodes`, `delegated_tasks`) oraz pełną synchronizację z `UnifiedStore`.

### Iteracja P2: Kompletne 19 Grup Routerów `RAE-Memory` (`:8000`) — 100% Parzystości z `apps/memory_api`
Rozbudowa `services/memory` (poprzez modularny plik `services/memory/extended_routes.go` + `extended_routes_test.go`) o wszystkie routery montowane w `apps/memory_api/main.py`:
1. **System & Observability**: `/health`, `/ready`, `/live`, `/metrics` (format Prometheus), `/docs`, `/openapi.json`.
2. **Memories V2 (`/v2/memories`)**: `POST /v2/memories`, `GET /v2/memories`, `GET /v2/memories/{id}`, `DELETE /v2/memories/{id}`, `GET /v2/memories/sessions/{session_id}`, `POST /v2/memories/reflections`, `GET /v2/memories/stats`, `POST /v2/memories/consolidate`, `POST /v2/memories/upgrade`, `POST /v2/memories/query`.
3. **Knowledge Graph+ (`/v2/graph`) — 19 endpointów**:
   - `POST /v2/graph/nodes`, `GET /v2/graph/nodes/{node_id}/metrics`, `POST /v2/graph/nodes/connected`, `POST /v2/graph/nodes/batch`,
   - `POST /v2/graph/edges`, `POST /v2/graph/edges/batch`, `PUT /v2/graph/edges/{edge_id}/weight`, `POST /v2/graph/edges/{edge_id}/deactivate`, `POST /v2/graph/edges/{edge_id}/activate`, `PUT /v2/graph/edges/{edge_id}/temporal`,
   - `POST /v2/graph/traverse`, `POST /v2/graph/path/shortest` (algorytm BFS/Dijkstra najkrótszej ścieżki w Go), `POST /v2/graph/cycles/detect`,
   - `POST /v2/graph/snapshots`, `GET /v2/graph/snapshots`, `GET /v2/graph/snapshots/{snapshot_id}`, `POST /v2/graph/snapshots/{snapshot_id}/restore`, `POST /v2/graph/statistics`, `GET /v2/graph/health`.
4. **Event Triggers & Workflows (`/v2/automation`) — 18 endpointów**:
   - CRUD wyzwalaczy (`POST /v2/automation/create`, `GET/PUT/DELETE /v2/automation/{trigger_id}`, `POST /v2/automation/{trigger_id}/enable`, `POST /v2/automation/{trigger_id}/disable`, `GET /v2/automation/list`),
   - Emitowanie zdarzeń i egzekucje (`POST /v2/automation/events/emit`, `GET /v2/automation/events/types`, `POST /v2/automation/executions`),
   - Przepływy pracy i szablony (`POST /v2/automation/workflows/create`, `GET /v2/automation/workflows`, `GET /v2/automation/workflows/{workflow_id}`, `GET /v2/automation/templates`, `GET /v2/automation/templates/{template_id}`, `POST /v2/automation/templates/{template_id}/instantiate`, `GET /v2/automation/health`, `GET /v2/automation/info`).
5. **Evaluation & Drift Detection (`/v2/evaluation`) — 12 endpointów**:
   - `POST /v2/evaluation/search` (obliczanie MRR, NDCG, Precision@K, Recall@K), `GET /v2/evaluation/metrics/supported`, `POST /v2/evaluation/drift/detect`, `GET /v2/evaluation/drift/severity-levels`, `POST /v2/evaluation/ab-test/create`, `POST /v2/evaluation/ab-test/{test_id}/compare`, `POST /v2/evaluation/quality/metrics`, `GET /v2/evaluation/quality/thresholds`, `POST /v2/evaluation/benchmark/run`, `GET /v2/evaluation/benchmark/suites`, `GET /v2/evaluation/health`, `GET /v2/evaluation/info`.
6. **Hybrid Search & Weight Profiles (`/v2/search`) — 11 endpointów**:
   - `POST /v2/search/hybrid`, `POST /v2/search/evidence`, `POST /v2/search/analyze`, `POST /v2/search/analyze/explain`, `GET /v2/search/weights/profiles`, `GET /v2/search/weights/profiles/{profile_name}`, `POST /v2/search/weights/calculate`, `POST /v2/search/compare`, `POST /v2/search/test/weights`, `GET /v2/search/health`, `GET /v2/search/info`.
7. **Mesh Federation V2 (`/v2/mesh`) — 10 endpointów**:
   - `POST /v2/mesh/invite`, `GET /v2/mesh/handshake/challenge`, `POST /v2/mesh/join`, `POST /v2/mesh/handshake`, `GET /v2/mesh/peers`, `POST /v2/mesh/peers`, `DELETE /v2/mesh/peers/{peer_id}`, `POST /v2/mesh/peers/{peer_id}/sync`, `GET /v2/mesh/peers/{peer_id}/status`, `POST /v2/mesh/sync/receive`.
8. **Reflections, Nodes, Dashboard, Compliance, Bridge, Agent, Feedback, Sync, Tuning, Token Savings, Federation & Tenants**:
   - Wszystkie endpointy `/v2/reflections/*`, `/v2/nodes/*`, `/v2/dashboard/*`, `/v2/compliance/*`, `/v2/bridge/*`, `/v2/agent/execute`, `/v2/feedback`, `/v2/sync/*`, `/v2/tuning/*`, `/v2/token-savings/*`, `/v2/federation/query`, `/system/tenants`, `/system/projects`, `/tenant/*`, `/admin/overview`, `/embeddings`, `/rerank`.

### Iteracja P3: Kompletne 26 Endpointów `RAE-Quality 2.0` (`:8010`) oraz 23 Endpointy `RAE-Suite Orchestrator` (`:8009`)
1. **Rozbudowa `services/quality/service.go`**:
   - Dodanie wszystkich wyspecjalizowanych bram audytowych z `packages/rae-quality/main.py`:
     - `GET /v2/quality/risk-coverage`, `POST /v2/quality/risk-coverage/verify`, `GET /v2/quality/gate-decision`,
     - `POST /v2/quality/dtp-audit` (audyt DTP/PDF), `POST /v2/quality/calculator-audit` (ochrona silników kalkulacji),
     - `GET /v2/quality/defect-metrics`, `GET /v2/quality/golden-set-audit`, `GET /v2/quality/sonarqube-audit`, `GET /v2/quality/sonar-strict-metrics`,
     - `POST /v2/quality/predictive-plan`, `GET /v2/quality/unit-test-inventory`, `GET /v2/quality/mutation-audit`,
     - `POST /v2/quality/policy`, `POST /v2/quality/analyze-diff`, `POST /v2/quality/security-scan`, `POST /v2/quality/generate-tests`, `POST /v2/quality/verify-fix`, `POST /v2/quality/check-architecture`, `POST /v2/quality/release-gate`, `GET /v2/quality/report`, `GET /v2/quality/manifest`, `POST /v2/quality/rpc`, `GET /mcp/sse`, `POST /mcp/messages`.
2. **Rozbudowa `services/orchestrator/service.go`**:
   - Dodanie wszystkich 23 endpointów z `RAE-Suite/main.py`:
     - `GET /v2/factory/reconcile`, `POST /v2/orchestrator/reconcile`, `POST /v2/departments/register`, `GET /v2/fabric/route`, `POST /v2/audit/evaluate`,
     - Zintegrowane bramki `/v2/quality/*`, `POST /v2/kernel/execute`, `POST /v2/triad/run` (podpięte pod `pkg/drivers/llm`), `GET /v2/portal/overview`, `GET /v2/portal/hitl/requests`, `POST /v2/portal/hitl/request`, `POST /v2/portal/hitl/approve`, `GET /v2/portal/hitl/iso-report`, `POST /v2/knowledge/resolve`.

### Iteracja P4: Kompletne Endpointy `RAE-Phoenix` (`:8012`), `RAE-Hive` (`:8013`), `RAE-Lab` (`:8011`) oraz `RAE-Portal` (`:8082`)
1. **`services/phoenix/service.go`**:
   - Dodanie `/v2/phoenix/create`, `/sessions/analyze`, `/report/{report_id}`, `/patterns/errors`, `/metrics`, `/metrics/prometheus`, `/mcp/sse`, `/mcp/messages` oraz integracji z klientem LLM dla generowania kandydatów naprawy.
2. **`services/hive/service.go`**:
   - Dodanie `/execute`, `/metrics`, `/metrics/prometheus`, `/mcp/sse`, `/mcp/messages`.
3. **`services/lab/service.go`**:
   - Obsługa zarówno `GET` jak i `POST` na `/lab/analyze/failures`, `/lab/experiment/run`, `/mcp/sse`, `/mcp/messages`.
4. **`services/portal/service.go`**:
   - Dodanie `/logout`, `/callback`, `/v2/portal/overview`, `/v2/portal/hitl/*` oraz serwowania statycznych zasobów SPA z `rae-portal/dist` (jeśli obecne) lub bogatego wielowidokowego interfejsu SPA wbudowanego w binarkę Go.

### Iteracja P5: Kompleksowa Macierz Weryfikacyjna 150+ Endpointów (`scripts/verify_python_vs_go_parity.py`)
1. Rozbudowa `scripts/verify_python_vs_go_parity.py`, aby automatycznie testował **wszystkie grupy routerów (ponad 150 punktów kontrolnych HTTP/MCP/CRL/LLM/DB)** i generował pełny raport w `docs/RAPORT_POROWNANIA_I_PARZYSTOSCI_PYTHON_VS_GO.md`.

### Iteracja P6: Testy `-race`, Kompilacja 14 Binarek (`CGO_ENABLED=0`), Git Flow (`v4.2.0`) i Zapis do RAE
1. Wykonanie `make test && make build && make build-cross`.
2. Scalenie gałęzi `feature/*` -> `develop` -> `main`, tag `v4.2.0`, push do `https://github.com/vproject111/RAE-Suite-go` oraz zapis zadania przez `record_task_completion.py`.
