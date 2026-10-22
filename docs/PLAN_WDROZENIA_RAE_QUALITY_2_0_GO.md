# Plan Wdrożenia RAE-Quality 2.0 (219 Reguł, 6 Silników, 8 Operacji Agentowych) w `RAE-Suite-go`

**Data opracowania**: 2026-10-19  
**Dokument źródłowy**: [`/home/grzegorz/cloud/RAE-Suite/docs/poziom-bezpieczeństwa-kodu.md`](file:///home/grzegorz/cloud/RAE-Suite/docs/poziom-bezpiecze%C5%84stwa-kodu.md)  
**Wzorzec referencyjny Python**: [`/home/grzegorz/cloud/RAE-Suite/packages/rae-quality/`](file:///home/grzegorz/cloud/RAE-Suite/packages/rae-quality/main.py)  
**Cel**: Wdrożenie 1:1 pełnej architektury **RAE-Quality 2.0** w natywnym Go (`RAE-Suite-go/services/quality` oraz `services/mcp`), rozszerzonej o natywne wzorce detekcyjne dla języka **Go** (umożliwiające głęboki audyt bezpieczeństwa i architektury projektów takich jak `backend-go`).

---

## 1. Diagnoza Luk Przed Wdrożeniem

W wyniku porównania [`RAE-Suite/packages/rae-quality`](file:///home/grzegorz/cloud/RAE-Suite/packages/rae-quality/main.py) z [`RAE-Suite-go/services/quality`](file:///home/grzegorz/cloud/RAE-Suite-go/services/quality/service.go) zidentyfikowano następujące braki funkcjonalne w wersji Go:

1. **Katalog 219 Reguł (`catalog_198.py`)**:
   - W Go istniała wyłącznie płaska tablica 210 nazw zagadnień (`All198Topics` + `AdditionalAISaaSTopics`), bez pełnych obiektów `QualityRule` (brak priorytetów `P0..P2`, etapów `before_coding..pre_prod_post_deploy`, wymagań dowodowych `evidence_required`, `triggers`, `autofix`) oraz **brak 9 reguł pakietu `RAE-EDITOR-SECURITY` (`RAE-EDT-001` .. `RAE-EDT-009`)**.
2. **Silnik Polityk (`PolicyEngine` — `policy_engine.py`)**:
   - Brak dynamicznego doboru reguł do zadania (`get_policy_for_task`) wg zmienianych plików, stosu technologicznego (`postgresql`, `redis`, `kafka`, `kubernetes`, `docker`, `go`) i 4 etapów wdrożenia (`etap_1_p0` .. `etap_4_prod`).
   - Brak walidacji twardych dowodów (`validate_rule_evidence` — odrzucającej niepoparte dowodami oświadczenia agenta).
   - Brak ochrony przed auto-modyfikacją reguł przez agenta (`check_agent_tampering` / `RAE-SEC-208`).
   - Brak generatora manifestu technicznego wszystkich 219 reguł (`generate_technical_manifest`).
3. **Bramka Bezpieczeństwa (`SecurityGateway` — `security_gateway.py`)**:
   - W Go `SecurityGateway.ScanCode` sprawdzał tylko 2 proste warunki, podczas gdy wersja Python posiada **24 aktywne reguły SAST/Semantyczne** (`RAE-SEC-001..010`, `020`, `023`, `024`, `032`, `033`, `034`, `199`, `207` oraz `RAE-EDT-001..009`), macierz 10 skanerów (`SCANNER_REGISTRY`), separację długu historycznego (`baseline_fingerprints`) oraz eksport do standardu **SARIF 2.1.0**.
   - Brak wzorców SAST specyficznych dla języka **Go** (np. `fmt.Sprintf` w zapytaniach SQL, `md5.Sum` dla haseł, porównywanie podpisów HMAC przez `==` zamiast `hmac.Equal`, `http.Get` bez walidacji SSRF).
4. **Silnik Architektury (`ArchitectureEngine` — `architecture_engine.py`)**:
   - Brak wykrywania zapytań **N+1 w pętlach** (`RAE-SEC-043` — dla Python/PHP/TS oraz pętli `for` w Go), brakujących migracji `down` (`RAE-SEC-047`), wieloetapowego zapisu zamówień bez transakcji (`RAE-SEC-048`), **Race Conditions** przy zapisie projektu bez wersji/ETag (`RAE-SEC-090`), zapisu do Redis bez TTL (`RAE-SEC-100`) oraz weryfikacji kompatybilności wstecznej **OpenAPI** (`RAE-SEC-128`, `RAE-SEC-129`).
5. **Silnik Infrastruktury (`InfrastructureEngine` — `infrastructure_engine.py`)**:
   - Brak audytu `Dockerfile` (`RAE-SEC-190` Multi-Stage, `RAE-SEC-191` Non-Root `USER`, `RAE-SEC-141` `HEALTHCHECK`), manifestów Kubernetes (`RAE-SEC-193` `resources.limits`, `RAE-SEC-142` `livenessProbe`/`readinessProbe`) oraz gotowości systemowej (`EvaluateSystemicReadiness`: backupy, TLS, RTO/RPO, OpenTelemetry, Branch Protection).
6. **Silnik Jakości Kodu i Integralności Testów (`CodeQualityEngine` + `TestIntegrityGuard`)**:
   - Brak generatora 10 kanonicznych scenariuszy testów negatywnych (`NEG-IDOR-001` .. `NEG-SECRET-010`) i niezależnego weryfikatora poprawek (`verify_fix`) blokującego osłabianie asercji lub pomijanie testów bezpieczeństwa przez agenta.
7. **Silnik Semantyczny AI Review i Pętla Auto-Naprawy (`AIReviewEngine`, `AutoRepairOrchestrator`, `AgentScorecardTracker`, `AuditTrailRecorder`)**:
   - Brak semantycznej analizy uprawnień `printshop_admin` (`RAE-EDT-003`), manipulacji logiką zamówień (`RAE-SEC-202`) i własności zasobów (`RAE-EDT-001`), pętli maksymalnie 3 prób auto-naprawy z eskalacją do człowieka, kart wyników agentów (`SOTA`/`STANDARD`/`PROBATION`/`RESTRICTED`) oraz łańcucha audytowego SHA-256 (`RAE-SEC-210`).
8. **Endpointy HTTP REST i Narzędzia MCP RAE-Quality 2.0**:
   - Brak 10 endpointów w `services/quality/service.go`: `POST /v2/quality/policy`, `POST /v2/quality/analyze-diff`, `POST /v2/quality/security-scan`, `POST /v2/quality/generate-tests`, `POST /v2/quality/verify-fix`, `POST /v2/quality/check-architecture`, `POST /v2/quality/release-gate`, `GET /v2/quality/report`, `GET /v2/quality/manifest`, `POST /v2/quality/rpc`.

---

## 2. Plan Realizacji Iteracyjnej

### Iteracja 1: Modele Danych RAE-Quality 2.0 (`models_v2.go`), Pełny Katalog 219 Reguł (`catalog_219.go`) oraz `PolicyEngine` (`policy_engine.go`)
- Utworzenie struktur danych 1:1 z `models/security_v2.py` (`QualityRule`, `SecurityFinding`, `EvidenceItem`, `SARIFReport`, `MandatoryGateCheck`, `PRQualityAssessment`, `AutoRepairTask`, `AgentQualityScorecard`).
- Zbudowanie pełnego katalogu **219 wersjonowanych reguł**:
  - `RAE-SEC-001` .. `RAE-SEC-198` (198 zagadnień w 12 modułach z priorytetami `P0..P2`),
  - `RAE-SEC-199` .. `RAE-SEC-210` (12 zagadnień AI & Multi-Tenant SaaS),
  - `RAE-EDT-001` .. `RAE-EDT-009` (9 reguł `RAE-EDITOR-SECURITY`).
- Implementacja `PolicyEngine`:
  - `GetPolicyForTask` (obsługa `etap_1_p0`..`etap_4_prod`, detekcja zmian kosmetycznych `.css`/`.scss`, dobór reguł dla zamówień, edytora DTP, bazy danych i infrastruktury),
  - `ValidateRuleEvidence` (odrzucanie gołych deklaracji agenta),
  - `CheckAgentTampering` (`RAE-SEC-208`),
  - `GenerateTechnicalManifest`.

### Iteracja 2: `SecurityGateway` (`security_gateway.go`) — 24+ Reguły SAST/Semantyczne (Python/PHP/TS/JS + Go Native) i SARIF 2.1.0
- Implementacja `ScannerRegistry` (10 skanerów: `semgrep`, `codeql`, `gitleaks`, `trivy`, `owasp_zap`, `sonarqube`, `schemathesis`, `playwright`, `k6`, `checkov`) oraz `GetCompatibleScanners`.
- Implementacja wszystkich 24 reguł detekcyjnych z `security_gateway.py` rozszerzonych o składnię **Go**:
  - `RAE-SEC-001`..`008`, `010`, `020`, `023`, `024`, `032`, `033`, `034`, `199`, `207` oraz `RAE-EDT-001`..`009`.
- Obsługa `baseline_fingerprints` (oddzielenie długu historycznego od nowych błędów) oraz eksport do formatu **SARIF 2.1.0**.

### Iteracja 3: Silniki `ArchitectureEngine`, `InfrastructureEngine`, `CodeQualityEngine`, `AIReviewEngine` oraz `AutoRepairOrchestrator` (`engines_v2.go`)
- Implementacja `ArchitectureEngine` (`CheckArchitecture` + `VerifyOpenAPICompatibility`: `RAE-SEC-043`, `047`, `048`, `090`, `100`, `128`, `129`, w tym detekcja N+1 i transakcji w kodzie Go).
- Implementacja `InfrastructureEngine` (`AuditDockerfile`, `AuditK8sManifest`, `EvaluateSystemicReadiness`).
- Implementacja `CodeQualityEngine` + `TestIntegrityGuard` (`GenerateTests` dla 10 scenariuszy negatywnych + `VerifyFix` blokujący manipulację testami Python i Go).
- Implementacja `AIReviewEngine` (`ReviewChangeSemantics` + generowanie kryptograficznych `EvidenceItem`).
- Implementacja `AutoRepairOrchestrator` (limit 3 prób -> eskalacja HITL), `AgentScorecardTracker` i `AuditTrailRecorder` (łańcuch SHA-256 `RAE-SEC-210`).

### Iteracja 4: Centralny `RAEQualityGatewayV2` (`gateway_v2.go`), 10 Endpointów REST, Integracja MCP i Skaner Katalogów Go (`scan-dir`)
- Implementacja `RAEQualityGatewayV2` realizującego wszystkie 8 operacji kanonicznych (`quality.get_policy`, `quality.analyze_diff`, `quality.security_scan`, `quality.generate_tests`, `quality.verify_fix`, `quality.check_architecture`, `quality.release_gate`, `quality.report`) oraz `DispatchRPC`.
- Podpięcie wszystkich 10 endpointów `/v2/quality/*` w `services/quality/service.go` oraz aktualizacja `AuditCode` i `services/mcp/service.go`.
- Dodanie trybu `scan-dir` w `cmd/rae-quality/main.go` do kompleksowego skanowania wieloplikowych repozytoriów Go/Python/TS/PHP.

### Iteracja 5: Testy Weryfikacyjne (9/9 Kryteriów Akceptacji), Audyt `backend-go` i Raport Końcowy
- Uruchomienie pełnego zestawu testów jednostkowych i integracyjnych dla wszystkich 219 reguł i 8 operacji `quality.*`.
- Przeprowadzenie rzeczywistego audytu repozytorium `/home/grzegorz/cloud/backend-go` za pomocą wdrożonego `RAE-Quality 2.0` w Go.
- Sporządzenie raportu z wdrożenia w `/home/grzegorz/cloud/RAE-Suite-go/docs/RAPORT_WDROZENIA_RAE_QUALITY_2_0_GO.md`.
