package epistemic_test

import (
	"testing"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/epistemic"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

func TestDAGMultiHopCycleDetectionAndTopologicalSort(t *testing.T) {
	edges := []models.ArtifactEdge{
		{SourceID: "A", TargetID: "B", RelationType: "refines"},
		{SourceID: "B", TargetID: "C", RelationType: "supports"},
	}
	// Adding C -> D is valid
	if err := epistemic.ValidateDAGAddition(edges, "C", "D", "tests"); err != nil {
		t.Fatalf("expected C -> D to be valid, got %v", err)
	}
	// Adding C -> A creates a 3-hop cycle A -> B -> C -> A
	if err := epistemic.ValidateDAGAddition(edges, "C", "A", "refines"); err == nil {
		t.Fatal("expected 3-hop cycle A -> B -> C -> A to be rejected")
	}
	// Invalid edge vocabulary
	if err := epistemic.ValidateDAGAddition(edges, "C", "D", "invalid_rel"); err == nil {
		t.Fatal("expected invalid relation vocabulary to be rejected")
	}

	order, err := epistemic.TopologicalSort([]string{"A", "B", "C"}, edges)
	if err != nil || len(order) != 3 || order[0] != "A" || order[2] != "C" {
		t.Fatalf("unexpected topological sort: %v (err=%v)", order, err)
	}
}

func TestOfflineEvidencePackageVerification(t *testing.T) {
	guard := security.NewHardFramesGuard("seal-key")
	now := time.Now().UTC().Truncate(time.Second)
	art := models.EpistemicArtifact{
		ID:         "art-1",
		VersionCID: "mh:1220abcd",
	}
	cborMap := map[string]string{
		"project":   "doktorat-kubus",
		"sbom_ref":  "sbom://v1",
		"vex":       "NOT_AFFECTED",
		"sealed_at": now.Format(time.RFC3339),
		"art_art-1": "mh:1220abcd",
	}
	cid := models.ComputeVersionCID(cborMap)
	sig := guard.SignTokenHMAC(cid + "|sbom://v1|NOT_AFFECTED")

	pkg := &models.EvidencePackage{
		PackageID:     "pkg-1",
		Project:       "doktorat-kubus",
		VersionCID:    cid,
		Artifacts:     []models.EpistemicArtifact{art},
		SBOMReference: "sbom://v1",
		VEXStatus:     "NOT_AFFECTED",
		SignatureHMAC: sig,
		SealedAt:      now,
	}
	if err := epistemic.VerifyEvidencePackageOffline(pkg, guard); err != nil {
		t.Fatalf("expected valid EvidencePackage to pass offline verification, got %v", err)
	}
	// Tamper with signature
	pkg.SignatureHMAC = "tampered"
	if err := epistemic.VerifyEvidencePackageOffline(pkg, guard); err == nil {
		t.Fatal("expected tampered signature to fail offline verification")
	}

	// Self-loop and empty relation type in ValidateDAGAddition
	if err := epistemic.ValidateDAGAddition(nil, "X", "Y", ""); err != nil {
		t.Fatalf("expected empty relationType to default to related_to, got %v", err)
	}
	if err := epistemic.ValidateDAGAddition(nil, "X", "X", "supports"); err != epistemic.ErrDAGCycleDetected {
		t.Fatalf("expected self-loop to return ErrDAGCycleDetected, got %v", err)
	}

	// TopologicalSort with implicit nodes in edges and with cycle
	orderImplicit, err := epistemic.TopologicalSort([]string{"A"}, []models.ArtifactEdge{
		{SourceID: "A", TargetID: "B"},
		{SourceID: "C", TargetID: "D"},
	})
	if err != nil || len(orderImplicit) != 4 {
		t.Fatalf("expected 4 nodes in topological sort, got %v (%v)", orderImplicit, err)
	}
	if _, err := epistemic.TopologicalSort([]string{"A", "B"}, []models.ArtifactEdge{
		{SourceID: "A", TargetID: "B"},
		{SourceID: "B", TargetID: "A"},
	}); err != epistemic.ErrDAGCycleDetected {
		t.Fatalf("expected cycle error in TopologicalSort, got %v", err)
	}

	// VerifyEvidencePackageOffline missing SBOM/VEX and CID mismatch
	if err := epistemic.VerifyEvidencePackageOffline(nil, guard); err != epistemic.ErrMissingSBOMOrVEX {
		t.Fatalf("expected ErrMissingSBOMOrVEX for nil package, got %v", err)
	}
	pkgBadCID := *pkg
	pkgBadCID.VersionCID = "mh:1220deadbeef"
	if err := epistemic.VerifyEvidencePackageOffline(&pkgBadCID, guard); err != epistemic.ErrInvalidPackageCID {
		t.Fatalf("expected ErrInvalidPackageCID, got %v", err)
	}
}

