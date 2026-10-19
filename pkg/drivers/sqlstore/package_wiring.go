package sqlstore

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*InMemoryStorage)(nil),
		(*PersistentState)(nil),
		(*PersistentStore)(nil),
		(*PostgreSQLStorage)(nil),
		(*SQLiteGraphStore)(nil),
		(*SQLiteStorage)(nil),
		(*SQLiteVectorStore)(nil),
	}
}
