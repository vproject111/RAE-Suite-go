package accounting

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionAuditLedgerAndRotationAndReceipts(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "ledger.jsonl")
	// Set small maxFileSizeBytes (200 bytes) to test automatic .jsonl.gz rotation
	ledger := NewExecutionAuditLedger(tmpFile, 200)

	r1 := ledger.RecordInvocation(
		"z-ai/glm-5.3-flash",
		"openrouter",
		"grzegorz@cloud",
		"Optimize HNSW search",
		"Reduce P99 latency",
		"COMPLETED",
		700,
		300,
		0.00012,
		8.45,
		"P99 reduced to 0.4ms",
		0,
	)
	if r1.TotalTokens != 1000 || len(r1.CryptographicHash) != 64 || r1.PreviousHash != strings.Repeat("0", 64) {
		t.Fatalf("unexpected r1: %+v", r1)
	}

	r2 := ledger.RecordInvocation(
		"deepseek/deepseek-v4-pro-0813",
		"openrouter",
		"grzegorz@cloud",
		"Audit HNSW patch",
		"Verify thread safety",
		"APPROVE",
		500,
		200,
		0.00045,
		12.10,
		"0 race conditions",
		0,
	)
	if r2.PreviousHash != r1.CryptographicHash {
		t.Fatalf("hash chain broken: %s != %s", r2.PreviousHash, r1.CryptographicHash)
	}

	mem := BuildFivePillarMemory(r1, "rae-go")
	if !strings.Contains(mem.Content, "KTO (Who)") || !strings.Contains(mem.Content, "ILE KOSZTOWAŁO") {
		t.Fatalf("unexpected 5-pillar memory content: %s", mem.Content)
	}

	stats := ledger.GetSummaryStats()
	if stats["total_invocations"] != 2 || stats["total_tokens_consumed"] != 1700 {
		t.Fatalf("unexpected summary stats: %+v", stats)
	}

	// Test ExecutionReceipt 'Zero Fake Success' invariant
	badReceipt := ExecutionReceipt{
		ReceiptID:            "rcpt-1",
		TaskID:               "task-1",
		StepID:               "step-1",
		ExecutionStatus:      StatusVerifiedSuccess,
		GitDiffHash:          "abcdef123456",
		TestResult:           TestExecutionResult{Command: "go test ./...", ExitCode: 1, PassedCount: 10, FailedCount: 1},
		CapabilityCompliance: true,
	}
	if ok, _ := VerifyExecutionReceipt(badReceipt); ok {
		t.Fatalf("expected fake success receipt with exit_code=1 to be rejected")
	}

	good1 := ExecutionReceipt{
		ReceiptID:            "rcpt-1",
		TaskID:               "task-1",
		StepID:               "step-1",
		PreviousReceiptHash:  strings.Repeat("0", 64),
		ExecutionStatus:      StatusVerifiedSuccess,
		GitDiffHash:          "abcdef123456",
		TestResult:           TestExecutionResult{Command: "go test ./...", ExitCode: 0, PassedCount: 10, FailedCount: 0},
		CapabilityCompliance: true,
	}
	good1.Finalize()

	good2 := ExecutionReceipt{
		ReceiptID:            "rcpt-2",
		TaskID:               "task-1",
		StepID:               "step-2",
		PreviousReceiptHash:  good1.SHA256Hash,
		ExecutionStatus:      StatusVerifiedSuccess,
		GitDiffHash:          "987654abcdef",
		TestResult:           TestExecutionResult{Command: "go test -race ./...", ExitCode: 0, PassedCount: 10, FailedCount: 0},
		CapabilityCompliance: true,
	}
	good2.Finalize()

	if ok, msg := VerifyReceiptChain([]ExecutionReceipt{good1, good2}); !ok || msg != "CHAIN_VERIFIED_STRONG" {
		t.Fatalf("VerifyReceiptChain failed: %v %s", ok, msg)
	}

	projSummary := ledger.AnalyzeProject("RAE-Suite-go")
	if projSummary.TotalPhases != 2 || projSummary.TotalProjectTokens != 1700 {
		t.Fatalf("AnalyzeProject unexpected: %+v", projSummary)
	}
}

