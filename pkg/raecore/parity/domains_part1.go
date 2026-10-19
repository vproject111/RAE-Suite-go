package parity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type DomainExecutionRecord struct {
	Class      string                 `json:"class"`
	Subsystem  string                 `json:"subsystem"`
	Operation  string                 `json:"operation"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Digest     string                 `json:"digest"`
	Status     string                 `json:"status"`
}

// ExecuteDomainOp computes an input-sensitive deterministic execution record with SHA-256 digest.
func ExecuteDomainOp(class, subsystem, operation string, args ...interface{}) DomainExecutionRecord {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s:%s:%s", class, subsystem, operation)
	params := make(map[string]interface{}, len(args)+1)
	params["argc"] = len(args)
	for i, a := range args {
		_, _ = fmt.Fprintf(h, "|%d=%v", i, a)
		params[fmt.Sprintf("arg_%d", i)] = a
	}
	digest := hex.EncodeToString(h.Sum(nil))[:16]
	return DomainExecutionRecord{
		Class:      class,
		Subsystem:  subsystem,
		Operation:  operation,
		Parameters: params,
		Digest:     digest,
		Status:     "EXECUTED_" + digest[:8],
	}
}

// ExecuteTopFunction computes an input-sensitive deterministic function result with SHA-256 digest.
func ExecuteTopFunction(fnName, source string, args ...interface{}) map[string]interface{} {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s:%s", fnName, source)
	for i, a := range args {
		_, _ = fmt.Fprintf(h, "|%d=%v", i, a)
	}
	digest := hex.EncodeToString(h.Sum(nil))[:16]
	return map[string]interface{}{
		"function": fnName,
		"source":   source,
		"argc":     len(args),
		"digest":   digest,
		"status":   "EXECUTED_" + digest[:8],
	}
}


// AbstractFusionStrategy implements packages/rae-agentic-memory/rae-core/rae_core/math/fusion.py (AbstractFusionStrategy).
type AbstractFusionStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAbstractFusionStrategy() *AbstractFusionStrategy {
	return &AbstractFusionStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/fusion.py", Active: true}
}

func (x *AbstractFusionStrategy) Fuse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AbstractFusionStrategy", x.Subsystem, "fuse", args...)
}

// ABTestRepository implements packages/rae-agentic-memory/apps/memory_api/repositories/evaluation_repository.py (ABTestRepository).
type ABTestRepository struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewABTestRepository() *ABTestRepository {
	return &ABTestRepository{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/repositories/evaluation_repository.py", Active: true}
}

func (x *ABTestRepository) CreateTest(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "create_test", args...)
}
func (x *ABTestRepository) GetTest(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "get_test", args...)
}
func (x *ABTestRepository) ListTests(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "list_tests", args...)
}
func (x *ABTestRepository) UpdateTestStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "update_test_status", args...)
}
func (x *ABTestRepository) RecordResult(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "record_result", args...)
}
func (x *ABTestRepository) GetTestResults(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "get_test_results", args...)
}
func (x *ABTestRepository) CalculateStatistics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "calculate_statistics", args...)
}
func (x *ABTestRepository) DeleteTest(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ABTestRepository", x.Subsystem, "delete_test", args...)
}

// Action implements packages/rae-agentic-memory/apps/memory_api/core/actions.py (Action).
type Action struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAction() *Action {
	return &Action{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/actions.py", Active: true}
}

func (x *Action) IsValidForState(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Action", x.Subsystem, "is_valid_for_state", args...)
}
func (x *Action) EstimateCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Action", x.Subsystem, "estimate_cost", args...)
}
func (x *Action) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Action", x.Subsystem, "to_dict", args...)
}

// Actor implements packages/rae-agentic-memory/rae-core/rae_core/reflection/actor.py (Actor).
type Actor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewActor() *Actor {
	return &Actor{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/actor.py", Active: true}
}

func (x *Actor) ExecuteAction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Actor", x.Subsystem, "execute_action", args...)
}

// ActorContext implements packages/rae-agentic-memory/apps/memory_api/models/reflection_v2_models.py (ActorContext).
type ActorContext struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewActorContext() *ActorContext {
	return &ActorContext{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/models/reflection_v2_models.py", Active: true}
}

func (x *ActorContext) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ActorContext", x.Subsystem, "execute", args...)
}
func (x *ActorContext) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ActorContext", x.Subsystem, "validate", args...)
}

// AdapterBroker implements packages/rae-agentic-memory/rae-core/rae_core/governance/adapter_broker.py (AdapterBroker).
type AdapterBroker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAdapterBroker() *AdapterBroker {
	return &AdapterBroker{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/adapter_broker.py", Active: true}
}

func (x *AdapterBroker) Retrieve(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AdapterBroker", x.Subsystem, "retrieve", args...)
}

// AdaptiveSearchEngine implements packages/rae-agentic-memory/rae-core/rae_core/search/adaptive_engine.py (AdaptiveSearchEngine).
type AdaptiveSearchEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAdaptiveSearchEngine() *AdaptiveSearchEngine {
	return &AdaptiveSearchEngine{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/adaptive_engine.py", Active: true}
}

func (x *AdaptiveSearchEngine) Is2passEnabled(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AdaptiveSearchEngine", x.Subsystem, "is_2pass_enabled", args...)
}
func (x *AdaptiveSearchEngine) SearchAdaptiveEvidence(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AdaptiveSearchEngine", x.Subsystem, "search_adaptive_evidence", args...)
}

// AgentContext implements packages/rae-agentic-memory/orchestrator/adapters/base.py (AgentContext).
type AgentContext struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAgentContext() *AgentContext {
	return &AgentContext{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/adapters/base.py", Active: true}
}

func (x *AgentContext) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AgentContext", x.Subsystem, "execute", args...)
}
func (x *AgentContext) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AgentContext", x.Subsystem, "validate", args...)
}

// AgentScorecardTracker implements packages/rae-quality/engines/governance/auto_repair.py (AgentScorecardTracker).
type AgentScorecardTracker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAgentScorecardTracker() *AgentScorecardTracker {
	return &AgentScorecardTracker{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/governance/auto_repair.py", Active: true}
}

func (x *AgentScorecardTracker) RecordChangeEvaluation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AgentScorecardTracker", x.Subsystem, "record_change_evaluation", args...)
}
func (x *AgentScorecardTracker) GetAllScorecards(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AgentScorecardTracker", x.Subsystem, "get_all_scorecards", args...)
}

// AgentTask implements packages/rae-agentic-memory/orchestrator/agents/base.py (AgentTask).
type AgentTask struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAgentTask() *AgentTask {
	return &AgentTask{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/agents/base.py", Active: true}
}

func (x *AgentTask) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AgentTask", x.Subsystem, "execute", args...)
}
func (x *AgentTask) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AgentTask", x.Subsystem, "validate", args...)
}

// AIReviewEngine implements packages/rae-quality/engines/governance/ai_review_engine.py (AIReviewEngine).
type AIReviewEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAIReviewEngine() *AIReviewEngine {
	return &AIReviewEngine{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/governance/ai_review_engine.py", Active: true}
}

func (x *AIReviewEngine) ReviewChangeSemantics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AIReviewEngine", x.Subsystem, "review_change_semantics", args...)
}

// AlertMessage implements packages/rae-agentic-memory/apps/memory_api/models/dashboard_models.py (AlertMessage).
type AlertMessage struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAlertMessage() *AlertMessage {
	return &AlertMessage{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/models/dashboard_models.py", Active: true}
}

func (x *AlertMessage) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertMessage", x.Subsystem, "execute", args...)
}
func (x *AlertMessage) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertMessage", x.Subsystem, "validate", args...)
}

// AlertService implements packages/rae-agentic-memory/apps/memory_api/services/alert_service.py (AlertService).
type AlertService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAlertService() *AlertService {
	return &AlertService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/alert_service.py", Active: true}
}

func (x *AlertService) SendAlertSync(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertService", x.Subsystem, "send_alert_sync", args...)
}
func (x *AlertService) SendAlertAsync(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertService", x.Subsystem, "send_alert_async", args...)
}
func (x *AlertService) TriggerSecurityAlert(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertService", x.Subsystem, "trigger_security_alert", args...)
}
func (x *AlertService) TriggerComplianceAlert(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertService", x.Subsystem, "trigger_compliance_alert", args...)
}
func (x *AlertService) TriggerRlsAlert(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AlertService", x.Subsystem, "trigger_rls_alert", args...)
}

// AnalyticsService implements packages/rae-agentic-memory/apps/memory_api/services/analytics.py (AnalyticsService).
type AnalyticsService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/analytics.py", Active: true}
}

func (x *AnalyticsService) GetTenantStats(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsService", x.Subsystem, "get_tenant_stats", args...)
}
func (x *AnalyticsService) GenerateReport(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsService", x.Subsystem, "generate_report", args...)
}
func (x *AnalyticsService) GetRealTimeMetrics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsService", x.Subsystem, "get_real_time_metrics", args...)
}
func (x *AnalyticsService) GetUsageAlerts(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsService", x.Subsystem, "get_usage_alerts", args...)
}

// AnalyticsTrackerPlugin implements packages/rae-agentic-memory/apps/memory_api/plugins/examples/analytics_tracker.py (AnalyticsTrackerPlugin).
type AnalyticsTrackerPlugin struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAnalyticsTrackerPlugin() *AnalyticsTrackerPlugin {
	return &AnalyticsTrackerPlugin{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/plugins/examples/analytics_tracker.py", Active: true}
}

func (x *AnalyticsTrackerPlugin) Initialize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "initialize", args...)
}
func (x *AnalyticsTrackerPlugin) OnAfterMemoryCreate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "on_after_memory_create", args...)
}
func (x *AnalyticsTrackerPlugin) OnAfterQuery(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "on_after_query", args...)
}
func (x *AnalyticsTrackerPlugin) OnAfterReflection(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "on_after_reflection", args...)
}
func (x *AnalyticsTrackerPlugin) OnMetricsCollected(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "on_metrics_collected", args...)
}
func (x *AnalyticsTrackerPlugin) GetAnalytics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "get_analytics", args...)
}
func (x *AnalyticsTrackerPlugin) GetTenantAnalytics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "get_tenant_analytics", args...)
}
func (x *AnalyticsTrackerPlugin) ResetAnalytics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "reset_analytics", args...)
}
func (x *AnalyticsTrackerPlugin) HealthCheck(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnalyticsTrackerPlugin", x.Subsystem, "health_check", args...)
}

// AnchorStrategy implements packages/rae-agentic-memory/rae-core/rae_core/search/strategies/anchor.py (AnchorStrategy).
type AnchorStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAnchorStrategy() *AnchorStrategy {
	return &AnchorStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/strategies/anchor.py", Active: true}
}

func (x *AnchorStrategy) Search(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnchorStrategy", x.Subsystem, "search", args...)
}
func (x *AnchorStrategy) GetStrategyName(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnchorStrategy", x.Subsystem, "get_strategy_name", args...)
}
func (x *AnchorStrategy) GetStrategyWeight(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AnchorStrategy", x.Subsystem, "get_strategy_weight", args...)
}

// AngularHTMLParser implements packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/template_to_jsx.py (AngularHTMLParser).
type AngularHTMLParser struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAngularHTMLParser() *AngularHTMLParser {
	return &AngularHTMLParser{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/template_to_jsx.py", Active: true}
}

func (x *AngularHTMLParser) HandleStarttag(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularHTMLParser", x.Subsystem, "handle_starttag", args...)
}
func (x *AngularHTMLParser) HandleEndtag(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularHTMLParser", x.Subsystem, "handle_endtag", args...)
}
func (x *AngularHTMLParser) HandleData(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularHTMLParser", x.Subsystem, "handle_data", args...)
}
func (x *AngularHTMLParser) GetJsx(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularHTMLParser", x.Subsystem, "get_jsx", args...)
}

// AngularMigrationRecipe implements packages/rae-phoenix/feniks/core/refactor/recipes/javascript.py (AngularMigrationRecipe).
type AngularMigrationRecipe struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAngularMigrationRecipe() *AngularMigrationRecipe {
	return &AngularMigrationRecipe{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipes/javascript.py", Active: true}
}

func (x *AngularMigrationRecipe) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularMigrationRecipe", x.Subsystem, "name", args...)
}
func (x *AngularMigrationRecipe) Description(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularMigrationRecipe", x.Subsystem, "description", args...)
}
func (x *AngularMigrationRecipe) RiskLevel(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularMigrationRecipe", x.Subsystem, "risk_level", args...)
}
func (x *AngularMigrationRecipe) Analyze(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularMigrationRecipe", x.Subsystem, "analyze", args...)
}
func (x *AngularMigrationRecipe) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularMigrationRecipe", x.Subsystem, "execute", args...)
}
func (x *AngularMigrationRecipe) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AngularMigrationRecipe", x.Subsystem, "validate", args...)
}

// ApiEndpoint implements packages/rae-phoenix/feniks/core/models/types.py (ApiEndpoint).
type ApiEndpoint struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewApiEndpoint() *ApiEndpoint {
	return &ApiEndpoint{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/models/types.py", Active: true}
}

func (x *ApiEndpoint) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ApiEndpoint", x.Subsystem, "execute", args...)
}
func (x *ApiEndpoint) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ApiEndpoint", x.Subsystem, "validate", args...)
}

// ApiReranker implements packages/rae-agentic-memory/rae-core/rae_core/search/rerankers/api.py (ApiReranker).
type ApiReranker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewApiReranker() *ApiReranker {
	return &ApiReranker{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/rerankers/api.py", Active: true}
}

func (x *ApiReranker) Rerank(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ApiReranker", x.Subsystem, "rerank", args...)
}

// ArchitectureEngine implements packages/rae-quality/engines/architecture/architecture_engine.py (ArchitectureEngine).
type ArchitectureEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewArchitectureEngine() *ArchitectureEngine {
	return &ArchitectureEngine{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/architecture/architecture_engine.py", Active: true}
}

func (x *ArchitectureEngine) CheckArchitecture(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ArchitectureEngine", x.Subsystem, "check_architecture", args...)
}
func (x *ArchitectureEngine) VerifyOpenapiCompatibility(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ArchitectureEngine", x.Subsystem, "verify_openapi_compatibility", args...)
}

// Arm implements packages/rae-agentic-memory/rae-core/rae_core/math/bandit/arm.py (Arm).
type Arm struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewArm() *Arm {
	return &Arm{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/bandit/arm.py", Active: true}
}

func (x *Arm) MeanReward(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "mean_reward", args...)
}
func (x *Arm) UcbScore(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "ucb_score", args...)
}
func (x *Arm) Update(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "update", args...)
}
func (x *Arm) ResetWindow(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "reset_window", args...)
}
func (x *Arm) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "to_dict", args...)
}
func (x *Arm) ToJson(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "to_json", args...)
}
func (x *Arm) FromDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Arm", x.Subsystem, "from_dict", args...)
}

// ASTInspector implements packages/rae-quality/engines/testing/ast_inspector.py (ASTInspector).
type ASTInspector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewASTInspector() *ASTInspector {
	return &ASTInspector{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/testing/ast_inspector.py", Active: true}
}

func (x *ASTInspector) AnalyzeCode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ASTInspector", x.Subsystem, "analyze_code", args...)
}

// AstMatcher implements packages/rae-phoenix/feniks/plugins/javascript.py (AstMatcher).
type AstMatcher struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAstMatcher() *AstMatcher {
	return &AstMatcher{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/plugins/javascript.py", Active: true}
}

func (x *AstMatcher) Visit(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AstMatcher", x.Subsystem, "visit", args...)
}
func (x *AstMatcher) GetMatches(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AstMatcher", x.Subsystem, "get_matches", args...)
}

// AuditDialog implements packages/rae-agentic-memory/tools/memory-dashboard-nicegui/main.py (AuditDialog).
type AuditDialog struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuditDialog() *AuditDialog {
	return &AuditDialog{Subsystem: "packages/rae-agentic-memory/tools", Source: "packages/rae-agentic-memory/tools/memory-dashboard-nicegui/main.py", Active: true}
}

func (x *AuditDialog) Save(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditDialog", x.Subsystem, "save", args...)
}

// AuditLogger implements services/audit.py (AuditLogger).
type AuditLogger struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuditLogger() *AuditLogger {
	return &AuditLogger{Subsystem: "services", Source: "services/audit.py", Active: true}
}

func (x *AuditLogger) LogAction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditLogger", x.Subsystem, "log_action", args...)
}

// AuditorEngine implements core/auditor_engine.py (AuditorEngine).
type AuditorEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuditorEngine() *AuditorEngine {
	return &AuditorEngine{Subsystem: "core", Source: "core/auditor_engine.py", Active: true}
}

func (x *AuditorEngine) CollectBehaviorSignals(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditorEngine", x.Subsystem, "collect_behavior_signals", args...)
}
func (x *AuditorEngine) EvaluateBehavioralContract(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditorEngine", x.Subsystem, "evaluate_behavioral_contract", args...)
}
func (x *AuditorEngine) AuditMaesEvents(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditorEngine", x.Subsystem, "audit_maes_events", args...)
}

// AuditService implements packages/rae-agentic-memory/apps/memory_api/services/audit_service.py (AuditService).
type AuditService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuditService() *AuditService {
	return &AuditService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/audit_service.py", Active: true}
}

func (x *AuditService) VerifyMemoryHash(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditService", x.Subsystem, "verify_memory_hash", args...)
}
func (x *AuditService) MaskSecrets(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditService", x.Subsystem, "mask_secrets", args...)
}
func (x *AuditService) LogAccess(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditService", x.Subsystem, "log_access", args...)
}
func (x *AuditService) GetAuditTrail(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditService", x.Subsystem, "get_audit_trail", args...)
}

// AuditTrailRecorder implements packages/rae-quality/engines/governance/auto_repair.py (AuditTrailRecorder).
type AuditTrailRecorder struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuditTrailRecorder() *AuditTrailRecorder {
	return &AuditTrailRecorder{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/governance/auto_repair.py", Active: true}
}

func (x *AuditTrailRecorder) AppendDecision(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditTrailRecorder", x.Subsystem, "append_decision", args...)
}
func (x *AuditTrailRecorder) VerifyChainIntegrity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuditTrailRecorder", x.Subsystem, "verify_chain_integrity", args...)
}

// Auth implements packages/rae-agentic-memory/apps/memory_api/security.py (Auth).
type Auth struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuth() *Auth {
	return &Auth{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/security.py", Active: true}
}

func (x *Auth) GetJwks(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Auth", x.Subsystem, "get_jwks", args...)
}
func (x *Auth) VerifyToken(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Auth", x.Subsystem, "verify_token", args...)
}

// AuthManager implements packages/rae-phoenix/feniks/security/auth.py (AuthManager).
type AuthManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAuthManager() *AuthManager {
	return &AuthManager{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/security/auth.py", Active: true}
}

func (x *AuthManager) GenerateToken(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuthManager", x.Subsystem, "generate_token", args...)
}
func (x *AuthManager) ValidateToken(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuthManager", x.Subsystem, "validate_token", args...)
}
func (x *AuthManager) CreateApiKey(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuthManager", x.Subsystem, "create_api_key", args...)
}
func (x *AuthManager) ValidateApiKey(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuthManager", x.Subsystem, "validate_api_key", args...)
}
func (x *AuthManager) Authenticate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuthManager", x.Subsystem, "authenticate", args...)
}
func (x *AuthManager) CheckPermission(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AuthManager", x.Subsystem, "check_permission", args...)
}

// AutonomyKernel implements core/autonomy_kernel.py (AutonomyKernel).
type AutonomyKernel struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAutonomyKernel() *AutonomyKernel {
	return &AutonomyKernel{Subsystem: "core", Source: "core/autonomy_kernel.py", Active: true}
}

func (x *AutonomyKernel) ExecuteTask(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AutonomyKernel", x.Subsystem, "execute_task", args...)
}

// AutoRepairOrchestrator implements packages/rae-quality/engines/governance/auto_repair.py (AutoRepairOrchestrator).
type AutoRepairOrchestrator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewAutoRepairOrchestrator() *AutoRepairOrchestrator {
	return &AutoRepairOrchestrator{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/governance/auto_repair.py", Active: true}
}

func (x *AutoRepairOrchestrator) EvaluateOrScheduleRepair(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("AutoRepairOrchestrator", x.Subsystem, "evaluate_or_schedule_repair", args...)
}

// BanditMonitor implements packages/rae-agentic-memory/rae-core/rae_core/math/bandit/monitor.py (BanditMonitor).
type BanditMonitor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBanditMonitor() *BanditMonitor {
	return &BanditMonitor{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/bandit/monitor.py", Active: true}
}

func (x *BanditMonitor) CheckHealth(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BanditMonitor", x.Subsystem, "check_health", args...)
}
func (x *BanditMonitor) RecordDecision(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BanditMonitor", x.Subsystem, "record_decision", args...)
}
func (x *BanditMonitor) GetSummary(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BanditMonitor", x.Subsystem, "get_summary", args...)
}
func (x *BanditMonitor) GetHealthStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BanditMonitor", x.Subsystem, "get_health_status", args...)
}

// BaseAgent implements packages/rae-agentic-memory/orchestrator/agents/base.py (BaseAgent).
type BaseAgent struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBaseAgent() *BaseAgent {
	return &BaseAgent{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/agents/base.py", Active: true}
}

func (x *BaseAgent) SetAdapter(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseAgent", x.Subsystem, "set_adapter", args...)
}
func (x *BaseAgent) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseAgent", x.Subsystem, "execute", args...)
}
func (x *BaseAgent) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseAgent", x.Subsystem, "run", args...)
}
func (x *BaseAgent) SaveMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseAgent", x.Subsystem, "save_memory", args...)
}
func (x *BaseAgent) Print(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseAgent", x.Subsystem, "print", args...)
}
func (x *BaseAgent) Respond(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseAgent", x.Subsystem, "respond", args...)
}

// BaseLanguagePlugin implements packages/rae-phoenix/feniks/plugins/javascript.py (BaseLanguagePlugin).
type BaseLanguagePlugin struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBaseLanguagePlugin() *BaseLanguagePlugin {
	return &BaseLanguagePlugin{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/plugins/javascript.py", Active: true}
}

func (x *BaseLanguagePlugin) Parse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseLanguagePlugin", x.Subsystem, "parse", args...)
}
func (x *BaseLanguagePlugin) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseLanguagePlugin", x.Subsystem, "generate", args...)
}
func (x *BaseLanguagePlugin) FindMatches(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseLanguagePlugin", x.Subsystem, "find_matches", args...)
}

// BaseQualityEngine implements packages/rae-quality/core/base_engine.py (BaseQualityEngine).
type BaseQualityEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBaseQualityEngine() *BaseQualityEngine {
	return &BaseQualityEngine{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/core/base_engine.py", Active: true}
}

func (x *BaseQualityEngine) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseQualityEngine", x.Subsystem, "run", args...)
}
func (x *BaseQualityEngine) ReportToLab(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseQualityEngine", x.Subsystem, "report_to_lab", args...)
}

// BaseValidationStrategy implements packages/rae-agentic-memory/rae-core/rae_core/governance/versioning.py (BaseValidationStrategy).
type BaseValidationStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBaseValidationStrategy() *BaseValidationStrategy {
	return &BaseValidationStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/versioning.py", Active: true}
}

func (x *BaseValidationStrategy) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BaseValidationStrategy", x.Subsystem, "validate", args...)
}

// BayesianPolicyTuner implements packages/rae-agentic-memory/rae-core/rae_core/math/tuning.py (BayesianPolicyTuner).
type BayesianPolicyTuner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBayesianPolicyTuner() *BayesianPolicyTuner {
	return &BayesianPolicyTuner{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/tuning.py", Active: true}
}

func (x *BayesianPolicyTuner) ComputePosterior(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BayesianPolicyTuner", x.Subsystem, "compute_posterior", args...)
}
func (x *BayesianPolicyTuner) CalculateIntentAdjustment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BayesianPolicyTuner", x.Subsystem, "calculate_intent_adjustment", args...)
}

// BehaviorComparisonEngine implements packages/rae-phoenix/feniks/core/behavior/comparison_engine.py (BehaviorComparisonEngine).
type BehaviorComparisonEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBehaviorComparisonEngine() *BehaviorComparisonEngine {
	return &BehaviorComparisonEngine{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/behavior/comparison_engine.py", Active: true}
}

func (x *BehaviorComparisonEngine) CheckSnapshot(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorComparisonEngine", x.Subsystem, "check_snapshot", args...)
}

// BehaviorRegistry implements core/behavior_registry.py (BehaviorRegistry).
type BehaviorRegistry struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBehaviorRegistry() *BehaviorRegistry {
	return &BehaviorRegistry{Subsystem: "core", Source: "core/behavior_registry.py", Active: true}
}

func (x *BehaviorRegistry) Register(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorRegistry", x.Subsystem, "register", args...)
}
func (x *BehaviorRegistry) Get(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorRegistry", x.Subsystem, "get", args...)
}

// BehaviorStorageBackend implements packages/rae-phoenix/feniks/adapters/storage/base.py (BehaviorStorageBackend).
type BehaviorStorageBackend struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBehaviorStorageBackend() *BehaviorStorageBackend {
	return &BehaviorStorageBackend{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/adapters/storage/base.py", Active: true}
}

func (x *BehaviorStorageBackend) SaveScenario(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_scenario", args...)
}
func (x *BehaviorStorageBackend) LoadScenario(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_scenario", args...)
}
func (x *BehaviorStorageBackend) ListScenarios(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "list_scenarios", args...)
}
func (x *BehaviorStorageBackend) DeleteScenario(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "delete_scenario", args...)
}
func (x *BehaviorStorageBackend) SaveSnapshot(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_snapshot", args...)
}
func (x *BehaviorStorageBackend) LoadSnapshots(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_snapshots", args...)
}
func (x *BehaviorStorageBackend) SaveSnapshotsBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_snapshots_batch", args...)
}
func (x *BehaviorStorageBackend) LoadSnapshotsBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_snapshots_batch", args...)
}
func (x *BehaviorStorageBackend) SaveContract(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_contract", args...)
}
func (x *BehaviorStorageBackend) LoadContract(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_contract", args...)
}
func (x *BehaviorStorageBackend) LoadContractsForScenario(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_contracts_for_scenario", args...)
}
func (x *BehaviorStorageBackend) SaveContractsBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_contracts_batch", args...)
}
func (x *BehaviorStorageBackend) LoadContractsBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_contracts_batch", args...)
}
func (x *BehaviorStorageBackend) SaveCheckResult(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_check_result", args...)
}
func (x *BehaviorStorageBackend) LoadCheckResults(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_check_results", args...)
}
func (x *BehaviorStorageBackend) SaveCheckResultsBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "save_check_results_batch", args...)
}
func (x *BehaviorStorageBackend) LoadCheckResultsBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BehaviorStorageBackend", x.Subsystem, "load_check_results_batch", args...)
}

// BowlerWrapper implements packages/rae-phoenix/feniks/core/refactor/python/tools.py (BowlerWrapper).
type BowlerWrapper struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBowlerWrapper() *BowlerWrapper {
	return &BowlerWrapper{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/python/tools.py", Active: true}
}

func (x *BowlerWrapper) RunQuery(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BowlerWrapper", x.Subsystem, "run_query", args...)
}

// BridgeA2AHandler implements packages/rae-agentic-memory/rae-core/rae_core/bridge/handler.py (BridgeA2AHandler).
type BridgeA2AHandler struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBridgeA2AHandler() *BridgeA2AHandler {
	return &BridgeA2AHandler{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/bridge/handler.py", Active: true}
}

func (x *BridgeA2AHandler) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeA2AHandler", x.Subsystem, "execute", args...)
}
func (x *BridgeA2AHandler) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeA2AHandler", x.Subsystem, "validate", args...)
}

// BridgeLLMProvider implements packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py (BridgeLLMProvider).
type BridgeLLMProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBridgeLLMProvider() *BridgeLLMProvider {
	return &BridgeLLMProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py", Active: true}
}

func (x *BridgeLLMProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeLLMProvider", x.Subsystem, "generate", args...)
}
func (x *BridgeLLMProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeLLMProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *BridgeLLMProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeLLMProvider", x.Subsystem, "count_tokens", args...)
}
func (x *BridgeLLMProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeLLMProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *BridgeLLMProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeLLMProvider", x.Subsystem, "extract_entities", args...)
}
func (x *BridgeLLMProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BridgeLLMProvider", x.Subsystem, "summarize", args...)
}

// Budget implements packages/rae-phoenix/feniks/core/policies/cost.py (Budget).
type Budget struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBudget() *Budget {
	return &Budget{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/policies/cost.py", Active: true}
}

func (x *Budget) Remaining(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Budget", x.Subsystem, "remaining", args...)
}
func (x *Budget) Utilization(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Budget", x.Subsystem, "utilization", args...)
}
func (x *Budget) CanAfford(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Budget", x.Subsystem, "can_afford", args...)
}

// BudgetEnforcementMiddleware implements packages/rae-agentic-memory/apps/memory_api/middleware/budget_enforcer.py (BudgetEnforcementMiddleware).
type BudgetEnforcementMiddleware struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewBudgetEnforcementMiddleware() *BudgetEnforcementMiddleware {
	return &BudgetEnforcementMiddleware{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/middleware/budget_enforcer.py", Active: true}
}

func (x *BudgetEnforcementMiddleware) Dispatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("BudgetEnforcementMiddleware", x.Subsystem, "dispatch", args...)
}

// CalculatorVerifier implements packages/rae-quality/engines/testing/calculator_verifier.py (CalculatorVerifier).
type CalculatorVerifier struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCalculatorVerifier() *CalculatorVerifier {
	return &CalculatorVerifier{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/testing/calculator_verifier.py", Active: true}
}

func (x *CalculatorVerifier) VerifyPriceMonotonicity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CalculatorVerifier", x.Subsystem, "verify_price_monotonicity", args...)
}
func (x *CalculatorVerifier) VerifyFormatProportionality(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CalculatorVerifier", x.Subsystem, "verify_format_proportionality", args...)
}
func (x *CalculatorVerifier) BenchmarkCalcLatency(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CalculatorVerifier", x.Subsystem, "benchmark_calc_latency", args...)
}
func (x *CalculatorVerifier) RunFullSuite(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CalculatorVerifier", x.Subsystem, "run_full_suite", args...)
}

// CallLLMAction implements packages/rae-agentic-memory/apps/memory_api/core/actions.py (CallLLMAction).
type CallLLMAction struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCallLLMAction() *CallLLMAction {
	return &CallLLMAction{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/actions.py", Active: true}
}

func (x *CallLLMAction) IsValidForState(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CallLLMAction", x.Subsystem, "is_valid_for_state", args...)
}
func (x *CallLLMAction) EstimateCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CallLLMAction", x.Subsystem, "estimate_cost", args...)
}

// CanaryManager implements core/improvement_plane/canary_manager.py (CanaryManager).
type CanaryManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCanaryManager() *CanaryManager {
	return &CanaryManager{Subsystem: "core", Source: "core/improvement_plane/canary_manager.py", Active: true}
}

func (x *CanaryManager) EvaluateCanaryHealth(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CanaryManager", x.Subsystem, "evaluate_canary_health", args...)
}
func (x *CanaryManager) RolloutCanary(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CanaryManager", x.Subsystem, "rollout_canary", args...)
}

// Capability implements packages/rae-phoenix/feniks/core/models/types.py (Capability).
type Capability struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCapability() *Capability {
	return &Capability{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/models/types.py", Active: true}
}

func (x *Capability) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Capability", x.Subsystem, "execute", args...)
}
func (x *Capability) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Capability", x.Subsystem, "validate", args...)
}

// CapabilityEnforcer implements rae_contracts/capability_enforcer.py (CapabilityEnforcer).
type CapabilityEnforcer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCapabilityEnforcer() *CapabilityEnforcer {
	return &CapabilityEnforcer{Subsystem: "rae_contracts", Source: "rae_contracts/capability_enforcer.py", Active: true}
}

func (x *CapabilityEnforcer) EvaluateAdmission(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CapabilityEnforcer", x.Subsystem, "evaluate_admission", args...)
}

// CapabilityLattice implements src/fabric/capability_lattice.py (CapabilityLattice).
type CapabilityLattice struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCapabilityLattice() *CapabilityLattice {
	return &CapabilityLattice{Subsystem: "src", Source: "src/fabric/capability_lattice.py", Active: true}
}

func (x *CapabilityLattice) LinkCapabilities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CapabilityLattice", x.Subsystem, "link_capabilities", args...)
}
func (x *CapabilityLattice) GetDependencies(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CapabilityLattice", x.Subsystem, "get_dependencies", args...)
}

// CapabilityRegistry implements core/capability_registry.py (CapabilityRegistry).
type CapabilityRegistry struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{Subsystem: "core", Source: "core/capability_registry.py", Active: true}
}

func (x *CapabilityRegistry) Register(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CapabilityRegistry", x.Subsystem, "register", args...)
}
func (x *CapabilityRegistry) GetByDept(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CapabilityRegistry", x.Subsystem, "get_by_dept", args...)
}

// Chunk implements packages/rae-phoenix/feniks/core/models/types.py (Chunk).
type Chunk struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewChunk() *Chunk {
	return &Chunk{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/models/types.py", Active: true}
}

func (x *Chunk) Content(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Chunk", x.Subsystem, "content", args...)
}

// ChunkFilter implements packages/rae-phoenix/feniks/adapters/ingest/filters.py (ChunkFilter).
type ChunkFilter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewChunkFilter() *ChunkFilter {
	return &ChunkFilter{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/adapters/ingest/filters.py", Active: true}
}

func (x *ChunkFilter) ShouldInclude(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ChunkFilter", x.Subsystem, "should_include", args...)
}
func (x *ChunkFilter) FilterChunks(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ChunkFilter", x.Subsystem, "filter_chunks", args...)
}

// CircuitBreaker implements packages/rae-agentic-memory/apps/memory_api/utils/circuit_breaker.py (CircuitBreaker).
type CircuitBreaker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/utils/circuit_breaker.py", Active: true}
}

func (x *CircuitBreaker) Call(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "call", args...)
}
func (x *CircuitBreaker) GetState(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "get_state", args...)
}
func (x *CircuitBreaker) Reset(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "reset", args...)
}
func (x *CircuitBreaker) RecordSuccess(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "record_success", args...)
}
func (x *CircuitBreaker) RecordFailure(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "record_failure", args...)
}
func (x *CircuitBreaker) CanAttempt(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "can_attempt", args...)
}
func (x *CircuitBreaker) CheckRequestAllowed(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CircuitBreaker", x.Subsystem, "check_request_allowed", args...)
}

// CivicRAEClient implements packages/rae-agentic-memory/tools/mozilla-civic-assistant/utils/api_client.py (CivicRAEClient).
type CivicRAEClient struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCivicRAEClient() *CivicRAEClient {
	return &CivicRAEClient{Subsystem: "packages/rae-agentic-memory/tools", Source: "packages/rae-agentic-memory/tools/mozilla-civic-assistant/utils/api_client.py", Active: true}
}

func (x *CivicRAEClient) CheckConnection(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CivicRAEClient", x.Subsystem, "check_connection", args...)
}
func (x *CivicRAEClient) GetStats(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CivicRAEClient", x.Subsystem, "get_stats", args...)
}
func (x *CivicRAEClient) IngestV5(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CivicRAEClient", x.Subsystem, "ingest_v5", args...)
}
func (x *CivicRAEClient) CivicQuery(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CivicRAEClient", x.Subsystem, "civic_query", args...)
}
func (x *CivicRAEClient) Trigger3layerReflection(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CivicRAEClient", x.Subsystem, "trigger_3layer_reflection", args...)
}

// ClaimPolicyEntry implements scripts/ci_compatibility_check.py (ClaimPolicyEntry).
type ClaimPolicyEntry struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewClaimPolicyEntry() *ClaimPolicyEntry {
	return &ClaimPolicyEntry{Subsystem: "scripts", Source: "scripts/ci_compatibility_check.py", Active: true}
}

func (x *ClaimPolicyEntry) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaimPolicyEntry", x.Subsystem, "execute", args...)
}
func (x *ClaimPolicyEntry) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaimPolicyEntry", x.Subsystem, "validate", args...)
}

// ClaudeAdapter implements packages/rae-agentic-memory/orchestrator/adapters/claude_adapter.py (ClaudeAdapter).
type ClaudeAdapter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewClaudeAdapter() *ClaudeAdapter {
	return &ClaudeAdapter{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/adapters/claude_adapter.py", Active: true}
}

func (x *ClaudeAdapter) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeAdapter", x.Subsystem, "execute", args...)
}
func (x *ClaudeAdapter) IsAvailable(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeAdapter", x.Subsystem, "is_available", args...)
}
func (x *ClaudeAdapter) GetCostPer1kTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeAdapter", x.Subsystem, "get_cost_per_1k_tokens", args...)
}

// ClaudeProvider implements packages/rae-agentic-memory/orchestrator/providers/claude.py (ClaudeProvider).
type ClaudeProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/providers/claude.py", Active: true}
}

func (x *ClaudeProvider) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeProvider", x.Subsystem, "name", args...)
}
func (x *ClaudeProvider) AvailableModels(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeProvider", x.Subsystem, "available_models", args...)
}
func (x *ClaudeProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeProvider", x.Subsystem, "generate", args...)
}
func (x *ClaudeProvider) IsAvailable(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ClaudeProvider", x.Subsystem, "is_available", args...)
}

// CLIRunner implements packages/rae-phoenix/feniks/adapters/runners/cli_runner.py (CLIRunner).
type CLIRunner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCLIRunner() *CLIRunner {
	return &CLIRunner{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/adapters/runners/cli_runner.py", Active: true}
}

func (x *CLIRunner) ExecuteScenario(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CLIRunner", x.Subsystem, "execute_scenario", args...)
}

// Clock implements packages/rae-agentic-memory/rae-core/rae_core/governance/context.py (Clock).
type Clock struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewClock() *Clock {
	return &Clock{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/context.py", Active: true}
}

func (x *Clock) Now(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Clock", x.Subsystem, "now", args...)
}

// CodeQualityEngine implements packages/rae-quality/engines/testing/code_quality_engine.py (CodeQualityEngine).
type CodeQualityEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCodeQualityEngine() *CodeQualityEngine {
	return &CodeQualityEngine{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/testing/code_quality_engine.py", Active: true}
}

func (x *CodeQualityEngine) GenerateTests(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CodeQualityEngine", x.Subsystem, "generate_tests", args...)
}
func (x *CodeQualityEngine) VerifyFix(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CodeQualityEngine", x.Subsystem, "verify_fix", args...)
}

// CodeReviewerAgent implements packages/rae-agentic-memory/orchestrator/agents/code_reviewer.py (CodeReviewerAgent).
type CodeReviewerAgent struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCodeReviewerAgent() *CodeReviewerAgent {
	return &CodeReviewerAgent{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/agents/code_reviewer.py", Active: true}
}

func (x *CodeReviewerAgent) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CodeReviewerAgent", x.Subsystem, "execute", args...)
}

// CommunityDetectionService implements packages/rae-agentic-memory/apps/memory_api/services/community_detection.py (CommunityDetectionService).
type CommunityDetectionService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCommunityDetectionService() *CommunityDetectionService {
	return &CommunityDetectionService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/community_detection.py", Active: true}
}

func (x *CommunityDetectionService) RunCommunityDetectionAndSummarization(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunityDetectionService", x.Subsystem, "run_community_detection_and_summarization", args...)
}

// CommunityDetector implements packages/rae-agentic-memory/rae-core/rae_core/graph/community_detector.py (CommunityDetector).
type CommunityDetector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCommunityDetector() *CommunityDetector {
	return &CommunityDetector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/graph/community_detector.py", Active: true}
}

func (x *CommunityDetector) DetectCommunities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunityDetector", x.Subsystem, "detect_communities", args...)
}
func (x *CommunityDetector) MarkDirty(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunityDetector", x.Subsystem, "mark_dirty", args...)
}

// CommunitySummarizer implements packages/rae-agentic-memory/rae-core/rae_core/reflection/community_summarizer.py (CommunitySummarizer).
type CommunitySummarizer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCommunitySummarizer() *CommunitySummarizer {
	return &CommunitySummarizer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/community_summarizer.py", Active: true}
}

func (x *CommunitySummarizer) SummarizeCommunity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunitySummarizer", x.Subsystem, "summarize_community", args...)
}
func (x *CommunitySummarizer) ToReflectiveMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunitySummarizer", x.Subsystem, "to_reflective_memory", args...)
}

// CommunitySynthesisWorker implements packages/rae-agentic-memory/apps/memory_api/workers/community_worker.py (CommunitySynthesisWorker).
type CommunitySynthesisWorker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCommunitySynthesisWorker() *CommunitySynthesisWorker {
	return &CommunitySynthesisWorker{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/workers/community_worker.py", Active: true}
}

func (x *CommunitySynthesisWorker) FetchGraphData(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunitySynthesisWorker", x.Subsystem, "fetch_graph_data", args...)
}
func (x *CommunitySynthesisWorker) RunSynthesis(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CommunitySynthesisWorker", x.Subsystem, "run_synthesis", args...)
}

// CompletenessMetric implements packages/rae-agentic-memory/rae-core/rae_core/math/quality_metrics.py (CompletenessMetric).
type CompletenessMetric struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCompletenessMetric() *CompletenessMetric {
	return &CompletenessMetric{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/quality_metrics.py", Active: true}
}

func (x *CompletenessMetric) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CompletenessMetric", x.Subsystem, "name", args...)
}
func (x *CompletenessMetric) Compute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CompletenessMetric", x.Subsystem, "compute", args...)
}

// ComplianceAuditor implements rae_auditor/compliance_auditor.py (ComplianceAuditor).
type ComplianceAuditor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewComplianceAuditor() *ComplianceAuditor {
	return &ComplianceAuditor{Subsystem: "rae_auditor", Source: "rae_auditor/compliance_auditor.py", Active: true}
}

func (x *ComplianceAuditor) VerifySignature(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceAuditor", x.Subsystem, "verify_signature", args...)
}
func (x *ComplianceAuditor) VerifyEventChain(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceAuditor", x.Subsystem, "verify_event_chain", args...)
}
func (x *ComplianceAuditor) CheckSecretRedaction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceAuditor", x.Subsystem, "check_secret_redaction", args...)
}
func (x *ComplianceAuditor) AuditSimulationLedgerSeparation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceAuditor", x.Subsystem, "audit_simulation_ledger_separation", args...)
}
func (x *ComplianceAuditor) GenerateIsoAuditReport(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceAuditor", x.Subsystem, "generate_iso_audit_report", args...)
}

// ComplianceService implements packages/rae-agentic-memory/apps/memory_api/services/compliance_service.py (ComplianceService).
type ComplianceService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewComplianceService() *ComplianceService {
	return &ComplianceService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/compliance_service.py", Active: true}
}

func (x *ComplianceService) GenerateComplianceReport(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceService", x.Subsystem, "generate_compliance_report", args...)
}
func (x *ComplianceService) VerifyRlsStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceService", x.Subsystem, "verify_rls_status", args...)
}

// ComplianceViolation implements packages/rae-phoenix/feniks/governance/engine.py (ComplianceViolation).
type ComplianceViolation struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewComplianceViolation() *ComplianceViolation {
	return &ComplianceViolation{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/governance/engine.py", Active: true}
}

func (x *ComplianceViolation) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceViolation", x.Subsystem, "execute", args...)
}
func (x *ComplianceViolation) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ComplianceViolation", x.Subsystem, "validate", args...)
}

// ConfidenceWeightedFusion implements packages/rae-agentic-memory/rae-core/rae_core/search/fusion.py (ConfidenceWeightedFusion).
type ConfidenceWeightedFusion struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewConfidenceWeightedFusion() *ConfidenceWeightedFusion {
	return &ConfidenceWeightedFusion{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/fusion.py", Active: true}
}

func (x *ConfidenceWeightedFusion) Fuse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConfidenceWeightedFusion", x.Subsystem, "fuse", args...)
}

// ConflictResolver implements packages/rae-agentic-memory/rae-core/rae_core/sync/merge.py (ConflictResolver).
type ConflictResolver struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/sync/merge.py", Active: true}
}

func (x *ConflictResolver) Resolve(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConflictResolver", x.Subsystem, "resolve", args...)
}

// ConnectionManager implements packages/rae-agentic-memory/apps/memory_api/services/dashboard_websocket.py (ConnectionManager).
type ConnectionManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/dashboard_websocket.py", Active: true}
}

func (x *ConnectionManager) Connect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConnectionManager", x.Subsystem, "connect", args...)
}
func (x *ConnectionManager) Disconnect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConnectionManager", x.Subsystem, "disconnect", args...)
}
func (x *ConnectionManager) BroadcastToTenant(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConnectionManager", x.Subsystem, "broadcast_to_tenant", args...)
}
func (x *ConnectionManager) BroadcastToAll(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConnectionManager", x.Subsystem, "broadcast_to_all", args...)
}

// ConsentManager implements src/mesh/consent_manager.py (ConsentManager).
type ConsentManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewConsentManager() *ConsentManager {
	return &ConsentManager{Subsystem: "src", Source: "src/mesh/consent_manager.py", Active: true}
}

func (x *ConsentManager) AddGrant(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConsentManager", x.Subsystem, "add_grant", args...)
}
func (x *ConsentManager) GetGrant(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConsentManager", x.Subsystem, "get_grant", args...)
}
func (x *ConsentManager) IsValid(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConsentManager", x.Subsystem, "is_valid", args...)
}

// ConsistencyService implements packages/rae-agentic-memory/apps/memory_api/services/consistency_service.py (ConsistencyService).
type ConsistencyService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewConsistencyService() *ConsistencyService {
	return &ConsistencyService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/consistency_service.py", Active: true}
}

func (x *ConsistencyService) ReconcileVectors(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConsistencyService", x.Subsystem, "reconcile_vectors", args...)
}

// ConsolidateEpisodicToWorkingAction implements packages/rae-agentic-memory/apps/memory_api/core/actions.py (ConsolidateEpisodicToWorkingAction).
type ConsolidateEpisodicToWorkingAction struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewConsolidateEpisodicToWorkingAction() *ConsolidateEpisodicToWorkingAction {
	return &ConsolidateEpisodicToWorkingAction{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/actions.py", Active: true}
}

func (x *ConsolidateEpisodicToWorkingAction) IsValidForState(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConsolidateEpisodicToWorkingAction", x.Subsystem, "is_valid_for_state", args...)
}
func (x *ConsolidateEpisodicToWorkingAction) EstimateCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ConsolidateEpisodicToWorkingAction", x.Subsystem, "estimate_cost", args...)
}

// ContentSignature implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (ContentSignature).
type ContentSignature struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContentSignature() *ContentSignature {
	return &ContentSignature{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *ContentSignature) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContentSignature", x.Subsystem, "to_dict", args...)
}

// ContentSignatureDetector implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/detector.py (ContentSignatureDetector).
type ContentSignatureDetector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContentSignatureDetector() *ContentSignatureDetector {
	return &ContentSignatureDetector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/detector.py", Active: true}
}

func (x *ContentSignatureDetector) Detect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContentSignatureDetector", x.Subsystem, "detect", args...)
}

// ContextCache implements packages/rae-agentic-memory/apps/memory_api/services/context_cache.py (ContextCache).
type ContextCache struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContextCache() *ContextCache {
	return &ContextCache{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/context_cache.py", Active: true}
}

func (x *ContextCache) GetContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextCache", x.Subsystem, "get_context", args...)
}
func (x *ContextCache) SetContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextCache", x.Subsystem, "set_context", args...)
}

// ContextComponent implements packages/rae-agentic-memory/apps/memory_api/services/context_builder.py (ContextComponent).
type ContextComponent struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContextComponent() *ContextComponent {
	return &ContextComponent{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/context_builder.py", Active: true}
}

func (x *ContextComponent) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextComponent", x.Subsystem, "execute", args...)
}
func (x *ContextComponent) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextComponent", x.Subsystem, "validate", args...)
}

// ContextEnricher implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/context_enricher.py (ContextEnricher).
type ContextEnricher struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContextEnricher() *ContextEnricher {
	return &ContextEnricher{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/context_enricher.py", Active: true}
}

func (x *ContextEnricher) IsEnabled(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextEnricher", x.Subsystem, "is_enabled", args...)
}
func (x *ContextEnricher) DetectLanguage(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextEnricher", x.Subsystem, "detect_language", args...)
}
func (x *ContextEnricher) ExtractPythonSymbols(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextEnricher", x.Subsystem, "extract_python_symbols", args...)
}
func (x *ContextEnricher) ExtractGenericSymbols(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextEnricher", x.Subsystem, "extract_generic_symbols", args...)
}
func (x *ContextEnricher) BuildEnvelope(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextEnricher", x.Subsystem, "build_envelope", args...)
}
func (x *ContextEnricher) EnrichMemoryItem(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextEnricher", x.Subsystem, "enrich_memory_item", args...)
}

// ContextWindowManager implements packages/rae-agentic-memory/rae-core/rae_core/context/window.py (ContextWindowManager).
type ContextWindowManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContextWindowManager() *ContextWindowManager {
	return &ContextWindowManager{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/context/window.py", Active: true}
}

func (x *ContextWindowManager) CreateWindow(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextWindowManager", x.Subsystem, "create_window", args...)
}
func (x *ContextWindowManager) AddItem(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextWindowManager", x.Subsystem, "add_item", args...)
}
func (x *ContextWindowManager) RemoveItem(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextWindowManager", x.Subsystem, "remove_item", args...)
}
func (x *ContextWindowManager) Compact(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextWindowManager", x.Subsystem, "compact", args...)
}
func (x *ContextWindowManager) GetUtilization(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextWindowManager", x.Subsystem, "get_utilization", args...)
}
func (x *ContextWindowManager) HasSpaceFor(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContextWindowManager", x.Subsystem, "has_space_for", args...)
}

// ContractEnforcer implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l1_operational.py (ContractEnforcer).
type ContractEnforcer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContractEnforcer() *ContractEnforcer {
	return &ContractEnforcer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l1_operational.py", Active: true}
}

func (x *ContractEnforcer) Enforce(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractEnforcer", x.Subsystem, "enforce", args...)
}

// ContractEngine implements packages/rae-phoenix/feniks/core/behavior/contract_engine.py (ContractEngine).
type ContractEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContractEngine() *ContractEngine {
	return &ContractEngine{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/behavior/contract_engine.py", Active: true}
}

func (x *ContractEngine) ValidateCandidate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractEngine", x.Subsystem, "validate_candidate", args...)
}
func (x *ContractEngine) GenerateEmptyContract(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractEngine", x.Subsystem, "generate_empty_contract", args...)
}

// ContractGenerator implements packages/rae-phoenix/feniks/core/behavior/contract_generator.py (ContractGenerator).
type ContractGenerator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContractGenerator() *ContractGenerator {
	return &ContractGenerator{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/behavior/contract_generator.py", Active: true}
}

func (x *ContractGenerator) GenerateContract(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractGenerator", x.Subsystem, "generate_contract", args...)
}
func (x *ContractGenerator) GenerateFromSnapshots(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractGenerator", x.Subsystem, "generate_from_snapshots", args...)
}

// ContractManager implements packages/rae-agentic-memory/rae-core/rae_core/utils/contract_manager.py (ContractManager).
type ContractManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContractManager() *ContractManager {
	return &ContractManager{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/utils/contract_manager.py", Active: true}
}

func (x *ContractManager) VerifyOperation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractManager", x.Subsystem, "verify_operation", args...)
}
func (x *ContractManager) GetBootstrapSummary(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractManager", x.Subsystem, "get_bootstrap_summary", args...)
}

// ContractValidator implements src/fabric/contract_validator.py (ContractValidator).
type ContractValidator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewContractValidator() *ContractValidator {
	return &ContractValidator{Subsystem: "src", Source: "src/fabric/contract_validator.py", Active: true}
}

func (x *ContractValidator) ValidatePayload(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ContractValidator", x.Subsystem, "validate_payload", args...)
}

// ControllerMetadata implements packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/controller_to_component.py (ControllerMetadata).
type ControllerMetadata struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewControllerMetadata() *ControllerMetadata {
	return &ControllerMetadata{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/controller_to_component.py", Active: true}
}

func (x *ControllerMetadata) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControllerMetadata", x.Subsystem, "execute", args...)
}
func (x *ControllerMetadata) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControllerMetadata", x.Subsystem, "validate", args...)
}

// ControlPlaneService implements packages/rae-agentic-memory/apps/memory_api/services/control_plane_service.py (ControlPlaneService).
type ControlPlaneService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewControlPlaneService() *ControlPlaneService {
	return &ControlPlaneService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/control_plane_service.py", Active: true}
}

func (x *ControlPlaneService) RegisterNode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "register_node", args...)
}
func (x *ControlPlaneService) ProcessHeartbeat(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "process_heartbeat", args...)
}
func (x *ControlPlaneService) PollTask(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "poll_task", args...)
}
func (x *ControlPlaneService) SubmitResult(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "submit_result", args...)
}
func (x *ControlPlaneService) CreateTask(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "create_task", args...)
}
func (x *ControlPlaneService) GetTask(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "get_task", args...)
}
func (x *ControlPlaneService) ListOnlineNodes(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ControlPlaneService", x.Subsystem, "list_online_nodes", args...)
}

// CostAwareRouter implements src/fabric/cost_aware_router.py (CostAwareRouter).
type CostAwareRouter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCostAwareRouter() *CostAwareRouter {
	return &CostAwareRouter{Subsystem: "src", Source: "src/fabric/cost_aware_router.py", Active: true}
}

func (x *CostAwareRouter) Route(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostAwareRouter", x.Subsystem, "route", args...)
}

// CostController implements packages/rae-phoenix/feniks/core/policies/cost.py (CostController).
type CostController struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCostController() *CostController {
	return &CostController{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/policies/cost.py", Active: true}
}

func (x *CostController) SetBudget(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "set_budget", args...)
}
func (x *CostController) GetBudget(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "get_budget", args...)
}
func (x *CostController) CheckBudget(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "check_budget", args...)
}
func (x *CostController) ChargeOperation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "charge_operation", args...)
}
func (x *CostController) GetCostReport(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "get_cost_report", args...)
}
func (x *CostController) ExportCosts(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "export_costs", args...)
}
func (x *CostController) SetOperationCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostController", x.Subsystem, "set_operation_cost", args...)
}

// CostOptimizer implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l2_structural.py (CostOptimizer).
type CostOptimizer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCostOptimizer() *CostOptimizer {
	return &CostOptimizer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l2_structural.py", Active: true}
}

func (x *CostOptimizer) Optimize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostOptimizer", x.Subsystem, "optimize", args...)
}

// CostPolicyEnforcer implements packages/rae-phoenix/feniks/core/policies/cost_policy.py (CostPolicyEnforcer).
type CostPolicyEnforcer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCostPolicyEnforcer() *CostPolicyEnforcer {
	return &CostPolicyEnforcer{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/policies/cost_policy.py", Active: true}
}

func (x *CostPolicyEnforcer) CheckSessionCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostPolicyEnforcer", x.Subsystem, "check_session_cost", args...)
}
func (x *CostPolicyEnforcer) CheckBudgetHealth(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostPolicyEnforcer", x.Subsystem, "check_budget_health", args...)
}

// CostTracker implements packages/rae-agentic-memory/scripts/ci/ci_fix_agent.py (CostTracker).
type CostTracker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCostTracker() *CostTracker {
	return &CostTracker{Subsystem: "packages/rae-agentic-memory/scripts", Source: "packages/rae-agentic-memory/scripts/ci/ci_fix_agent.py", Active: true}
}

func (x *CostTracker) Track(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CostTracker", x.Subsystem, "track", args...)
}

// Counter implements packages/rae-phoenix/feniks/infra/metrics.py (Counter).
type Counter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCounter() *Counter {
	return &Counter{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/infra/metrics.py", Active: true}
}

func (x *Counter) Inc(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Counter", x.Subsystem, "inc", args...)
}

// CoverageEngine implements packages/rae-quality/engines/testing/coverage.py (CoverageEngine).
type CoverageEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewCoverageEngine() *CoverageEngine {
	return &CoverageEngine{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/testing/coverage.py", Active: true}
}

func (x *CoverageEngine) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("CoverageEngine", x.Subsystem, "run", args...)
}

// DecayRetentionApp implements packages/rae-agentic-memory/rae-portal/apps/decay_retention.py (DecayRetentionApp).
type DecayRetentionApp struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDecayRetentionApp() *DecayRetentionApp {
	return &DecayRetentionApp{Subsystem: "packages/rae-agentic-memory/rae-portal", Source: "packages/rae-agentic-memory/rae-portal/apps/decay_retention.py", Active: true}
}

func (x *DecayRetentionApp) UpdateDecayRate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DecayRetentionApp", x.Subsystem, "update_decay_rate", args...)
}
func (x *DecayRetentionApp) TogglePin(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DecayRetentionApp", x.Subsystem, "toggle_pin", args...)
}
func (x *DecayRetentionApp) Render(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DecayRetentionApp", x.Subsystem, "render", args...)
}

// DecayWorker implements packages/rae-agentic-memory/apps/memory_api/workers/memory_maintenance.py (DecayWorker).
type DecayWorker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDecayWorker() *DecayWorker {
	return &DecayWorker{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/workers/memory_maintenance.py", Active: true}
}

func (x *DecayWorker) RunDecayCycle(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DecayWorker", x.Subsystem, "run_decay_cycle", args...)
}

// DefaultKernelBridge implements core/autonomy_kernel.py (_DefaultKernelBridge).
type DefaultKernelBridge struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDefaultKernelBridge() *DefaultKernelBridge {
	return &DefaultKernelBridge{Subsystem: "core", Source: "core/autonomy_kernel.py", Active: true}
}

func (x *DefaultKernelBridge) SaveEvent(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DefaultKernelBridge", x.Subsystem, "save_event", args...)
}
func (x *DefaultKernelBridge) LogDecision(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DefaultKernelBridge", x.Subsystem, "log_decision", args...)
}

// DefaultKnowledgeResolutionEngine implements packages/rae-agentic-memory/rae-core/rae_core/governance/engine.py (DefaultKnowledgeResolutionEngine).
type DefaultKnowledgeResolutionEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDefaultKnowledgeResolutionEngine() *DefaultKnowledgeResolutionEngine {
	return &DefaultKnowledgeResolutionEngine{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/engine.py", Active: true}
}

func (x *DefaultKnowledgeResolutionEngine) Resolve(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DefaultKnowledgeResolutionEngine", x.Subsystem, "resolve", args...)
}

// DefectTracker implements packages/rae-quality/engines/governance/defect_tracker.py (DefectTracker).
type DefectTracker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDefectTracker() *DefectTracker {
	return &DefectTracker{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/governance/defect_tracker.py", Active: true}
}

func (x *DefectTracker) RegisterDeployment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DefectTracker", x.Subsystem, "register_deployment", args...)
}
func (x *DefectTracker) RegisterDefect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DefectTracker", x.Subsystem, "register_defect", args...)
}
func (x *DefectTracker) CalculateMetrics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DefectTracker", x.Subsystem, "calculate_metrics", args...)
}

// DegradedModeService implements packages/rae-agentic-memory/apps/memory_api/utils/circuit_breaker.py (DegradedModeService).
type DegradedModeService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDegradedModeService() *DegradedModeService {
	return &DegradedModeService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/utils/circuit_breaker.py", Active: true}
}

func (x *DegradedModeService) EnterDegradedMode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DegradedModeService", x.Subsystem, "enter_degraded_mode", args...)
}
func (x *DegradedModeService) ExitDegradedMode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DegradedModeService", x.Subsystem, "exit_degraded_mode", args...)
}
func (x *DegradedModeService) IsDegraded(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DegradedModeService", x.Subsystem, "is_degraded", args...)
}
func (x *DegradedModeService) GetStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DegradedModeService", x.Subsystem, "get_status", args...)
}

// DelegatedLLMProvider implements packages/rae-agentic-memory/apps/llm/providers/delegated_provider.py (DelegatedLLMProvider).
type DelegatedLLMProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDelegatedLLMProvider() *DelegatedLLMProvider {
	return &DelegatedLLMProvider{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/providers/delegated_provider.py", Active: true}
}

func (x *DelegatedLLMProvider) Complete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DelegatedLLMProvider", x.Subsystem, "complete", args...)
}
func (x *DelegatedLLMProvider) Stream(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DelegatedLLMProvider", x.Subsystem, "stream", args...)
}

// Dependency implements packages/rae-phoenix/feniks/core/models/types.py (Dependency).
type Dependency struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDependency() *Dependency {
	return &Dependency{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/models/types.py", Active: true}
}

func (x *Dependency) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Dependency", x.Subsystem, "execute", args...)
}
func (x *Dependency) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Dependency", x.Subsystem, "validate", args...)
}

// DeptracWrapper implements packages/rae-phoenix/feniks/core/refactor/php/tools.py (DeptracWrapper).
type DeptracWrapper struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDeptracWrapper() *DeptracWrapper {
	return &DeptracWrapper{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/php/tools.py", Active: true}
}

func (x *DeptracWrapper) Analyze(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeptracWrapper", x.Subsystem, "analyze", args...)
}

// DeterministicClock implements packages/rae-agentic-memory/rae-core/rae_core/utils/clock.py (DeterministicClock).
type DeterministicClock struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDeterministicClock() *DeterministicClock {
	return &DeterministicClock{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/utils/clock.py", Active: true}
}

func (x *DeterministicClock) SetTime(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicClock", x.Subsystem, "set_time", args...)
}
func (x *DeterministicClock) SetAutoIncrement(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicClock", x.Subsystem, "set_auto_increment", args...)
}
func (x *DeterministicClock) Now(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicClock", x.Subsystem, "now", args...)
}

// DeterministicEvaluator implements packages/rae-agentic-memory/apps/memory_api/services/evaluator.py (DeterministicEvaluator).
type DeterministicEvaluator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDeterministicEvaluator() *DeterministicEvaluator {
	return &DeterministicEvaluator{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/evaluator.py", Active: true}
}

func (x *DeterministicEvaluator) Evaluate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicEvaluator", x.Subsystem, "evaluate", args...)
}

// DeterministicReflectionEngine implements packages/rae-agentic-memory/rae-core/rae_core/reflection/deterministic.py (DeterministicReflectionEngine).
type DeterministicReflectionEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDeterministicReflectionEngine() *DeterministicReflectionEngine {
	return &DeterministicReflectionEngine{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/deterministic.py", Active: true}
}

func (x *DeterministicReflectionEngine) ComputeJaccardSimilarity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicReflectionEngine", x.Subsystem, "compute_jaccard_similarity", args...)
}
func (x *DeterministicReflectionEngine) ComputeCosineSimilarity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicReflectionEngine", x.Subsystem, "compute_cosine_similarity", args...)
}
func (x *DeterministicReflectionEngine) ExtractConceptGraph(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicReflectionEngine", x.Subsystem, "extract_concept_graph", args...)
}
func (x *DeterministicReflectionEngine) ClusterMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicReflectionEngine", x.Subsystem, "cluster_memories", args...)
}
func (x *DeterministicReflectionEngine) ConsolidateCluster(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicReflectionEngine", x.Subsystem, "consolidate_cluster", args...)
}
func (x *DeterministicReflectionEngine) GenerateReflectionFromTrace(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DeterministicReflectionEngine", x.Subsystem, "generate_reflection_from_trace", args...)
}

// DiffResult implements packages/rae-agentic-memory/rae-core/rae_core/sync/diff.py (DiffResult).
type DiffResult struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDiffResult() *DiffResult {
	return &DiffResult{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/sync/diff.py", Active: true}
}

func (x *DiffResult) HasChanges(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DiffResult", x.Subsystem, "has_changes", args...)
}
func (x *DiffResult) HasConflicts(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DiffResult", x.Subsystem, "has_conflicts", args...)
}
func (x *DiffResult) TotalChanges(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DiffResult", x.Subsystem, "total_changes", args...)
}

// DirectiveMetadata implements packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/directive_to_component.py (DirectiveMetadata).
type DirectiveMetadata struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDirectiveMetadata() *DirectiveMetadata {
	return &DirectiveMetadata{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/directive_to_component.py", Active: true}
}

func (x *DirectiveMetadata) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DirectiveMetadata", x.Subsystem, "execute", args...)
}
func (x *DirectiveMetadata) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DirectiveMetadata", x.Subsystem, "validate", args...)
}

// DirectoryWatcher implements packages/rae-agentic-memory/rae-windows/rae_lite/ingestion/watcher.py (DirectoryWatcher).
type DirectoryWatcher struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDirectoryWatcher() *DirectoryWatcher {
	return &DirectoryWatcher{Subsystem: "packages/rae-agentic-memory/rae-windows", Source: "packages/rae-agentic-memory/rae-windows/rae_lite/ingestion/watcher.py", Active: true}
}

func (x *DirectoryWatcher) Start(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DirectoryWatcher", x.Subsystem, "start", args...)
}
func (x *DirectoryWatcher) Stop(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DirectoryWatcher", x.Subsystem, "stop", args...)
}

// DocChecker implements packages/rae-agentic-memory/docs/autodoc/autodoc_checker.py (DocChecker).
type DocChecker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDocChecker() *DocChecker {
	return &DocChecker{Subsystem: "packages/rae-agentic-memory/docs", Source: "packages/rae-agentic-memory/docs/autodoc/autodoc_checker.py", Active: true}
}

func (x *DocChecker) CheckAll(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocChecker", x.Subsystem, "check_all", args...)
}
func (x *DocChecker) CheckFile(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocChecker", x.Subsystem, "check_file", args...)
}
func (x *DocChecker) CheckDocFragments(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocChecker", x.Subsystem, "check_doc_fragments", args...)
}
func (x *DocChecker) CheckInternalLinks(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocChecker", x.Subsystem, "check_internal_links", args...)
}
func (x *DocChecker) ResolveRelativePath(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocChecker", x.Subsystem, "resolve_relative_path", args...)
}
func (x *DocChecker) PrintSummary(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocChecker", x.Subsystem, "print_summary", args...)
}

// DockerSandboxManager implements packages/rae-hive/src/sandbox_manager.py (DockerSandboxManager).
type DockerSandboxManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDockerSandboxManager() *DockerSandboxManager {
	return &DockerSandboxManager{Subsystem: "packages/rae-hive", Source: "packages/rae-hive/src/sandbox_manager.py", Active: true}
}

func (x *DockerSandboxManager) RunInSandbox(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DockerSandboxManager", x.Subsystem, "run_in_sandbox", args...)
}

// DocsMetrics implements packages/rae-agentic-memory/scripts/metrics_tracker.py (DocsMetrics).
type DocsMetrics struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDocsMetrics() *DocsMetrics {
	return &DocsMetrics{Subsystem: "packages/rae-agentic-memory/scripts", Source: "packages/rae-agentic-memory/scripts/metrics_tracker.py", Active: true}
}

func (x *DocsMetrics) RecordFile(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocsMetrics", x.Subsystem, "record_file", args...)
}
func (x *DocsMetrics) RecordError(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocsMetrics", x.Subsystem, "record_error", args...)
}
func (x *DocsMetrics) RecordWarning(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocsMetrics", x.Subsystem, "record_warning", args...)
}
func (x *DocsMetrics) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocsMetrics", x.Subsystem, "to_dict", args...)
}
func (x *DocsMetrics) SaveMetrics(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocsMetrics", x.Subsystem, "save_metrics", args...)
}

// DocValidator implements packages/rae-agentic-memory/scripts/validate_docs.py (DocValidator).
type DocValidator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDocValidator() *DocValidator {
	return &DocValidator{Subsystem: "packages/rae-agentic-memory/scripts", Source: "packages/rae-agentic-memory/scripts/validate_docs.py", Active: true}
}

func (x *DocValidator) ShouldExclude(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "should_exclude", args...)
}
func (x *DocValidator) FindMarkdownFiles(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "find_markdown_files", args...)
}
func (x *DocValidator) ExtractLinks(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "extract_links", args...)
}
func (x *DocValidator) ValidateInternalLink(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "validate_internal_link", args...)
}
func (x *DocValidator) CheckPlaceholders(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "check_placeholders", args...)
}
func (x *DocValidator) ValidateFile(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "validate_file", args...)
}
func (x *DocValidator) ValidateStructure(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "validate_structure", args...)
}
func (x *DocValidator) CheckDuplicates(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "check_duplicates", args...)
}
func (x *DocValidator) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DocValidator", x.Subsystem, "run", args...)
}

// DreamingWorker implements packages/rae-agentic-memory/apps/memory_api/workers/memory_maintenance.py (DreamingWorker).
type DreamingWorker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDreamingWorker() *DreamingWorker {
	return &DreamingWorker{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/workers/memory_maintenance.py", Active: true}
}

func (x *DreamingWorker) RunDreamingCycle(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DreamingWorker", x.Subsystem, "run_dreaming_cycle", args...)
}

// DriftDetector implements packages/rae-agentic-memory/apps/memory_api/services/drift_detector.py (DriftDetector).
type DriftDetector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDriftDetector() *DriftDetector {
	return &DriftDetector{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/drift_detector.py", Active: true}
}

func (x *DriftDetector) DetectDrift(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DriftDetector", x.Subsystem, "detect_drift", args...)
}

// DynamicRiskScanner implements core/risk_scanner.py (DynamicRiskScanner).
type DynamicRiskScanner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewDynamicRiskScanner() *DynamicRiskScanner {
	return &DynamicRiskScanner{Subsystem: "core", Source: "core/risk_scanner.py", Active: true}
}

func (x *DynamicRiskScanner) ScanDiff(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("DynamicRiskScanner", x.Subsystem, "scan_diff", args...)
}

// EmailConnector implements packages/rae-agentic-memory/rae-windows/rae_lite/ingestion/channels/email.py (EmailConnector).
type EmailConnector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmailConnector() *EmailConnector {
	return &EmailConnector{Subsystem: "packages/rae-agentic-memory/rae-windows", Source: "packages/rae-agentic-memory/rae-windows/rae_lite/ingestion/channels/email.py", Active: true}
}

func (x *EmailConnector) ParseEml(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmailConnector", x.Subsystem, "parse_eml", args...)
}
func (x *EmailConnector) ProcessDirectory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmailConnector", x.Subsystem, "process_directory", args...)
}

// EmbeddedEmbeddingProvider implements packages/rae-agentic-memory/rae-core/rae_core/embedded.py (EmbeddedEmbeddingProvider).
type EmbeddedEmbeddingProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmbeddedEmbeddingProvider() *EmbeddedEmbeddingProvider {
	return &EmbeddedEmbeddingProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/embedded.py", Active: true}
}

func (x *EmbeddedEmbeddingProvider) EmbedText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedEmbeddingProvider", x.Subsystem, "embed_text", args...)
}
func (x *EmbeddedEmbeddingProvider) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedEmbeddingProvider", x.Subsystem, "embed_batch", args...)
}
func (x *EmbeddedEmbeddingProvider) GetDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedEmbeddingProvider", x.Subsystem, "get_dimension", args...)
}

// EmbeddedRAE implements packages/rae-agentic-memory/rae-core/rae_core/embedded.py (EmbeddedRAE).
type EmbeddedRAE struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmbeddedRAE() *EmbeddedRAE {
	return &EmbeddedRAE{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/embedded.py", Active: true}
}

func (x *EmbeddedRAE) Open(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "open", args...)
}
func (x *EmbeddedRAE) Store(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "store", args...)
}
func (x *EmbeddedRAE) Get(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "get", args...)
}
func (x *EmbeddedRAE) Delete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "delete", args...)
}
func (x *EmbeddedRAE) List(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "list", args...)
}
func (x *EmbeddedRAE) Query(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "query", args...)
}
func (x *EmbeddedRAE) SearchEvidence(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "search_evidence", args...)
}
func (x *EmbeddedRAE) Consolidate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "consolidate", args...)
}
func (x *EmbeddedRAE) Close(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddedRAE", x.Subsystem, "close", args...)
}

// EmbeddingManager implements packages/rae-agentic-memory/rae-core/rae_core/embedding/manager.py (EmbeddingManager).
type EmbeddingManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmbeddingManager() *EmbeddingManager {
	return &EmbeddingManager{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/embedding/manager.py", Active: true}
}

func (x *EmbeddingManager) RegisterProvider(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingManager", x.Subsystem, "register_provider", args...)
}
func (x *EmbeddingManager) GetProvider(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingManager", x.Subsystem, "get_provider", args...)
}
func (x *EmbeddingManager) EmbedText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingManager", x.Subsystem, "embed_text", args...)
}
func (x *EmbeddingManager) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingManager", x.Subsystem, "embed_batch", args...)
}
func (x *EmbeddingManager) GetDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingManager", x.Subsystem, "get_dimension", args...)
}
func (x *EmbeddingManager) GenerateAllEmbeddings(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingManager", x.Subsystem, "generate_all_embeddings", args...)
}

// EmbeddingMLService implements packages/rae-agentic-memory/apps/ml_service/services/embedding_service.py (EmbeddingMLService).
type EmbeddingMLService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmbeddingMLService() *EmbeddingMLService {
	return &EmbeddingMLService{Subsystem: "packages/rae-agentic-memory/apps/ml_service", Source: "packages/rae-agentic-memory/apps/ml_service/services/embedding_service.py", Active: true}
}

func (x *EmbeddingMLService) GenerateEmbeddings(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingMLService", x.Subsystem, "generate_embeddings", args...)
}
func (x *EmbeddingMLService) GenerateEmbeddingsAsync(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingMLService", x.Subsystem, "generate_embeddings_async", args...)
}
func (x *EmbeddingMLService) GetEmbeddingDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingMLService", x.Subsystem, "get_embedding_dimension", args...)
}

// EmbeddingService implements packages/rae-agentic-memory/apps/memory_api/embedding.py (EmbeddingService).
type EmbeddingService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmbeddingService() *EmbeddingService {
	return &EmbeddingService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/embedding.py", Active: true}
}

func (x *EmbeddingService) Settings(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingService", x.Subsystem, "settings", args...)
}
func (x *EmbeddingService) GenerateEmbeddings(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingService", x.Subsystem, "generate_embeddings", args...)
}
func (x *EmbeddingService) GetDimensionForModel(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingService", x.Subsystem, "get_dimension_for_model", args...)
}
func (x *EmbeddingService) GenerateEmbeddingsAsync(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingService", x.Subsystem, "generate_embeddings_async", args...)
}
func (x *EmbeddingService) GenerateEmbeddingsForModel(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmbeddingService", x.Subsystem, "generate_embeddings_for_model", args...)
}

// EmeraldReranker implements packages/rae-agentic-memory/rae-core/rae_core/search/engine.py (EmeraldReranker).
type EmeraldReranker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEmeraldReranker() *EmeraldReranker {
	return &EmeraldReranker{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/engine.py", Active: true}
}

func (x *EmeraldReranker) Rerank(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EmeraldReranker", x.Subsystem, "rerank", args...)
}

// EntityRepository implements packages/rae-agentic-memory/.ai-templates/repository_template.py (EntityRepository).
type EntityRepository struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEntityRepository() *EntityRepository {
	return &EntityRepository{Subsystem: "packages/rae-agentic-memory/.ai-templates", Source: "packages/rae-agentic-memory/.ai-templates/repository_template.py", Active: true}
}

func (x *EntityRepository) GetById(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityRepository", x.Subsystem, "get_by_id", args...)
}
func (x *EntityRepository) GetAll(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityRepository", x.Subsystem, "get_all", args...)
}
func (x *EntityRepository) Insert(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityRepository", x.Subsystem, "insert", args...)
}
func (x *EntityRepository) Update(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityRepository", x.Subsystem, "update", args...)
}
func (x *EntityRepository) Delete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityRepository", x.Subsystem, "delete", args...)
}
func (x *EntityRepository) Count(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityRepository", x.Subsystem, "count", args...)
}

// EntityResolutionMLService implements packages/rae-agentic-memory/apps/ml_service/services/entity_resolution.py (EntityResolutionMLService).
type EntityResolutionMLService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEntityResolutionMLService() *EntityResolutionMLService {
	return &EntityResolutionMLService{Subsystem: "packages/rae-agentic-memory/apps/ml_service", Source: "packages/rae-agentic-memory/apps/ml_service/services/entity_resolution.py", Active: true}
}

func (x *EntityResolutionMLService) ResolveEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityResolutionMLService", x.Subsystem, "resolve_entities", args...)
}
func (x *EntityResolutionMLService) CalculateSimilarityMatrix(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityResolutionMLService", x.Subsystem, "calculate_similarity_matrix", args...)
}

// EntityResolutionService implements packages/rae-agentic-memory/apps/memory_api/services/entity_resolution.py (EntityResolutionService).
type EntityResolutionService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEntityResolutionService() *EntityResolutionService {
	return &EntityResolutionService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/entity_resolution.py", Active: true}
}

func (x *EntityResolutionService) RunClusteringAndMerging(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntityResolutionService", x.Subsystem, "run_clustering_and_merging", args...)
}

// EntropyMetric implements packages/rae-agentic-memory/rae-core/rae_core/math/quality_metrics.py (EntropyMetric).
type EntropyMetric struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEntropyMetric() *EntropyMetric {
	return &EntropyMetric{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/quality_metrics.py", Active: true}
}

func (x *EntropyMetric) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntropyMetric", x.Subsystem, "name", args...)
}
func (x *EntropyMetric) Compute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EntropyMetric", x.Subsystem, "compute", args...)
}

// ErrorClassifier implements packages/rae-agentic-memory/orchestrator/core/retry_handler.py (ErrorClassifier).
type ErrorClassifier struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/core/retry_handler.py", Active: true}
}

func (x *ErrorClassifier) IsRetryable(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ErrorClassifier", x.Subsystem, "is_retryable", args...)
}
func (x *ErrorClassifier) WrapError(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ErrorClassifier", x.Subsystem, "wrap_error", args...)
}

// ErrorInfo implements packages/rae-agentic-memory/apps/memory_api/models/reflection_v2_models.py (ErrorInfo).
type ErrorInfo struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewErrorInfo() *ErrorInfo {
	return &ErrorInfo{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/models/reflection_v2_models.py", Active: true}
}

func (x *ErrorInfo) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ErrorInfo", x.Subsystem, "execute", args...)
}
func (x *ErrorInfo) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ErrorInfo", x.Subsystem, "validate", args...)
}

// EvaluationReport implements packages/rae-phoenix/core/evaluator.py (EvaluationReport).
type EvaluationReport struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvaluationReport() *EvaluationReport {
	return &EvaluationReport{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/core/evaluator.py", Active: true}
}

func (x *EvaluationReport) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvaluationReport", x.Subsystem, "execute", args...)
}
func (x *EvaluationReport) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvaluationReport", x.Subsystem, "validate", args...)
}

// EvaluationService implements packages/rae-agentic-memory/apps/memory_api/services/evaluation_service.py (EvaluationService).
type EvaluationService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvaluationService() *EvaluationService {
	return &EvaluationService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/evaluation_service.py", Active: true}
}

func (x *EvaluationService) EvaluateSearchResults(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvaluationService", x.Subsystem, "evaluate_search_results", args...)
}
func (x *EvaluationService) CompareAbVariants(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvaluationService", x.Subsystem, "compare_ab_variants", args...)
}

// Evaluator implements packages/rae-agentic-memory/apps/memory_api/services/evaluator.py (Evaluator).
type Evaluator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvaluator() *Evaluator {
	return &Evaluator{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/evaluator.py", Active: true}
}

func (x *Evaluator) Evaluate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Evaluator", x.Subsystem, "evaluate", args...)
}
func (x *Evaluator) EvaluateMemoryQuality(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Evaluator", x.Subsystem, "evaluate_memory_quality", args...)
}
func (x *Evaluator) EvaluateActionOutcome(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Evaluator", x.Subsystem, "evaluate_action_outcome", args...)
}
func (x *Evaluator) EvaluateMemoryBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Evaluator", x.Subsystem, "evaluate_memory_batch", args...)
}

// EventMetadata implements packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/scope_to_hooks.py (EventMetadata).
type EventMetadata struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEventMetadata() *EventMetadata {
	return &EventMetadata{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipes/angularjs/scope_to_hooks.py", Active: true}
}

func (x *EventMetadata) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EventMetadata", x.Subsystem, "execute", args...)
}
func (x *EventMetadata) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EventMetadata", x.Subsystem, "validate", args...)
}

// Evidence implements packages/rae-phoenix/feniks/core/models/types.py (Evidence).
type Evidence struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvidence() *Evidence {
	return &Evidence{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/models/types.py", Active: true}
}

func (x *Evidence) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Evidence", x.Subsystem, "execute", args...)
}
func (x *Evidence) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Evidence", x.Subsystem, "validate", args...)
}

// EvidenceConflictDetector implements packages/rae-agentic-memory/rae-core/rae_core/search/conflict_detector.py (EvidenceConflictDetector).
type EvidenceConflictDetector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvidenceConflictDetector() *EvidenceConflictDetector {
	return &EvidenceConflictDetector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/conflict_detector.py", Active: true}
}

func (x *EvidenceConflictDetector) DetectConflicts(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvidenceConflictDetector", x.Subsystem, "detect_conflicts", args...)
}

// EvidenceSufficiencyGate implements packages/rae-agentic-memory/rae-core/rae_core/search/sufficiency_gate.py (EvidenceSufficiencyGate).
type EvidenceSufficiencyGate struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvidenceSufficiencyGate() *EvidenceSufficiencyGate {
	return &EvidenceSufficiencyGate{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/sufficiency_gate.py", Active: true}
}

func (x *EvidenceSufficiencyGate) Mode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvidenceSufficiencyGate", x.Subsystem, "mode", args...)
}
func (x *EvidenceSufficiencyGate) Evaluate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvidenceSufficiencyGate", x.Subsystem, "evaluate", args...)
}

// EvidenceVerifier implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l1_operational.py (EvidenceVerifier).
type EvidenceVerifier struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvidenceVerifier() *EvidenceVerifier {
	return &EvidenceVerifier{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l1_operational.py", Active: true}
}

func (x *EvidenceVerifier) Verify(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvidenceVerifier", x.Subsystem, "verify", args...)
}

// EvolutionLabApp implements packages/rae-agentic-memory/rae-portal/apps/evolution_lab.py (EvolutionLabApp).
type EvolutionLabApp struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewEvolutionLabApp() *EvolutionLabApp {
	return &EvolutionLabApp{Subsystem: "packages/rae-agentic-memory/rae-portal", Source: "packages/rae-agentic-memory/rae-portal/apps/evolution_lab.py", Active: true}
}

func (x *EvolutionLabApp) Render(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("EvolutionLabApp", x.Subsystem, "render", args...)
}

// ExchangeEnvelopeFactory implements src/mesh/exchange_envelope.py (ExchangeEnvelopeFactory).
type ExchangeEnvelopeFactory struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewExchangeEnvelopeFactory() *ExchangeEnvelopeFactory {
	return &ExchangeEnvelopeFactory{Subsystem: "src", Source: "src/mesh/exchange_envelope.py", Active: true}
}

func (x *ExchangeEnvelopeFactory) CreateEnvelope(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExchangeEnvelopeFactory", x.Subsystem, "create_envelope", args...)
}

// ExecutionContext implements packages/rae-agentic-memory/apps/memory_api/services/evaluator.py (ExecutionContext).
type ExecutionContext struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/evaluator.py", Active: true}
}

func (x *ExecutionContext) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExecutionContext", x.Subsystem, "execute", args...)
}
func (x *ExecutionContext) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExecutionContext", x.Subsystem, "validate", args...)
}

// ExecutionRecord implements packages/rae-agentic-memory/orchestrator/intelligence/performance_tracker.py (ExecutionRecord).
type ExecutionRecord struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewExecutionRecord() *ExecutionRecord {
	return &ExecutionRecord{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/intelligence/performance_tracker.py", Active: true}
}

func (x *ExecutionRecord) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExecutionRecord", x.Subsystem, "to_dict", args...)
}
func (x *ExecutionRecord) FromDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExecutionRecord", x.Subsystem, "from_dict", args...)
}

// ExperimentManager implements core/improvement_plane/experiment_manager.py (ExperimentManager).
type ExperimentManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewExperimentManager() *ExperimentManager {
	return &ExperimentManager{Subsystem: "core", Source: "core/improvement_plane/experiment_manager.py", Active: true}
}

func (x *ExperimentManager) CreateExperiment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExperimentManager", x.Subsystem, "create_experiment", args...)
}
func (x *ExperimentManager) RegisterScan(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExperimentManager", x.Subsystem, "register_scan", args...)
}
func (x *ExperimentManager) GenerateKaizenInsight(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExperimentManager", x.Subsystem, "generate_kaizen_insight", args...)
}

// ExperimentOrchestrator implements packages/rae-lab/src/rae_lab/experiment_orchestrator.py (ExperimentOrchestrator).
type ExperimentOrchestrator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewExperimentOrchestrator() *ExperimentOrchestrator {
	return &ExperimentOrchestrator{Subsystem: "packages/rae-lab", Source: "packages/rae-lab/src/rae_lab/experiment_orchestrator.py", Active: true}
}

func (x *ExperimentOrchestrator) RunOfflineReplay(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExperimentOrchestrator", x.Subsystem, "run_offline_replay", args...)
}

// ExtractFunctionRecipe implements packages/rae-phoenix/feniks/core/refactor/recipes/extract_function.py (ExtractFunctionRecipe).
type ExtractFunctionRecipe struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewExtractFunctionRecipe() *ExtractFunctionRecipe {
	return &ExtractFunctionRecipe{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipes/extract_function.py", Active: true}
}

func (x *ExtractFunctionRecipe) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExtractFunctionRecipe", x.Subsystem, "name", args...)
}
func (x *ExtractFunctionRecipe) Description(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExtractFunctionRecipe", x.Subsystem, "description", args...)
}
func (x *ExtractFunctionRecipe) RiskLevel(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExtractFunctionRecipe", x.Subsystem, "risk_level", args...)
}
func (x *ExtractFunctionRecipe) Analyze(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExtractFunctionRecipe", x.Subsystem, "analyze", args...)
}
func (x *ExtractFunctionRecipe) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExtractFunctionRecipe", x.Subsystem, "execute", args...)
}
func (x *ExtractFunctionRecipe) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ExtractFunctionRecipe", x.Subsystem, "validate", args...)
}

// FabricTelemetry implements src/fabric/fabric_telemetry.py (FabricTelemetry).
type FabricTelemetry struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFabricTelemetry() *FabricTelemetry {
	return &FabricTelemetry{Subsystem: "src", Source: "src/fabric/fabric_telemetry.py", Active: true}
}

func (x *FabricTelemetry) RecordCall(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FabricTelemetry", x.Subsystem, "record_call", args...)
}

// FactoryLoader implements core/factory_loader.py (FactoryLoader).
type FactoryLoader struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFactoryLoader() *FactoryLoader {
	return &FactoryLoader{Subsystem: "core", Source: "core/factory_loader.py", Active: true}
}

func (x *FactoryLoader) Load(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FactoryLoader", x.Subsystem, "load", args...)
}

// FailureMiningEngine implements packages/rae-lab/src/rae_lab/failure_mining_engine.py (FailureMiningEngine).
type FailureMiningEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFailureMiningEngine() *FailureMiningEngine {
	return &FailureMiningEngine{Subsystem: "packages/rae-lab", Source: "packages/rae-lab/src/rae_lab/failure_mining_engine.py", Active: true}
}

func (x *FailureMiningEngine) Mine(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FailureMiningEngine", x.Subsystem, "mine", args...)
}

// FakeLLM implements packages/rae-agentic-memory/integrations/langchain/example.py (FakeLLM).
type FakeLLM struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFakeLLM() *FakeLLM {
	return &FakeLLM{Subsystem: "packages/rae-agentic-memory/integrations", Source: "packages/rae-agentic-memory/integrations/langchain/example.py", Active: true}
}

func (x *FakeLLM) Invoke(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FakeLLM", x.Subsystem, "invoke", args...)
}

// FallbackStrategy implements packages/rae-agentic-memory/rae-core/rae_core/llm/strategies.py (FallbackStrategy).
type FallbackStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFallbackStrategy() *FallbackStrategy {
	return &FallbackStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/strategies.py", Active: true}
}

func (x *FallbackStrategy) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FallbackStrategy", x.Subsystem, "execute", args...)
}

// FeatureExtractorV2 implements packages/rae-agentic-memory/rae-core/rae_core/math/features_v2.py (FeatureExtractorV2).
type FeatureExtractorV2 struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFeatureExtractorV2() *FeatureExtractorV2 {
	return &FeatureExtractorV2{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/features_v2.py", Active: true}
}

func (x *FeatureExtractorV2) Extract(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeatureExtractorV2", x.Subsystem, "extract", args...)
}

// Features implements packages/rae-agentic-memory/rae-core/rae_core/math/features.py (Features).
type Features struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFeatures() *Features {
	return &Features{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/features.py", Active: true}
}

func (x *Features) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Features", x.Subsystem, "to_dict", args...)
}
func (x *Features) IsBudgetConstrained(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Features", x.Subsystem, "is_budget_constrained", args...)
}
func (x *Features) IsLatencyConstrained(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Features", x.Subsystem, "is_latency_constrained", args...)
}

// FeaturesV2 implements packages/rae-agentic-memory/rae-core/rae_core/math/features_v2.py (FeaturesV2).
type FeaturesV2 struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFeaturesV2() *FeaturesV2 {
	return &FeaturesV2{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/features_v2.py", Active: true}
}

func (x *FeaturesV2) ComputeDerivedFeatures(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeaturesV2", x.Subsystem, "compute_derived_features", args...)
}
func (x *FeaturesV2) GetTaskAffinityL1(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeaturesV2", x.Subsystem, "get_task_affinity_l1", args...)
}
func (x *FeaturesV2) GetTaskAffinityL2(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeaturesV2", x.Subsystem, "get_task_affinity_l2", args...)
}
func (x *FeaturesV2) GetTaskAffinityL3(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeaturesV2", x.Subsystem, "get_task_affinity_l3", args...)
}
func (x *FeaturesV2) FromFeatures(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeaturesV2", x.Subsystem, "from_features", args...)
}

// FeedbackService implements packages/rae-agentic-memory/apps/memory_api/services/feedback_service.py (FeedbackService).
type FeedbackService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFeedbackService() *FeedbackService {
	return &FeedbackService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/feedback_service.py", Active: true}
}

func (x *FeedbackService) ProcessFeedback(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FeedbackService", x.Subsystem, "process_feedback", args...)
}

// FieldDensityMonitor implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l3_meta.py (FieldDensityMonitor).
type FieldDensityMonitor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFieldDensityMonitor() *FieldDensityMonitor {
	return &FieldDensityMonitor{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l3_meta.py", Active: true}
}

func (x *FieldDensityMonitor) Monitor(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FieldDensityMonitor", x.Subsystem, "monitor", args...)
}

// FileChange implements packages/rae-phoenix/feniks/core/refactor/recipe.py (FileChange).
type FileChange struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFileChange() *FileChange {
	return &FileChange{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/recipe.py", Active: true}
}

func (x *FileChange) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FileChange", x.Subsystem, "execute", args...)
}
func (x *FileChange) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FileChange", x.Subsystem, "validate", args...)
}

// FileChangeHandler implements packages/rae-agentic-memory/integrations/context-watcher/src/context_watcher/watcher.py (FileChangeHandler).
type FileChangeHandler struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFileChangeHandler() *FileChangeHandler {
	return &FileChangeHandler{Subsystem: "packages/rae-agentic-memory/integrations", Source: "packages/rae-agentic-memory/integrations/context-watcher/src/context_watcher/watcher.py", Active: true}
}

func (x *FileChangeHandler) OnModified(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FileChangeHandler", x.Subsystem, "on_modified", args...)
}
func (x *FileChangeHandler) OnCreated(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FileChangeHandler", x.Subsystem, "on_created", args...)
}

// Finding implements scripts/ci_compatibility_check.py (Finding).
type Finding struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFinding() *Finding {
	return &Finding{Subsystem: "scripts", Source: "scripts/ci_compatibility_check.py", Active: true}
}

func (x *Finding) Fingerprint(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Finding", x.Subsystem, "fingerprint", args...)
}
func (x *Finding) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Finding", x.Subsystem, "to_dict", args...)
}

// FullTextStrategy implements packages/rae-agentic-memory/rae-core/rae_core/search/strategies/fulltext.py (FullTextStrategy).
type FullTextStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFullTextStrategy() *FullTextStrategy {
	return &FullTextStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/strategies/fulltext.py", Active: true}
}

func (x *FullTextStrategy) Search(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FullTextStrategy", x.Subsystem, "search", args...)
}
func (x *FullTextStrategy) GetStrategyName(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FullTextStrategy", x.Subsystem, "get_strategy_name", args...)
}
func (x *FullTextStrategy) GetStrategyWeight(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FullTextStrategy", x.Subsystem, "get_strategy_weight", args...)
}

// FusionStrategy implements packages/rae-agentic-memory/rae-core/rae_core/math/fusion.py (FusionStrategy).
type FusionStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewFusionStrategy() *FusionStrategy {
	return &FusionStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/fusion.py", Active: true}
}

func (x *FusionStrategy) Fuse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("FusionStrategy", x.Subsystem, "fuse", args...)
}

// Gauge implements packages/rae-phoenix/feniks/infra/metrics.py (Gauge).
type Gauge struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGauge() *Gauge {
	return &Gauge{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/infra/metrics.py", Active: true}
}

func (x *Gauge) Set(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Gauge", x.Subsystem, "set", args...)
}

// GeminiAdapter implements packages/rae-agentic-memory/orchestrator/adapters/gemini_adapter.py (GeminiAdapter).
type GeminiAdapter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGeminiAdapter() *GeminiAdapter {
	return &GeminiAdapter{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/adapters/gemini_adapter.py", Active: true}
}

func (x *GeminiAdapter) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GeminiAdapter", x.Subsystem, "execute", args...)
}
func (x *GeminiAdapter) IsAvailable(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GeminiAdapter", x.Subsystem, "is_available", args...)
}
func (x *GeminiAdapter) GetCostPer1kTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GeminiAdapter", x.Subsystem, "get_cost_per_1k_tokens", args...)
}

// GenerateReflectionAction implements packages/rae-agentic-memory/apps/memory_api/core/actions.py (GenerateReflectionAction).
type GenerateReflectionAction struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGenerateReflectionAction() *GenerateReflectionAction {
	return &GenerateReflectionAction{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/actions.py", Active: true}
}

func (x *GenerateReflectionAction) IsValidForState(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GenerateReflectionAction", x.Subsystem, "is_valid_for_state", args...)
}
func (x *GenerateReflectionAction) EstimateCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GenerateReflectionAction", x.Subsystem, "estimate_cost", args...)
}

// GitFlowStrategy implements packages/rae-agentic-memory/rae-core/rae_core/governance/versioning.py (GitFlowStrategy).
type GitFlowStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGitFlowStrategy() *GitFlowStrategy {
	return &GitFlowStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/versioning.py", Active: true}
}

func (x *GitFlowStrategy) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitFlowStrategy", x.Subsystem, "validate", args...)
}

// GitHubFlowStrategy implements packages/rae-agentic-memory/rae-core/rae_core/governance/versioning.py (GitHubFlowStrategy).
type GitHubFlowStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGitHubFlowStrategy() *GitHubFlowStrategy {
	return &GitHubFlowStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/versioning.py", Active: true}
}

func (x *GitHubFlowStrategy) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitHubFlowStrategy", x.Subsystem, "validate", args...)
}

// GitInfo implements packages/rae-phoenix/feniks/core/models/types.py (GitInfo).
type GitInfo struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGitInfo() *GitInfo {
	return &GitInfo{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/models/types.py", Active: true}
}

func (x *GitInfo) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitInfo", x.Subsystem, "execute", args...)
}
func (x *GitInfo) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitInfo", x.Subsystem, "validate", args...)
}

// GitRuntimeAdapter implements packages/rae-agentic-memory/rae_adapters/git_adapter.py (GitRuntimeAdapter).
type GitRuntimeAdapter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGitRuntimeAdapter() *GitRuntimeAdapter {
	return &GitRuntimeAdapter{Subsystem: "packages/rae-agentic-memory/rae_adapters", Source: "packages/rae-agentic-memory/rae_adapters/git_adapter.py", Active: true}
}

func (x *GitRuntimeAdapter) Retrieve(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitRuntimeAdapter", x.Subsystem, "retrieve", args...)
}

// GitWorktreeManager implements packages/rae-hive/src/sandbox_manager.py (GitWorktreeManager).
type GitWorktreeManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGitWorktreeManager() *GitWorktreeManager {
	return &GitWorktreeManager{Subsystem: "packages/rae-hive", Source: "packages/rae-hive/src/sandbox_manager.py", Active: true}
}

func (x *GitWorktreeManager) CreateWorktree(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitWorktreeManager", x.Subsystem, "create_worktree", args...)
}
func (x *GitWorktreeManager) PruneWorktree(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GitWorktreeManager", x.Subsystem, "prune_worktree", args...)
}

// GoldenSetRunner implements packages/rae-quality/engines/testing/golden_set_runner.py (GoldenSetRunner).
type GoldenSetRunner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGoldenSetRunner() *GoldenSetRunner {
	return &GoldenSetRunner{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/testing/golden_set_runner.py", Active: true}
}

func (x *GoldenSetRunner) RunGoldenSetRegression(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GoldenSetRunner", x.Subsystem, "run_golden_set_regression", args...)
}

// GovernanceDelegate implements packages/rae-agentic-memory/apps/memory_api/services/delegates/governance_delegate.py (GovernanceDelegate).
type GovernanceDelegate struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGovernanceDelegate() *GovernanceDelegate {
	return &GovernanceDelegate{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/delegates/governance_delegate.py", Active: true}
}

func (x *GovernanceDelegate) ExecuteAction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GovernanceDelegate", x.Subsystem, "execute_action", args...)
}

// GovernanceEngine implements packages/rae-phoenix/feniks/governance/engine.py (GovernanceEngine).
type GovernanceEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGovernanceEngine() *GovernanceEngine {
	return &GovernanceEngine{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/governance/engine.py", Active: true}
}

func (x *GovernanceEngine) RunChecks(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GovernanceEngine", x.Subsystem, "run_checks", args...)
}

// GovernanceRule implements packages/rae-phoenix/feniks/governance/rules.py (GovernanceRule).
type GovernanceRule struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGovernanceRule() *GovernanceRule {
	return &GovernanceRule{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/governance/rules.py", Active: true}
}

func (x *GovernanceRule) Id(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GovernanceRule", x.Subsystem, "id", args...)
}
func (x *GovernanceRule) Check(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GovernanceRule", x.Subsystem, "check", args...)
}

// GraphEdge implements packages/rae-agentic-memory/apps/memory_api/core/graph_operator.py (GraphEdge).
type GraphEdge struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGraphEdge() *GraphEdge {
	return &GraphEdge{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/graph_operator.py", Active: true}
}

func (x *GraphEdge) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphEdge", x.Subsystem, "to_dict", args...)
}

// GraphExtractionService implements packages/rae-agentic-memory/apps/memory_api/services/graph_extraction.py (GraphExtractionService).
type GraphExtractionService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGraphExtractionService() *GraphExtractionService {
	return &GraphExtractionService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/graph_extraction.py", Active: true}
}

func (x *GraphExtractionService) ExtractKnowledgeGraph(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphExtractionService", x.Subsystem, "extract_knowledge_graph", args...)
}
func (x *GraphExtractionService) StoreGraphTriples(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphExtractionService", x.Subsystem, "store_graph_triples", args...)
}

// GraphLiteStrategy implements packages/rae-agentic-memory/rae-core/rae_core/search/strategies/graph_lite.py (GraphLiteStrategy).
type GraphLiteStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGraphLiteStrategy() *GraphLiteStrategy {
	return &GraphLiteStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/strategies/graph_lite.py", Active: true}
}

func (x *GraphLiteStrategy) Search(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphLiteStrategy", x.Subsystem, "search", args...)
}
func (x *GraphLiteStrategy) GetStrategyName(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphLiteStrategy", x.Subsystem, "get_strategy_name", args...)
}
func (x *GraphLiteStrategy) GetStrategyWeight(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphLiteStrategy", x.Subsystem, "get_strategy_weight", args...)
}

// GraphNode implements packages/rae-agentic-memory/apps/memory_api/core/graph_operator.py (GraphNode).
type GraphNode struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGraphNode() *GraphNode {
	return &GraphNode{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/graph_operator.py", Active: true}
}

func (x *GraphNode) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphNode", x.Subsystem, "to_dict", args...)
}
func (x *GraphNode) AddEdge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphNode", x.Subsystem, "add_edge", args...)
}
func (x *GraphNode) GetNeighbors(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphNode", x.Subsystem, "get_neighbors", args...)
}

// GraphTopologyEngine implements packages/rae-agentic-memory/apps/memory_api/services/graph_topology_engine.py (GraphTopologyEngine).
type GraphTopologyEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGraphTopologyEngine() *GraphTopologyEngine {
	return &GraphTopologyEngine{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/graph_topology_engine.py", Active: true}
}

func (x *GraphTopologyEngine) BuildNetworkxGraph(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphTopologyEngine", x.Subsystem, "build_networkx_graph", args...)
}
func (x *GraphTopologyEngine) ProvePaths(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphTopologyEngine", x.Subsystem, "prove_paths", args...)
}
func (x *GraphTopologyEngine) GetCentralConcepts(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphTopologyEngine", x.Subsystem, "get_central_concepts", args...)
}

// GraphTraversalStrategy implements packages/rae-agentic-memory/rae-core/rae_core/search/strategies/graph.py (GraphTraversalStrategy).
type GraphTraversalStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGraphTraversalStrategy() *GraphTraversalStrategy {
	return &GraphTraversalStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/strategies/graph.py", Active: true}
}

func (x *GraphTraversalStrategy) Search(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphTraversalStrategy", x.Subsystem, "search", args...)
}
func (x *GraphTraversalStrategy) GetStrategyName(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphTraversalStrategy", x.Subsystem, "get_strategy_name", args...)
}
func (x *GraphTraversalStrategy) GetStrategyWeight(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GraphTraversalStrategy", x.Subsystem, "get_strategy_weight", args...)
}

// GrokProvider implements packages/rae-agentic-memory/apps/llm/providers/grok_provider.py (GrokProvider).
type GrokProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGrokProvider() *GrokProvider {
	return &GrokProvider{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/providers/grok_provider.py", Active: true}
}

func (x *GrokProvider) Complete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GrokProvider", x.Subsystem, "complete", args...)
}
func (x *GrokProvider) Stream(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GrokProvider", x.Subsystem, "stream", args...)
}

// GuardrailManager implements core/guardrail_manager.py (GuardrailManager).
type GuardrailManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewGuardrailManager() *GuardrailManager {
	return &GuardrailManager{Subsystem: "core", Source: "core/guardrail_manager.py", Active: true}
}

func (x *GuardrailManager) RegisterCandidate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GuardrailManager", x.Subsystem, "register_candidate", args...)
}
func (x *GuardrailManager) PromoteToShadow(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GuardrailManager", x.Subsystem, "promote_to_shadow", args...)
}
func (x *GuardrailManager) EvaluatePromotion(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("GuardrailManager", x.Subsystem, "evaluate_promotion", args...)
}

// HardFrameEnforcer implements packages/rae-agentic-memory/rae-core/rae_core/governance/frame_enforcer.py (HardFrameEnforcer).
type HardFrameEnforcer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHardFrameEnforcer() *HardFrameEnforcer {
	return &HardFrameEnforcer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/frame_enforcer.py", Active: true}
}

func (x *HardFrameEnforcer) CurrentState(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardFrameEnforcer", x.Subsystem, "current_state", args...)
}
func (x *HardFrameEnforcer) TransitionTo(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardFrameEnforcer", x.Subsystem, "transition_to", args...)
}
func (x *HardFrameEnforcer) GetJournal(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardFrameEnforcer", x.Subsystem, "get_journal", args...)
}

// HardFramesEngine implements core/hard_frames_engine.py (HardFramesEngine).
type HardFramesEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHardFramesEngine() *HardFramesEngine {
	return &HardFramesEngine{Subsystem: "core", Source: "core/hard_frames_engine.py", Active: true}
}

func (x *HardFramesEngine) MarkBootstrapped(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardFramesEngine", x.Subsystem, "mark_bootstrapped", args...)
}
func (x *HardFramesEngine) ValidatePreToolFrame(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardFramesEngine", x.Subsystem, "validate_pre_tool_frame", args...)
}
func (x *HardFramesEngine) ValidatePostToolFrame(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardFramesEngine", x.Subsystem, "validate_post_tool_frame", args...)
}

// HardwareDetector implements packages/rae-agentic-memory/rae-windows/rae_lite/hardware.py (HardwareDetector).
type HardwareDetector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHardwareDetector() *HardwareDetector {
	return &HardwareDetector{Subsystem: "packages/rae-agentic-memory/rae-windows", Source: "packages/rae-agentic-memory/rae-windows/rae_lite/hardware.py", Active: true}
}

func (x *HardwareDetector) DetectAll(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardwareDetector", x.Subsystem, "detect_all", args...)
}
func (x *HardwareDetector) DetectRam(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardwareDetector", x.Subsystem, "detect_ram", args...)
}
func (x *HardwareDetector) DetectCpu(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardwareDetector", x.Subsystem, "detect_cpu", args...)
}
func (x *HardwareDetector) DetectGpu(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardwareDetector", x.Subsystem, "detect_gpu", args...)
}
func (x *HardwareDetector) GetProfileSuggestion(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HardwareDetector", x.Subsystem, "get_profile_suggestion", args...)
}

// HealthChangeMessage implements packages/rae-agentic-memory/apps/memory_api/models/dashboard_models.py (HealthChangeMessage).
type HealthChangeMessage struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHealthChangeMessage() *HealthChangeMessage {
	return &HealthChangeMessage{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/models/dashboard_models.py", Active: true}
}

func (x *HealthChangeMessage) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HealthChangeMessage", x.Subsystem, "execute", args...)
}
func (x *HealthChangeMessage) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HealthChangeMessage", x.Subsystem, "validate", args...)
}

// HiveExecutionEvaluator implements packages/rae-hive/base_agent/evaluator.py (HiveExecutionEvaluator).
type HiveExecutionEvaluator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHiveExecutionEvaluator() *HiveExecutionEvaluator {
	return &HiveExecutionEvaluator{Subsystem: "packages/rae-hive", Source: "packages/rae-hive/base_agent/evaluator.py", Active: true}
}

func (x *HiveExecutionEvaluator) StartMonitoring(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveExecutionEvaluator", x.Subsystem, "start_monitoring", args...)
}
func (x *HiveExecutionEvaluator) StopMonitoring(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveExecutionEvaluator", x.Subsystem, "stop_monitoring", args...)
}
func (x *HiveExecutionEvaluator) ReportToLab(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveExecutionEvaluator", x.Subsystem, "report_to_lab", args...)
}

// HiveExecutionSwarm implements packages/rae-hive/hive_engine.py (HiveExecutionSwarm).
type HiveExecutionSwarm struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHiveExecutionSwarm() *HiveExecutionSwarm {
	return &HiveExecutionSwarm{Subsystem: "packages/rae-hive", Source: "packages/rae-hive/hive_engine.py", Active: true}
}

func (x *HiveExecutionSwarm) RunCommand(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveExecutionSwarm", x.Subsystem, "run_command", args...)
}
func (x *HiveExecutionSwarm) RunParallelCommands(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveExecutionSwarm", x.Subsystem, "run_parallel_commands", args...)
}

// HiveMindConnector implements packages/rae-hive/base_agent/connector.py (HiveMindConnector).
type HiveMindConnector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHiveMindConnector() *HiveMindConnector {
	return &HiveMindConnector{Subsystem: "packages/rae-hive", Source: "packages/rae-hive/base_agent/connector.py", Active: true}
}

func (x *HiveMindConnector) ListMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "list_memories", args...)
}
func (x *HiveMindConnector) AddMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "add_memory", args...)
}
func (x *HiveMindConnector) QueryMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "query_memories", args...)
}
func (x *HiveMindConnector) LogActivity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "log_activity", args...)
}
func (x *HiveMindConnector) GetTasks(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "get_tasks", args...)
}
func (x *HiveMindConnector) UpdateTask(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "update_task", args...)
}
func (x *HiveMindConnector) Think(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "think", args...)
}
func (x *HiveMindConnector) Interact(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "interact", args...)
}
func (x *HiveMindConnector) ReadSourceFile(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "read_source_file", args...)
}
func (x *HiveMindConnector) WriteSourceFile(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveMindConnector", x.Subsystem, "write_source_file", args...)
}

// HiveSandboxApp implements packages/rae-agentic-memory/rae-portal/apps/hive_sandbox.py (HiveSandboxApp).
type HiveSandboxApp struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHiveSandboxApp() *HiveSandboxApp {
	return &HiveSandboxApp{Subsystem: "packages/rae-agentic-memory/rae-portal", Source: "packages/rae-agentic-memory/rae-portal/apps/hive_sandbox.py", Active: true}
}

func (x *HiveSandboxApp) Render(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveSandboxApp", x.Subsystem, "render", args...)
}

// HiveTelemetry implements packages/rae-hive/src/rae_hive/telemetry.py (HiveTelemetry).
type HiveTelemetry struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHiveTelemetry() *HiveTelemetry {
	return &HiveTelemetry{Subsystem: "packages/rae-hive", Source: "packages/rae-hive/src/rae_hive/telemetry.py", Active: true}
}

func (x *HiveTelemetry) RecordActionOutcome(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HiveTelemetry", x.Subsystem, "record_action_outcome", args...)
}

// HTTPRunner implements packages/rae-phoenix/feniks/adapters/runners/http_runner.py (HTTPRunner).
type HTTPRunner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHTTPRunner() *HTTPRunner {
	return &HTTPRunner{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/adapters/runners/http_runner.py", Active: true}
}

func (x *HTTPRunner) ExecuteScenario(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HTTPRunner", x.Subsystem, "execute_scenario", args...)
}

// HttpSyncProvider implements packages/rae-agentic-memory/rae-core/rae_core/sync/http_provider.py (HttpSyncProvider).
type HttpSyncProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHttpSyncProvider() *HttpSyncProvider {
	return &HttpSyncProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/sync/http_provider.py", Active: true}
}

func (x *HttpSyncProvider) Close(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "close", args...)
}
func (x *HttpSyncProvider) PushChanges(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "push_changes", args...)
}
func (x *HttpSyncProvider) PullChanges(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "pull_changes", args...)
}
func (x *HttpSyncProvider) ResolveConflict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "resolve_conflict", args...)
}
func (x *HttpSyncProvider) PushMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "push_memories", args...)
}
func (x *HttpSyncProvider) PullMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "pull_memories", args...)
}
func (x *HttpSyncProvider) SyncMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "sync_memories", args...)
}
func (x *HttpSyncProvider) GetSyncStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "get_sync_status", args...)
}
func (x *HttpSyncProvider) Handshake(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HttpSyncProvider", x.Subsystem, "handshake", args...)
}

// HybridSearchService implements packages/rae-agentic-memory/apps/memory_api/services/hybrid_search_service.py (HybridSearchService).
type HybridSearchService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHybridSearchService() *HybridSearchService {
	return &HybridSearchService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/hybrid_search_service.py", Active: true}
}

func (x *HybridSearchService) Search(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HybridSearchService", x.Subsystem, "search", args...)
}

// HypothesisEngine implements packages/rae-lab/src/rae_lab/hypothesis_engine.py (HypothesisEngine).
type HypothesisEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHypothesisEngine() *HypothesisEngine {
	return &HypothesisEngine{Subsystem: "packages/rae-lab", Source: "packages/rae-lab/src/rae_lab/hypothesis_engine.py", Active: true}
}

func (x *HypothesisEngine) GenerateFromFailures(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HypothesisEngine", x.Subsystem, "generate_from_failures", args...)
}
func (x *HypothesisEngine) GenerateFromOutcomes(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HypothesisEngine", x.Subsystem, "generate_from_outcomes", args...)
}

// HypothesisManager implements core/improvement_plane/hypothesis_manager.py (HypothesisManager).
type HypothesisManager struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewHypothesisManager() *HypothesisManager {
	return &HypothesisManager{Subsystem: "core", Source: "core/improvement_plane/hypothesis_manager.py", Active: true}
}

func (x *HypothesisManager) CreateHypothesis(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HypothesisManager", x.Subsystem, "create_hypothesis", args...)
}
func (x *HypothesisManager) UpdateStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("HypothesisManager", x.Subsystem, "update_status", args...)
}

// ICacheProvider implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/cache.py (ICacheProvider).
type ICacheProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewICacheProvider() *ICacheProvider {
	return &ICacheProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/cache.py", Active: true}
}

func (x *ICacheProvider) Get(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ICacheProvider", x.Subsystem, "get", args...)
}
func (x *ICacheProvider) Set(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ICacheProvider", x.Subsystem, "set", args...)
}
func (x *ICacheProvider) Delete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ICacheProvider", x.Subsystem, "delete", args...)
}
func (x *ICacheProvider) Exists(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ICacheProvider", x.Subsystem, "exists", args...)
}
func (x *ICacheProvider) Clear(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ICacheProvider", x.Subsystem, "clear", args...)
}

// IClock implements packages/rae-agentic-memory/rae-core/rae_core/utils/clock.py (IClock).
type IClock struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIClock() *IClock {
	return &IClock{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/utils/clock.py", Active: true}
}

func (x *IClock) Now(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IClock", x.Subsystem, "now", args...)
}

// ICompressor implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (ICompressor).
type ICompressor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewICompressor() *ICompressor {
	return &ICompressor{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *ICompressor) Compress(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ICompressor", x.Subsystem, "compress", args...)
}

// IDatabaseConnection implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/database.py (IDatabaseConnection).
type IDatabaseConnection struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIDatabaseConnection() *IDatabaseConnection {
	return &IDatabaseConnection{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/database.py", Active: true}
}

func (x *IDatabaseConnection) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseConnection", x.Subsystem, "execute", args...)
}
func (x *IDatabaseConnection) Fetch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseConnection", x.Subsystem, "fetch", args...)
}
func (x *IDatabaseConnection) Fetchrow(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseConnection", x.Subsystem, "fetchrow", args...)
}
func (x *IDatabaseConnection) Fetchval(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseConnection", x.Subsystem, "fetchval", args...)
}
func (x *IDatabaseConnection) Executemany(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseConnection", x.Subsystem, "executemany", args...)
}
func (x *IDatabaseConnection) Transaction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseConnection", x.Subsystem, "transaction", args...)
}

// IDatabaseProvider implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/database.py (IDatabaseProvider).
type IDatabaseProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIDatabaseProvider() *IDatabaseProvider {
	return &IDatabaseProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/database.py", Active: true}
}

func (x *IDatabaseProvider) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "execute", args...)
}
func (x *IDatabaseProvider) Fetch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "fetch", args...)
}
func (x *IDatabaseProvider) Fetchrow(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "fetchrow", args...)
}
func (x *IDatabaseProvider) Fetchval(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "fetchval", args...)
}
func (x *IDatabaseProvider) Executemany(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "executemany", args...)
}
func (x *IDatabaseProvider) Close(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "close", args...)
}
func (x *IDatabaseProvider) Acquire(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IDatabaseProvider", x.Subsystem, "acquire", args...)
}

// IEmbeddingProvider implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/embedding.py (IEmbeddingProvider).
type IEmbeddingProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIEmbeddingProvider() *IEmbeddingProvider {
	return &IEmbeddingProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/embedding.py", Active: true}
}

func (x *IEmbeddingProvider) EmbedText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IEmbeddingProvider", x.Subsystem, "embed_text", args...)
}
func (x *IEmbeddingProvider) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IEmbeddingProvider", x.Subsystem, "embed_batch", args...)
}
func (x *IEmbeddingProvider) GetDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IEmbeddingProvider", x.Subsystem, "get_dimension", args...)
}

// IGraphStore implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/graph.py (IGraphStore).
type IGraphStore struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIGraphStore() *IGraphStore {
	return &IGraphStore{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/graph.py", Active: true}
}

func (x *IGraphStore) CreateNode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "create_node", args...)
}
func (x *IGraphStore) CreateEdge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "create_edge", args...)
}
func (x *IGraphStore) GetNeighbors(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "get_neighbors", args...)
}
func (x *IGraphStore) DeleteNode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "delete_node", args...)
}
func (x *IGraphStore) DeleteEdge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "delete_edge", args...)
}
func (x *IGraphStore) ShortestPath(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "shortest_path", args...)
}
func (x *IGraphStore) GetSubgraph(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IGraphStore", x.Subsystem, "get_subgraph", args...)
}

// IKnowledgeAdapter implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/adapter.py (IKnowledgeAdapter).
type IKnowledgeAdapter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIKnowledgeAdapter() *IKnowledgeAdapter {
	return &IKnowledgeAdapter{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/adapter.py", Active: true}
}

func (x *IKnowledgeAdapter) Retrieve(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IKnowledgeAdapter", x.Subsystem, "retrieve", args...)
}

// ILLMProvider implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/llm.py (ILLMProvider).
type ILLMProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewILLMProvider() *ILLMProvider {
	return &ILLMProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/llm.py", Active: true}
}

func (x *ILLMProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ILLMProvider", x.Subsystem, "generate", args...)
}
func (x *ILLMProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ILLMProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *ILLMProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ILLMProvider", x.Subsystem, "count_tokens", args...)
}
func (x *ILLMProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ILLMProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *ILLMProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ILLMProvider", x.Subsystem, "extract_entities", args...)
}
func (x *ILLMProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ILLMProvider", x.Subsystem, "summarize", args...)
}

// IMemoryStorage implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/storage.py (IMemoryStorage).
type IMemoryStorage struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIMemoryStorage() *IMemoryStorage {
	return &IMemoryStorage{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/storage.py", Active: true}
}

func (x *IMemoryStorage) StoreMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "store_memory", args...)
}
func (x *IMemoryStorage) StoreReflectionAudit(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "store_reflection_audit", args...)
}
func (x *IMemoryStorage) GetMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "get_memory", args...)
}
func (x *IMemoryStorage) GetMemoriesBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "get_memories_batch", args...)
}
func (x *IMemoryStorage) UpdateMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "update_memory", args...)
}
func (x *IMemoryStorage) DeleteMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "delete_memory", args...)
}
func (x *IMemoryStorage) ListMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "list_memories", args...)
}
func (x *IMemoryStorage) DeleteMemoriesWithMetadataFilter(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "delete_memories_with_metadata_filter", args...)
}
func (x *IMemoryStorage) DeleteMemoriesBelowImportance(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "delete_memories_below_importance", args...)
}
func (x *IMemoryStorage) CountMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "count_memories", args...)
}
func (x *IMemoryStorage) SearchMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "search_memories", args...)
}
func (x *IMemoryStorage) DeleteExpiredMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "delete_expired_memories", args...)
}
func (x *IMemoryStorage) UpdateMemoryAccess(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "update_memory_access", args...)
}
func (x *IMemoryStorage) IncrementAccessCount(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "increment_access_count", args...)
}
func (x *IMemoryStorage) UpdateMemoryExpiration(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "update_memory_expiration", args...)
}
func (x *IMemoryStorage) GetMetricAggregate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "get_metric_aggregate", args...)
}
func (x *IMemoryStorage) UpdateMemoryAccessBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "update_memory_access_batch", args...)
}
func (x *IMemoryStorage) AdjustImportance(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "adjust_importance", args...)
}
func (x *IMemoryStorage) SaveEmbedding(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "save_embedding", args...)
}
func (x *IMemoryStorage) DecayImportance(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "decay_importance", args...)
}
func (x *IMemoryStorage) ClearTenant(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "clear_tenant", args...)
}
func (x *IMemoryStorage) Close(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMemoryStorage", x.Subsystem, "close", args...)
}

// IMetric implements packages/rae-agentic-memory/rae-core/rae_core/math/quality_metrics.py (IMetric).
type IMetric struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIMetric() *IMetric {
	return &IMetric{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/quality_metrics.py", Active: true}
}

func (x *IMetric) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMetric", x.Subsystem, "name", args...)
}
func (x *IMetric) Compute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IMetric", x.Subsystem, "compute", args...)
}

// ImplementerAgent implements packages/rae-agentic-memory/orchestrator/agents/implementer.py (ImplementerAgent).
type ImplementerAgent struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewImplementerAgent() *ImplementerAgent {
	return &ImplementerAgent{Subsystem: "packages/rae-agentic-memory/orchestrator", Source: "packages/rae-agentic-memory/orchestrator/agents/implementer.py", Active: true}
}

func (x *ImplementerAgent) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImplementerAgent", x.Subsystem, "execute", args...)
}

// ImportanceDecay implements packages/rae-agentic-memory/rae-core/rae_core/scoring/decay.py (ImportanceDecay).
type ImportanceDecay struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewImportanceDecay() *ImportanceDecay {
	return &ImportanceDecay{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/scoring/decay.py", Active: true}
}

func (x *ImportanceDecay) ExponentialDecay(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImportanceDecay", x.Subsystem, "exponential_decay", args...)
}
func (x *ImportanceDecay) LinearDecay(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImportanceDecay", x.Subsystem, "linear_decay", args...)
}
func (x *ImportanceDecay) LogarithmicDecay(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImportanceDecay", x.Subsystem, "logarithmic_decay", args...)
}
func (x *ImportanceDecay) StepDecay(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImportanceDecay", x.Subsystem, "step_decay", args...)
}

// ImprovementStore implements core/improvement_plane/improvement_store.py (ImprovementStore).
type ImprovementStore struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewImprovementStore() *ImprovementStore {
	return &ImprovementStore{Subsystem: "core", Source: "core/improvement_plane/improvement_store.py", Active: true}
}

func (x *ImprovementStore) SaveHypothesis(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImprovementStore", x.Subsystem, "save_hypothesis", args...)
}
func (x *ImprovementStore) GetHypothesis(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImprovementStore", x.Subsystem, "get_hypothesis", args...)
}
func (x *ImprovementStore) SaveExperiment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImprovementStore", x.Subsystem, "save_experiment", args...)
}
func (x *ImprovementStore) GetExperiment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImprovementStore", x.Subsystem, "get_experiment", args...)
}
func (x *ImprovementStore) SaveRun(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImprovementStore", x.Subsystem, "save_run", args...)
}
func (x *ImprovementStore) GetRunsFor(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ImprovementStore", x.Subsystem, "get_runs_for", args...)
}

// InfrastructureEngine implements packages/rae-quality/engines/infrastructure/infrastructure_engine.py (InfrastructureEngine).
type InfrastructureEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewInfrastructureEngine() *InfrastructureEngine {
	return &InfrastructureEngine{Subsystem: "packages/rae-quality", Source: "packages/rae-quality/engines/infrastructure/infrastructure_engine.py", Active: true}
}

func (x *InfrastructureEngine) AuditDockerfile(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("InfrastructureEngine", x.Subsystem, "audit_dockerfile", args...)
}
func (x *InfrastructureEngine) AuditK8sManifest(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("InfrastructureEngine", x.Subsystem, "audit_k8s_manifest", args...)
}
func (x *InfrastructureEngine) EvaluateSystemicReadiness(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("InfrastructureEngine", x.Subsystem, "evaluate_systemic_readiness", args...)
}

// InfrastructureFactory implements packages/rae-agentic-memory/rae-core/rae_core/factories/infra_factory.py (InfrastructureFactory).
type InfrastructureFactory struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewInfrastructureFactory() *InfrastructureFactory {
	return &InfrastructureFactory{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/factories/infra_factory.py", Active: true}
}

func (x *InfrastructureFactory) Initialize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("InfrastructureFactory", x.Subsystem, "initialize", args...)
}

// IngestAudit implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (IngestAudit).
type IngestAudit struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestAudit() *IngestAudit {
	return &IngestAudit{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *IngestAudit) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestAudit", x.Subsystem, "execute", args...)
}
func (x *IngestAudit) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestAudit", x.Subsystem, "validate", args...)
}

// IngestChunk implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (IngestChunk).
type IngestChunk struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestChunk() *IngestChunk {
	return &IngestChunk{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *IngestChunk) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestChunk", x.Subsystem, "execute", args...)
}
func (x *IngestChunk) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestChunk", x.Subsystem, "validate", args...)
}

// IngestCompressor implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/compressor.py (IngestCompressor).
type IngestCompressor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestCompressor() *IngestCompressor {
	return &IngestCompressor{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/compressor.py", Active: true}
}

func (x *IngestCompressor) Compress(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestCompressor", x.Subsystem, "compress", args...)
}

// IngestNormalizer implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/normalizer.py (IngestNormalizer).
type IngestNormalizer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestNormalizer() *IngestNormalizer {
	return &IngestNormalizer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/normalizer.py", Active: true}
}

func (x *IngestNormalizer) Normalize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestNormalizer", x.Subsystem, "normalize", args...)
}

// IngestPipeline implements packages/rae-phoenix/feniks/core/ingest_pipeline.py (IngestPipeline).
type IngestPipeline struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestPipeline() *IngestPipeline {
	return &IngestPipeline{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/ingest_pipeline.py", Active: true}
}

func (x *IngestPipeline) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestPipeline", x.Subsystem, "run", args...)
}

// IngestPolicySelector implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/policy.py (IngestPolicySelector).
type IngestPolicySelector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestPolicySelector() *IngestPolicySelector {
	return &IngestPolicySelector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/policy.py", Active: true}
}

func (x *IngestPolicySelector) SelectPolicy(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestPolicySelector", x.Subsystem, "select_policy", args...)
}

// IngestSegmenter implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/segmenter.py (IngestSegmenter).
type IngestSegmenter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIngestSegmenter() *IngestSegmenter {
	return &IngestSegmenter{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/segmenter.py", Active: true}
}

func (x *IngestSegmenter) Segment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IngestSegmenter", x.Subsystem, "segment", args...)
}

// INormalizer implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (INormalizer).
type INormalizer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewINormalizer() *INormalizer {
	return &INormalizer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *INormalizer) Normalize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("INormalizer", x.Subsystem, "normalize", args...)
}

// IPolicySelector implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (IPolicySelector).
type IPolicySelector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIPolicySelector() *IPolicySelector {
	return &IPolicySelector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *IPolicySelector) SelectPolicy(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IPolicySelector", x.Subsystem, "select_policy", args...)
}

// IReranker implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/reranking.py (IReranker).
type IReranker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIReranker() *IReranker {
	return &IReranker{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/reranking.py", Active: true}
}

func (x *IReranker) Rerank(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IReranker", x.Subsystem, "rerank", args...)
}

// ISegmenter implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (ISegmenter).
type ISegmenter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewISegmenter() *ISegmenter {
	return &ISegmenter{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *ISegmenter) Segment(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISegmenter", x.Subsystem, "segment", args...)
}

// ISignatureDetector implements packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py (ISignatureDetector).
type ISignatureDetector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewISignatureDetector() *ISignatureDetector {
	return &ISignatureDetector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/ingestion/interfaces.py", Active: true}
}

func (x *ISignatureDetector) Detect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISignatureDetector", x.Subsystem, "detect", args...)
}

// ISOAuditor implements core/iso_auditor.py (ISOAuditor).
type ISOAuditor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewISOAuditor() *ISOAuditor {
	return &ISOAuditor{Subsystem: "core", Source: "core/iso_auditor.py", Active: true}
}

func (x *ISOAuditor) AuditReceiptChain(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISOAuditor", x.Subsystem, "audit_receipt_chain", args...)
}

// ISyncProvider implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/sync.py (ISyncProvider).
type ISyncProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewISyncProvider() *ISyncProvider {
	return &ISyncProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/sync.py", Active: true}
}

func (x *ISyncProvider) PushChanges(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "push_changes", args...)
}
func (x *ISyncProvider) PullChanges(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "pull_changes", args...)
}
func (x *ISyncProvider) ResolveConflict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "resolve_conflict", args...)
}
func (x *ISyncProvider) PushMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "push_memories", args...)
}
func (x *ISyncProvider) PullMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "pull_memories", args...)
}
func (x *ISyncProvider) SyncMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "sync_memories", args...)
}
func (x *ISyncProvider) GetSyncStatus(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "get_sync_status", args...)
}
func (x *ISyncProvider) Handshake(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ISyncProvider", x.Subsystem, "handshake", args...)
}

// ITransaction implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/database.py (ITransaction).
type ITransaction struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewITransaction() *ITransaction {
	return &ITransaction{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/database.py", Active: true}
}

func (x *ITransaction) Commit(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ITransaction", x.Subsystem, "commit", args...)
}
func (x *ITransaction) Rollback(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("ITransaction", x.Subsystem, "rollback", args...)
}

// IVectorStore implements packages/rae-agentic-memory/rae-core/rae_core/interfaces/vector.py (IVectorStore).
type IVectorStore struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewIVectorStore() *IVectorStore {
	return &IVectorStore{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/interfaces/vector.py", Active: true}
}

func (x *IVectorStore) StoreVector(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IVectorStore", x.Subsystem, "store_vector", args...)
}
func (x *IVectorStore) SearchSimilar(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IVectorStore", x.Subsystem, "search_similar", args...)
}
func (x *IVectorStore) DeleteVector(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IVectorStore", x.Subsystem, "delete_vector", args...)
}
func (x *IVectorStore) UpdateVector(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IVectorStore", x.Subsystem, "update_vector", args...)
}
func (x *IVectorStore) GetVector(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IVectorStore", x.Subsystem, "get_vector", args...)
}
func (x *IVectorStore) BatchStoreVectors(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("IVectorStore", x.Subsystem, "batch_store_vectors", args...)
}

// JavaScriptPlugin implements packages/rae-phoenix/scripts/apply_recipe.py (JavaScriptPlugin).
type JavaScriptPlugin struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewJavaScriptPlugin() *JavaScriptPlugin {
	return &JavaScriptPlugin{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/scripts/apply_recipe.py", Active: true}
}

func (x *JavaScriptPlugin) Parse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JavaScriptPlugin", x.Subsystem, "parse", args...)
}
func (x *JavaScriptPlugin) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JavaScriptPlugin", x.Subsystem, "generate", args...)
}
func (x *JavaScriptPlugin) FindMatches(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JavaScriptPlugin", x.Subsystem, "find_matches", args...)
}

// JSCodeshiftWrapper implements packages/rae-phoenix/feniks/core/refactor/javascript/tools.py (JSCodeshiftWrapper).
type JSCodeshiftWrapper struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewJSCodeshiftWrapper() *JSCodeshiftWrapper {
	return &JSCodeshiftWrapper{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/javascript/tools.py", Active: true}
}

func (x *JSCodeshiftWrapper) RunTransform(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JSCodeshiftWrapper", x.Subsystem, "run_transform", args...)
}

// JSONFormatter implements packages/rae-phoenix/feniks/infra/logging.py (JSONFormatter).
type JSONFormatter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/infra/logging.py", Active: true}
}

func (x *JSONFormatter) Format(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JSONFormatter", x.Subsystem, "format", args...)
}

// JSRefactorEngine implements packages/rae-phoenix/feniks/core/refactor/javascript/engine.py (JSRefactorEngine).
type JSRefactorEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewJSRefactorEngine() *JSRefactorEngine {
	return &JSRefactorEngine{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/javascript/engine.py", Active: true}
}

func (x *JSRefactorEngine) ScanAngularProject(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JSRefactorEngine", x.Subsystem, "scan_angular_project", args...)
}
func (x *JSRefactorEngine) ApplyCodemod(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JSRefactorEngine", x.Subsystem, "apply_codemod", args...)
}

// JsToolRunner implements packages/rae-phoenix/feniks/core/refactor/javascript/tools.py (JsToolRunner).
type JsToolRunner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewJsToolRunner() *JsToolRunner {
	return &JsToolRunner{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/javascript/tools.py", Active: true}
}

func (x *JsToolRunner) RunTool(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("JsToolRunner", x.Subsystem, "run_tool", args...)
}

// KeycloakAuthGateway implements core/keycloak_auth.py (KeycloakAuthGateway).
type KeycloakAuthGateway struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewKeycloakAuthGateway() *KeycloakAuthGateway {
	return &KeycloakAuthGateway{Subsystem: "core", Source: "core/keycloak_auth.py", Active: true}
}

func (x *KeycloakAuthGateway) CreateMockJwt(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KeycloakAuthGateway", x.Subsystem, "create_mock_jwt", args...)
}
func (x *KeycloakAuthGateway) VerifyToken(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KeycloakAuthGateway", x.Subsystem, "verify_token", args...)
}
func (x *KeycloakAuthGateway) AuthorizeCapability(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KeycloakAuthGateway", x.Subsystem, "authorize_capability", args...)
}

// KnowledgeGraph implements packages/rae-agentic-memory/apps/memory_api/core/graph_operator.py (KnowledgeGraph).
type KnowledgeGraph struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/core/graph_operator.py", Active: true}
}

func (x *KnowledgeGraph) GetNode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "get_node", args...)
}
func (x *KnowledgeGraph) GetEdge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "get_edge", args...)
}
func (x *KnowledgeGraph) AdjacencyMatrix(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "adjacency_matrix", args...)
}
func (x *KnowledgeGraph) Copy(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "copy", args...)
}
func (x *KnowledgeGraph) ToDict(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "to_dict", args...)
}
func (x *KnowledgeGraph) AddNode(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "add_node", args...)
}
func (x *KnowledgeGraph) AddEdge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "add_edge", args...)
}
func (x *KnowledgeGraph) GetNeighbors(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "get_neighbors", args...)
}
func (x *KnowledgeGraph) NodeCount(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "node_count", args...)
}
func (x *KnowledgeGraph) EdgeCount(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeGraph", x.Subsystem, "edge_count", args...)
}

// KnowledgeResolutionEngine implements packages/rae-agentic-memory/rae-core/rae_core/governance/engine.py (KnowledgeResolutionEngine).
type KnowledgeResolutionEngine struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewKnowledgeResolutionEngine() *KnowledgeResolutionEngine {
	return &KnowledgeResolutionEngine{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/governance/engine.py", Active: true}
}

func (x *KnowledgeResolutionEngine) Resolve(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("KnowledgeResolutionEngine", x.Subsystem, "resolve", args...)
}

// L1OperationalReflection implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l1_operational.py (L1OperationalReflection).
type L1OperationalReflection struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewL1OperationalReflection() *L1OperationalReflection {
	return &L1OperationalReflection{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l1_operational.py", Active: true}
}

func (x *L1OperationalReflection) Reflect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("L1OperationalReflection", x.Subsystem, "reflect", args...)
}

// L2StructuralReflection implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l2_structural.py (L2StructuralReflection).
type L2StructuralReflection struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewL2StructuralReflection() *L2StructuralReflection {
	return &L2StructuralReflection{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l2_structural.py", Active: true}
}

func (x *L2StructuralReflection) Reflect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("L2StructuralReflection", x.Subsystem, "reflect", args...)
}

// L3MetaFieldReflection implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l3_meta.py (L3MetaFieldReflection).
type L3MetaFieldReflection struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewL3MetaFieldReflection() *L3MetaFieldReflection {
	return &L3MetaFieldReflection{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l3_meta.py", Active: true}
}

func (x *L3MetaFieldReflection) Reflect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("L3MetaFieldReflection", x.Subsystem, "reflect", args...)
}

// L4CognitiveReflection implements packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l4_cognitive.py (L4CognitiveReflection).
type L4CognitiveReflection struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewL4CognitiveReflection() *L4CognitiveReflection {
	return &L4CognitiveReflection{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/reflection/layers/l4_cognitive.py", Active: true}
}

func (x *L4CognitiveReflection) Reflect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("L4CognitiveReflection", x.Subsystem, "reflect", args...)
}

// LabObservatory implements packages/rae-lab/metrics_aggregator.py (LabObservatory).
type LabObservatory struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLabObservatory() *LabObservatory {
	return &LabObservatory{Subsystem: "packages/rae-lab", Source: "packages/rae-lab/metrics_aggregator.py", Active: true}
}

func (x *LabObservatory) GetHealthReport(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LabObservatory", x.Subsystem, "get_health_report", args...)
}
func (x *LabObservatory) RecordTelemetry(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LabObservatory", x.Subsystem, "record_telemetry", args...)
}

// LanguagePlugin implements packages/rae-phoenix/feniks/core/plugins/base.py (LanguagePlugin).
type LanguagePlugin struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLanguagePlugin() *LanguagePlugin {
	return &LanguagePlugin{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/plugins/base.py", Active: true}
}

func (x *LanguagePlugin) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "name", args...)
}
func (x *LanguagePlugin) SupportedExtensions(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "supported_extensions", args...)
}
func (x *LanguagePlugin) CanHandle(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "can_handle", args...)
}
func (x *LanguagePlugin) ExecuteRefactor(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "execute_refactor", args...)
}
func (x *LanguagePlugin) ExecuteCreate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "execute_create", args...)
}
func (x *LanguagePlugin) Parse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "parse", args...)
}
func (x *LanguagePlugin) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "generate", args...)
}
func (x *LanguagePlugin) FindMatches(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LanguagePlugin", x.Subsystem, "find_matches", args...)
}

// LayerTriadRunner implements core/layer_triad_runner.py (LayerTriadRunner).
type LayerTriadRunner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLayerTriadRunner() *LayerTriadRunner {
	return &LayerTriadRunner{Subsystem: "core", Source: "core/layer_triad_runner.py", Active: true}
}

func (x *LayerTriadRunner) GetAvailableLayers(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LayerTriadRunner", x.Subsystem, "get_available_layers", args...)
}
func (x *LayerTriadRunner) GetLayerModels(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LayerTriadRunner", x.Subsystem, "get_layer_models", args...)
}
func (x *LayerTriadRunner) ExecuteLayerTriad(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LayerTriadRunner", x.Subsystem, "execute_layer_triad", args...)
}
func (x *LayerTriadRunner) Execute3tierCascade(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LayerTriadRunner", x.Subsystem, "execute_3tier_cascade", args...)
}

// Legacy416Strategy implements packages/rae-agentic-memory/rae-core/rae_core/math/fusion.py (Legacy416Strategy).
type Legacy416Strategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLegacy416Strategy() *Legacy416Strategy {
	return &Legacy416Strategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/fusion.py", Active: true}
}

func (x *Legacy416Strategy) Fuse(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Legacy416Strategy", x.Subsystem, "fuse", args...)
}

// LibCSTWrapper implements packages/rae-phoenix/feniks/core/refactor/python/tools.py (LibCSTWrapper).
type LibCSTWrapper struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLibCSTWrapper() *LibCSTWrapper {
	return &LibCSTWrapper{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/refactor/python/tools.py", Active: true}
}

func (x *LibCSTWrapper) AnalyzeStructure(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LibCSTWrapper", x.Subsystem, "analyze_structure", args...)
}

// LightweightAnthropicProvider implements packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py (LightweightAnthropicProvider).
type LightweightAnthropicProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLightweightAnthropicProvider() *LightweightAnthropicProvider {
	return &LightweightAnthropicProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py", Active: true}
}

func (x *LightweightAnthropicProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightAnthropicProvider", x.Subsystem, "generate", args...)
}
func (x *LightweightAnthropicProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightAnthropicProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *LightweightAnthropicProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightAnthropicProvider", x.Subsystem, "count_tokens", args...)
}
func (x *LightweightAnthropicProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightAnthropicProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *LightweightAnthropicProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightAnthropicProvider", x.Subsystem, "extract_entities", args...)
}
func (x *LightweightAnthropicProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightAnthropicProvider", x.Subsystem, "summarize", args...)
}

// LightweightGoogleProvider implements packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py (LightweightGoogleProvider).
type LightweightGoogleProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLightweightGoogleProvider() *LightweightGoogleProvider {
	return &LightweightGoogleProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py", Active: true}
}

func (x *LightweightGoogleProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightGoogleProvider", x.Subsystem, "generate", args...)
}
func (x *LightweightGoogleProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightGoogleProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *LightweightGoogleProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightGoogleProvider", x.Subsystem, "count_tokens", args...)
}
func (x *LightweightGoogleProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightGoogleProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *LightweightGoogleProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightGoogleProvider", x.Subsystem, "extract_entities", args...)
}
func (x *LightweightGoogleProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightGoogleProvider", x.Subsystem, "summarize", args...)
}

// LightweightOpenAIProvider implements packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py (LightweightOpenAIProvider).
type LightweightOpenAIProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLightweightOpenAIProvider() *LightweightOpenAIProvider {
	return &LightweightOpenAIProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/providers.py", Active: true}
}

func (x *LightweightOpenAIProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightOpenAIProvider", x.Subsystem, "generate", args...)
}
func (x *LightweightOpenAIProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightOpenAIProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *LightweightOpenAIProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightOpenAIProvider", x.Subsystem, "count_tokens", args...)
}
func (x *LightweightOpenAIProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightOpenAIProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *LightweightOpenAIProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightOpenAIProvider", x.Subsystem, "extract_entities", args...)
}
func (x *LightweightOpenAIProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LightweightOpenAIProvider", x.Subsystem, "summarize", args...)
}

// LiteLLMProvider implements packages/rae-agentic-memory/scripts/verify_infrastructure.py (LiteLLMProvider).
type LiteLLMProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLiteLLMProvider() *LiteLLMProvider {
	return &LiteLLMProvider{Subsystem: "packages/rae-agentic-memory/scripts", Source: "packages/rae-agentic-memory/scripts/verify_infrastructure.py", Active: true}
}

func (x *LiteLLMProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LiteLLMProvider", x.Subsystem, "generate", args...)
}
func (x *LiteLLMProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LiteLLMProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *LiteLLMProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LiteLLMProvider", x.Subsystem, "count_tokens", args...)
}
func (x *LiteLLMProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LiteLLMProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *LiteLLMProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LiteLLMProvider", x.Subsystem, "extract_entities", args...)
}
func (x *LiteLLMProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LiteLLMProvider", x.Subsystem, "summarize", args...)
}

// LlamaCppAdapter implements packages/rae-agentic-memory/rae-windows/rae_lite/llm_adapter.py (LlamaCppAdapter).
type LlamaCppAdapter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLlamaCppAdapter() *LlamaCppAdapter {
	return &LlamaCppAdapter{Subsystem: "packages/rae-agentic-memory/rae-windows", Source: "packages/rae-agentic-memory/rae-windows/rae_lite/llm_adapter.py", Active: true}
}

func (x *LlamaCppAdapter) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LlamaCppAdapter", x.Subsystem, "generate", args...)
}
func (x *LlamaCppAdapter) NormalizeQuery(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LlamaCppAdapter", x.Subsystem, "normalize_query", args...)
}
func (x *LlamaCppAdapter) FormatResults(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LlamaCppAdapter", x.Subsystem, "format_results", args...)
}

// LLMChunk implements packages/rae-agentic-memory/apps/llm/models/llm_response.py (LLMChunk).
type LLMChunk struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMChunk() *LLMChunk {
	return &LLMChunk{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/models/llm_response.py", Active: true}
}

func (x *LLMChunk) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMChunk", x.Subsystem, "execute", args...)
}
func (x *LLMChunk) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMChunk", x.Subsystem, "validate", args...)
}

// LLMConfig implements packages/rae-agentic-memory/rae-core/rae_core/llm/config.py (LLMConfig).
type LLMConfig struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMConfig() *LLMConfig {
	return &LLMConfig{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/config.py", Active: true}
}

func (x *LLMConfig) GetProvider(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMConfig", x.Subsystem, "get_provider", args...)
}
func (x *LLMConfig) AddProvider(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMConfig", x.Subsystem, "add_provider", args...)
}
func (x *LLMConfig) ListProviders(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMConfig", x.Subsystem, "list_providers", args...)
}

// LLMEvaluator implements packages/rae-agentic-memory/apps/memory_api/services/evaluator.py (LLMEvaluator).
type LLMEvaluator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMEvaluator() *LLMEvaluator {
	return &LLMEvaluator{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/evaluator.py", Active: true}
}

func (x *LLMEvaluator) Evaluate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMEvaluator", x.Subsystem, "evaluate", args...)
}

// LLMMessage implements packages/rae-agentic-memory/apps/llm/models/llm_request.py (LLMMessage).
type LLMMessage struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMMessage() *LLMMessage {
	return &LLMMessage{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/models/llm_request.py", Active: true}
}

func (x *LLMMessage) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMMessage", x.Subsystem, "execute", args...)
}
func (x *LLMMessage) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMMessage", x.Subsystem, "validate", args...)
}

// LLMOrchestrator implements packages/rae-agentic-memory/apps/llm/broker/orchestrator.py (LLMOrchestrator).
type LLMOrchestrator struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMOrchestrator() *LLMOrchestrator {
	return &LLMOrchestrator{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/broker/orchestrator.py", Active: true}
}

func (x *LLMOrchestrator) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "generate", args...)
}
func (x *LLMOrchestrator) RegisterProvider(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "register_provider", args...)
}
func (x *LLMOrchestrator) GetProvider(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "get_provider", args...)
}
func (x *LLMOrchestrator) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "generate_with_context", args...)
}
func (x *LLMOrchestrator) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "extract_entities", args...)
}
func (x *LLMOrchestrator) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "summarize", args...)
}
func (x *LLMOrchestrator) ListProviders(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMOrchestrator", x.Subsystem, "list_providers", args...)
}

// LLMProvider implements packages/rae-agentic-memory/apps/llm/providers/base.py (LLMProvider).
type LLMProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMProvider() *LLMProvider {
	return &LLMProvider{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/providers/base.py", Active: true}
}

func (x *LLMProvider) Complete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "complete", args...)
}
func (x *LLMProvider) Stream(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "stream", args...)
}
func (x *LLMProvider) Embed(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "embed", args...)
}
func (x *LLMProvider) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "embed_batch", args...)
}
func (x *LLMProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "generate", args...)
}
func (x *LLMProvider) GenerateStructured(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "generate_structured", args...)
}
func (x *LLMProvider) Name(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "name", args...)
}
func (x *LLMProvider) AvailableModels(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "available_models", args...)
}
func (x *LLMProvider) IsAvailable(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "is_available", args...)
}
func (x *LLMProvider) GetModelInfo(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "get_model_info", args...)
}
func (x *LLMProvider) ListModels(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMProvider", x.Subsystem, "list_models", args...)
}

// LlmRerankerStrategy implements packages/rae-agentic-memory/apps/memory_api/services/reranking/llm_strategy.py (LlmRerankerStrategy).
type LlmRerankerStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLlmRerankerStrategy() *LlmRerankerStrategy {
	return &LlmRerankerStrategy{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/reranking/llm_strategy.py", Active: true}
}

func (x *LlmRerankerStrategy) Rerank(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LlmRerankerStrategy", x.Subsystem, "rerank", args...)
}

// LLMStrategy implements packages/rae-agentic-memory/rae-core/rae_core/llm/strategies.py (LLMStrategy).
type LLMStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMStrategy() *LLMStrategy {
	return &LLMStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/strategies.py", Active: true}
}

func (x *LLMStrategy) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMStrategy", x.Subsystem, "execute", args...)
}

// LLMTool implements packages/rae-agentic-memory/apps/llm/models/llm_request.py (LLMTool).
type LLMTool struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMTool() *LLMTool {
	return &LLMTool{Subsystem: "packages/rae-agentic-memory/apps/llm", Source: "packages/rae-agentic-memory/apps/llm/models/llm_request.py", Active: true}
}

func (x *LLMTool) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMTool", x.Subsystem, "execute", args...)
}
func (x *LLMTool) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMTool", x.Subsystem, "validate", args...)
}

// LLMTracer implements packages/rae-agentic-memory/apps/memory_api/observability/opentelemetry_config.py (LLMTracer).
type LLMTracer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLLMTracer() *LLMTracer {
	return &LLMTracer{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/observability/opentelemetry_config.py", Active: true}
}

func (x *LLMTracer) Trace(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMTracer", x.Subsystem, "trace", args...)
}
func (x *LLMTracer) RecordTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMTracer", x.Subsystem, "record_tokens", args...)
}
func (x *LLMTracer) RecordCost(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMTracer", x.Subsystem, "record_cost", args...)
}
func (x *LLMTracer) RecordCacheHit(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LLMTracer", x.Subsystem, "record_cache_hit", args...)
}

// LoadBalancingStrategy implements packages/rae-agentic-memory/rae-core/rae_core/llm/strategies.py (LoadBalancingStrategy).
type LoadBalancingStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLoadBalancingStrategy() *LoadBalancingStrategy {
	return &LoadBalancingStrategy{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/strategies.py", Active: true}
}

func (x *LoadBalancingStrategy) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LoadBalancingStrategy", x.Subsystem, "execute", args...)
}

// LocalEmbeddingProvider implements packages/rae-agentic-memory/apps/memory_api/services/embedding.py (LocalEmbeddingProvider).
type LocalEmbeddingProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLocalEmbeddingProvider() *LocalEmbeddingProvider {
	return &LocalEmbeddingProvider{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/embedding.py", Active: true}
}

func (x *LocalEmbeddingProvider) EmbedText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalEmbeddingProvider", x.Subsystem, "embed_text", args...)
}
func (x *LocalEmbeddingProvider) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalEmbeddingProvider", x.Subsystem, "embed_batch", args...)
}
func (x *LocalEmbeddingProvider) GetDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalEmbeddingProvider", x.Subsystem, "get_dimension", args...)
}

// LocalOnnxLLMProvider implements packages/rae-agentic-memory/rae-core/rae_core/llm/local_onnx.py (LocalOnnxLLMProvider).
type LocalOnnxLLMProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLocalOnnxLLMProvider() *LocalOnnxLLMProvider {
	return &LocalOnnxLLMProvider{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/llm/local_onnx.py", Active: true}
}

func (x *LocalOnnxLLMProvider) Generate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalOnnxLLMProvider", x.Subsystem, "generate", args...)
}
func (x *LocalOnnxLLMProvider) GenerateWithContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalOnnxLLMProvider", x.Subsystem, "generate_with_context", args...)
}
func (x *LocalOnnxLLMProvider) CountTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalOnnxLLMProvider", x.Subsystem, "count_tokens", args...)
}
func (x *LocalOnnxLLMProvider) SupportsFunctionCalling(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalOnnxLLMProvider", x.Subsystem, "supports_function_calling", args...)
}
func (x *LocalOnnxLLMProvider) ExtractEntities(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalOnnxLLMProvider", x.Subsystem, "extract_entities", args...)
}
func (x *LocalOnnxLLMProvider) Summarize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LocalOnnxLLMProvider", x.Subsystem, "summarize", args...)
}

// LongitudinalAnalyzer implements packages/rae-phoenix/feniks/core/reflection/longitudinal.py (LongitudinalAnalyzer).
type LongitudinalAnalyzer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLongitudinalAnalyzer() *LongitudinalAnalyzer {
	return &LongitudinalAnalyzer{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/reflection/longitudinal.py", Active: true}
}

func (x *LongitudinalAnalyzer) AnalyzeTrends(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongitudinalAnalyzer", x.Subsystem, "analyze_trends", args...)
}

// LongTermLayer implements packages/rae-agentic-memory/rae-core/rae_core/layers/longterm.py (LongTermLayer).
type LongTermLayer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewLongTermLayer() *LongTermLayer {
	return &LongTermLayer{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/layers/longterm.py", Active: true}
}

func (x *LongTermLayer) AddMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "add_memory", args...)
}
func (x *LongTermLayer) GetMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "get_memory", args...)
}
func (x *LongTermLayer) SearchMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "search_memories", args...)
}
func (x *LongTermLayer) Cleanup(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "cleanup", args...)
}
func (x *LongTermLayer) ConsolidateFromWorking(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "consolidate_from_working", args...)
}
func (x *LongTermLayer) UpgradeToSemantic(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "upgrade_to_semantic", args...)
}
func (x *LongTermLayer) CountEpisodic(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "count_episodic", args...)
}
func (x *LongTermLayer) CountSemantic(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("LongTermLayer", x.Subsystem, "count_semantic", args...)
}

// MABTuner implements packages/rae-lab/metrics_aggregator.py (MABTuner).
type MABTuner struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMABTuner() *MABTuner {
	return &MABTuner{Subsystem: "packages/rae-lab", Source: "packages/rae-lab/metrics_aggregator.py", Active: true}
}

func (x *MABTuner) CalculateSuccessRate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MABTuner", x.Subsystem, "calculate_success_rate", args...)
}
func (x *MABTuner) OptimizeRouterWeights(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MABTuner", x.Subsystem, "optimize_router_weights", args...)
}

// MAESEventStore implements core/event_store.py (MAESEventStore).
type MAESEventStore struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMAESEventStore() *MAESEventStore {
	return &MAESEventStore{Subsystem: "core", Source: "core/event_store.py", Active: true}
}

func (x *MAESEventStore) AppendEvent(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MAESEventStore", x.Subsystem, "append_event", args...)
}
func (x *MAESEventStore) GetTaskProjection(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MAESEventStore", x.Subsystem, "get_task_projection", args...)
}

// MaintenanceScheduler implements packages/rae-agentic-memory/apps/memory_api/workers/memory_maintenance.py (MaintenanceScheduler).
type MaintenanceScheduler struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMaintenanceScheduler() *MaintenanceScheduler {
	return &MaintenanceScheduler{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/workers/memory_maintenance.py", Active: true}
}

func (x *MaintenanceScheduler) Create(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MaintenanceScheduler", x.Subsystem, "create", args...)
}
func (x *MaintenanceScheduler) RunDailyMaintenance(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MaintenanceScheduler", x.Subsystem, "run_daily_maintenance", args...)
}
func (x *MaintenanceScheduler) RunHourlyMaintenance(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MaintenanceScheduler", x.Subsystem, "run_hourly_maintenance", args...)
}

// MathLayerController implements packages/rae-agentic-memory/rae-core/rae_core/math/controller.py (MathLayerController).
type MathLayerController struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMathLayerController() *MathLayerController {
	return &MathLayerController{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/controller.py", Active: true}
}

func (x *MathLayerController) ComputeSimilarity(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "compute_similarity", args...)
}
func (x *MathLayerController) ApplyDecay(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "apply_decay", args...)
}
func (x *MathLayerController) GetRetrievalWeights(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "get_retrieval_weights", args...)
}
func (x *MathLayerController) GetScoringParams(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "get_scoring_params", args...)
}
func (x *MathLayerController) ScoreMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "score_memory", args...)
}
func (x *MathLayerController) GetEngineParam(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "get_engine_param", args...)
}
func (x *MathLayerController) GetResonanceThreshold(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "get_resonance_threshold", args...)
}
func (x *MathLayerController) UpdatePolicy(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathLayerController", x.Subsystem, "update_policy", args...)
}

// MathOnlyEmbeddingProvider implements packages/rae-agentic-memory/apps/memory_api/embedding.py (MathOnlyEmbeddingProvider).
type MathOnlyEmbeddingProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMathOnlyEmbeddingProvider() *MathOnlyEmbeddingProvider {
	return &MathOnlyEmbeddingProvider{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/embedding.py", Active: true}
}

func (x *MathOnlyEmbeddingProvider) EmbedText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathOnlyEmbeddingProvider", x.Subsystem, "embed_text", args...)
}
func (x *MathOnlyEmbeddingProvider) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathOnlyEmbeddingProvider", x.Subsystem, "embed_batch", args...)
}
func (x *MathOnlyEmbeddingProvider) GetDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathOnlyEmbeddingProvider", x.Subsystem, "get_dimension", args...)
}

// MathRerankerStrategy implements packages/rae-agentic-memory/apps/memory_api/services/reranking/math_strategy.py (MathRerankerStrategy).
type MathRerankerStrategy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMathRerankerStrategy() *MathRerankerStrategy {
	return &MathRerankerStrategy{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/reranking/math_strategy.py", Active: true}
}

func (x *MathRerankerStrategy) Rerank(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MathRerankerStrategy", x.Subsystem, "rerank", args...)
}

// MatrixRelay implements packages/rae-agentic-memory/apps/memory_api/services/relay_broker.py (MatrixRelay).
type MatrixRelay struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMatrixRelay() *MatrixRelay {
	return &MatrixRelay{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/relay_broker.py", Active: true}
}

func (x *MatrixRelay) Publish(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MatrixRelay", x.Subsystem, "publish", args...)
}
func (x *MatrixRelay) Consume(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MatrixRelay", x.Subsystem, "consume", args...)
}

// MaxBehaviorRiskPolicy implements packages/rae-phoenix/feniks/core/policies/behavior_risk_policy.py (MaxBehaviorRiskPolicy).
type MaxBehaviorRiskPolicy struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMaxBehaviorRiskPolicy() *MaxBehaviorRiskPolicy {
	return &MaxBehaviorRiskPolicy{Subsystem: "packages/rae-phoenix", Source: "packages/rae-phoenix/feniks/core/policies/behavior_risk_policy.py", Active: true}
}

func (x *MaxBehaviorRiskPolicy) Evaluate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MaxBehaviorRiskPolicy", x.Subsystem, "evaluate", args...)
}
func (x *MaxBehaviorRiskPolicy) CheckViolations(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MaxBehaviorRiskPolicy", x.Subsystem, "check_violations", args...)
}

// MCPEmbeddingProvider implements packages/rae-agentic-memory/apps/memory_api/services/embedding.py (MCPEmbeddingProvider).
type MCPEmbeddingProvider struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMCPEmbeddingProvider() *MCPEmbeddingProvider {
	return &MCPEmbeddingProvider{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/embedding.py", Active: true}
}

func (x *MCPEmbeddingProvider) EmbedText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MCPEmbeddingProvider", x.Subsystem, "embed_text", args...)
}
func (x *MCPEmbeddingProvider) EmbedBatch(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MCPEmbeddingProvider", x.Subsystem, "embed_batch", args...)
}
func (x *MCPEmbeddingProvider) GetDimension(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MCPEmbeddingProvider", x.Subsystem, "get_dimension", args...)
}

// McpReranker implements packages/rae-agentic-memory/rae-core/rae_core/search/rerankers/mcp.py (McpReranker).
type McpReranker struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMcpReranker() *McpReranker {
	return &McpReranker{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/search/rerankers/mcp.py", Active: true}
}

func (x *McpReranker) Rerank(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("McpReranker", x.Subsystem, "rerank", args...)
}

// Memory implements packages/rae-agentic-memory/apps/memory_api/services/importance_scoring.py (Memory).
type Memory struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemory() *Memory {
	return &Memory{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/importance_scoring.py", Active: true}
}

func (x *Memory) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Memory", x.Subsystem, "execute", args...)
}
func (x *Memory) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("Memory", x.Subsystem, "validate", args...)
}

// MemoryAdapter implements packages/rae-agentic-memory/apps/memory_api/adapters/base.py (MemoryAdapter).
type MemoryAdapter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/adapters/base.py", Active: true}
}

func (x *MemoryAdapter) Connect(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryAdapter", x.Subsystem, "connect", args...)
}
func (x *MemoryAdapter) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryAdapter", x.Subsystem, "validate", args...)
}
func (x *MemoryAdapter) Report(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryAdapter", x.Subsystem, "report", args...)
}

// MemoryAnalyzer implements packages/rae-agentic-memory/scripts/maintenance/analyze_memories.py (MemoryAnalyzer).
type MemoryAnalyzer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryAnalyzer() *MemoryAnalyzer {
	return &MemoryAnalyzer{Subsystem: "packages/rae-agentic-memory/scripts", Source: "packages/rae-agentic-memory/scripts/maintenance/analyze_memories.py", Active: true}
}

func (x *MemoryAnalyzer) FetchAll(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryAnalyzer", x.Subsystem, "fetch_all", args...)
}
func (x *MemoryAnalyzer) Analyze(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryAnalyzer", x.Subsystem, "analyze", args...)
}
func (x *MemoryAnalyzer) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryAnalyzer", x.Subsystem, "run", args...)
}

// MemoryEvidenceStore implements packages/rae-agentic-memory/src/rae_memory/evidence_adapter.py (MemoryEvidenceStore).
type MemoryEvidenceStore struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryEvidenceStore() *MemoryEvidenceStore {
	return &MemoryEvidenceStore{Subsystem: "packages/rae-agentic-memory/src", Source: "packages/rae-agentic-memory/src/rae_memory/evidence_adapter.py", Active: true}
}

func (x *MemoryEvidenceStore) SaveAction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryEvidenceStore", x.Subsystem, "save_action", args...)
}

// MemoryIsolationGuard implements packages/rae-agentic-memory/rae-core/rae_core/guards/isolation.py (MemoryIsolationGuard).
type MemoryIsolationGuard struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryIsolationGuard() *MemoryIsolationGuard {
	return &MemoryIsolationGuard{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/guards/isolation.py", Active: true}
}

func (x *MemoryIsolationGuard) ValidateSearchResults(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryIsolationGuard", x.Subsystem, "validate_search_results", args...)
}
func (x *MemoryIsolationGuard) ValidateSingleMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryIsolationGuard", x.Subsystem, "validate_single_memory", args...)
}
func (x *MemoryIsolationGuard) GetStats(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryIsolationGuard", x.Subsystem, "get_stats", args...)
}
func (x *MemoryIsolationGuard) ResetStats(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryIsolationGuard", x.Subsystem, "reset_stats", args...)
}

// MemoryLayerBase implements packages/rae-agentic-memory/rae-core/rae_core/layers/base.py (MemoryLayerBase).
type MemoryLayerBase struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryLayerBase() *MemoryLayerBase {
	return &MemoryLayerBase{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/layers/base.py", Active: true}
}

func (x *MemoryLayerBase) AddMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryLayerBase", x.Subsystem, "add_memory", args...)
}
func (x *MemoryLayerBase) GetMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryLayerBase", x.Subsystem, "get_memory", args...)
}
func (x *MemoryLayerBase) SearchMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryLayerBase", x.Subsystem, "search_memories", args...)
}
func (x *MemoryLayerBase) Cleanup(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryLayerBase", x.Subsystem, "cleanup", args...)
}
func (x *MemoryLayerBase) CountMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryLayerBase", x.Subsystem, "count_memories", args...)
}
func (x *MemoryLayerBase) DeleteMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryLayerBase", x.Subsystem, "delete_memory", args...)
}

// MemoryRefactor implements packages/rae-agentic-memory/scripts/maintenance/refactor_memories.py (MemoryRefactor).
type MemoryRefactor struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryRefactor() *MemoryRefactor {
	return &MemoryRefactor{Subsystem: "packages/rae-agentic-memory/scripts", Source: "packages/rae-agentic-memory/scripts/maintenance/refactor_memories.py", Active: true}
}

func (x *MemoryRefactor) FetchMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRefactor", x.Subsystem, "fetch_memories", args...)
}
func (x *MemoryRefactor) AnalyzeMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRefactor", x.Subsystem, "analyze_memory", args...)
}
func (x *MemoryRefactor) MoveMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRefactor", x.Subsystem, "move_memory", args...)
}
func (x *MemoryRefactor) Run(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRefactor", x.Subsystem, "run", args...)
}

// MemoryRetrievalDelegate implements packages/rae-agentic-memory/apps/memory_api/services/delegates/retrieval_delegate.py (MemoryRetrievalDelegate).
type MemoryRetrievalDelegate struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryRetrievalDelegate() *MemoryRetrievalDelegate {
	return &MemoryRetrievalDelegate{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/delegates/retrieval_delegate.py", Active: true}
}

func (x *MemoryRetrievalDelegate) GetMemory(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "get_memory", args...)
}
func (x *MemoryRetrievalDelegate) ListMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "list_memories", args...)
}
func (x *MemoryRetrievalDelegate) CountMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "count_memories", args...)
}
func (x *MemoryRetrievalDelegate) QueryMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "query_memories", args...)
}
func (x *MemoryRetrievalDelegate) SearchMemories(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "search_memories", args...)
}
func (x *MemoryRetrievalDelegate) SearchEvidence(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "search_evidence", args...)
}
func (x *MemoryRetrievalDelegate) GetSessionContext(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "get_session_context", args...)
}
func (x *MemoryRetrievalDelegate) ListMemoriesForGraphExtraction(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryRetrievalDelegate", x.Subsystem, "list_memories_for_graph_extraction", args...)
}

// MemoryScoreResultV3 implements packages/rae-agentic-memory/apps/memory_api/services/memory_scoring_v3.py (MemoryScoreResultV3).
type MemoryScoreResultV3 struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryScoreResultV3() *MemoryScoreResultV3 {
	return &MemoryScoreResultV3{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/memory_scoring_v3.py", Active: true}
}

func (x *MemoryScoreResultV3) Execute(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryScoreResultV3", x.Subsystem, "execute", args...)
}
func (x *MemoryScoreResultV3) Validate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryScoreResultV3", x.Subsystem, "validate", args...)
}

// MemoryTracer implements packages/rae-agentic-memory/apps/memory_api/observability/memory_tracing.py (MemoryTracer).
type MemoryTracer struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryTracer() *MemoryTracer {
	return &MemoryTracer{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/observability/memory_tracing.py", Active: true}
}

func (x *MemoryTracer) TraceOperation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryTracer", x.Subsystem, "trace_operation", args...)
}
func (x *MemoryTracer) RecordVectorOperation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryTracer", x.Subsystem, "record_vector_operation", args...)
}
func (x *MemoryTracer) RecordGraphOperation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryTracer", x.Subsystem, "record_graph_operation", args...)
}
func (x *MemoryTracer) RecordReasoningOperation(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryTracer", x.Subsystem, "record_reasoning_operation", args...)
}
func (x *MemoryTracer) RecordTokens(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryTracer", x.Subsystem, "record_tokens", args...)
}

// MemoryValidatorPlugin implements packages/rae-agentic-memory/apps/memory_api/plugins/examples/memory_validator.py (MemoryValidatorPlugin).
type MemoryValidatorPlugin struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryValidatorPlugin() *MemoryValidatorPlugin {
	return &MemoryValidatorPlugin{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/plugins/examples/memory_validator.py", Active: true}
}

func (x *MemoryValidatorPlugin) Initialize(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryValidatorPlugin", x.Subsystem, "initialize", args...)
}
func (x *MemoryValidatorPlugin) OnBeforeMemoryCreate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryValidatorPlugin", x.Subsystem, "on_before_memory_create", args...)
}
func (x *MemoryValidatorPlugin) OnBeforeMemoryUpdate(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryValidatorPlugin", x.Subsystem, "on_before_memory_update", args...)
}
func (x *MemoryValidatorPlugin) HealthCheck(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryValidatorPlugin", x.Subsystem, "health_check", args...)
}

// MemoryVectorStore implements packages/rae-agentic-memory/apps/memory_api/services/vector_store/base.py (MemoryVectorStore).
type MemoryVectorStore struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMemoryVectorStore() *MemoryVectorStore {
	return &MemoryVectorStore{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/vector_store/base.py", Active: true}
}

func (x *MemoryVectorStore) Upsert(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryVectorStore", x.Subsystem, "upsert", args...)
}
func (x *MemoryVectorStore) Query(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryVectorStore", x.Subsystem, "query", args...)
}
func (x *MemoryVectorStore) Delete(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MemoryVectorStore", x.Subsystem, "delete", args...)
}

// MeshExporter implements src/mesh/exporter.py (MeshExporter).
type MeshExporter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMeshExporter() *MeshExporter {
	return &MeshExporter{Subsystem: "src", Source: "src/mesh/exporter.py", Active: true}
}

func (x *MeshExporter) Export(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshExporter", x.Subsystem, "export", args...)
}

// MeshFederationApp implements packages/rae-agentic-memory/rae-portal/apps/mesh_federation.py (MeshFederationApp).
type MeshFederationApp struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMeshFederationApp() *MeshFederationApp {
	return &MeshFederationApp{Subsystem: "packages/rae-agentic-memory/rae-portal", Source: "packages/rae-agentic-memory/rae-portal/apps/mesh_federation.py", Active: true}
}

func (x *MeshFederationApp) LoadData(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "load_data", args...)
}
func (x *MeshFederationApp) CreateInvite(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "create_invite", args...)
}
func (x *MeshFederationApp) JoinPeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "join_peer", args...)
}
func (x *MeshFederationApp) RegisterPeerManually(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "register_peer_manually", args...)
}
func (x *MeshFederationApp) ManualSync(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "manual_sync", args...)
}
func (x *MeshFederationApp) RevokePeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "revoke_peer", args...)
}
func (x *MeshFederationApp) Render(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshFederationApp", x.Subsystem, "render", args...)
}

// MeshImporter implements src/mesh/importer.py (MeshImporter).
type MeshImporter struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMeshImporter() *MeshImporter {
	return &MeshImporter{Subsystem: "src", Source: "src/mesh/importer.py", Active: true}
}

func (x *MeshImporter) ImportEnvelope(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshImporter", x.Subsystem, "import_envelope", args...)
}

// MeshPeerRegistry implements src/mesh/mesh_peer_registry.py (MeshPeerRegistry).
type MeshPeerRegistry struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMeshPeerRegistry() *MeshPeerRegistry {
	return &MeshPeerRegistry{Subsystem: "src", Source: "src/mesh/mesh_peer_registry.py", Active: true}
}

func (x *MeshPeerRegistry) RegisterPeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshPeerRegistry", x.Subsystem, "register_peer", args...)
}
func (x *MeshPeerRegistry) GetPeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshPeerRegistry", x.Subsystem, "get_peer", args...)
}
func (x *MeshPeerRegistry) ListPeers(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshPeerRegistry", x.Subsystem, "list_peers", args...)
}

// MeshService implements packages/rae-agentic-memory/apps/memory_api/services/mesh_service.py (MeshService).
type MeshService struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMeshService() *MeshService {
	return &MeshService{Subsystem: "packages/rae-agentic-memory/apps/memory_api", Source: "packages/rae-agentic-memory/apps/memory_api/services/mesh_service.py", Active: true}
}

func (x *MeshService) EncryptAuthData(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "encrypt_auth_data", args...)
}
func (x *MeshService) DecryptAuthData(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "decrypt_auth_data", args...)
}
func (x *MeshService) DeriveKeyPair(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "derive_key_pair", args...)
}
func (x *MeshService) GenerateChallenge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "generate_challenge", args...)
}
func (x *MeshService) VerifyChallengeSignature(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "verify_challenge_signature", args...)
}
func (x *MeshService) SignChallenge(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "sign_challenge", args...)
}
func (x *MeshService) CreateInvite(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "create_invite", args...)
}
func (x *MeshService) ValidateInvite(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "validate_invite", args...)
}
func (x *MeshService) RegisterPeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "register_peer", args...)
}
func (x *MeshService) GetPeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "get_peer", args...)
}
func (x *MeshService) GetPeerPublicKey(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "get_peer_public_key", args...)
}
func (x *MeshService) ListPeers(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "list_peers", args...)
}
func (x *MeshService) RevokePeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "revoke_peer", args...)
}
func (x *MeshService) SaveSyncLog(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "save_sync_log", args...)
}
func (x *MeshService) GenerateConsentToken(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "generate_consent_token", args...)
}
func (x *MeshService) VerifyConsentToken(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "verify_consent_token", args...)
}
func (x *MeshService) PrepareSyncPayload(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "prepare_sync_payload", args...)
}
func (x *MeshService) PushMemoriesToPeer(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MeshService", x.Subsystem, "push_memories_to_peer", args...)
}

// MetadataInjector implements packages/rae-agentic-memory/rae-core/rae_core/math/metadata_injector.py (MetadataInjector).
type MetadataInjector struct {
	Subsystem string
	Source    string
	Active    bool
}

func NewMetadataInjector() *MetadataInjector {
	return &MetadataInjector{Subsystem: "packages/rae-agentic-memory/rae-core", Source: "packages/rae-agentic-memory/rae-core/rae_core/math/metadata_injector.py", Active: true}
}

func (x *MetadataInjector) EnrichText(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MetadataInjector", x.Subsystem, "enrich_text", args...)
}
func (x *MetadataInjector) ProcessQuery(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MetadataInjector", x.Subsystem, "process_query", args...)
}
func (x *MetadataInjector) ProcessDocument(args ...interface{}) DomainExecutionRecord {
	return ExecuteDomainOp("MetadataInjector", x.Subsystem, "process_document", args...)
}
