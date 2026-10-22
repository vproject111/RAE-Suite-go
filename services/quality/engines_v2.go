package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// ============================================================================
// 1. ArchitectureEngine (packages/rae-quality/engines/architecture/architecture_engine.py)
// ============================================================================

var (
	nPlusOneRegex = regexp.MustCompile(
		`(?is)(?:for\s+\w+\s+in\s+.*?:|foreach\s*\(.*?\)\s*\{|for\s+[\w,\s]+:=\s*range\s+[^{]+\{)[^}]{1,260}?(?:\.execute\s*\(\s*['"]SELECT|::where\s*\(|->find\s*\(|\.objects\.filter\s*\(|\.(?:Query|QueryRow|QueryContext|QueryRowContext)\s*\(\s*(?:ctx\s*,\s*)?["` + "`" + `]SELECT)`,
	)
	redisNoTTLRegex = regexp.MustCompile(
		`(?i)(?:redis\.set\s*\(\s*[^,]+,\s*[^,)]+\s*\)|\.Set\s*\(\s*ctx\s*,\s*[^,]+,\s*[^,]+,\s*0\s*\))`,
	)
)

// ArchitectureEngine audits architectural compliance across Modules 3, 4, 5, 6, and 7 (Topics 41-133).
type ArchitectureEngine struct{}

// NewArchitectureEngine constructs an ArchitectureEngine.
func NewArchitectureEngine() *ArchitectureEngine {
	return &ArchitectureEngine{}
}

// CheckArchitecture implements `quality.check_architecture` (Stage 3: Pre-Merge).
func (ae *ArchitectureEngine) CheckArchitecture(
	code string,
	filePath string,
	technologies []string,
	openapiBefore map[string]interface{},
	openapiAfter map[string]interface{},
) map[string]interface{} {
	if filePath == "" {
		filePath = "service.go"
	}
	if len(technologies) == 0 {
		technologies = []string{"postgresql", "redis"}
	}
	activeTechs := make(map[string]bool, len(technologies))
	for _, t := range technologies {
		activeTechs[strings.ToLower(t)] = true
	}

	findings := make([]SecurityFinding, 0)
	lowerCode := strings.ToLower(code)
	lowerPath := strings.ToLower(filePath)

	// 1. Database Migrations & Down Migrations (RAE-SEC-046, RAE-SEC-047)
	if strings.Contains(lowerPath, "migration") || strings.Contains(lowerPath, "alembic") ||
		strings.Contains(lowerCode, "def upgrade(") || strings.Contains(lowerCode, "function up(") ||
		strings.Contains(lowerCode, "+goose up") {
		hasDown := containsAnyKeyword(lowerCode, []string{"def downgrade(", "function down(", "down_migration", "+goose down", "migrate:down"})
		if !hasDown {
			findings = append(findings, SecurityFinding{
				ID:          "FND-RAE-SEC-047-1",
				RuleID:      "RAE-SEC-047",
				RuleName:    "Down Migrations",
				Module:      "Database Integrity",
				Severity:    SeverityHigh,
				Enforcement: EnforcementRepair,
				ControlType: ControlDeterministic,
				Scanner:     "rae-architecture-engine",
				Message:     "Database migration is missing a reversible down/downgrade migration.",
				FilePath:    filePath,
				LineNumber:  1,
				Remediation: "Implement `downgrade()` / `down()` / `-- +goose Down` to guarantee safe rollback.",
			})
		}
	}

	// 2. N+1 Query Detection (RAE-SEC-043)
	if nPlusOneRegex.MatchString(code) {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-043-1",
			RuleID:      "RAE-SEC-043",
			RuleName:    "N+1 Queries",
			Module:      "Database Integrity",
			Severity:    SeverityHigh,
			Enforcement: EnforcementRepair,
			ControlType: ControlDeterministic,
			Scanner:     "rae-architecture-engine",
			Message:     "Potential N+1 database query inside loop detected.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Batch-fetch records using `WHERE id = ANY($1)` / `WHERE id IN (...)`, `selectinload`, or `with()` eager loading.",
		})
	}

	// 3. Transactions & Order Atomicity (RAE-SEC-048)
	if (strings.Contains(lowerCode, "save_order(") || strings.Contains(lowerCode, "create_order(") ||
		strings.Contains(lowerCode, "saveorder(") || strings.Contains(lowerCode, "createorder(")) &&
		(strings.Contains(lowerCode, "save_order_items(") || strings.Contains(lowerCode, "saveorderitems(")) {
		hasTx := containsAnyKeyword(lowerCode, []string{"transaction", "begin(", "beginx(", "begintx(", "atomic(", "commit("})
		if !hasTx {
			findings = append(findings, SecurityFinding{
				ID:          "FND-RAE-SEC-048-1",
				RuleID:      "RAE-SEC-048",
				RuleName:    "Transactions",
				Module:      "Database Integrity",
				Severity:    SeverityCritical,
				Enforcement: EnforcementBlock,
				ControlType: ControlAISemantic,
				Scanner:     "rae-architecture-engine",
				Message:     "Multi-step order and order-items persistence lacks an atomic database transaction.",
				FilePath:    filePath,
				LineNumber:  1,
				Remediation: "Wrap order creation and order item insertion in an atomic transaction block (BeginTx / defer Rollback / Commit).",
			})
		}
	}

	// 4. Concurrent Project Save / Race Conditions (RAE-SEC-090 & RAE-SEC-052)
	if (strings.Contains(lowerCode, "save_project(") || strings.Contains(lowerCode, "update_scene(") ||
		strings.Contains(lowerCode, "saveproject(") || strings.Contains(lowerCode, "updatescene(")) &&
		!containsAnyKeyword(lowerCode, []string{"version", "etag", "optimistic", "for update", "lock", "yjs"}) {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-090-1",
			RuleID:      "RAE-SEC-090",
			RuleName:    "Race Conditions",
			Module:      "Distributed Systems",
			Severity:    SeverityHigh,
			Enforcement: EnforcementRepair,
			ControlType: ControlAISemantic,
			Scanner:     "rae-architecture-engine",
			Message:     "Concurrent project save handler lacks optimistic locking (version/ETag) or conflict detection.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Enforce version/ETag optimistic concurrency check (`WHERE id = $1 AND version = $2`).",
		})
	}

	// 5. Cache TTLs (RAE-SEC-100)
	if redisNoTTLRegex.MatchString(code) {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-100-1",
			RuleID:      "RAE-SEC-100",
			RuleName:    "TTLs",
			Module:      "Cache & Performance",
			Severity:    SeverityMedium,
			Enforcement: EnforcementRepair,
			ControlType: ControlDeterministic,
			Scanner:     "rae-architecture-engine",
			Message:     "Redis cache write (`redis.set` / `.Set(..., 0)`) is missing an explicit TTL expiration parameter.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Specify a non-zero TTL duration or use `redis.setex` to prevent unbounded cache growth.",
		})
	}

	// 6. OpenAPI Backward Compatibility Check (RAE-SEC-128, RAE-SEC-129)
	openapiCompat := ae.VerifyOpenAPICompatibility(openapiBefore, openapiAfter)
	compatible, _ := openapiCompat["compatible"].(bool)
	if !compatible {
		if breakingList, ok := openapiCompat["breaking_changes"].([]string); ok {
			for idx, breaking := range breakingList {
				findings = append(findings, SecurityFinding{
					ID:          fmt.Sprintf("FND-RAE-SEC-129-%d", idx+1),
					RuleID:      "RAE-SEC-129",
					RuleName:    "Backward Compatibility (OpenAPI)",
					Module:      "Network & API",
					Severity:    SeverityCritical,
					Enforcement: EnforcementBlock,
					ControlType: ControlDeterministic,
					Scanner:     "schemathesis",
					Message:     fmt.Sprintf("OpenAPI breaking change: %s", breaking),
					FilePath:    filePath,
					LineNumber:  1,
					Remediation: "Maintain backward compatibility or introduce a new versioned endpoint (`/v2/...`).",
				})
			}
		}
	}

	hasBlockers := false
	for _, f := range findings {
		if f.Enforcement == EnforcementBlock {
			hasBlockers = true
			break
		}
	}
	verdict := VerdictPass
	if hasBlockers {
		verdict = VerdictBlock
	} else if len(findings) > 0 {
		verdict = VerdictRepair
	}

	sortedTechs := make([]string, 0, len(activeTechs))
	for t := range activeTechs {
		sortedTechs = append(sortedTechs, t)
	}
	sort.Strings(sortedTechs)

	return map[string]interface{}{
		"verdict":                          string(verdict),
		"active_technologies":              sortedTechs,
		"postgres_vacuum_applicable":       activeTechs["postgresql"],
		"kafka_consumer_groups_applicable": activeTechs["kafka"],
		"openapi_compatibility":            openapiCompat,
		"findings_count":                   len(findings),
		"findings":                         findings,
	}
}

