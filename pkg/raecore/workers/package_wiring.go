package workers

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*ActionExecutor)(nil),
		(*BenchmarkRepository)(nil),
		(*CommunityWorker)(nil),
		(*ContextBuilder)(nil),
		(*DashboardWebSocketService)(nil),
		(*EnhancedGraphRepository)(nil),
		(*GraphAlgorithmsService)(nil),
		(*GraphRepository)(nil),
		(*GraphUpdateOperator)(nil),
		(*HybridSearchCache)(nil),
		(*MemoryConsolidationService)(nil),
		(*MemoryIngestionDelegate)(nil),
		(*MemoryLifecycleDelegate)(nil),
		(*MemoryMaintenanceWorker)(nil),
		(*MetricsRepository)(nil),
		(*MetricsTracker)(nil),
		(*QueryAnalyzer)(nil),
		(*RAECoreService)(nil),
		(*ReflectionPipeline)(nil),
		(*RewardFunction)(nil),
		(*SemanticExtractor)(nil),
		(*SemanticSearchPipeline)(nil),
		(*TriggerRepository)(nil),
	}
}
