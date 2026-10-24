package quality

import (
	"fmt"
	"strings"
)

// ModuleRange represents a numeric topic range [Start, End] inclusive.
type ModuleRange struct {
	Start int
	End   int
}

// ModuleDef represents one of the 12 Control Modules from Section 2 of poziom-bezpieczeństwa-kodu.md.
type ModuleDef struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Ranges         []ModuleRange  `json:"ranges"`
	Priority       ModulePriority `json:"priority"`
	DefaultScanner string         `json:"default_scanner"`
}

// ModuleDefinitions maps all 198 topics into 12 engineering/security control modules.
var ModuleDefinitions = []ModuleDef{
	{
		ID:             "security_owasp",
		Name:           "Security & OWASP",
		Ranges:         []ModuleRange{{1, 16}, {33, 40}},
		Priority:       PriorityP0,
		DefaultScanner: "semgrep+codeql+trivy",
	},
	{
		ID:             "identity_secrets",
		Name:           "Identity & Secrets",
		Ranges:         []ModuleRange{{17, 32}},
		Priority:       PriorityP0,
		DefaultScanner: "gitleaks+semgrep",
	},
	{
		ID:             "database_integrity",
		Name:           "Database Integrity",
		Ranges:         []ModuleRange{{41, 75}},
		Priority:       PriorityP1,
		DefaultScanner: "rae-architecture-engine",
	},
	{
		ID:             "distributed_systems",
		Name:           "Distributed Systems",
		Ranges:         []ModuleRange{{76, 97}},
		Priority:       PriorityP2,
		DefaultScanner: "rae-architecture-engine+k6",
	},
	{
		ID:             "cache_performance",
		Name:           "Cache & Performance",
		Ranges:         []ModuleRange{{98, 106}},
		Priority:       PriorityP2,
		DefaultScanner: "rae-architecture-engine",
	},
	{
		ID:             "messaging_jobs",
		Name:           "Messaging & Jobs",
		Ranges:         []ModuleRange{{107, 115}},
		Priority:       PriorityP1P2,
		DefaultScanner: "rae-architecture-engine",
	},
	{
		ID:             "network_api",
		Name:           "Network & API",
		Ranges:         []ModuleRange{{116, 133}},
		Priority:       PriorityP1,
		DefaultScanner: "schemathesis+owasp-zap",
	},
	{
		ID:             "infrastructure_security",
		Name:           "Infrastructure Security",
		Ranges:         []ModuleRange{{134, 150}},
		Priority:       PriorityP1,
		DefaultScanner: "checkov+trivy",
	},
	{
		ID:             "reliability_operations",
		Name:           "Reliability & Operations",
		Ranges:         []ModuleRange{{151, 167}},
		Priority:       PriorityP1P2,
		DefaultScanner: "k6+rae-infra-engine",
	},
	{
		ID:             "observability",
		Name:           "Observability",
		Ranges:         []ModuleRange{{168, 179}},
		Priority:       PriorityP1,
		DefaultScanner: "rae-infra-engine",
	},
	{
		ID:             "delivery_containers",
		Name:           "Delivery & Containers",
		Ranges:         []ModuleRange{{180, 194}},
		Priority:       PriorityP0P1,
		DefaultScanner: "trivy+checkov",
	},
	{
		ID:             "supply_chain_git",
		Name:           "Supply Chain & Git",
		Ranges:         []ModuleRange{{195, 198}},
		Priority:       PriorityP0,
		DefaultScanner: "trivy+rae-git-guard",
	},
}

