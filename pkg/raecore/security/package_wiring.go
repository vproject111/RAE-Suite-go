package security

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*DeletionAuditEntry)(nil),
		(*GovernancePolicyVersion)(nil),
		(*RuleExecutionResult)(nil),
		(*TenantBudget)(nil),
		(*ContextQualityAudit)(nil),
		(*DecisionContext)(nil),
		(*DecisionRecord)(nil),
	}
}
