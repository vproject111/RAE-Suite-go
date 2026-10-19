package workflows

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*A2AEnvelope)(nil),
		(*HardwareProfile)(nil),
		(*ReplayHistoryEntry)(nil),
		(*SagaCoordinator)(nil),
		(*SagaResult)(nil),
		(*SagaStep)(nil),
		(*ScrubResult)(nil),
		(*ShadowGuardrail)(nil),
		(*SpeculativeResult)(nil),
		(*StreamingFunctionComposer)(nil),
		(*StreamingStepResult)(nil),
		(*SynthesizedTool)(nil),
		(*piiRule)(nil),
	}
}
