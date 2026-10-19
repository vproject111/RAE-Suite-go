package portal

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*CentralAuditorApp)(nil),
		(*MissionControlApp)(nil),
		(*PDKnowledgeHubApp)(nil),
		(*PhoenixRepairApp)(nil),
		(*RAESuitePortal)(nil),
	}
}
