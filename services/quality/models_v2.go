package quality

import "time"

// GatewayVerdict represents the 4-stage Quality Gateway decision.
type GatewayVerdict string

const (
	VerdictPass   GatewayVerdict = "PASS"
	VerdictRepair GatewayVerdict = "REPAIR"
	VerdictBlock  GatewayVerdict = "BLOCK"
	VerdictReview GatewayVerdict = "REVIEW"
)

// ControlType represents the 3 control categories from Section 3 of poziom-bezpieczeństwa-kodu.md.
type ControlType string

const (
	ControlDeterministic ControlType = "deterministic"
	ControlAISemantic    ControlType = "ai_semantic"
	ControlSystemic      ControlType = "systemic"
)

// ControlStage represents the 4 lifecycle stages from Section 1 of poziom-bezpieczeństwa-kodu.md.
type ControlStage string

const (
	StageBeforeCoding      ControlStage = "before_coding"
	StageDuringWork        ControlStage = "during_work"
	StagePreMerge          ControlStage = "pre_merge"
	StagePreProdPostDeploy ControlStage = "pre_prod_post_deploy"
)

// RuleEnforcement represents the action taken when a rule is violated.
type RuleEnforcement string

const (
	EnforcementBlock  RuleEnforcement = "block"
	EnforcementRepair RuleEnforcement = "repair"
	EnforcementReview RuleEnforcement = "review"
	EnforcementWarn   RuleEnforcement = "warn"
)

// RuleSeverity represents the severity level of a rule or finding.
type RuleSeverity string

const (
	SeverityCritical RuleSeverity = "critical"
	SeverityHigh     RuleSeverity = "high"
	SeverityMedium   RuleSeverity = "medium"
	SeverityLow      RuleSeverity = "low"
)

// ModulePriority represents the rollout priority from Section 2.
type ModulePriority string

const (
	PriorityP0   ModulePriority = "P0"
	PriorityP0P1 ModulePriority = "P0/P1"
	PriorityP1   ModulePriority = "P1"
	PriorityP1P2 ModulePriority = "P1/P2"
	PriorityP2   ModulePriority = "P2"
)

// RuleAppliesTo defines the language, component, and technology scope of a QualityRule.
type RuleAppliesTo struct {
	Languages    []string `json:"languages"`
	Components   []string `json:"components"`
	Technologies []string `json:"technologies"`
}

// RuleVerification specifies which verification mechanisms a rule requires.
type RuleVerification struct {
	StaticAnalysis  bool `json:"static_analysis"`
	AIReview        bool `json:"ai_review"`
	IntegrationTest bool `json:"integration_test"`
	NegativeTest    bool `json:"negative_test"`
	SystemicCheck   bool `json:"systemic_check"`
}

// AutofixSpec configures automated agent repair behavior for a rule.
type AutofixSpec struct {
	Enabled             bool `json:"enabled"`
	RequireRevalidation bool `json:"require_revalidation"`
	MaxAttempts         int  `json:"max_attempts"`
}

// QualityRule represents one of the 219 versioned rules in RAE-Quality 2.0.
type QualityRule struct {
	ID               string           `json:"id"`
	Version          string           `json:"version"`
	TopicNumber      int              `json:"topic_number,omitempty"`
	Name             string           `json:"name"`
	Category         string           `json:"category"`
	Module           string           `json:"module"`
	Priority         ModulePriority   `json:"priority"`
	Severity         RuleSeverity     `json:"severity"`
	Enforcement      RuleEnforcement  `json:"enforcement"`
	ControlType      ControlType      `json:"control_type"`
	Stage            ControlStage     `json:"stage"`
	AppliesTo        RuleAppliesTo    `json:"applies_to"`
	Triggers         []string         `json:"triggers"`
	Verification     RuleVerification `json:"verification"`
	Requirements     []string         `json:"requirements"`
	EvidenceRequired []string         `json:"evidence_required"`
	Autofix          AutofixSpec      `json:"autofix"`
	Scanner          string           `json:"scanner"`
}

// EvidenceItem represents verifiable proof for a rule check (Section 5).
type EvidenceItem struct {
	RuleID            string      `json:"rule_id"`
	ControlType       ControlType `json:"control_type,omitempty"`
	SourceLocation    string      `json:"source_location,omitempty"`
	AffectedEndpoint  string      `json:"affected_endpoint,omitempty"`
	AuthorizationPath string      `json:"authorization_path,omitempty"`
	TestResult        string      `json:"test_result,omitempty"`
	ScannerOutput     string      `json:"scanner_output,omitempty"`
	DataFlowTrace     string      `json:"data_flow_trace,omitempty"`
	ApprovedReviewID  string      `json:"approved_review_id,omitempty"`
	VerifiedByTool    bool        `json:"verified_by_tool"`
	AgentStatement    string      `json:"agent_statement,omitempty"`
}

