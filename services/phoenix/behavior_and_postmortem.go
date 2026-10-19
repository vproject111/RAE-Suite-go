package phoenix

import (
	"math"
	"regexp"
	"strings"
)

// BehaviorSnapshot represents an observed execution snapshot (legacy baseline or refactored candidate).
type BehaviorSnapshot struct {
	HTTPStatus    int                    `json:"http_status"`
	HTTPHeaders   map[string]string      `json:"http_headers,omitempty"`
	LatencyMs     float64                `json:"latency_ms"`
	CLIExitCode   int                    `json:"cli_exit_code"`
	CLIStdout     string                 `json:"cli_stdout,omitempty"`
	CLIStderr     string                 `json:"cli_stderr,omitempty"`
	DOMSelectors  []string               `json:"dom_selectors,omitempty"`
	LogLines      []string               `json:"log_lines,omitempty"`
	JSONPayload   map[string]interface{} `json:"json_payload,omitempty"`
}

// BehaviorCheckResult6D captures the 6-dimensional behavioral parity check from feniks/core/behavior/comparison_engine.py.
type BehaviorCheckResult6D struct {
	Passed          bool     `json:"passed"`
	HTTPParityOK    bool     `json:"http_parity_ok"`
	CLIParityOK     bool     `json:"cli_parity_ok"`
	DOMParityOK     bool     `json:"dom_parity_ok"`
	LogsParityOK    bool     `json:"logs_parity_ok"`
	JSONPathOK      bool     `json:"json_path_ok"`
	PatternMatchOK  bool     `json:"pattern_match_ok"`
	RiskScore       float64  `json:"risk_score"`
	Violations      []string `json:"violations"`
}

// CompareBehavior6D evaluates behavioral parity across HTTP, CLI, DOM selectors, Logs, JSON DTO keys, and regex patterns.
func CompareBehavior6D(baseline, candidate BehaviorSnapshot, requiredRegex string) BehaviorCheckResult6D {
	var violations []string
	risk := 0.0

	// 1. _check_http
	httpOK := baseline.HTTPStatus == candidate.HTTPStatus
	if !httpOK {
		violations = append(violations, "HTTP status mismatch")
		risk += 0.25
	}

	// 2. _check_cli
	cliOK := baseline.CLIExitCode == candidate.CLIExitCode
	if !cliOK {
		violations = append(violations, "CLI exit_code mismatch")
		risk += 0.20
	}

	// 3. _check_dom
	domOK := true
	candDOM := make(map[string]bool, len(candidate.DOMSelectors))
	for _, s := range candidate.DOMSelectors {
		candDOM[s] = true
	}
	for _, reqSel := range baseline.DOMSelectors {
		if !candDOM[reqSel] {
			domOK = false
			violations = append(violations, "Missing DOM selector: "+reqSel)
			risk += 0.15
		}
	}

	// 4. _check_logs
	logsOK := true
	for _, line := range candidate.LogLines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "panic:") || strings.Contains(lower, "traceback") || strings.Contains(lower, "exception") {
			logsOK = false
			violations = append(violations, "New error in logs: "+line)
			risk += 0.20
		}
	}

	// 5. _check_json_path
	jsonOK := true
	for k := range baseline.JSONPayload {
		if _, exists := candidate.JSONPayload[k]; !exists {
			jsonOK = false
			violations = append(violations, "Missing JSON DTO key: "+k)
			risk += 0.15
		}
	}

	// 6. _check_pattern
	patternOK := true
	if requiredRegex != "" {
		if re, err := regexp.Compile(requiredRegex); err == nil {
			if !re.MatchString(candidate.CLIStdout) {
				patternOK = false
				violations = append(violations, "CLI stdout failed pattern check: "+requiredRegex)
				risk += 0.10
			}
		}
	}

	if risk > 1.0 {
		risk = 1.0
	}
	passed := httpOK && cliOK && domOK && logsOK && jsonOK && patternOK && risk <= 0.20
	return BehaviorCheckResult6D{
		Passed:         passed,
		HTTPParityOK:   httpOK,
		CLIParityOK:    cliOK,
		DOMParityOK:    domOK,
		LogsParityOK:   logsOK,
		JSONPathOK:     jsonOK,
		PatternMatchOK: patternOK,
		RiskScore:      math.Round(risk*10000) / 10000,
		Violations:     violations,
	}
}

// PostMortemReport captures longitudinal failure analysis from feniks/core/reflection/post_mortem.py.
type PostMortemReport struct {
	SessionID              string   `json:"session_id"`
	FailedAttempts         int      `json:"failed_attempts"`
	RootCauseCategory      string   `json:"root_cause_category"`
	EstimatedWastedUSD     float64  `json:"estimated_wasted_usd"`
	BrokenContracts        []string `json:"broken_contracts"`
	ReflectiveLesson       string   `json:"reflective_lesson"`
	RecommendedHermesRoute bool     `json:"recommended_hermes_route"`
}

// AnalyzePostMortem inspects a repair session and produces a longitudinal PostMortemReport.
func AnalyzePostMortem(sessionID string, attempts int, errorTraces []string, costUSD float64) PostMortemReport {
	if sessionID == "" {
		sessionID = "pm-session-1"
	}
	category := "CONTRACT_DRIFT"
	joined := strings.ToLower(strings.Join(errorTraces, " "))
	if strings.Contains(joined, "nil") || strings.Contains(joined, "pointer") {
		category = "NIL_DEREFERENCE"
	} else if strings.Contains(joined, "dimension") || strings.Contains(joined, "vector") {
		category = "VECTOR_DIMENSION_MISMATCH"
	}
	if costUSD <= 0 {
		costUSD = float64(attempts) * 0.0045
	}
	return PostMortemReport{
		SessionID:              sessionID,
		FailedAttempts:         attempts,
		RootCauseCategory:      category,
		EstimatedWastedUSD:     math.Round(costUSD*10000) / 10000,
		BrokenContracts:        errorTraces,
		ReflectiveLesson:       "Store negative knowledge in RAE-CRL as FAILED_BUT_INFORMATIVE and route L3 to Hermes.",
		RecommendedHermesRoute: attempts >= 3,
	}
}
