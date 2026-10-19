# Raport Wdrożenia Behawioralnego, Porównania Szybkości Python vs Go oraz Publikacji Obrazu Produkcyjnego (`RAE-Suite-go v5.1.0`)

**Data wdrożenia:** 2026-10-19  
**Wersja docelowa:** `RAE-Suite-go v5.1.0`  
**Raport badania wstępnego:** `docs/RAPORT_DOKLADNEGO_SPRAWDZENIA_BRAKOW_V5_1_0.md`  
**Zrzut audytu behawioralnego:** `docs/BADANIE_BEHAWIORALNE_PY_GO_V5_1_0.json`  
**Zrzut benchmarku wydajności:** `docs/BENCHMARK_PYTHON_VS_GO_V5_1_0.json`

---

## 1. Wyniki Wszystkich 3 Audytorów Parzystości (`v5.1.0` — 100% Zero Braków)

### 1.1. Audytor Behawioralny, Sub-Tras HTTP, CLI i Input-Sensitivity (`scripts/deep_behavioral_and_route_audit.py`)

| Wymiar badania behawioralnego | Stan przed (`v5.0.0`) | Stan po (`v5.1.0`) | Status |
| :--- | :---: | :---: | :---: |
| **1. Nieodwzorowane sub-trasy HTTP (`unmatched_http_subroutes_count`)** | `18` | **`0`** | **100% ZREALIZOWANE** |
| **2. Stałe / niewrażliwe na argumenty metody domenowe (`constant_domain_methods_count`)** | `1691` | **`0`** | **100% ZREALIZOWANE** |
| **3. Stałe / niewrażliwe na argumenty funkcje top-level (`constant_top_funcs_count`)** | `522` | **`0`** | **100% ZREALIZOWANE** |
| **4. Brakujące podkomendy CLI w `cmd/rae` (`missing_cli_subcommands_count`)** | `6` | **`0`** | **100% ZREALIZOWANE** |

### 1.2. 5-Wymiarowy Audytor Forensyczny AST/Runtime (`scripts/deep_forensic_gap_investigation.py`)

| Wymiar forensyczny | Wynik w `v5.1.0` |
| :--- | :---: |
| **1. Nieodwzorowane klasy domenowe Python (`total_unmatched_domain_classes`)** | **`0`** |
| **2. Puste struktury znacznikowe w Go (`total_empty_marker_structs_in_go`)** | **`0`** |
| **3. Klasy uproszczone / z deficytem metod (`total_method_deficit_classes`)** | **`0`** |
| **4. Niepodpięte struktury Go (`total_unwired_go_structs`)** | **`0`** |
| **5. Nieodwzorowane funkcje top-level `>=15 LOC` (`total_unmatched_top_funcs_ge_15loc`)** | **`0`** |

### 1.3. 5-Etapowy Audytor Kontraktowy (`scripts/reusable_5stage_parity_auditor.py`)
- **Stage 1 (AST Symbol & Endpoint Mapping Matrix):** `PASSED` (`335` zarejestrowanych tras HTTP, `903` struktury Go, `3986` funkcji Go)
- **Stage 2 (Static Anti-Stub & Dead-Logic Scanner):** `PASSED` (`0` stubów w `351` handlerach)
- **Stage 3 (Differential Input-Sensitivity Probing `f(x1) != f(x2)`):** `PASSED` (`17/17` sond różnicowych)
- **Stage 4 (Stateful End-to-End Workflow Invariant Chain):** `PASSED` (`6/6` kroków stanowych E2E)
- **Stage 5 (Database Schema, SQL DDL & Environment Variable Contract Audit):** `PASSED` (`12` tabel + `16/16` zmiennych środowiskowych)

---

## 2. Porównanie Szybkości Działania i Zasobów: Python (`RAE-Suite v2.9.4`) vs Go (`RAE-Suite-go v5.1.0`)

Pomiary wykonano za pomocą skryptu `scripts/benchmark_python_vs_go.py` na tym samym hoście (`linux/amd64`):

| Test wydajnościowy / Zasób | Python (`RAE-Suite`) | Go (`RAE-Suite-go v5.1.0`) | Przyspieszenie / Redukcja |
| :--- | :---: | :---: | :---: |
| **1. Podobieństwo cosinusowe wektorów 384D (`5 000` iteracji)** | `212.716 ms` | **`1.831 ms`** | **116.18x szybciej w Go** |
| **2. Obsługa 100 zapytań HTTP (`GET /health` round-trip)** | `595.144 ms` | **`12.857 ms`** | **46.29x szybciej w Go** |
| **3. Scoring L1/L2/L3 MathController + krzywa Ebbinghausa (`10 000` iteracji)** | `7.191 ms` | **`3.526 ms`** | **2.04x szybciej w Go** |
| **4. Wyliczanie 16-kubełkowego drzewa Merkle (`50 x 1 000` wspomnień)** | `43.589 ms` | **`27.974 ms`** | **1.56x szybciej w Go** |
| **5. Zapis 1 000 wspomnień w RAM z indeksacją i strażnikiem ISO 42001** | `1.264 ms` | **`1.230 ms`** | **1.03x szybciej w Go** |
| **6. Łączny rozmiar obrazów produkcyjnych Docker (cały stos RAE)** | `~10.90 GB` (10 obrazów Python) | **`~42 MB`** (1 obraz `distroless` Go z 10 binariami) | **~250x mniejszy obraz Docker** |
| **7. Czas zimnego startu wszystkich 9 modułów (`--all`)** | `~14.5 s` (Uvicorn + Celery) | **`< 15 ms`** (`rae --all`) | **~960x szybszy zimny start** |

---

## 3. Zbudowanie Obrazu Produkcyjnego, Publikacja na Harbor i Lokalnie oraz Czyszczenie Cache

1. **Obraz produkcyjny (`Dockerfile`, multi-stage `golang:1.23-alpine` -> `gcr.io/distroless/static-debian12:nonroot`, `CGO_ENABLED=0`):**
   - Zawiera wszystkie 10 statycznych binariów: `rae`, `rae-memory`, `rae-crl`, `rae-mcp`, `rae-suite`, `rae-quality`, `rae-lab`, `rae-phoenix`, `rae-hive`, `rae-portal`.
   - Otagowany i wypchnięty na Harbor oraz zapisany lokalnie:
     - `harbor.cloud.printworks.pl/dreamsoft/rae-suite-go:5.1.0`
     - `harbor.cloud.printworks.pl/dreamsoft/rae-suite-go:latest`
     - `rae-suite-go:5.1.0`
     - `rae-suite-go:latest`
   - Lokalny katalog `bin/` przebudowany (`make build`) do najnowszej wersji `v5.1.0`.
2. **Czyszczenie cache:**
   - Wyczyszczono cache kompilacji i testów Go (`go clean -cache -testcache`), tymczasowe pliki `__pycache__` oraz cache budowania Docker (`docker builder prune -f`).
