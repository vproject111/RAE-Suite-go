package graph

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*EntityResolutionResult)(nil),
		(*MergeDecision)(nil),
		(*EmergingPattern)(nil),
		(*GraphSnapshot)(nil),
		(*TemporalComparison)(nil),
	}
}
