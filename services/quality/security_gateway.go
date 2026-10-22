package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ScannerInfo describes an external scanner integrated into RAE-Quality 2.0 (Section 7).
type ScannerInfo struct {
	Purpose              string   `json:"purpose"`
	SupportedLanguages   []string `json:"supported_languages"`
	UnsupportedLanguages []string `json:"unsupported_languages,omitempty"`
	FallbackForPHP       string   `json:"fallback_for_php,omitempty"`
	OutputFormat         string   `json:"output_format"`
}

// ScannerRegistry holds the 10 integrated scanners from Section 7 of poziom-bezpieczeństwa-kodu.md.
var ScannerRegistry = map[string]ScannerInfo{
	"semgrep": {
		Purpose:            "SAST, custom PHP/JS/TS/Python/Go rules, dangerous patterns",
		SupportedLanguages: []string{"php", "javascript", "typescript", "python", "go"},
		OutputFormat:       "SARIF",
	},
	"codeql": {
		Purpose:              "Data flow analysis, taint tracking, inter-procedural vulnerabilities",
		SupportedLanguages:   []string{"javascript", "typescript", "python", "go", "java", "cpp", "csharp"},
		UnsupportedLanguages: []string{"php"},
		FallbackForPHP:       "semgrep+sonarqube",
		OutputFormat:         "SARIF",
	},
	"gitleaks": {
		Purpose:            "Secret detection in source code and Git history",
		SupportedLanguages: []string{"all"},
		OutputFormat:       "SARIF",
	},
	"trivy": {
		Purpose:            "SCA dependency vulnerabilities, container images, and IaC misconfigurations",
		SupportedLanguages: []string{"all", "docker", "kubernetes", "terraform"},
		OutputFormat:       "SARIF",
	},
	"owasp_zap": {
		Purpose:            "Dynamic Application Security Testing (DAST)",
		SupportedLanguages: []string{"http_api", "web"},
		OutputFormat:       "SARIF",
	},
	"sonarqube": {
		Purpose:            "Code quality, maintainability, security hotspots, duplications",
		SupportedLanguages: []string{"php", "javascript", "typescript", "python", "go"},
		OutputFormat:       "SARIF",
	},
	"schemathesis": {
		Purpose:            "Automated property-based API testing from OpenAPI specs",
		SupportedLanguages: []string{"openapi", "rest_api"},
		OutputFormat:       "JSON/SARIF",
	},
	"playwright": {
		Purpose:            "E2E tests, permissions, sessions, multi-tenant isolation",
		SupportedLanguages: []string{"typescript", "javascript", "web"},
		OutputFormat:       "JSON",
	},
	"k6": {
		Purpose:            "Load testing, performance SLA, and concurrency/race condition verification",
		SupportedLanguages: []string{"http_api", "websocket"},
		OutputFormat:       "JSON",
	},
	"checkov": {
		Purpose:            "Infrastructure as Code (IaC), Kubernetes, Dockerfile, and Terraform security",
		SupportedLanguages: []string{"dockerfile", "kubernetes", "helm", "terraform"},
		OutputFormat:       "SARIF",
	},
}

// SecurityPattern defines an executable SAST/Semantic detection pattern in SecurityGateway.
type SecurityPattern struct {
	RuleID      string
	RuleName    string
	Module      string
	Severity    RuleSeverity
	Enforcement RuleEnforcement
	ControlType ControlType
	Scanner     string
	Regex       *regexp.Regexp
	Message     string
	Remediation string
}

