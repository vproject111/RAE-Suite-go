package accounting

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*ISOAuditMetadata)(nil),
		(*MinimumAuditableEvent)(nil),
		(*TestExecutionResult)(nil),
	}
}
