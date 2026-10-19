package vectorstore

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*HNSWIndex)(nil),
		(*InMemoryCache)(nil),
		(*InMemoryVectorStore)(nil),
		(*QdrantClient)(nil),
		(*QdrantVectorStore)(nil),
		(*VectorHit)(nil),
	}
}
