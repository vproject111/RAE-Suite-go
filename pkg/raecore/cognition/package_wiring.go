package cognition

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*CognitivePlanResult)(nil),
		(*CognitivePlanStep)(nil),
		(*ConsensusResult)(nil),
		(*ModelSpec)(nil),
		(*ToTNode)(nil),
		(*scoredModel)(nil),
	}
}