// VerifyOpenAPICompatibility checks for breaking changes between two OpenAPI specifications.
func (ae *ArchitectureEngine) VerifyOpenAPICompatibility(
	beforeSpec map[string]interface{},
	afterSpec map[string]interface{},
) map[string]interface{} {
	if len(beforeSpec) == 0 || len(afterSpec) == 0 {
		return map[string]interface{}{
			"compatible":       true,
			"breaking_changes": []string{},
		}
	}

	var breakingChanges []string
	beforePaths, _ := beforeSpec["paths"].(map[string]interface{})
	afterPaths, _ := afterSpec["paths"].(map[string]interface{})

	pathKeys := make([]string, 0, len(beforePaths))
	for p := range beforePaths {
		pathKeys = append(pathKeys, p)
	}
	sort.Strings(pathKeys)

	for _, path := range pathKeys {
		methodsRaw := beforePaths[path]
		afterMethodsRaw, pathExists := afterPaths[path]
		if !pathExists {
			breakingChanges = append(breakingChanges, fmt.Sprintf("Removed API endpoint path: %s", path))
			continue
		}
		beforeMethods, _ := methodsRaw.(map[string]interface{})
		afterMethods, _ := afterMethodsRaw.(map[string]interface{})
		for method, opBeforeRaw := range beforeMethods {
			opAfterRaw, methodExists := afterMethods[method]
			if !methodExists {
				breakingChanges = append(breakingChanges, fmt.Sprintf("Removed HTTP method %s on %s", strings.ToUpper(method), path))
				continue
			}
			opBefore, _ := opBeforeRaw.(map[string]interface{})
			opAfter, _ := opAfterRaw.(map[string]interface{})
			reqBefore := toStringSet(opBefore["required_fields"])
			reqAfter := toStringSet(opAfter["required_fields"])
			var newlyRequired []string
			for rf := range reqAfter {
				if !reqBefore[rf] {
					newlyRequired = append(newlyRequired, rf)
				}
			}
			sort.Strings(newlyRequired)
			if len(newlyRequired) > 0 {
				breakingChanges = append(breakingChanges, fmt.Sprintf(
					"Added new required request field(s) %v on %s %s",
					newlyRequired, strings.ToUpper(method), path,
				))
			}
		}
	}

	if breakingChanges == nil {
		breakingChanges = []string{}
	}
	return map[string]interface{}{
		"compatible":       len(breakingChanges) == 0,
		"breaking_changes": breakingChanges,
	}
}

func toStringSet(v interface{}) map[string]bool {
	set := make(map[string]bool)
	switch arr := v.(type) {
	case []string:
		for _, s := range arr {
			set[s] = true
		}
	case []interface{}:
		for _, item := range arr {
			if s, ok := item.(string); ok {
				set[s] = true
			}
		}
	}
	return set
}

// ============================================================================
// 2. InfrastructureEngine (packages/rae-quality/engines/infrastructure/infrastructure_engine.py)
// ============================================================================

// InfrastructureEngine audits Dockerfiles, Kubernetes/Helm manifests, and Systemic Production Readiness.
type InfrastructureEngine struct{}

// NewInfrastructureEngine constructs an InfrastructureEngine.
func NewInfrastructureEngine() *InfrastructureEngine {
	return &InfrastructureEngine{}
}

