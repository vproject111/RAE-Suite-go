package orchestrator

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*AdaptiveModelRouter)(nil),
		(*ModelRouter)(nil),
		(*ModelRouterV2)(nil),
		(*OrchestratorTelemetry)(nil),
		(*PerformanceAnalytics)(nil),
		(*PerformanceTracker)(nil),
		(*QualityGate)(nil),
		(*RunLogger)(nil),
	}
}