// AISaaSTopicDetails maps topics 199..210 with their canonical Polish descriptions from Section 11.
var AISaaSTopicDetails = []struct {
	Number      int
	Name        string
	Description string
}{
	{199, "AI Prompt Injection", "Złośliwe instrukcje w repozytorium, dokumentach i danych wejściowych"},
	{200, "Agent Tool Permissions", "Ograniczenie uprawnień agentów do narzędzi, repozytoriów i produkcji"},
	{201, "Tenant Isolation Testing", "Automatyczne próby przekraczania granic klientów SaaS"},
	{202, "Business Logic Security", "Manipulacja cenami, zamówieniami, rabatami i uprawnieniami"},
	{203, "SBOM & Provenance", "Identyfikowalność zależności i pochodzenia artefaktów"},
	{204, "Artifact Signing", "Potwierdzenie, że wdrażany jest dokładnie zatwierdzony artefakt"},
	{205, "Security Regression Tests", "Każda naprawiona podatność pozostawia trwały test"},
	{206, "Data Privacy & Retention", "Ochrona danych osobowych, zdjęć, EXIF i zasad retencji"},
	{207, "Safe Deserialization", "Ochrona przed niebezpiecznym odtwarzaniem obiektów"},
	{208, "Agent Self-Modification Protection", "Agent nie może osłabiać własnych reguł kontroli"},
	{209, "Resource Exhaustion", "Limity CPU, RAM, rozmiaru projektów i kosztów AI"},
	{210, "Audit Trail Integrity", "Niezmienialna historia decyzji i zatwierdzeń bezpieczeństwa"},
}