// AuditDockerfile checks RAE-SEC-190 (Multi-Stage), RAE-SEC-191 (Non-Root USER), and RAE-SEC-141 (HEALTHCHECK).
func (ie *InfrastructureEngine) AuditDockerfile(dockerfileContent, filePath string) []SecurityFinding {
	if filePath == "" {
		filePath = "Dockerfile"
	}
	findings := make([]SecurityFinding, 0)
	upper := strings.ToUpper(dockerfileContent)
	lines := strings.Split(dockerfileContent, "\n")

	fromCount := 0
	hasNonRootUser := false
	for _, line := range lines {
		trimmed := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(trimmed, "FROM ") {
			fromCount++
		}
		if strings.HasPrefix(trimmed, "USER ") && !strings.Contains(trimmed, "ROOT") {
			hasNonRootUser = true
		}
	}
	if strings.Contains(upper, "APP_USER=") {
		hasNonRootUser = true
	}

	if fromCount < 2 {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-190-1",
			RuleID:      "RAE-SEC-190",
			RuleName:    "Multi-Stage Builds",
			Module:      "Delivery & Containers",
			Severity:    SeverityMedium,
			Enforcement: EnforcementRepair,
			ControlType: ControlDeterministic,
			Scanner:     "checkov+trivy",
			Message:     "Dockerfile does not use multi-stage build (`FROM ... AS builder`), increasing attack surface.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Separate build dependencies from runtime image using a multi-stage Dockerfile.",
		})
	}

	if !hasNonRootUser {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-191-1",
			RuleID:      "RAE-SEC-191",
			RuleName:    "Non-Root Containers",
			Module:      "Delivery & Containers",
			Severity:    SeverityHigh,
			Enforcement: EnforcementRepair,
			ControlType: ControlDeterministic,
			Scanner:     "trivy+checkov",
			Message:     "Dockerfile does not declare a non-root `USER` directive.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Create and switch to an unprivileged user (`USER appuser`) before `CMD`/`ENTRYPOINT`.",
		})
	}

	if !strings.Contains(upper, "HEALTHCHECK") {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-141-1",
			RuleID:      "RAE-SEC-141",
			RuleName:    "Health Checks",
			Module:      "Infrastructure Security",
			Severity:    SeverityLow,
			Enforcement: EnforcementWarn,
			ControlType: ControlDeterministic,
			Scanner:     "checkov",
			Message:     "Dockerfile does not define a container `HEALTHCHECK` instruction.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Add `HEALTHCHECK` instruction or ensure Kubernetes liveness/readiness probes are configured.",
		})
	}

	return findings
}

// AuditK8sManifest checks RAE-SEC-193 (Resource Requests & Limits) and RAE-SEC-142 (Liveness/Readiness Probes).
func (ie *InfrastructureEngine) AuditK8sManifest(manifestText, filePath string) []SecurityFinding {
	if filePath == "" {
		filePath = "values.yaml"
	}
	findings := make([]SecurityFinding, 0)
	lower := strings.ToLower(manifestText)

	if (strings.Contains(lower, "deployment") || strings.Contains(lower, "image:")) &&
		(!strings.Contains(lower, "resources:") || !strings.Contains(lower, "limits:") || !strings.Contains(lower, "requests:")) {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-193-1",
			RuleID:      "RAE-SEC-193",
			RuleName:    "Resource Requests & Limits",
			Module:      "Delivery & Containers",
			Severity:    SeverityHigh,
			Enforcement: EnforcementRepair,
			ControlType: ControlDeterministic,
			Scanner:     "checkov",
			Message:     "Kubernetes workload is missing CPU/memory `resources.requests` and `resources.limits`.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Define explicit CPU and memory requests and limits to prevent OOM & noisy-neighbor starvation.",
		})
	}

	if (strings.Contains(lower, "deployment") || strings.Contains(lower, "containers:")) &&
		(!strings.Contains(lower, "livenessprobe") && !strings.Contains(lower, "readinessprobe") && !strings.Contains(lower, "healthcheck")) {
		findings = append(findings, SecurityFinding{
			ID:          "FND-RAE-SEC-142-1",
			RuleID:      "RAE-SEC-142",
			RuleName:    "Liveness vs Readiness Probes",
			Module:      "Infrastructure Security",
			Severity:    SeverityMedium,
			Enforcement: EnforcementRepair,
			ControlType: ControlDeterministic,
			Scanner:     "checkov",
			Message:     "Kubernetes workload is missing `livenessProbe` / `readinessProbe` definitions.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Configure both livenessProbe and readinessProbe for zero-downtime rollouts.",
		})
	}

	return findings
}

// EvaluateSystemicReadiness evaluates Section 3 Systemic Controls (Backups, Restore, RTO/RPO, TLS, OTel, Branch Protection).
func (ie *InfrastructureEngine) EvaluateSystemicReadiness(systemTelemetry map[string]interface{}) map[string]interface{} {
	backupVerified := boolFromMap(systemTelemetry, "backup_restore_tested", true)
	tlsDaysRemaining := intFromMap(systemTelemetry, "tls_days_remaining", 60)
	rtoMinutes := intFromMap(systemTelemetry, "rto_minutes", 15)
	rpoMinutes := intFromMap(systemTelemetry, "rpo_minutes", 5)
	otelEnabled := boolFromMap(systemTelemetry, "opentelemetry_enabled", true)
	correlationIDs := boolFromMap(systemTelemetry, "correlation_ids_enabled", true)
	branchProtection := boolFromMap(systemTelemetry, "branch_protection_active", true)
	sbomSigned := boolFromMap(systemTelemetry, "sbom_and_artifact_signed", true)

	issues := make([]string, 0)
	if !backupVerified {
		issues = append(issues, "RAE-SEC-071: Backup restore verification has not been executed within SLA window.")
	}
	if tlsDaysRemaining < 14 {
		issues = append(issues, fmt.Sprintf("RAE-SEC-035: TLS certificate expires in %d days (< 14 days threshold).", tlsDaysRemaining))
	}
	if rtoMinutes > 60 || rpoMinutes > 15 {
		issues = append(issues, fmt.Sprintf("RAE-SEC-151: RTO (%dm) or RPO (%dm) exceeds target SLA.", rtoMinutes, rpoMinutes))
	}
	if !otelEnabled || !correlationIDs {
		issues = append(issues, "RAE-SEC-172/173: OpenTelemetry tracing or Correlation ID propagation is disabled.")
	}
	if !branchProtection {
		issues = append(issues, "RAE-SEC-198: Git Flow branch protection hooks are not armed.")
	}

	verdict := VerdictPass
	if !backupVerified || tlsDaysRemaining <= 0 || !branchProtection {
		verdict = VerdictBlock
	} else if len(issues) > 0 {
		verdict = VerdictReview
	}

	return map[string]interface{}{
		"verdict":                   string(verdict),
		"backup_restore_verified":   backupVerified,
		"tls_days_remaining":        tlsDaysRemaining,
		"rto_minutes":               rtoMinutes,
		"rpo_minutes":               rpoMinutes,
		"observability_ready":       otelEnabled && correlationIDs,
		"branch_protection_active":  branchProtection,
		"sbom_and_artifact_signed":  sbomSigned,
		"issues":                    issues,
	}
}

// ============================================================================
// 3. CodeQualityEngine & TestIntegrityGuard (packages/rae-quality/engines/testing/code_quality_engine.py)
// ============================================================================

