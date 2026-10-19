package raesdk

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*CIFixAgent)(nil),
		(*FailureAnalyzer)(nil),
		(*MLServiceClient)(nil),
		(*MemoryClient)(nil),
		(*NodeAgent)(nil),
		(*RAEAPIClient)(nil),
		(*RAEClient)(nil),
	}
}
