package mdp

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*BudgetState)(nil),
		(*GraphTopologicalState)(nil),
		(*MDPMetrics)(nil),
		(*MemoryDistributionState)(nil),
		(*RAEState)(nil),
		(*RewardHyperparameters)(nil),
		(*TransitionOutcome)(nil),
		(*WorkingContextState)(nil),
	}
}