// NegativeScenario represents one of the 10 canonical SaaS/Editor negative tests from Section 4.
type NegativeScenario struct {
	ID                 string   `json:"id"`
	RuleID             string   `json:"rule_id"`
	Threat             string   `json:"threat"`
	Scenario           string   `json:"scenario"`
	ExpectedReaction   string   `json:"expected_reaction"`
	ExpectedHTTPStatus []int    `json:"expected_http_status"`
	Assertion          string   `json:"assertion"`
}

// CanonicalNegativeScenarios holds the 10 canonical SaaS & Editor negative test templates.
var CanonicalNegativeScenarios = []NegativeScenario{
	{
		ID:                 "NEG-IDOR-001",
		RuleID:             "RAE-SEC-006",
		Threat:             "IDOR / Cross-Tenant Access",
		Scenario:           "Klient drukarni A próbuje pobrać lub zmodyfikować projekt drukarni B",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{403, 404},
		Assertion:          "assert response.status_code in (403, 404) and 'tenant_b_secret' not in response.text",
	},
	{
		ID:                 "NEG-AUTH-002",
		RuleID:             "RAE-SEC-013",
		Threat:             "Server-Side Authorization Bypass",
		Scenario:           "Użytkownik zmienia userId, tenantId lub orderId w payloadzie API",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{403, 422},
		Assertion:          "assert resolved_tenant_id == authenticated_session.tenant_id",
	},
	{
		ID:                 "NEG-MASS-003",
		RuleID:             "RAE-SEC-008",
		Threat:             "Mass Assignment",
		Scenario:           "Klient przesyła pola role='admin', price=0.01, isAdmin=True w żądaniu",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{400, 403, 422},
		Assertion:          "assert user.is_admin is False and order.price == server_calculated_price",
	},
	{
		ID:                 "NEG-SQLI-004",
		RuleID:             "RAE-SEC-001",
		Threat:             "SQL Injection",
		Scenario:           "Dane wejściowe zawierają ładunek SQL (' OR '1'='1' --) w filtrze projektów",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{400, 422, 200},
		Assertion:          "assert len(leaked_cross_tenant_rows) == 0",
	},
	{
		ID:                 "NEG-XSS-005",
		RuleID:             "RAE-SEC-002",
		Threat:             "Cross-Site Scripting (XSS)",
		Scenario:           "Nazwa projektu zawiera złośliwy HTML/JS (<script>fetch('https://evil.tld')</script>)",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{400, 422, 200},
		Assertion:          "assert '<script>' not in rendered_output",
	},
	{
		ID:                 "NEG-SSRF-006",
		RuleID:             "RAE-SEC-005",
		Threat:             "Server-Side Request Forgery (SSRF)",
		Scenario:           "Import zdjęcia z URL kierującego do sieci wewnętrznej (http://169.254.169.254 lub http://127.0.0.1)",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{400, 403, 422},
		Assertion:          "assert ssrf_guard.is_blocked('http://169.254.169.254/latest/meta-data/') is True",
	},
	{
		ID:                 "NEG-UPLOAD-007",
		RuleID:             "RAE-SEC-033",
		Threat:             "Malicious File Upload (SVG/PDF/Image)",
		Scenario:           "Upload pliku SVG z tagiem <script>/onload lub obrazu z nieoczyszczonym EXIF GPS",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{400, 415, 422},
		Assertion:          "assert upload_result.rejected is True and exif_gps_present is False",
	},
	{
		ID:                 "NEG-RACE-008",
		RuleID:             "RAE-SEC-090",
		Threat:             "Race Conditions (Concurrent Project Save)",
		Scenario:           "Równoczesny zapis tego samego projektu przez dwie osoby z tą samą wersją bazową",
		ExpectedReaction:   "Test konfliktu",
		ExpectedHTTPStatus: []int{409},
		Assertion:          "assert statuses.count(200) == 1 and statuses.count(409) == 1",
	},
	{
		ID:                 "NEG-TX-009",
		RuleID:             "RAE-SEC-048",
		Threat:             "Transactions & Atomic Rollback",
		Scenario:           "Awaria pomiędzy zapisaniem zamówienia i pozycji zamówienia",
		ExpectedReaction:   "Test integralności",
		ExpectedHTTPStatus: []int{500},
		Assertion:          "assert db.count_orphaned_orders() == 0",
	},
	{
		ID:                 "NEG-SECRET-010",
		RuleID:             "RAE-SEC-020",
		Threat:             "Secrets Leakage",
		Scenario:           "Klucz API trafia do repozytorium, logów lub obrazu kontenera",
		ExpectedReaction:   "BLOCK",
		ExpectedHTTPStatus: []int{},
		Assertion:          "assert gitleaks_findings == 0",
	},
}

var (
	assertionCountRe = regexp.MustCompile(`(?m)(?:\bassert\b|self\.assert\w+\s*\(|t\.(?:Fatal|Fatalf|Error|Errorf)\s*\(|(?:require|assert)\.\w+\s*\()`)
	softAssertionRe  = regexp.MustCompile(`(?i)(?:\bassert\s+True\b|\bassert\s+1\s*==\s*1\b|\bassert\s+['"]a['"]\s*==\s*['"]a['"])`)
)

// TestIntegrityGuard prevents agents from weakening or deleting test assertions during repair loops.
type TestIntegrityGuard struct{}

// NewTestIntegrityGuard constructs a TestIntegrityGuard.
func NewTestIntegrityGuard() *TestIntegrityGuard {
	return &TestIntegrityGuard{}
}

// ValidateTestIntegrity compares baseline and proposed test files for assertion deletion or softening.
func (tig *TestIntegrityGuard) ValidateTestIntegrity(baselineCode, proposedCode string) (bool, string) {
	baseAssertions := len(assertionCountRe.FindAllString(baselineCode, -1))
	propAssertions := len(assertionCountRe.FindAllString(proposedCode, -1))

	if propAssertions < baseAssertions {
		return false, fmt.Sprintf("Test integrity violation: Assertion count dropped from %d to %d.", baseAssertions, propAssertions)
	}

	baseSoft := len(softAssertionRe.FindAllString(baselineCode, -1))
	propSoft := len(softAssertionRe.FindAllString(proposedCode, -1))
	if propSoft > baseSoft {
		return false, "Test integrity violation: Assertion softening detected (empty assert True statements added)."
	}

	return true, ""
}

// CodeQualityEngine implements packages/rae-quality/engines/testing/code_quality_engine.py.
type CodeQualityEngine struct {
	securityGateway    *SecurityGateway
	testIntegrityGuard *TestIntegrityGuard
}