// RAEEditorSecurityRules defines the 9 custom SaaS & DTP Editor security rules from Section 4.
var RAEEditorSecurityRules = []QualityRule{
	{
		ID:          "RAE-EDT-001",
		Version:     "1.0.0",
		Name:        "Project, Page, Asset and Order Level Authorization",
		Category:    "editor_authorization",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "typescript", "javascript", "php", "python"},
			Components:   []string{"editor", "api", "backend"},
			Technologies: []string{},
		},
		Triggers:     []string{"editor_resource_access", "order_mutation", "page_mutation", "asset_access"},
		Verification: RuleVerification{StaticAnalysis: true, AIReview: true, IntegrationTest: true, NegativeTest: true},
		Requirements: []string{
			"Verify ownership at project, page, asset, and order granularity",
			"Deny cross-project asset injection or page cloning without ownership proof",
		},
		EvidenceRequired: []string{"source_location", "affected_endpoint", "authorization_path", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "semgrep+playwright",
	},
	{
		ID:          "RAE-EDT-002",
		Version:     "1.0.0",
		Name:        "Multi-Printshop Data Isolation (Cache, WebSockets, Queues, Files)",
		Category:    "tenant_isolation",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "typescript", "javascript", "php", "python"},
			Components:   []string{"editor", "api", "backend", "cache", "websocket", "queue"},
			Technologies: []string{},
		},
		Triggers:     []string{"cache_key_defined", "websocket_room_joined", "queue_job_dispatched", "storage_path_resolved"},
		Verification: RuleVerification{StaticAnalysis: true, AIReview: true, IntegrationTest: true, NegativeTest: true},
		Requirements: []string{
			"Prefix all Redis cache keys, WebSocket rooms, queue payloads, and S3/local file paths with tenant_id",
			"Block cross-printshop event broadcasting or shared cache poisoning",
		},
		EvidenceRequired: []string{"source_location", "authorization_path", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "semgrep+rae-security-gateway",
	},
	{
		ID:          "RAE-EDT-003",
		Version:     "1.0.0",
		Name:        "Printshop Admin Project Access & Impersonation Audit",
		Category:    "authorization",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "typescript", "javascript", "php", "python"},
			Components:   []string{"editor", "backend", "api"},
			Technologies: []string{},
		},
		Triggers:     []string{"admin_impersonation", "project_support_access"},
		Verification: RuleVerification{StaticAnalysis: true, AIReview: true, IntegrationTest: true},
		Requirements: []string{
			"Printshop admin may only access projects belonging to their own printshop tenant",
			"Emit immutable audit log entry whenever admin opens or edits a customer project",
		},
		EvidenceRequired: []string{"source_location", "authorization_path", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "rae-security-gateway",
	},
	{
		ID:          "RAE-EDT-004",
		Version:     "1.0.0",
		Name:        "Multi-User Collaborative Project Editing Safety",
		Category:    "concurrency",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP1,
		Severity:    SeverityHigh,
		Enforcement: EnforcementRepair,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "typescript", "javascript", "python"},
			Components:   []string{"editor", "websocket", "api"},
			Technologies: []string{},
		},
		Triggers:     []string{"collaborative_sync_changed", "scene_save_changed"},
		Verification: RuleVerification{StaticAnalysis: true, IntegrationTest: true, NegativeTest: true},
		Requirements: []string{
			"Prevent lost updates during simultaneous saves via optimistic locking (version/ETag) or CRDT/Yjs conflict resolution",
			"Enforce per-collaborator role permissions (viewer vs editor)",
		},
		EvidenceRequired: []string{"source_location", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "playwright+k6",
	},
	{
		ID:          "RAE-EDT-005",
		Version:     "1.0.0",
		Name:        "Production Parameter Server-Side Validation (Format, Bleed, Pages, Price)",
		Category:    "business_logic",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "php", "typescript", "javascript", "python"},
			Components:   []string{"editor", "backend", "api", "calculator"},
			Technologies: []string{},
		},
		Triggers:     []string{"pricing_logic_changed", "order_checkout_changed", "dtp_spec_changed"},
		Verification: RuleVerification{StaticAnalysis: true, AIReview: true, IntegrationTest: true, NegativeTest: true},
		Requirements: []string{
			"Never trust client-submitted price, discount, page count, bleed, or format dimensions",
			"Recalculate price and validate DTP geometry server-side before order acceptance",
		},
		EvidenceRequired: []string{"source_location", "affected_endpoint", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "rae-security-gateway+schemathesis",
	},
	{
		ID:          "RAE-EDT-006",
		Version:     "1.0.0",
		Name:        "Production PDF File Tampering & Replacement Protection",
		Category:    "file_integrity",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "php", "typescript", "javascript", "python"},
			Components:   []string{"editor", "backend", "api", "storage"},
			Technologies: []string{},
		},
		Triggers:     []string{"pdf_export_changed", "production_file_upload"},
		Verification: RuleVerification{StaticAnalysis: true, IntegrationTest: true, NegativeTest: true},
		Requirements: []string{
			"Protect approved production PDFs with cryptographic SHA-256 checksums and signed URLs",
			"Block unauthorized overwrite of print-ready PDF after order lock",
		},
		EvidenceRequired: []string{"source_location", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "rae-security-gateway",
	},
	{
		ID:          "RAE-EDT-007",
		Version:     "1.0.0",
		Name:        "Safe Image Processing & EXIF Geolocation Metadata Sanitization",
		Category:    "privacy_upload",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP1,
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "php", "typescript", "javascript", "python"},
			Components:   []string{"editor", "backend", "api"},
			Technologies: []string{},
		},
		Triggers:     []string{"image_upload_changed", "asset_pipeline_changed"},
		Verification: RuleVerification{StaticAnalysis: true, IntegrationTest: true, NegativeTest: true},
		Requirements: []string{
			"Strip sensitive EXIF metadata (GPS coordinates, camera serials) while preserving ICC color profiles for DTP",
			"Sanitize SVG uploads against embedded <script>, onload, and XXE payloads",
		},
		EvidenceRequired: []string{"source_location", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "semgrep+rae-security-gateway",
	},
	{
		ID:          "RAE-EDT-008",
		Version:     "1.0.0",
		Name:        "AI Model Cost Quota & Invocation Permission Control",
		Category:    "ai_governance",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "python", "typescript", "javascript", "php"},
			Components:   []string{"editor", "api", "backend", "ai"},
			Technologies: []string{},
		},
		Triggers:     []string{"llm_call_added", "ai_image_generation_changed"},
		Verification: RuleVerification{StaticAnalysis: true, IntegrationTest: true},
		Requirements: []string{
			"Enforce per-tenant token/cost budgets and rate limits on all AI/LLM endpoints",
			"Verify user role authorization before invoking paid external AI models",
		},
		EvidenceRequired: []string{"source_location", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "rae-security-gateway",
	},
	{
		ID:          "RAE-EDT-009",
		Version:     "1.0.0",
		Name:        "Prompt Injection Protection in Documents, Images, Descriptions and External Sources",
		Category:    "ai_security",
		Module:      "RAE-EDITOR-SECURITY",
		Priority:    PriorityP0,
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go", "python", "typescript", "javascript", "php"},
			Components:   []string{"editor", "api", "backend", "ai"},
			Technologies: []string{},
		},
		Triggers:     []string{"prompt_template_changed", "external_data_ingestion_changed", "document_ocr_changed"},
		Verification: RuleVerification{StaticAnalysis: true, AIReview: true, NegativeTest: true},
		Requirements: []string{
			"Isolate untrusted document text, OCR output, image metadata, and external URLs from system instructions",
			"Scan ingested content for indirect prompt injection and tool-hijacking payloads",
		},
		EvidenceRequired: []string{"source_location", "test_result"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "rae-security-gateway",
	},
}

