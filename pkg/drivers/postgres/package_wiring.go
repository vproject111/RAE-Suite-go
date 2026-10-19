package postgres

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*DelegatedTaskRecord)(nil),
		(*TokenSavingEntry)(nil),
	}
}