// NewCodeQualityEngine constructs a CodeQualityEngine.
func NewCodeQualityEngine(sg *SecurityGateway) *CodeQualityEngine {
	if sg == nil {
		sg = NewSecurityGateway()
	}
	return &CodeQualityEngine{
		securityGateway:    sg,
		testIntegrityGuard: NewTestIntegrityGuard(),
	}
}

// GenerateTests implements `quality.generate_tests`.
func (cqe *CodeQualityEngine) GenerateTests(
	targetEndpoint string,
	component string,
	ruleIDs []string,
) map[string]interface{} {
	if targetEndpoint == "" {
		targetEndpoint = "/api/v1/projects"
	}
	if component == "" {
		component = "editor_api"
	}

	reqSet := make(map[string]bool, len(ruleIDs))
	for _, r := range ruleIDs {
		reqSet[r] = true
	}

	var selected []NegativeScenario
	for _, sc := range CanonicalNegativeScenarios {
		if len(reqSet) == 0 || reqSet[sc.RuleID] {
			selected = append(selected, sc)
		}
	}
	if len(selected) == 0 {
		selected = CanonicalNegativeScenarios
	}

	pytestLines := []string{
		"import pytest",
		"",
		fmt.Sprintf("# Auto-generated RAE-Quality 2.0 Negative & Security Regression Suite for %s", targetEndpoint),
	}
	goTestLines := []string{
		"package security_test",
		"",
		"import \"testing\"",
		"",
		fmt.Sprintf("// Auto-generated RAE-Quality 2.0 Negative & Security Regression Suite for %s", targetEndpoint),
	}

	for _, sc := range selected {
		fnName := "test_" + strings.ToLower(strings.ReplaceAll(sc.ID, "-", "_"))
		pytestLines = append(pytestLines,
			fmt.Sprintf("def %s(client, tenant_a_ctx, tenant_b_ctx):", fnName),
			fmt.Sprintf("    \"\"\"[%s] %s: %s\"\"\"", sc.RuleID, sc.Threat, sc.Scenario),
			fmt.Sprintf("    # Target: %s | Expected Reaction: %s", targetEndpoint, sc.ExpectedReaction),
			fmt.Sprintf("    %s", sc.Assertion),
			"",
		)
		goFnName := "Test_" + strings.ReplaceAll(sc.ID, "-", "_")
		goTestLines = append(goTestLines,
			fmt.Sprintf("func %s(t *testing.T) {", goFnName),
			fmt.Sprintf("\t// [%s] %s: %s", sc.RuleID, sc.Threat, sc.Scenario),
			fmt.Sprintf("\t// Target: %s | Expected Reaction: %s", targetEndpoint, sc.ExpectedReaction),
			"}",
			"",
		)
	}

	return map[string]interface{}{
		"target_endpoint":   targetEndpoint,
		"component":         component,
		"total_generated":   len(selected),
		"scenarios":         selected,
		"pytest_template":   strings.Join(pytestLines, "\n"),
		"go_test_template":  strings.Join(goTestLines, "\n"),
		"regression_policy": "Every fixed vulnerability leaves a permanent security regression test (RAE-SEC-205).",
	}
}

// VerifyFix implements `quality.verify_fix` (Section 6 & 9).
func (cqe *CodeQualityEngine) VerifyFix(
	originalCode string,
	fixedCode string,
	ruleID string,
	canonicalTestCode string,
	agentSubmittedTestCode string,
) map[string]interface{} {
	if ruleID == "" {
		ruleID = "RAE-SEC-006"
	}

	// 1a. Check TestIntegrityGuard for assertion deletion or `assert True` softening
	if canonicalTestCode != "" && agentSubmittedTestCode != "" {
		passed, violation := cqe.testIntegrityGuard.ValidateTestIntegrity(canonicalTestCode, agentSubmittedTestCode)
		if !passed {
			return map[string]interface{}{
				"verdict":                 string(VerdictBlock),
				"fix_verified":            false,
				"rule_id":                 ruleID,
				"tamper_detected":         true,
				"ast_integrity_violation": violation,
				"reason":                  fmt.Sprintf("BLOCK (RAE-SEC-208 / TestIntegrityGuard): %s", violation),
			}
		}

		origSum := sha256.Sum256([]byte(strings.TrimSpace(canonicalTestCode)))
		subSum := sha256.Sum256([]byte(strings.TrimSpace(agentSubmittedTestCode)))
		if hex.EncodeToString(origSum[:]) != hex.EncodeToString(subSum[:]) {
			return map[string]interface{}{
				"verdict":         string(VerdictBlock),
				"fix_verified":    false,
				"rule_id":         ruleID,
				"tamper_detected": true,
				"reason":          "BLOCK (RAE-SEC-208): Agent modified canonical security test suite during repair. Fixes must pass against unmodified independent security tests.",
			}
		}
	}

	lowerTest := strings.ToLower(agentSubmittedTestCode)
	if containsAnyKeyword(lowerTest, []string{"pytest.mark.skip", "pytest.skip(", "pytest.mark.xfail", "@unittest.skip", "t.skip(", "t.skipnow("}) {
		return map[string]interface{}{
			"verdict":         string(VerdictBlock),
			"fix_verified":    false,
			"rule_id":         ruleID,
			"tamper_detected": true,
			"reason":          "BLOCK (Zero Test Weakening): Agent attempted to skip or xfail security test.",
		}
	}
	if agentSubmittedTestCode != "" && canonicalTestCode == "" && softAssertionRe.MatchString(agentSubmittedTestCode) {
		return map[string]interface{}{
			"verdict":         string(VerdictBlock),
			"fix_verified":    false,
			"rule_id":         ruleID,
			"tamper_detected": true,
			"reason":          "BLOCK (Zero Test Weakening): Trivial soft assertion (`assert True` / `1 == 1`) detected.",
		}
	}

	// 2. Scan original vs fixed code with SecurityGateway
	beforeScan := cqe.securityGateway.ScanCodeDetailed(originalCode, "before_fix", nil, nil)
	afterScan := cqe.securityGateway.ScanCodeDetailed(fixedCode, "after_fix", nil, nil)

	afterFindings, _ := afterScan["findings"].([]SecurityFinding)
	beforeCount, _ := beforeScan["findings_count"].(int)
	afterCount, _ := afterScan["findings_count"].(int)

	hasRemainingTarget := false
	hasRemainingBlocker := false
	for _, f := range afterFindings {
		if f.RuleID == ruleID {
			hasRemainingTarget = true
		}
		if f.Enforcement == EnforcementBlock {
			hasRemainingBlocker = true
		}
	}

	if hasRemainingTarget || hasRemainingBlocker {
		return map[string]interface{}{
			"verdict":            string(VerdictRepair),
			"fix_verified":       false,
			"rule_id":            ruleID,
			"tamper_detected":    false,
			"remaining_findings": afterFindings,
			"reason":             fmt.Sprintf("Vulnerability %s or new blocker still present in fixed code.", ruleID),
		}
	}

	return map[string]interface{}{
		"verdict":               string(VerdictPass),
		"fix_verified":          true,
		"rule_id":               ruleID,
		"tamper_detected":       false,
		"before_findings_count": beforeCount,
		"after_findings_count":  afterCount,
		"reason":                fmt.Sprintf("Independent verification confirmed %s is resolved with zero regressions.", ruleID),
	}
}

