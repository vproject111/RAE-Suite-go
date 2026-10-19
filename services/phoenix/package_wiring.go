package phoenix

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*AnalysisPipeline)(nil),
		(*BehaviorGuardIntegration)(nil),
		(*BehaviorStore)(nil),
		(*CapabilityDetector)(nil),
		(*ControllerToComponentRecipe)(nil),
		(*DirectiveToComponentRecipe)(nil),
		(*EnhancedRAEClient)(nil),
		(*FeniksMemoryRouter)(nil),
		(*PhoenixRefactorer)(nil),
		(*PostgresBackend)(nil),
		(*QdrantBackend)(nil),
		(*RAEFormatter)(nil),
		(*RefactorEngine)(nil),
		(*ReflectionRuleSet)(nil),
		(*ReportGenerator)(nil),
		(*RoutingToAppRouterRecipe)(nil),
		(*ScenarioLibrary)(nil),
		(*ScopeToHooksRecipe)(nil),
		(*TemplateToJsxRecipe)(nil),
		(*UIRunner)(nil),
	}
}
