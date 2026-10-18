package epistemic

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
)

var (
	ErrDAGCycleDetected       = errors.New("RAE-CRL Section 5.3 violation: cycle detected in Typed Provenance DAG")
	ErrInvalidEdgeGrammar     = errors.New("RAE-CRL Section 5.2 violation: unrecognized edge relation vocabulary")
	ErrMissingSBOMOrVEX       = errors.New("RAE-CRL Section 13/20 violation: EvidencePackage missing mandatory SBOM or VEX")
	ErrInvalidPackageCID      = errors.New("RAE-CRL Section 20.4 violation: deterministic CBOR VersionCID mismatch")
	ErrInvalidPackageSignature = errors.New("RAE-CRL Section 20.2 violation: HMAC-SHA256 package signature verification failed")
)

var validEdgeVocabulary = map[string]bool{
	"tests":                          true,
	"supports":                       true,
	"contradicts":                    true,
	"refines":                        true,
	"has_alternative_interpretation": true,
	"derived_from":                   true,
	"related_to":                     true,
	"supersedes":                     true,
}

// ValidateDAGAddition verifies the edge vocabulary and runs a full DFS reachability check
// to ensure adding sourceID -> targetID does not introduce a cycle of any depth.
func ValidateDAGAddition(existing []models.ArtifactEdge, sourceID, targetID, relationType string) error {
	if relationType == "" {
		relationType = "related_to"
	}
	if !validEdgeVocabulary[relationType] {
		return ErrInvalidEdgeGrammar
	}
	if sourceID == targetID {
		return ErrDAGCycleDetected
	}
	adj := make(map[string][]string)
	for _, e := range existing {
		adj[e.SourceID] = append(adj[e.SourceID], e.TargetID)
	}
	// If sourceID is reachable from targetID, adding sourceID -> targetID creates a cycle.
	visited := make(map[string]bool)
	var dfs func(curr string) bool
	dfs = func(curr string) bool {
		if curr == sourceID {
			return true
		}
		visited[curr] = true
		for _, nxt := range adj[curr] {
			if !visited[nxt] && dfs(nxt) {
				return true
			}
		}
		return false
	}
	if dfs(targetID) {
		return ErrDAGCycleDetected
	}
	return nil
}

// TopologicalSort returns a deterministic topological ordering of DAG node IDs.
func TopologicalSort(nodes []string, edges []models.ArtifactEdge) ([]string, error) {
	inDegree := make(map[string]int, len(nodes))
	adj := make(map[string][]string, len(nodes))
	for _, n := range nodes {
		inDegree[n] = 0
	}
	for _, e := range edges {
		if _, ok := inDegree[e.SourceID]; !ok {
			inDegree[e.SourceID] = 0
		}
		if _, ok := inDegree[e.TargetID]; !ok {
			inDegree[e.TargetID] = 0
		}
		adj[e.SourceID] = append(adj[e.SourceID], e.TargetID)
		inDegree[e.TargetID]++
	}

	var queue []string
	for n, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, n)
		}
	}
	sort.Strings(queue)

	var order []string
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		neighbors := adj[curr]
		sort.Strings(neighbors)
		for _, nxt := range neighbors {
			inDegree[nxt]--
			if inDegree[nxt] == 0 {
				queue = append(queue, nxt)
			}
		}
	}
	if len(order) != len(inDegree) {
		return nil, ErrDAGCycleDetected
	}
	return order, nil
}

// VerifyEvidencePackageOffline performs full offline verification of a sealed RAE-CRL EvidencePackage.
func VerifyEvidencePackageOffline(pkg *models.EvidencePackage, guard *security.HardFramesGuard) error {
	if pkg == nil || strings.TrimSpace(pkg.SBOMReference) == "" || strings.TrimSpace(pkg.VEXStatus) == "" {
		return ErrMissingSBOMOrVEX
	}
	cborMap := map[string]string{
		"project":   pkg.Project,
		"sbom_ref":  pkg.SBOMReference,
		"vex":       pkg.VEXStatus,
		"sealed_at": pkg.SealedAt.UTC().Format(time.RFC3339),
	}
	for _, a := range pkg.Artifacts {
		cborMap["art_"+a.ID] = a.VersionCID
	}
	expectedCID := models.ComputeVersionCID(cborMap)
	if pkg.VersionCID != expectedCID {
		return ErrInvalidPackageCID
	}
	payload := pkg.VersionCID + "|" + pkg.SBOMReference + "|" + pkg.VEXStatus
	if !guard.VerifyTokenHMAC(payload, pkg.SignatureHMAC) {
		return ErrInvalidPackageSignature
	}
	return nil
}
