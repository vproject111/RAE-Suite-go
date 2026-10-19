package math

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*HybridSearchEngine)(nil),
		(*MultiArmedBandit)(nil),
		(*RAEEngine)(nil),
		(*RAERuntime)(nil),
		(*Reflector)(nil),
		(*IBSelectionResult)(nil),
		(*ModelPricingEntry)(nil),
		(*SixFactorImportanceBreakdown)(nil),
	}
}