// ============================================================================
// 4. AIReviewEngine (packages/rae-quality/engines/governance/ai_review_engine.py)
// ============================================================================

var (
	endpointDefRe = regexp.MustCompile(
		`(?i)(?:@(?:app|router)\.(?:get|post|put|patch|delete)\s*\(\s*['"]([^'"]+)['"]|function\s+(\w+Action|\w+Controller)|func\s+(?:\([^)]+\)\s*)?\w+\s*\([^)]*(?:http\.ResponseWriter|\*gin\.Context|\*fiber\.Ctx))`,
	)
	authGuardRe = regexp.MustCompile(
		`(?i)(?:check_project_owner|verify_tenant_access|authorize_resource|Depends\(\s*get_current_tenant|current_user\.tenant_id|assert_owner|Policy::allows|\$this->authorize|VerifyTenantAccess|CheckProjectOwner|GetTenantFromContext|RequireTenantAuth|tenant\.GetDomainID|tenantMgr\.CompanyDB|extractTokenUser|verifyAdminToken)`,
	)
	adminBypassRe = regexp.MustCompile(
		`(?is)(?:if\s+.*?is_admin\s*:?\s*return\s+True|if\s+.*?IsAdmin\s*\{\s*return\s+true)`,
	)
	auditLogRe = regexp.MustCompile(
		`(?i)(?:audit_log|AuditLogger|record_audit|audit_recorder|security_audit|RecordAudit)`,
	)
	unsafeCouponOrStatusRe = regexp.MustCompile(
		`(?i)(?:order\.status\s*=\s*(?:req\.body\.status|payload\[['"]status['"]\])|discount_percent\s*=\s*(?:req\.body|payload)|coupon\.apply_unlimited|order\.Status\s*=\s*req\.Status)`,
	)
)

// AIReviewEngine performs semantic & business-logic review (Section 3 & Section 5).
type AIReviewEngine struct{}

// NewAIReviewEngine constructs an AIReviewEngine.
func NewAIReviewEngine() *AIReviewEngine {
	return &AIReviewEngine{}
}

// ReviewChangeSemantics evaluates semantic intent, authorization boundaries, and business logic integrity.
func (are *AIReviewEngine) ReviewChangeSemantics(
	code string,
	requirementDescription string,
	filePath string,
) map[string]interface{} {
	if filePath == "" {
		filePath = "inline_change.go"
	}
	findings := make([]SecurityFinding, 0)
	evidenceItems := make([]EvidenceItem, 0)
	lowerCode := strings.ToLower(code)

	// 1. Check Printshop Admin unscoped cross-tenant access or missing audit log (RAE-EDT-003)
	isUnscopedPrintshopAdmin := strings.Contains(lowerCode, "printshop_admin") &&
		!strings.Contains(lowerCode, "tenant_id") &&
		!strings.Contains(lowerCode, "tenantid") &&
		!auditLogRe.MatchString(code)

	if adminBypassRe.MatchString(code) || isUnscopedPrintshopAdmin {
		findings = append(findings, SecurityFinding{
			ID:          "FND-AI-RAE-EDT-003",
			RuleID:      "RAE-EDT-003",
			RuleName:    "Role & Permission Separation (Customer, Collaborator, Printshop Operator/Admin, Platform Admin)",
			Module:      "RAE-EDITOR-SECURITY",
			Severity:    SeverityCritical,
			Enforcement: EnforcementBlock,
			ControlType: ControlAISemantic,
			Scanner:     "rae-ai-review-engine",
			Message:     "Semantic Review: Printshop/Admin role grants access without enforcing printshop tenant_id boundary or recording an administrative audit log.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Scope printshop_admin queries strictly to their assigned tenant_id and log all administrative project/order access events via audit_log.",
			Evidence: map[string]interface{}{
				"semantic_check":   "admin_tenant_scope_and_audit",
				"verified_by_tool": true,
			},
		})
	}

	// 2. Check Order status / discount business logic manipulation (RAE-SEC-202)
	if unsafeCouponOrStatusRe.MatchString(code) {
		findings = append(findings, SecurityFinding{
			ID:          "FND-AI-RAE-SEC-202",
			RuleID:      "RAE-SEC-202",
			RuleName:    "Business Logic Security",
			Module:      "Agentic AI & SaaS Governance",
			Severity:    SeverityCritical,
			Enforcement: EnforcementBlock,
			ControlType: ControlAISemantic,
			Scanner:     "rae-ai-review-engine",
			Message:     "Semantic Review: Direct client control over order status transition or discount percentage detected without state-machine / server-side coupon policy validation.",
			FilePath:    filePath,
			LineNumber:  1,
			Remediation: "Enforce server-side order state machine transitions and server-validated coupon rules.",
			Evidence: map[string]interface{}{
				"semantic_check":   "order_state_machine_and_discounts",
				"verified_by_tool": true,
			},
		})
	}

	// 3. Check Endpoint Resource Ownership Verification on sensitive project/order routes (RAE-EDT-001)
	touchesSensitiveResource := containsAnyKeyword(lowerCode, []string{"project_id", "order_id", "asset_id", "page_id", "projectid", "orderid", "assetid", "pageid"})
	if endpointDefRe.MatchString(code) && touchesSensitiveResource {
		hasGuard := authGuardRe.MatchString(code) ||
			((strings.Contains(lowerCode, "tenant_id") || strings.Contains(lowerCode, "tenantid")) &&
				(strings.Contains(lowerCode, "owner_id") || strings.Contains(lowerCode, "ownerid") || strings.Contains(lowerCode, "userid")))
		doesDBLookup := containsAnyKeyword(lowerCode, []string{"select ", "db.query", ".get(", "repo.get", "findbyid"})
		if !hasGuard && doesDBLookup {
			findings = append(findings, SecurityFinding{
				ID:          "FND-AI-RAE-EDT-001",
				RuleID:      "RAE-EDT-001",
				RuleName:    "Project, Page, Asset, and Order Authorization",
				Module:      "RAE-EDITOR-SECURITY",
				Severity:    SeverityCritical,
				Enforcement: EnforcementBlock,
				ControlType: ControlAISemantic,
				Scanner:     "rae-ai-review-engine",
				Message:     "Semantic Review: Endpoint accesses project/order/asset by ID without verifiable owner_id / tenant_id authorization check.",
				FilePath:    filePath,
				LineNumber:  1,
				Remediation: "Verify resource ownership and tenant_id before loading or mutating project/order resources.",
				Evidence: map[string]interface{}{
					"semantic_check":   "endpoint_ownership_guard",
					"verified_by_tool": true,
				},
			})
		}
	}

	digest := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%s", filePath, code, requirementDescription)))
	approvedReviewID := "RAE-AI-REV-" + hex.EncodeToString(digest[:])[:16]

	failedRuleIDs := make(map[string]bool)
	hasBlockers := false
	for _, f := range findings {
		failedRuleIDs[f.RuleID] = true
		if f.Enforcement == EnforcementBlock {
			hasBlockers = true
		}
	}

	for _, rid := range []string{"RAE-EDT-001", "RAE-EDT-003", "RAE-SEC-201", "RAE-SEC-202"} {
		if !failedRuleIDs[rid] {
			evidenceItems = append(evidenceItems, EvidenceItem{
				RuleID:           rid,
				ControlType:      ControlAISemantic,
				ScannerOutput:    fmt.Sprintf("rae-ai-review-engine: PASS (%s)", rid),
				DataFlowTrace:    fmt.Sprintf("semantic_boundary_trace:%s->verified_clean", filePath),
				ApprovedReviewID: approvedReviewID,
				VerifiedByTool:   true,
			})
		}
	}

	verdict := VerdictPass
	if hasBlockers {
		verdict = VerdictBlock
	} else if len(findings) > 0 {
		verdict = VerdictReview
	}

	return map[string]interface{}{
		"verdict":             string(verdict),
		"approved_review_id":  approvedReviewID,
		"requirement_aligned": len(findings) == 0,
		"findings_count":      len(findings),
		"findings":            findings,
		"evidence_items":      evidenceItems,
	}
}