func resolveModuleForTopic(topicNum int) ModuleDef {
	for _, mod := range ModuleDefinitions {
		for _, r := range mod.Ranges {
			if topicNum >= r.Start && topicNum <= r.End {
				return mod
			}
		}
	}
	return ModuleDef{
		ID:             "agentic_ai_security",
		Name:           "Agentic AI & SaaS Governance",
		Priority:       PriorityP0,
		DefaultScanner: "rae-security-gateway",
	}
}

func determineControlType(topicNum int) ControlType {
	systemicNums := map[int]bool{
		35: true, 63: true, 64: true, 70: true, 71: true, 72: true,
		134: true, 135: true, 136: true, 137: true, 138: true,
		146: true, 147: true, 148: true, 150: true, 151: true,
		152: true, 153: true, 154: true, 155: true, 156: true,
		157: true, 158: true, 159: true,
	}
	if systemicNums[topicNum] {
		return ControlSystemic
	}
	if (topicNum >= 1 && topicNum <= 12) || (topicNum >= 17 && topicNum <= 24) ||
		topicNum == 33 || topicNum == 36 || topicNum == 38 || topicNum == 39 ||
		(topicNum >= 190 && topicNum <= 198) ||
		topicNum == 203 || topicNum == 204 || topicNum == 207 || topicNum == 208 || topicNum == 210 {
		return ControlDeterministic
	}
	return ControlAISemantic
}

func determineTechnologies(name string) []string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "postgres") {
		return []string{"postgresql"}
	}
	if strings.Contains(lower, "kafka") {
		return []string{"kafka"}
	}
	if strings.Contains(lower, "redis") {
		return []string{"redis"}
	}
	if strings.Contains(lower, "graphql") || strings.Contains(lower, "grpc") || strings.Contains(lower, "protocol buffers") {
		return []string{"grpc", "protobuf"}
	}
	if strings.Contains(lower, "terraform") {
		return []string{"terraform"}
	}
	if strings.Contains(lower, "pod disruption") || strings.Contains(lower, "liveness vs readiness") || strings.Contains(lower, "ingress") {
		return []string{"kubernetes"}
	}
	return []string{}
}