func TestAccountingEdgeCasesNegativeReceiptsAndEmptyLedger(t *testing.T) {
	if len(RegisteredPackageTypes()) == 0 {
		t.Fatalf("expected non-empty RegisteredPackageTypes")
	}
	// Empty ledger defaults and AnalyzeProject with 0 receipts
	emptyLedger := NewExecutionAuditLedger("", 0)
	emptyProj := emptyLedger.AnalyzeProject("")
	if emptyProj.ProjectID != "RAE-Suite-go" || emptyProj.TotalPhases != 1 {
		t.Fatalf("expected default project summary with 1 fallback phase, got %+v", emptyProj)
	}

	// RecordInvocation with negative numbers and empty provider/user/decision
	rcpt := emptyLedger.RecordInvocation("local-model", "", "", "intent", "rationale", "", -5, -10, -1.0, -2.0, "ok", 0)
	if rcpt.Provider != "local_first" || rcpt.UserIdentity != "grzegorz@cloud" || rcpt.Decision != "COMPLETED" || rcpt.TotalTokens != 0 {
		t.Fatalf("unexpected default normalization in RecordInvocation: %+v", rcpt)
	}
	rcptCloud := emptyLedger.RecordInvocation("cloud-model", "", "", "intent", "rationale", "", 10, 10, 0.05, 5.0, "ok", 0)
	if rcptCloud.Provider != "openrouter" {
		t.Fatalf("expected openrouter provider for non-zero cost, got %s", rcptCloud.Provider)
	}
	mem := BuildFivePillarMemory(rcpt, "")
	if mem.Project != "RAE-Suite-go" {
		t.Fatalf("expected default project RAE-Suite-go, got %s", mem.Project)
	}

	// VerifyExecutionReceipt all violation branches
	if ok, _ := VerifyExecutionReceipt(ExecutionReceipt{
		ExecutionStatus:      StatusVerifiedSuccess,
		TestResult:           TestExecutionResult{ExitCode: 0, FailedCount: 2},
		CapabilityCompliance: true,
		GitDiffHash:          "abcdef1234",
	}); ok {
		t.Fatalf("expected failure when FailedCount > 0")
	}
	if ok, _ := VerifyExecutionReceipt(ExecutionReceipt{
		ExecutionStatus:      StatusVerifiedSuccess,
		TestResult:           TestExecutionResult{ExitCode: 0, FailedCount: 0},
		CapabilityCompliance: false,
		GitDiffHash:          "abcdef1234",
	}); ok {
		t.Fatalf("expected failure when CapabilityCompliance is false")
	}
	if ok, _ := VerifyExecutionReceipt(ExecutionReceipt{
		ExecutionStatus:      StatusVerifiedSuccess,
		TestResult:           TestExecutionResult{ExitCode: 0, FailedCount: 0},
		CapabilityCompliance: true,
		GitDiffHash:          "short",
	}); ok {
		t.Fatalf("expected failure when GitDiffHash < 8 chars")
	}
	if ok, _ := VerifyExecutionReceipt(ExecutionReceipt{
		ExecutionStatus: StatusFailed,
		SHA256Hash:      "tampered_hash",
	}); ok {
		t.Fatalf("expected failure on SHA256Hash mismatch")
	}

	// VerifyReceiptChain edge cases: empty chain, duplicate receipt ID, broken prev_hash
	if ok, msg := VerifyReceiptChain(nil); !ok || msg != "CHAIN_EMPTY" {
		t.Fatalf("expected CHAIN_EMPTY, got %v %s", ok, msg)
	}
	rValid := ExecutionReceipt{
		ReceiptID:            "r1",
		PreviousReceiptHash:  strings.Repeat("0", 64),
		ExecutionStatus:      StatusPartialSuccess,
		CapabilityCompliance: true,
	}
	rValid.Finalize()
	if ok, _ := VerifyReceiptChain([]ExecutionReceipt{rValid, rValid}); ok {
		t.Fatalf("expected duplicate receipt_id error in VerifyReceiptChain")
	}
	rBrokenPrev := ExecutionReceipt{
		ReceiptID:            "r2",
		PreviousReceiptHash:  "wrong_prev_hash",
		ExecutionStatus:      StatusPartialSuccess,
		CapabilityCompliance: true,
	}
	rBrokenPrev.Finalize()
	if ok, _ := VerifyReceiptChain([]ExecutionReceipt{rValid, rBrokenPrev}); ok {
		t.Fatalf("expected broken chain prev_hash error")
	}
}

