package agnostic

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*BackendStatus)(nil),
		(*QdrantRESTAdapter)(nil),
		(*RedisRESPClient)(nil),
	}
}