// BuildFullRuleCatalog builds the complete versioned dictionary of 219 rules:
// - RAE-SEC-001 .. RAE-SEC-198 (198 engineering & security catalog topics)
// - RAE-SEC-199 .. RAE-SEC-210 (12 AI & Multi-Tenant SaaS extensions)
// - RAE-EDT-001 .. RAE-EDT-009 (9 RAE-EDITOR-SECURITY rules)
func BuildFullRuleCatalog() map[string]QualityRule {
	catalog := make(map[string]QualityRule, 219)

	p0Set := map[int]bool{
		1: true, 2: true, 5: true, 6: true, 7: true, 8: true,
		13: true, 14: true, 17: true, 18: true, 20: true, 33: true,
		38: true, 191: true, 195: true, 198: true,
	}
	negTestSet := map[int]bool{
		1: true, 2: true, 5: true, 6: true, 7: true, 8: true,
		13: true, 14: true, 33: true, 48: true, 90: true,
	}

	for idx, topicName := range All198Topics {
		topicNum := idx + 1
		ruleID := fmt.Sprintf("RAE-SEC-%03d", topicNum)
		mod := resolveModuleForTopic(topicNum)
		cType := determineControlType(topicNum)
		techs := determineTechnologies(topicName)

		if topicNum == 6 {
			catalog[ruleID] = QualityRule{
				ID:          "RAE-SEC-006",
				Version:     "1.0.0",
				TopicNumber: 6,
				Name:        "Cross-tenant object access (IDOR)",
				Category:    "authorization",
				Module:      mod.Name,
				Priority:    PriorityP0,
				Severity:    SeverityCritical,
				Enforcement: EnforcementBlock,
				ControlType: ControlAISemantic,
				Stage:       StagePreMerge,
				AppliesTo: RuleAppliesTo{
					Languages:    []string{"go", "php", "typescript", "javascript", "python"},
					Components:   []string{"api", "backend"},
					Technologies: []string{},
				},
				Triggers: []string{
					"resource_access_changed",
					"authorization_logic_changed",
					"tenant_context_changed",
				},
				Verification: RuleVerification{
					StaticAnalysis:  true,
					AIReview:        true,
					IntegrationTest: true,
					NegativeTest:    true,
				},
				Requirements: []string{
					"Server must resolve authenticated identity",
					"Resource must belong to authorized tenant",
					"Client-provided tenant ID is untrusted",
					"Access must be denied across tenants",
				},
				EvidenceRequired: []string{
					"source_location",
					"affected_endpoint",
					"authorization_path",
					"test_result",
				},
				Autofix: AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
				Scanner: "semgrep+rae-security-gateway+playwright",
			}
			continue
		}

		isP0 := mod.Priority == PriorityP0 || p0Set[topicNum]
		sev := SeverityMedium
		if isP0 {
			sev = SeverityCritical
		} else if strings.Contains(string(mod.Priority), "P1") {
			sev = SeverityHigh
		}
		enf := EnforcementRepair
		if isP0 {
			enf = EnforcementBlock
		}
		stg := StagePreMerge
		if cType == ControlSystemic {
			stg = StagePreProdPostDeploy
		}
		evReq := []string{"scanner_output"}
		if isP0 {
			evReq = []string{"source_location", "test_result"}
		}

		catalog[ruleID] = QualityRule{
			ID:          ruleID,
			Version:     "1.0.0",
			TopicNumber: topicNum,
			Name:        topicName,
			Category:    mod.ID,
			Module:      mod.Name,
			Priority:    mod.Priority,
			Severity:    sev,
			Enforcement: enf,
			ControlType: cType,
			Stage:       stg,
			AppliesTo: RuleAppliesTo{
				Languages:    []string{"go", "python", "php", "typescript", "javascript"},
				Components:   []string{"api", "backend", "frontend", "editor", "infra"},
				Technologies: techs,
			},
			Triggers: []string{fmt.Sprintf("%s_changed", mod.ID)},
			Verification: RuleVerification{
				StaticAnalysis:  cType == ControlDeterministic,
				AIReview:        cType == ControlAISemantic,
				IntegrationTest: isP0,
				NegativeTest:    negTestSet[topicNum],
				SystemicCheck:   cType == ControlSystemic,
			},
			Requirements:     []string{fmt.Sprintf("Enforce %s compliance in %s", topicName, mod.Name)},
			EvidenceRequired: evReq,
			Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
			Scanner:          mod.DefaultScanner,
		}
	}

	// Add 12 AI & SaaS Topics (199..210)
	for _, aiTopic := range AISaaSTopicDetails {
		ruleID := fmt.Sprintf("RAE-SEC-%03d", aiTopic.Number)
		cType := determineControlType(aiTopic.Number)
		catalog[ruleID] = QualityRule{
			ID:          ruleID,
			Version:     "1.0.0",
			TopicNumber: aiTopic.Number,
			Name:        aiTopic.Name,
			Category:    "agentic_ai_security",
			Module:      "Agentic AI & SaaS Governance",
			Priority:    PriorityP0,
			Severity:    SeverityCritical,
			Enforcement: EnforcementBlock,
			ControlType: cType,
			Stage:       StagePreMerge,
			AppliesTo: RuleAppliesTo{
				Languages:    []string{"go", "python", "php", "typescript", "javascript"},
				Components:   []string{"api", "backend", "editor", "agent", "infra"},
				Technologies: []string{},
			},
			Triggers: []string{"agent_code_change", "security_policy_touch", "tenant_boundary_touch"},
			Verification: RuleVerification{
				StaticAnalysis:  true,
				AIReview:        true,
				IntegrationTest: true,
				NegativeTest:    true,
			},
			Requirements:     []string{aiTopic.Description},
			EvidenceRequired: []string{"source_location", "test_result", "scanner_output"},
			Autofix:          AutofixSpec{Enabled: aiTopic.Number != 208, RequireRevalidation: true, MaxAttempts: 3},
			Scanner:          "rae-security-gateway",
		}
	}

	// Add 9 RAE-EDITOR-SECURITY rules
	for _, edt := range RAEEditorSecurityRules {
		catalog[edt.ID] = edt
	}

	return catalog
}

