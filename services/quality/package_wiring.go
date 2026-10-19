package quality

// RegisteredPackageTypes exposes package domain and internal structs for runtime introspection.
func RegisteredPackageTypes() []interface{} {
	return []interface{}{
		(*CalculatorVerificationResult)(nil),
		(*DTPSpotLayer)(nil),
		(*PDFDtpAuditResult)(nil),
		(*QualityTopic)(nil),
		(*RAEQualityGatewayV2)(nil),
		(*RiskCoverageEngine)(nil),
		(*RiskPathItem)(nil),
		(*SecurityGateway)(nil),
	}
}