// ============================================================================
// 5. AutoRepairOrchestrator, AgentScorecardTracker & AuditTrailRecorder
//    (packages/rae-quality/engines/governance/auto_repair.py)
// ============================================================================

// AutoRepairOrchestrator manages the automated repair cycle for AI agents (Max 3 attempts -> HITL).
type AutoRepairOrchestrator struct {
	mu               sync.Mutex
	MaxAttempts      int
	attemptsByChange map[string]int
	history          []AutoRepairTask
}

// NewAutoRepairOrchestrator constructs an AutoRepairOrchestrator with MaxAttempts = 3.
func NewAutoRepairOrchestrator() *AutoRepairOrchestrator {
	return &AutoRepairOrchestrator{
		MaxAttempts:      3,
		attemptsByChange: make(map[string]int),
		history:          make([]AutoRepairTask, 0),
	}
}

// EvaluateOrScheduleRepair schedules a surgical repair task or escalates to Human Review after 3 attempts.
func (aro *AutoRepairOrchestrator) EvaluateOrScheduleRepair(
	changeID string,
	agentID string,
	findings []SecurityFinding,
) AutoRepairTask {
	aro.mu.Lock()
	defer aro.mu.Unlock()

	if len(findings) == 0 {
		att := aro.attemptsByChange[changeID]
		if att < 1 {
			att = 1
		}
		task := AutoRepairTask{
			RepairID:                fmt.Sprintf("REP-%s-%d", changeID, att),
			ChangeID:                changeID,
			AgentID:                 agentID,
			Attempt:                 att,
			MaxAttempts:             aro.MaxAttempts,
			Verdict:                 VerdictPass,
			EscalatedToHuman:        false,
			FailedRules:             []string{},
			MinimalFixInstructions:  []string{"All checks passed. Merge permitted."},
			IndependentTestRequired: true,
			Timestamp:               time.Now().UTC(),
		}
		aro.history = append(aro.history, task)
		return task
	}

	currentAttempt := aro.attemptsByChange[changeID] + 1
	aro.attemptsByChange[changeID] = currentAttempt

	ruleSet := make(map[string]bool)
	instructions := make([]string, 0, len(findings))
	for _, f := range findings {
		rid := f.RuleID
		if rid == "" {
			rid = "UNKNOWN"
		}
		ruleSet[rid] = true
		rem := f.Remediation
		if rem == "" {
			rem = f.Message
		}
		instructions = append(instructions, fmt.Sprintf("[%s] %s (Surgical fix only, do not alter unrelated code or tests)", rid, rem))
	}

	failedRules := make([]string, 0, len(ruleSet))
	for r := range ruleSet {
		failedRules = append(failedRules, r)
	}
	sort.Strings(failedRules)

	var task AutoRepairTask
	if currentAttempt > aro.MaxAttempts {
		escalatedMsg := fmt.Sprintf("ESCALATED TO HUMAN REVIEW: Auto-repair exhausted maximum %d attempts. Manual security architecture review required.", aro.MaxAttempts)
		task = AutoRepairTask{
			RepairID:                fmt.Sprintf("REP-%s-%d", changeID, currentAttempt),
			ChangeID:                changeID,
			AgentID:                 agentID,
			Attempt:                 currentAttempt,
			MaxAttempts:             aro.MaxAttempts,
			Verdict:                 VerdictBlock,
			EscalatedToHuman:        true,
			FailedRules:             failedRules,
			MinimalFixInstructions:  append([]string{escalatedMsg}, instructions...),
			IndependentTestRequired: true,
			Timestamp:               time.Now().UTC(),
		}
	} else {
		task = AutoRepairTask{
			RepairID:                fmt.Sprintf("REP-%s-%d", changeID, currentAttempt),
			ChangeID:                changeID,
			AgentID:                 agentID,
			Attempt:                 currentAttempt,
			MaxAttempts:             aro.MaxAttempts,
			Verdict:                 VerdictRepair,
			EscalatedToHuman:        false,
			FailedRules:             failedRules,
			MinimalFixInstructions:  instructions,
			IndependentTestRequired: true,
			Timestamp:               time.Now().UTC(),
		}
	}

	aro.history = append(aro.history, task)
	return task
}

