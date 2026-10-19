package ingestion

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*DocumentChunk)(nil),
		(*GraphTriple)(nil),
		(*IngestionResult)(nil),
	}
}