// SecurityPatterns contains all 24 canonical security rules (OWASP, Identity/Secrets, AI/SaaS, and 9/9 RAE-EDITOR-SECURITY)
// supporting Python, PHP, TypeScript/JavaScript, and Go (`backend-go`).
var SecurityPatterns = []SecurityPattern{
	// --- OWASP & Core Security ---
	{
		RuleID:      "RAE-SEC-001",
		RuleName:    "SQL Injection",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:SELECT|INSERT|UPDATE|DELETE)\s+.*?(?:\+\s*[a-zA-Z_$]|\{[a-zA-Z0-9_.\[\]'"]+\}|\.\s*\$_(?:GET|POST|REQUEST)|%[svq]\b)`,
		),
		Message:     "Unsanitized variable interpolated, formatted (fmt.Sprintf), or concatenated directly into SQL query.",
		Remediation: "Use parameterized queries / prepared statements ($1, $2 or ?) with bound parameters.",
	},
	{
		RuleID:      "RAE-SEC-002",
		RuleName:    "Cross-Site Scripting (XSS)",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:dangerouslySetInnerHTML|\.innerHTML\s*=|echo\s+\$_(?:GET|POST|REQUEST)|document\.write\s*\(|template\.HTML\s*\()`,
		),
		Message:     "Unescaped HTML/JavaScript sink detected (XSS risk in project/resource rendering).",
		Remediation: "Sanitize HTML with DOMPurify or use framework-escaped text bindings.",
	},
	{
		RuleID:      "RAE-SEC-003",
		RuleName:    "CSRF Protection",
		Module:      "Security & OWASP",
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:@csrf_exempt|verify_csrf\s*=\s*False|csrfProtection\s*:\s*false|DisableCSRF\s*:\s*true)`,
		),
		Message:     "CSRF protection explicitly disabled on state-changing endpoint.",
		Remediation: "Enable CSRF token verification or SameSite=Strict/Lax session cookies on state-changing routes.",
	},
	{
		RuleID:      "RAE-SEC-004",
		RuleName:    "CORS Misconfiguration",
		Module:      "Security & OWASP",
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?is)(?:Access-Control-Allow-Origin['"]?\s*[:=]\s*['"]\*['"].*?Access-Control-Allow-Credentials['"]?\s*[:=]\s*['"]?true|allow_origins\s*=\s*\[\s*['"]\*['"]\s*\].*?allow_credentials\s*=\s*True|AllowOrigins\s*:\s*\[\]string\{\s*"\*"\s*\}.*?AllowCredentials\s*:\s*true)`,
		),
		Message:     "Insecure CORS configuration combining wildcard origin ('*') with credentials=True.",
		Remediation: "Specify an explicit allowlist of trusted tenant domains when allow_credentials is enabled.",
	},
	{
		RuleID:      "RAE-SEC-005",
		RuleName:    "Server-Side Request Forgery (SSRF)",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:169\.254\.169\.254|(?:requests\.(?:get|post)|httpx\.(?:get|post)|file_get_contents|curl_exec|http\.(?:Get|Post))\s*\(\s*(?:req\.|request\.|\$_(?:GET|POST)|user_url|import_url|rawURL|targetURL))`,
		),
		Message:     "Potential SSRF: Fetching user-controlled or internal network URL without allowlist validation.",
		Remediation: "Validate URL scheme/host against strict allowlist and block RFC1918/link-local IPs.",
	},
	{
		RuleID:      "RAE-SEC-006",
		RuleName:    "Cross-tenant object access (IDOR)",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:req\.(?:body|query|params)\.(?:tenantId|tenant_id)|\$_(?:GET|POST|REQUEST)\[['"](?:tenant_id|tenantId)['"]\]|payload\.get\(['"](?:tenant_id|tenantId)['"]\)|c\.Query\(['"](?:tenant_id|tenantId)['"]\)|r\.URL\.Query\(\)\.Get\(['"](?:tenant_id|tenantId)['"]\))`,
		),
		Message:     "Cross-tenant IDOR risk: Client-provided tenantId is trusted instead of server-authenticated tenant context.",
		Remediation: "Resolve tenant_id strictly from authenticated server session/JWT claims and enforce tenant ownership on resource lookup.",
	},
	{
		RuleID:      "RAE-SEC-007",
		RuleName:    "Path Traversal",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:\.\./|\.\.\\|open\s*\(\s*f['"].*?\{.*?(?:filename|path|file)\}|os\.ReadFile\s*\(\s*(?:req\.|c\.Query|r\.URL\.Query))`,
		),
		Message:     "Path traversal pattern detected in file access operation.",
		Remediation: "Normalize path with filepath.Clean / Path.resolve() and verify it starts with the allowed base directory.",
	},
	{
		RuleID:      "RAE-SEC-008",
		RuleName:    "Mass Assignment",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:::create\s*\(\s*\$request->all\(\)|Object\.assign\s*\(\s*\w+\s*,\s*req\.body\)|(?:\*\*request\.json\(\)|\*\*payload).*?(?:isAdmin|is_admin|role|price)|db\.Model\(&user\)\.Updates\(req\.Body\))`,
		),
		Message:     "Mass Assignment vulnerability: Unfiltered client payload bound to model/entity (risk of overwriting role, price, or isAdmin).",
		Remediation: "Use explicit DTO allowlists excluding privileged fields (role, isAdmin, price, tenant_id).",
	},
	{
		RuleID:      "RAE-SEC-010",
		RuleName:    "Content Security Policy (CSP)",
		Module:      "Security & OWASP",
		Severity:    SeverityHigh,
		Enforcement: EnforcementRepair,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)Content-Security-Policy.*?unsafe-eval`,
		),
		Message:     "CSP header allows 'unsafe-eval', weakening XSS mitigation in editor/browser context.",
		Remediation: "Remove 'unsafe-eval' and use nonce-based or strict hash-based Content-Security-Policy.",
	},
	// --- Identity & Secrets ---
	{
		RuleID:      "RAE-SEC-020",
		RuleName:    "Secret Scanning",
		Module:      "Identity & Secrets",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "gitleaks",
		Regex: regexp.MustCompile(
			`(?:sk-[a-zA-Z0-9]{20,}|ghp_[a-zA-Z0-9]{25,}|AKIA[0-9A-Z]{16}|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----|(?:api_key|secret_key|password|jwt_secret)\s*[:=]\s*['"][a-zA-Z0-9_\-!@#$%^&*]{12,}['"])`,
		),
		Message:     "Hardcoded secret, private key, or API token detected in source code.",
		Remediation: "Move secrets to environment variables / secret manager and rotate compromised credentials.",
	},
	{
		RuleID:      "RAE-SEC-023",
		RuleName:    "Password Hashing (Argon2id/bcrypt)",
		Module:      "Identity & Secrets",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:hashlib\.(?:md5|sha1)\s*\(\s*\w*pass|md5\s*\(\s*\$\w*pass|sha1\s*\(\s*\$\w*pass|(?:md5|sha1)\.Sum\s*\(\s*\[\]byte\(\w*pass)`,
		),
		Message:     "Weak password hashing algorithm (MD5/SHA1) detected.",
		Remediation: "Hash passwords exclusively with Argon2id or bcrypt with appropriate work factor.",
	},
	{
		RuleID:      "RAE-SEC-024",
		RuleName:    "JWT Security",
		Module:      "Identity & Secrets",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:algorithms\s*=\s*\[\s*['"]none['"]\s*\]|verify_signature['"]?\s*:\s*False|verify_exp['"]?\s*:\s*False|jwt\.WithoutClaimsValidation\s*\(\)|jwt\.SigningMethodNone)`,
		),
		Message:     "Insecure JWT validation (alg='none' or signature/expiration verification disabled).",
		Remediation: "Enforce strong asymmetric/HMAC algorithms and require both signature and exp/aud verification.",
	},
	{
		RuleID:      "RAE-SEC-032",
		RuleName:    "Timing Attack Protection",
		Module:      "Security & OWASP",
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:(?:hmac_signature|webhook_signature|api_token|secret_token|signature)\s*==\s*(?:expected_signature|provided_signature|req_token|expectedMAC))`,
		),
		Message:     "Non-constant-time string comparison (`==`) used for secret token or HMAC signature verification.",
		Remediation: "Use constant-time comparison (`hmac.compare_digest`, `hash_equals`, `hmac.Equal`, or `subtle.ConstantTimeCompare`).",
	},
	{
		RuleID:      "RAE-SEC-033",
		RuleName:    "File Upload Validation",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?is)(?:<svg[^>]*>.*?<script|onload\s*=|\.save\s*\(\s*upload\.filename\s*\)|SaveUploadedFile\s*\(\s*file\s*,\s*file\.Filename\s*\))`,
		),
		Message:     "Unsafe file upload handling or active script inside uploaded SVG/document.",
		Remediation: "Validate MIME & magic bytes, sanitize SVG XML, strip dangerous scripts, and generate random storage filenames.",
	},
	{
		RuleID:      "RAE-SEC-034",
		RuleName:    "Webhook Verification",
		Module:      "Security & OWASP",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:verify_webhook_signature\s*=\s*False|skip_webhook_hmac\s*=\s*True|SkipWebhookVerification\s*:\s*true)`,
		),
		Message:     "Webhook endpoint processes external payment/printshop events with signature verification disabled.",
		Remediation: "Verify webhook HMAC signature and timestamp replay window before processing payload.",
	},
	// --- Agentic AI & SaaS Governance ---
	{
		RuleID:      "RAE-SEC-199",
		RuleName:    "AI Prompt Injection",
		Module:      "Agentic AI & SaaS Governance",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:ignore\s+(?:all\s+)?previous\s+instructions|system\s+override\s*:|you\s+are\s+now\s+in\s+developer\s+mode)`,
		),
		Message:     "AI Prompt Injection payload detected in code, document, or untrusted input.",
		Remediation: "Isolate untrusted content in delimited user blocks and strip instruction-override tokens.",
	},
	{
		RuleID:      "RAE-SEC-207",
		RuleName:    "Safe Deserialization",
		Module:      "Agentic AI & SaaS Governance",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "semgrep",
		Regex: regexp.MustCompile(
			`(?i)(?:pickle\.loads\s*\(|yaml\.load\s*\([^,)]+\)|unserialize\s*\(\s*\$_(?:GET|POST|REQUEST|COOKIE))`,
		),
		Message:     "Unsafe deserialization of untrusted data detected.",
		Remediation: "Use safe parsers (json.loads, yaml.safe_load, json.Unmarshal) and never unserialize untrusted objects.",
	},
	// --- Complete 9/9 RAE-EDITOR-SECURITY Pack (RAE-EDT-001 .. RAE-EDT-009) ---
	{
		RuleID:      "RAE-EDT-001",
		RuleName:    "Project, Page, Asset, and Order Authorization",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:Project\.get_by_id_unscoped\s*\(|Order\.find_without_tenant\s*\(|skip_owner_check\s*=\s*True|GetProjectUnscoped\s*\(|FindOrderWithoutTenant\s*\()`,
		),
		Message:     "Direct project/page/asset/order lookup bypassing owner and tenant authorization.",
		Remediation: "Enforce server-side owner_id and tenant_id verification on every project, page, asset, and order operation.",
	},
	{
		RuleID:      "RAE-EDT-002",
		RuleName:    "Multi-Printshop Data Isolation (Cache, WebSockets, Queues, Files)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:redis\.(?:get|set)\s*\(\s*f?['"]project:\{?(?:project_id|id)\}?['"]|socket\.join\s*\(\s*f?['"]project_room_|rdb\.(?:Get|Set)\s*\([^,]+,\s*fmt\.Sprintf\s*\(\s*"project:%[sd]")`,
		),
		Message:     "Missing tenant_id namespace prefix in Redis cache key or WebSocket room name.",
		Remediation: "Prefix all cache keys and WebSocket rooms with tenant_id (e.g. `tenant:{tenant_id}:project:{project_id}`).",
	},
	{
		RuleID:      "RAE-EDT-003",
		RuleName:    "Role & Permission Separation (Customer, Collaborator, Printshop Operator/Admin, Platform Admin)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlAISemantic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:allow_cross_printshop_admin\s*=\s*True|role\s*==\s*['"]printshop_admin['"]\s*(?:and|&&)\s*skip_tenant_filter)`,
		),
		Message:     "Printshop admin granted cross-printshop access or unlogged administrative override.",
		Remediation: "Restrict printshop_admin strictly to their own printshop tenant_id and log all platform_admin access.",
	},
	{
		RuleID:      "RAE-EDT-004",
		RuleName:    "Collaborative Editing & Autosave Protection (Optimistic Locking / Versioning)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityHigh,
		Enforcement: EnforcementRepair,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:autosave_overwrite_without_version\s*=\s*True|save_canvas_state\s*\([^)]*ignore_version_conflict\s*=\s*True|IgnoreVersionConflict\s*:\s*true)`,
		),
		Message:     "Collaborative project autosave overwrites scene state without version/optimistic lock verification.",
		Remediation: "Verify base_version / ETag on every autosave and return HTTP 409 Conflict on concurrent modification.",
	},
	{
		RuleID:      "RAE-EDT-005",
		RuleName:    "Production Parameter Server-Side Validation (Format, Bleed, Pages, Price)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:order\.total_price\s*=\s*(?:req\.body\.price|payload\[['"]price['"]\]|\$_POST\[['"]price['"]\])|trust_client_price\s*=\s*True|order\.TotalPrice\s*=\s*req\.Price)`,
		),
		Message:     "Business Logic Violation: Order accepts client-submitted price without server-side calculation.",
		Remediation: "Always recalculate total price and validate format/bleed/pages via server-side CalculationEngine.",
	},
	{
		RuleID:      "RAE-EDT-006",
		RuleName:    "File Upload, Asset Import & EXIF Metadata Sanitization",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:preserve_exif_gps\s*=\s*True|strip_exif\s*=\s*False|allow_embedded_pdf_js\s*=\s*True|StripEXIF\s*:\s*false)`,
		),
		Message:     "Uploaded image/PDF asset retains sensitive EXIF GPS metadata or embedded PDF JavaScript.",
		Remediation: "Strip EXIF GPS/camera metadata on image ingest and reject/sanitize embedded scripts in SVG/PDF files.",
	},
	{
		RuleID:      "RAE-EDT-007",
		RuleName:    "Share Links & Project Previews Security (Token Scope, Expiry, Revocation)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:share_token\s*=\s*(?:str\(project_id\)|md5\(project_id\))|share_link_never_expires\s*=\s*True|ShareToken\s*=\s*fmt\.Sprintf\("%d",\s*projectID\))`,
		),
		Message:     "Predictable or non-expiring project share/preview link token detected.",
		Remediation: "Generate cryptographically random share tokens (crypto/rand or secrets.token_urlsafe) with explicit scope, TTL, and revocation support.",
	},
	{
		RuleID:      "RAE-EDT-008",
		RuleName:    "Render & Preflight Worker Isolation (Sandboxing, Resource Limits, Timeouts)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityCritical,
		Enforcement: EnforcementBlock,
		ControlType: ControlDeterministic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:render_worker_privileged\s*=\s*True|pdf_preflight_timeout\s*=\s*None|subprocess\.run\s*\([^)]*shell\s*=\s*True[^)]*ghostscript|exec\.Command\(\s*"(?:sh|bash)"\s*,\s*"-c")`,
		),
		Message:     "PDF/Print render worker or command executes without sandbox isolation, timeout, or safe argument array.",
		Remediation: "Run workers in unprivileged sandboxes with strict CPU/RAM quotas, network isolation, execution timeouts, and without shell=True / sh -c.",
	},
	{
		RuleID:      "RAE-EDT-009",
		RuleName:    "Client Data Privacy, Retention & Right to Erasure (RODO/GDPR)",
		Module:      "RAE-EDITOR-SECURITY",
		Severity:    SeverityHigh,
		Enforcement: EnforcementBlock,
		ControlType: ControlSystemic,
		Scanner:     "rae-security-gateway",
		Regex: regexp.MustCompile(
			`(?i)(?:skip_storage_cleanup_on_gdpr_delete\s*=\s*True|(?:logger|log)\.(?:info|debug|error|Printf|Println)\s*\([^)]*(?:customer_pesel|credit_card_number|raw_address_pii))`,
		),
		Message:     "GDPR/RODO Violation: Incomplete asset erasure on account deletion or raw customer PII written to logs.",
		Remediation: "Purge associated S3/MinIO project assets and thumbnails upon GDPR erasure and mask PII in application logs.",
	},
}

// SecurityGateway implements packages/rae-quality/engines/security/security_gateway.py.
type SecurityGateway struct{}

// NewSecurityGateway constructs a new SecurityGateway instance.
func NewSecurityGateway() *SecurityGateway {
	return &SecurityGateway{}
}

// ComputeFindingFingerprint computes a stable 16-char SHA-256 fingerprint for baseline debt separation (Section 10).
func (sg *SecurityGateway) ComputeFindingFingerprint(ruleID, filePath, snippet string) string {
	raw := fmt.Sprintf("%s:%s:%s", ruleID, filePath, strings.TrimSpace(snippet))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])[:16]
}

// GetCompatibleScanners returns scanners appropriate for the given languages, respecting Section 7 constraints
// (e.g., CodeQL does not natively support PHP and falls back to semgrep+sonarqube).
func (sg *SecurityGateway) GetCompatibleScanners(languages []string) map[string]interface{} {
	if len(languages) == 0 {
		languages = []string{"go", "python", "php", "typescript"}
	}
	normSet := make(map[string]bool, len(languages))
	var normList []string
	for _, l := range languages {
		nl := strings.ToLower(strings.TrimSpace(l))
		if nl != "" && !normSet[nl] {
			normSet[nl] = true
			normList = append(normList, nl)
		}
	}
	sort.Strings(normList)

	selected := make(map[string]ScannerInfo)
	var warnings []string

	scannerIDs := make([]string, 0, len(ScannerRegistry))
	for id := range ScannerRegistry {
		scannerIDs = append(scannerIDs, id)
	}
	sort.Strings(scannerIDs)

	for _, scannerID := range scannerIDs {
		meta := ScannerRegistry[scannerID]
		allUnsupported := len(normList) > 0
		unsupportedMap := make(map[string]bool)
		for _, u := range meta.UnsupportedLanguages {
			unsupportedMap[strings.ToLower(u)] = true
		}
		for _, nl := range normList {
			if !unsupportedMap[nl] {
				allUnsupported = false
				break
			}
		}
		if allUnsupported && len(meta.UnsupportedLanguages) > 0 {
			fb := meta.FallbackForPHP
			if fb == "" {
				fb = "semgrep"
			}
			warnings = append(warnings, fmt.Sprintf(
				"Scanner '%s' skipped for %v; using fallback '%s'.",
				scannerID, normList, fb,
			))
			continue
		}

		supportedMap := make(map[string]bool)
		for _, s := range meta.SupportedLanguages {
			supportedMap[strings.ToLower(s)] = true
		}
		if supportedMap["all"] || len(normList) == 0 {
			selected[scannerID] = meta
			continue
		}
		for _, nl := range normList {
			if supportedMap[nl] {
				selected[scannerID] = meta
				break
			}
		}
	}

	return map[string]interface{}{
		"languages":           normList,
		"selected_scanners":   selected,
		"compatibility_notes": warnings,
	}
}

// ScanCodeDetailed executes all 24 canonical security patterns, separates baseline debt, and returns SARIF 2.1.0.
func (sg *SecurityGateway) ScanCodeDetailed(
	code string,
	filePath string,
	languages []string,
	baselineFingerprints []string,
) map[string]interface{} {
	if filePath == "" {
		filePath = "inline_snippet.go"
	}
	if len(languages) == 0 {
		languages = []string{"go", "python", "php", "typescript"}
	}
	baselineSet := make(map[string]bool, len(baselineFingerprints))
	for _, fp := range baselineFingerprints {
		baselineSet[fp] = true
	}

	var allDetected []SecurityFinding
	lines := strings.Split(code, "\n")
	if len(lines) == 0 {
		lines = []string{code}
	}

	findingSeq := 0
	var nonCommentLines []string
	for _, l := range lines {
		tr := strings.TrimSpace(l)
		if strings.HasPrefix(tr, "//") || strings.HasPrefix(tr, "#") || strings.HasPrefix(tr, "--") || strings.HasPrefix(tr, "/*") || strings.HasPrefix(tr, "*") {
			nonCommentLines = append(nonCommentLines, "")
		} else {
			nonCommentLines = append(nonCommentLines, l)
		}
	}
	nonCommentCode := strings.Join(nonCommentLines, "\n")

	for _, pat := range SecurityPatterns {
		matchedRule := false
		for idx, line := range nonCommentLines {
			if line == "" {
				continue
			}
			if pat.Regex.MatchString(line) {
				matchedRule = true
				findingSeq++
				fp := sg.ComputeFindingFingerprint(pat.RuleID, filePath, line)
				snippet := strings.TrimSpace(line)
				if len(snippet) > 200 {
					snippet = snippet[:200]
				}
				allDetected = append(allDetected, SecurityFinding{
					ID:          fmt.Sprintf("FND-%s-%d", pat.RuleID, findingSeq),
					RuleID:      pat.RuleID,
					RuleName:    pat.RuleName,
					Module:      pat.Module,
					Severity:    pat.Severity,
					Enforcement: pat.Enforcement,
					ControlType: pat.ControlType,
					Scanner:     pat.Scanner,
					Message:     pat.Message,
					FilePath:    filePath,
					LineNumber:  idx + 1,
					Remediation: pat.Remediation,
					Evidence: map[string]interface{}{
						"matched_line":     snippet,
						"fingerprint":      fp,
						"is_baseline_debt": baselineSet[fp],
						"verified_by_tool": true,
					},
				})
			}
		}
		// Check multi-line patterns (e.g. SVG with <script> or multi-line CORS)
		if !matchedRule && pat.Regex.MatchString(nonCommentCode) {
			findingSeq++
			snippet := code
			if len(snippet) > 200 {
				snippet = snippet[:200]
			}
			fp := sg.ComputeFindingFingerprint(pat.RuleID, filePath, snippet)
			allDetected = append(allDetected, SecurityFinding{
				ID:          fmt.Sprintf("FND-%s-%d", pat.RuleID, findingSeq),
				RuleID:      pat.RuleID,
				RuleName:    pat.RuleName,
				Module:      pat.Module,
				Severity:    pat.Severity,
				Enforcement: pat.Enforcement,
				ControlType: pat.ControlType,
				Scanner:     pat.Scanner,
				Message:     pat.Message,
				FilePath:    filePath,
				LineNumber:  1,
				Remediation: pat.Remediation,
				Evidence: map[string]interface{}{
					"matched_snippet":  snippet,
					"fingerprint":      fp,
					"is_baseline_debt": baselineSet[fp],
					"verified_by_tool": true,
				},
			})
		}
	}

	activeFindings := make([]SecurityFinding, 0)
	baselineDebtFindings := make([]SecurityFinding, 0)
	blockingCount := 0

	for _, f := range allDetected {
		isDebt, _ := f.Evidence["is_baseline_debt"].(bool)
		if isDebt {
			baselineDebtFindings = append(baselineDebtFindings, f)
		} else {
			activeFindings = append(activeFindings, f)
			if f.Enforcement == EnforcementBlock {
				blockingCount++
			}
		}
	}

	verdict := VerdictPass
	if blockingCount > 0 {
		verdict = VerdictBlock
	} else if len(activeFindings) > 0 {
		verdict = VerdictRepair
	}

	sarif := sg.ToSARIFReport(activeFindings, verdict)

	return map[string]interface{}{
		"verdict":                string(verdict),
		"findings_count":         len(activeFindings),
		"blocking_count":         blockingCount,
		"baseline_debt_count":    len(baselineDebtFindings),
		"findings":               activeFindings,
		"baseline_debt_findings": baselineDebtFindings,
		"sarif":                  sarif,
		"scanner_matrix":         sg.GetCompatibleScanners(languages),
	}
}

// ScanCode provides a simple slice-returning helper for backward compatibility.
func (sg *SecurityGateway) ScanCode(code, filePath string) []SecurityFinding {
	res := sg.ScanCodeDetailed(code, filePath, nil, nil)
	if f, ok := res["findings"].([]SecurityFinding); ok {
		return f
	}
	return nil
}

// ToSARIFReport normalizes RAE-Quality findings into standard SARIF 2.1.0 format.
func (sg *SecurityGateway) ToSARIFReport(findings []SecurityFinding, verdict GatewayVerdict) SARIFReport {
	results := make([]SARIFResult, 0, len(findings))
	for _, f := range findings {
		level := "warning"
		if f.Severity == SeverityCritical || f.Severity == SeverityHigh {
			level = "error"
		}
		fileURI := f.FilePath
		if fileURI == "" {
			fileURI = "unknown"
		}
		line := f.LineNumber
		if line <= 0 {
			line = 1
		}
		fp, _ := f.Evidence["fingerprint"].(string)
		results = append(results, SARIFResult{
			RuleID: f.RuleID,
			Level:  level,
			Message: map[string]string{
				"text": fmt.Sprintf("[%s] %s | Fix: %s", f.RuleName, f.Message, f.Remediation),
			},
			Locations: []map[string]interface{}{
				{
					"physicalLocation": map[string]interface{}{
						"artifactLocation": map[string]interface{}{"uri": fileURI},
						"region":           map[string]interface{}{"startLine": line},
					},
				},
			},
			Properties: map[string]interface{}{
				"module":       f.Module,
				"enforcement":  string(f.Enforcement),
				"control_type": string(f.ControlType),
				"scanner":      f.Scanner,
				"fingerprint":  fp,
			},
		})
	}

	run := SARIFRun{
		Tool: map[string]interface{}{
			"driver": map[string]interface{}{
				"name":           "RAE-Quality Security Gateway",
				"version":        "2.0.0",
				"informationUri": "https://github.com/vproject111/RAE-Quality",
			},
		},
		Results:     results,
		Invocations: []map[string]interface{}{{"executionSuccessful": true}},
	}

	return SARIFReport{
		Version:   "2.1.0",
		SchemaURI: "https://json.schemastore.org/sarif-2.1.0.json",
		Runs:      []SARIFRun{run},
		RAEMetadata: map[string]interface{}{
			"gateway_verdict": string(verdict),
			"total_findings":  len(findings),
		},
	}
}