// SecurityFinding represents a detected violation from SAST, Architecture, Infra, or AI Semantic review.
type SecurityFinding struct {
	ID          string                 `json:"id"`
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Module      string                 `json:"module"`
	Severity    RuleSeverity           `json:"severity"`
	Enforcement RuleEnforcement        `json:"enforcement"`
	ControlType ControlType            `json:"control_type"`
	Scanner     string                 `json:"scanner"`
	Message     string                 `json:"message"`
	FilePath    string                 `json:"file_path,omitempty"`
	LineNumber  int                    `json:"line_number,omitempty"`
	Remediation string                 `json:"remediation"`
	Evidence    map[string]interface{} `json:"evidence,omitempty"`
}

// SARIFResult represents an entry in a SARIF 2.1.0 run.
type SARIFResult struct {
	RuleID     string                   `json:"ruleId"`
	Level      string                   `json:"level"`
	Message    map[string]string        `json:"message"`
	Locations  []map[string]interface{} `json:"locations"`
	Properties map[string]interface{}   `json:"properties,omitempty"`
}

// SARIFRun represents a single tool execution run in SARIF 2.1.0.
type SARIFRun struct {
	Tool        map[string]interface{}   `json:"tool"`
	Results     []SARIFResult            `json:"results"`
	Invocations []map[string]interface{} `json:"invocations"`
}

// SARIFReport represents a normalized SARIF 2.1.0 report with RAE metadata.
type SARIFReport struct {
	Version     string                 `json:"version"`
	SchemaURI   string                 `json:"$schema"`
	Runs        []SARIFRun             `json:"runs"`
	RAEMetadata map[string]interface{} `json:"rae_metadata"`
}

// MandatoryGateCheck represents an unconditional release gate check (Section 8).
type MandatoryGateCheck struct {
	GateID   string         `json:"gate_id"`
	Name     string         `json:"name"`
	Verdict  GatewayVerdict `json:"verdict"`
	Blocking bool           `json:"blocking"`
	Details  string         `json:"details"`
	RuleIDs  []string       `json:"rule_ids"`
}

// PRQualityAssessment represents the Dual Risk Assessment output from quality.release_gate (Section 8).
type PRQualityAssessment struct {
	AssessmentID    string               `json:"assessment_id"`
	Timestamp       time.Time            `json:"timestamp"`
	Verdict         GatewayVerdict       `json:"verdict"`
	MergeAllowed    bool                 `json:"merge_allowed"`
	QualityScore    int                  `json:"quality_score"`
	TestsPassed     int                  `json:"tests_passed"`
	TestsTotal      int                  `json:"tests_total"`
	TestsFailed     int                  `json:"tests_failed"`
	MandatoryGates  []MandatoryGateCheck `json:"mandatory_gates"`
	DecisionSummary string               `json:"decision_summary"`
	Findings        []SecurityFinding    `json:"findings"`
	AuditSHA256     string               `json:"audit_sha256"`
}

// AutoRepairTask represents a surgical repair task scheduled for an AI agent (max 3 attempts -> HITL).
type AutoRepairTask struct {
	RepairID                 string         `json:"repair_id"`
	ChangeID                 string         `json:"change_id"`
	AgentID                  string         `json:"agent_id"`
	Attempt                  int            `json:"attempt"`
	MaxAttempts              int            `json:"max_attempts"`
	Verdict                  GatewayVerdict `json:"verdict"`
	EscalatedToHuman         bool           `json:"escalated_to_human"`
	FailedRules              []string       `json:"failed_rules"`
	MinimalFixInstructions   []string       `json:"minimal_fix_instructions"`
	IndependentTestRequired  bool           `json:"independent_test_required"`
	Timestamp                time.Time      `json:"timestamp"`
}

// AgentQualityScorecard tracks empirical reliability metrics of an AI coding agent (Section 8).
type AgentQualityScorecard struct {
	AgentID                   string  `json:"agent_id"`
	TotalChanges              int     `json:"total_changes"`
	TotalDefects              int     `json:"total_defects"`
	DefectsPerChange          float64 `json:"defects_per_change"`
	FirstTimeFixCount         int     `json:"first_time_fix_count"`
	TotalFixAttempts          int     `json:"total_fix_attempts"`
	FirstTimeFixRatePct       float64 `json:"first_time_fix_rate_pct"`
	RegressionsIntroduced     int     `json:"regressions_introduced"`
	PostDeployVulnerabilities int     `json:"post_deploy_vulnerabilities"`
	TamperAttemptsBlocked     int     `json:"tamper_attempts_blocked"`
	ReliabilityTier           string  `json:"reliability_tier"`
}