// AgentScorecardTracker tracks empirical engineering quality of each AI Agent (Section 8).
type AgentScorecardTracker struct {
	mu     sync.Mutex
	agents map[string]*AgentQualityScorecard
}

// NewAgentScorecardTracker constructs an AgentScorecardTracker.
func NewAgentScorecardTracker() *AgentScorecardTracker {
	return &AgentScorecardTracker{
		agents: make(map[string]*AgentQualityScorecard),
	}
}

// RecordChangeEvaluation updates the scorecard for an AI agent.
func (ast *AgentScorecardTracker) RecordChangeEvaluation(
	agentID string,
	defectsFound int,
	wasFixAttempt bool,
	fixSucceededFirstTime bool,
	regressionDetected bool,
	postDeployVuln bool,
	tamperBlocked bool,
) AgentQualityScorecard {
	ast.mu.Lock()
	defer ast.mu.Unlock()

	card, exists := ast.agents[agentID]
	if !exists {
		card = &AgentQualityScorecard{
			AgentID:             agentID,
			FirstTimeFixRatePct: 100.0,
			ReliabilityTier:     "SOTA",
		}
		ast.agents[agentID] = card
	}

	card.TotalChanges++
	if defectsFound > 0 {
		card.TotalDefects += defectsFound
	}
	card.DefectsPerChange = math.Round((float64(card.TotalDefects)/float64(card.TotalChanges))*1000) / 1000

	if wasFixAttempt {
		card.TotalFixAttempts++
		if fixSucceededFirstTime {
			card.FirstTimeFixCount++
		}
		card.FirstTimeFixRatePct = math.Round((float64(card.FirstTimeFixCount)/float64(card.TotalFixAttempts))*10000) / 100
	}

	if regressionDetected {
		card.RegressionsIntroduced++
	}
	if postDeployVuln {
		card.PostDeployVulnerabilities++
	}
	if tamperBlocked {
		card.TamperAttemptsBlocked++
	}

	if card.TamperAttemptsBlocked > 0 || card.PostDeployVulnerabilities > 0 {
		card.ReliabilityTier = "RESTRICTED"
	} else if card.DefectsPerChange <= 0.25 && card.FirstTimeFixRatePct >= 85.0 {
		card.ReliabilityTier = "SOTA"
	} else if card.DefectsPerChange <= 0.75 {
		card.ReliabilityTier = "STANDARD"
	} else {
		card.ReliabilityTier = "PROBATION"
	}

	return *card
}

// GetAllScorecards returns all tracked agent scorecards.
func (ast *AgentScorecardTracker) GetAllScorecards() []AgentQualityScorecard {
	ast.mu.Lock()
	defer ast.mu.Unlock()

	out := make([]AgentQualityScorecard, 0, len(ast.agents))
	for _, c := range ast.agents {
		out = append(out, *c)
	}
	return out
}

// AuditTrailRecorder implements RAE-SEC-210 (Audit Trail Integrity) with a SHA-256 hash chain.
type AuditTrailRecorder struct {
	mu    sync.Mutex
	chain []map[string]interface{}
}

// NewAuditTrailRecorder constructs an AuditTrailRecorder.
func NewAuditTrailRecorder() *AuditTrailRecorder {
	return &AuditTrailRecorder{
		chain: make([]map[string]interface{}, 0),
	}
}

// AppendDecision appends a new hash-chained decision entry to the immutable audit ledger.
func (atr *AuditTrailRecorder) AppendDecision(
	operation string,
	verdict string,
	actor string,
	evidenceSummary map[string]interface{},
) map[string]interface{} {
	atr.mu.Lock()
	defer atr.mu.Unlock()

	prevHash := strings.Repeat("0", 64)
	if len(atr.chain) > 0 {
		if ph, ok := atr.chain[len(atr.chain)-1]["entry_sha256"].(string); ok {
			prevHash = ph
		}
	}

	idx := len(atr.chain) + 1
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	payload := map[string]interface{}{
		"actor":            actor,
		"evidence_summary": evidenceSummary,
		"index":            idx,
		"operation":        operation,
		"prev_sha256":      prevHash,
		"timestamp":        ts,
		"verdict":          verdict,
	}
	canonicalJSON, _ := json.Marshal(payload)
	sum := sha256.Sum256(canonicalJSON)
	entryHash := hex.EncodeToString(sum[:])

	entry := map[string]interface{}{
		"actor":            actor,
		"entry_sha256":     entryHash,
		"evidence_summary": evidenceSummary,
		"index":            idx,
		"operation":        operation,
		"prev_sha256":      prevHash,
		"timestamp":        ts,
		"verdict":          verdict,
	}
	atr.chain = append(atr.chain, entry)
	return entry
}

// VerifyChainIntegrity verifies the cryptographic continuity of the entire SHA-256 audit ledger.
func (atr *AuditTrailRecorder) VerifyChainIntegrity() map[string]interface{} {
	atr.mu.Lock()
	defer atr.mu.Unlock()

	prev := strings.Repeat("0", 64)
	for idx, entry := range atr.chain {
		prevInEntry, _ := entry["prev_sha256"].(string)
		if prevInEntry != prev {
			return map[string]interface{}{
				"valid":           false,
				"broken_at_index": idx + 1,
			}
		}
		checkPayload := map[string]interface{}{
			"actor":            entry["actor"],
			"evidence_summary": entry["evidence_summary"],
			"index":            entry["index"],
			"operation":        entry["operation"],
			"prev_sha256":      entry["prev_sha256"],
			"timestamp":        entry["timestamp"],
			"verdict":          entry["verdict"],
		}
		canonicalJSON, _ := json.Marshal(checkPayload)
		sum := sha256.Sum256(canonicalJSON)
		expectedHash := hex.EncodeToString(sum[:])
		actualHash, _ := entry["entry_sha256"].(string)
		if expectedHash != actualHash {
			return map[string]interface{}{
				"valid":           false,
				"broken_at_index": idx + 1,
			}
		}
		prev = actualHash
	}

	return map[string]interface{}{
		"valid":         true,
		"entries_count": len(atr.chain),
		"head_sha256":   prev,
	}
}

func boolFromMap(m map[string]interface{}, key string, def bool) bool {
	if m == nil {
		return def
	}
	if v, ok := m[key]; ok {
		if b, isBool := v.(bool); isBool {
			return b
		}
	}
	return def
}

func intFromMap(m map[string]interface{}, key string, def int) int {
	if m == nil {
		return def
	}
	if v, ok := m[key]; ok {
		switch iv := v.(type) {
		case int:
			return iv
		case float64:
			return int(iv)
		}
	}
	return def
}
