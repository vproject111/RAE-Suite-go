package quality

import (
	"fmt"
	"sort"
	"strings"
)

// ProtectedQualityPaths lists governance paths that coding agents are forbidden from weakening.
var ProtectedQualityPaths = []string{
	"packages/rae-quality/engines/policy",
	"packages/rae-quality/engines/security",
	"packages/rae-quality/engines/governance",
	"services/quality/policy_engine.go",
	"services/quality/security_gateway.go",
	"services/quality/gateway_v2.go",
	"scripts/hard_contract_gatekeeper.py",
	"AGENT_CORE_PROTOCOL.md",
}

// RolloutPhases maps Section 10 rollout stages to active priorities.
var RolloutPhases = map[string][]string{
	"etap_1_p0":   {"P0"},
	"etap_2_p1":   {"P0", "P0/P1", "P1"},
	"etap_3_p2":   {"P0", "P0/P1", "P1", "P1/P2", "P2"},
	"etap_4_prod": {"P0", "P0/P1", "P1", "P1/P2", "P2", "P3"},
}

// PolicyEngine implements packages/rae-quality/engines/policy/policy_engine.py.
type PolicyEngine struct {
	catalog map[string]QualityRule
}

// NewPolicyEngine creates a PolicyEngine initialized with all 219 rules.
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		catalog: BuildFullRuleCatalog(),
	}
}

// Catalog returns the full dictionary of 219 rules.
func (pe *PolicyEngine) Catalog() map[string]QualityRule {
	return pe.catalog
}

// GetRule looks up a rule by ID.
func (pe *PolicyEngine) GetRule(ruleID string) (QualityRule, bool) {
	r, ok := pe.catalog[ruleID]
	return r, ok
}

