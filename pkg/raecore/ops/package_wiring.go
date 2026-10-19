package ops

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*ClusterNode)(nil),
		(*EmbeddingDriftDetector)(nil),
		(*EscalationResult)(nil),
		(*KeycloakClaims)(nil),
		(*PromotionDecision)(nil),
		(*RateLimitStatus)(nil),
		(*RollbackDecision)(nil),
		(*tokenBucket)(nil),
		(*GitOpsActionResult)(nil),
		(*GitOpsDaemon)(nil),
		(*GlobalSearchResponse)(nil),
		(*GlobalSearchResultItem)(nil),
		(*HermesOpenCodeBridge)(nil),
		(*HermesOpenCodeResponse)(nil),
	}
}
