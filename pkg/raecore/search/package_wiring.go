package search

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*ClassificationResult)(nil),
		(*RoutingPlan)(nil),
		(*EvidenceConflict)(nil),
		(*RewritePlan)(nil),
		(*DriftEvaluationResult)(nil),
		(*IREvaluationMetrics)(nil),
	}
}