// OBS001ObservabilityRule returns the canonical OBS-001 (Observability & Trace Propagation Gate) quality rule definition.
func OBS001ObservabilityRule() QualityRule {
	return QualityRule{
		ID:          "OBS-001",
		Version:     "1.0.0",
		TopicNumber: 138,
		Name:        "Observability & Trace Propagation Gate (W3C traceparent, Canonical slog JSON, Zero PII Leak)",
		Category:    "observability_monitoring",
		Module:      "Observability & Monitoring",
		Priority:    PriorityP0,
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Stage:       StagePreMerge,
		AppliesTo: RuleAppliesTo{
			Languages:    []string{"go"},
			Components:   []string{"api", "backend", "frontend", "editor", "manager", "rae-suite"},
			Technologies: []string{"opentelemetry", "slog"},
		},
		Triggers: []string{"observability_monitoring_changed", "http_middleware_changed"},
		Verification: RuleVerification{
			StaticAnalysis:  true,
			AIReview:        true,
			IntegrationTest: true,
			NegativeTest:    true,
		},
		Requirements: []string{
			"Zero fmt.Println or log.Printf in HTTP handlers/services outside CLI tools",
			"Outgoing HTTP requests must propagate W3C traceparent header",
			"All logs must be single-line canonical JSON with PII redaction",
		},
		EvidenceRequired: []string{"source_location", "test_result", "trace_waterfall"},
		Autofix:          AutofixSpec{Enabled: true, RequireRevalidation: true, MaxAttempts: 3},
		Scanner:          "rae-quality-obs001-gate",
	}
}

// VerifyOBS001ObservabilityGate evaluates Go source snippets, HTTP headers, and emitted log lines against OBS-001.
func VerifyOBS001ObservabilityGate(goCode string, isCLI bool, traceparentHeader string, sampleLogJSON string) (bool, []string) {
	var violations []string
	if !isCLI {
		if strings.Contains(goCode, "fmt.Println(") || strings.Contains(goCode, "log.Printf(") {
			violations = append(violations, "OBS-001: forbidden fmt.Println/log.Printf in non-CLI service code")
		}
	}
	if !strings.HasPrefix(strings.TrimSpace(traceparentHeader), "00-") || len(strings.TrimSpace(traceparentHeader)) != 55 {
		violations = append(violations, "OBS-001: missing or invalid W3C traceparent header")
	}
	if strings.Contains(sampleLogJSON, "Bearer eyJ") || strings.Contains(sampleLogJSON, "password\":\"secret") {
		violations = append(violations, "OBS-001: unmasked PII or secret token detected in log output")
	}
	return len(violations) == 0, violations
}
