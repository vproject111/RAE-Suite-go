package reflection

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*ReflectionReport)(nil),
		(*CommunitySummary)(nil),
		(*DreamInsight)(nil),
		(*MemoryLifecycleRecord)(nil),
	}
}