// GetPolicyForTask implements `quality.get_policy` (Stage 1: Before Coding & Risk Profiling).
func (pe *PolicyEngine) GetPolicyForTask(
	taskDescription string,
	modifiedFiles []string,
	technologies []string,
	stage string,
	rolloutPhase string,
) map[string]interface{} {
	if len(technologies) == 0 {
		technologies = []string{"postgresql", "redis", "docker", "kubernetes"}
	}
	activeTechs := make(map[string]bool, len(technologies))
	for _, t := range technologies {
		activeTechs[strings.ToLower(t)] = true
	}

	phase := rolloutPhase
	if phase == "" {
		phase = "etap_4_prod"
	}
	allowedSlice, ok := RolloutPhases[phase]
	if !ok {
		allowedSlice = RolloutPhases["etap_4_prod"]
	}
	allowedPriorities := make(map[string]bool, len(allowedSlice))
	for _, p := range allowedSlice {
		allowedPriorities[p] = true
	}

	lowerDesc := strings.ToLower(taskDescription)
	lowerFiles := strings.ToLower(strings.Join(modifiedFiles, " "))
	combinedCtx := lowerDesc + " " + lowerFiles

	// Detect if change is purely cosmetic UI (e.g. button color change in .css/.scss/.less)
	isCosmeticUIOnly := len(modifiedFiles) > 0
	for _, f := range modifiedFiles {
		lf := strings.ToLower(f)
		if !strings.HasSuffix(lf, ".css") && !strings.HasSuffix(lf, ".scss") && !strings.HasSuffix(lf, ".less") {
			isCosmeticUIOnly = false
			break
		}
	}
	if isCosmeticUIOnly {
		for _, kw := range []string{"auth", "order", "api", "db", "sql", "tenant", "price"} {
			if strings.Contains(lowerDesc, kw) {
				isCosmeticUIOnly = false
				break
			}
		}
	}

	var selectedRules []QualityRule
	mandatoryDomains := make(map[string]bool)
	riskLevel := "HIGH"

	if isCosmeticUIOnly {
		for _, rid := range []string{"RAE-SEC-002", "RAE-SEC-010", "RAE-SEC-020", "RAE-SEC-208"} {
			if r, exists := pe.catalog[rid]; exists {
				selectedRules = append(selectedRules, r)
			}
		}
		mandatoryDomains["xss_csp"] = true
		mandatoryDomains["secrets_scan"] = true
		riskLevel = "LOW"
	} else {
		selectedIDs := map[string]bool{
			"RAE-SEC-001": true, // SQL Injection
			"RAE-SEC-002": true, // XSS
			"RAE-SEC-005": true, // SSRF
			"RAE-SEC-006": true, // IDOR / Cross-tenant object access
			"RAE-SEC-007": true, // Path Traversal
			"RAE-SEC-008": true, // Mass Assignment
			"RAE-SEC-013": true, // Server-Side Authorization
			"RAE-SEC-014": true, // Row-Level Security
			"RAE-SEC-017": true, // Secrets Management
			"RAE-SEC-020": true, // Secret Scanning
			"RAE-SEC-033": true, // File Upload Validation
			"RAE-SEC-199": true, // AI Prompt Injection
			"RAE-SEC-201": true, // Tenant Isolation Testing
			"RAE-SEC-208": true, // Agent Self-Modification Protection
			"RAE-SEC-210": true, // Audit Trail Integrity
		}
		mandatoryDomains["authorization"] = true
		mandatoryDomains["tenant_isolation"] = true
		mandatoryDomains["secrets_scan"] = true
		mandatoryDomains["sast"] = true

		// Orders / Checkout / API / Pricing context
		if containsAnyKeyword(combinedCtx, []string{"order", "zamow", "cart", "koszyk", "price", "calc", "api", "controller", "endpoint", "handler"}) {
			for _, id := range []string{
				"RAE-SEC-048", "RAE-SEC-049", "RAE-SEC-050", "RAE-SEC-090",
				"RAE-SEC-128", "RAE-SEC-129", "RAE-SEC-202", "RAE-EDT-001", "RAE-EDT-005",
			} {
				selectedIDs[id] = true
			}
			mandatoryDomains["transactions"] = true
			mandatoryDomains["openapi_compatibility"] = true
			mandatoryDomains["order_regression"] = true
			mandatoryDomains["business_logic"] = true
		}

		// Editor / DTP / PDF / Canvas / Upload context
		if containsAnyKeyword(combinedCtx, []string{"editor", "pdf", "dtp", "svg", "image", "upload", "canvas", "scene", "bleed", "exif"}) {
			for _, edt := range RAEEditorSecurityRules {
				selectedIDs[edt.ID] = true
			}
			mandatoryDomains["editor_security"] = true
			mandatoryDomains["file_upload_security"] = true
			mandatoryDomains["race_conditions"] = true
		}

		// Database / Migration context
		if containsAnyKeyword(combinedCtx, []string{"migration", "schema", "sql", "db", "database", "model", "repository", "query", "postgres"}) {
			for _, id := range []string{
				"RAE-SEC-041", "RAE-SEC-043", "RAE-SEC-045", "RAE-SEC-046",
				"RAE-SEC-047", "RAE-SEC-048", "RAE-SEC-052", "RAE-SEC-056",
			} {
				selectedIDs[id] = true
			}
			if activeTechs["postgresql"] {
				selectedIDs["RAE-SEC-068"] = true // Postgres VACUUM
			}
			mandatoryDomains["database_integrity"] = true
		}

		// Infrastructure / Docker / K8s / CI context
		if containsAnyKeyword(combinedCtx, []string{"dockerfile", "docker-compose", "k8s", "helm", "values.yaml", "ci", "deploy", "infra"}) {
			for _, id := range []string{
				"RAE-SEC-141", "RAE-SEC-142", "RAE-SEC-143", "RAE-SEC-190",
				"RAE-SEC-191", "RAE-SEC-192", "RAE-SEC-193", "RAE-SEC-194",
				"RAE-SEC-195", "RAE-SEC-196", "RAE-SEC-198",
			} {
				selectedIDs[id] = true
			}
			mandatoryDomains["infrastructure_containers"] = true
		}

		sortedIDs := make([]string, 0, len(selectedIDs))
		for id := range selectedIDs {
			sortedIDs = append(sortedIDs, id)
		}
		sort.Strings(sortedIDs)

		for _, rid := range sortedIDs {
			rule, exists := pe.catalog[rid]
			if !exists {
				continue
			}
			if len(rule.AppliesTo.Technologies) > 0 {
				techMatch := false
				for _, rt := range rule.AppliesTo.Technologies {
					if activeTechs[strings.ToLower(rt)] {
						techMatch = true
						break
					}
				}
				if !techMatch {
					continue
				}
			}
			if !allowedPriorities[string(rule.Priority)] {
				continue
			}
			if stage != "" && string(rule.Stage) != stage && rule.Priority != PriorityP0 {
				continue
			}
			selectedRules = append(selectedRules, rule)
		}

		for _, r := range selectedRules {
			if r.Severity == SeverityCritical {
				riskLevel = "CRITICAL"
				break
			}
		}
	}

	sortedTechs := make([]string, 0, len(activeTechs))
	for t := range activeTechs {
		sortedTechs = append(sortedTechs, t)
	}
	sort.Strings(sortedTechs)

	sortedDomains := make([]string, 0, len(mandatoryDomains))
	for d := range mandatoryDomains {
		sortedDomains = append(sortedDomains, d)
	}
	sort.Strings(sortedDomains)

	sortedAllowed := make([]string, 0, len(allowedPriorities))
	for p := range allowedPriorities {
		sortedAllowed = append(sortedAllowed, p)
	}
	sort.Strings(sortedAllowed)

	return map[string]interface{}{
		"task_description":     taskDescription,
		"modified_files":       modifiedFiles,
		"active_technologies":  sortedTechs,
		"rollout_phase":        phase,
		"allowed_priorities":   sortedAllowed,
		"baseline_policy":      "zero_tolerance_for_new_code_baseline_debt_tracked_separately",
		"is_cosmetic_ui_only":  isCosmeticUIOnly,
		"risk_level":           riskLevel,
		"mandatory_domains":    sortedDomains,
		"selected_rules_count": len(selectedRules),
		"total_catalog_rules":  len(pe.catalog),
		"rules":                selectedRules,
		"agent_constraints": map[string]interface{}{
			"can_disable_mandatory_controls": false,
			"can_self_approve_exceptions":    false,
			"can_modify_security_tests":      false,
			"requires_verifiable_evidence":   true,
		},
	}
}

// ValidateRuleEvidence enforces Section 5: bare agent statements without tool-verified proof are rejected.
func (pe *PolicyEngine) ValidateRuleEvidence(ruleID string, evidence EvidenceItem) map[string]interface{} {
	rule, exists := pe.catalog[ruleID]
	if !exists {
		return map[string]interface{}{
			"valid":  false,
			"reason": fmt.Sprintf("Unknown rule_id: %s", ruleID),
		}
	}

	hasConcreteProof := (evidence.TestResult != "" ||
		evidence.ScannerOutput != "" ||
		evidence.DataFlowTrace != "" ||
		evidence.ApprovedReviewID != "") && evidence.VerifiedByTool

	if !hasConcreteProof {
		return map[string]interface{}{
			"valid":   false,
			"rule_id": ruleID,
			"reason":  "REJECTED: Unverified agent statement is not valid evidence. Must provide verifiable scanner output, test execution result, data-flow trace, or approved review.",
		}
	}

	var missingFields []string
	for _, reqField := range rule.EvidenceRequired {
		switch reqField {
		case "source_location":
			if evidence.SourceLocation == "" {
				missingFields = append(missingFields, reqField)
			}
		case "affected_endpoint":
			if evidence.AffectedEndpoint == "" {
				missingFields = append(missingFields, reqField)
			}
		case "authorization_path":
			if evidence.AuthorizationPath == "" {
				missingFields = append(missingFields, reqField)
			}
		case "test_result":
			if evidence.TestResult == "" {
				missingFields = append(missingFields, reqField)
			}
		case "scanner_output":
			if evidence.ScannerOutput == "" {
				missingFields = append(missingFields, reqField)
			}
		}
	}

	if len(missingFields) > 0 {
		return map[string]interface{}{
			"valid":          false,
			"rule_id":        ruleID,
			"reason":         fmt.Sprintf("Missing required evidence fields for %s: %s", ruleID, strings.Join(missingFields, ", ")),
			"missing_fields": missingFields,
		}
	}

	return map[string]interface{}{
		"valid":   true,
		"rule_id": ruleID,
		"reason":  "Evidence verified by independent tool/test.",
	}
}

// CheckAgentTampering enforces RAE-SEC-208 (Agent Self-Modification Protection).
func (pe *PolicyEngine) CheckAgentTampering(
	agentID string,
	modifiedFiles []string,
	requestedOverride bool,
	disableRules []string,
) map[string]interface{} {
	var violations []string
	if requestedOverride {
		violations = append(violations, fmt.Sprintf("Agent '%s' attempted unauthorized quality gate override.", agentID))
	}
	if len(disableRules) > 0 {
		violations = append(violations, fmt.Sprintf("Agent '%s' attempted to disable mandatory rules: %v", agentID, disableRules))
	}
	for _, f := range modifiedFiles {
		norm := strings.ReplaceAll(f, "\\", "/")
		for _, prot := range ProtectedQualityPaths {
			if strings.Contains(norm, prot) {
				violations = append(violations, fmt.Sprintf("Agent '%s' attempted to modify protected quality governance path: %s", agentID, f))
				break
			}
		}
	}
	enforcement := "PASS"
	if len(violations) > 0 {
		enforcement = "BLOCK"
	}
	return map[string]interface{}{
		"tamper_detected": len(violations) > 0,
		"rule_id":         "RAE-SEC-208",
		"enforcement":     enforcement,
		"violations":      violations,
	}
}

// GenerateTechnicalManifest generates the Section 13 technical implementation manifest across all 219 rules.
func (pe *PolicyEngine) GenerateTechnicalManifest() map[string]interface{} {
	modulesSummary := make([]map[string]interface{}, 0, len(ModuleDefinitions))
	for _, mod := range ModuleDefinitions {
		count := 0
		for _, r := range pe.catalog {
			if r.Category == mod.ID || r.Module == mod.Name {
				count++
			}
		}
		modulesSummary = append(modulesSummary, map[string]interface{}{
			"module_id":       mod.ID,
			"module_name":     mod.Name,
			"priority":        string(mod.Priority),
			"default_scanner": mod.DefaultScanner,
			"rules_count":     count,
		})
	}

	return map[string]interface{}{
		"version":                     "2.0.0",
		"total_198_topics":            len(All198Topics),
		"total_ai_saas_extensions":    len(AISaaSTopicDetails),
		"total_editor_security_rules": len(RAEEditorSecurityRules),
		"total_rules_in_engine":       len(pe.catalog),
		"modules":                     modulesSummary,
		"control_types":               []string{"deterministic", "ai_semantic", "systemic"},
		"stages":                      []string{"before_coding", "during_work", "pre_merge", "pre_prod_post_deploy"},
	}
}

func containsAnyKeyword(haystack string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(haystack, kw) {
			return true
		}
	}
	return false
}
