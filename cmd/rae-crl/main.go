package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
)

func main() {
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	svc := crl.NewService(store, store, guard)
	ctx := context.Background()

	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "init":
			proj := "default-research"
			if len(os.Args) > 2 {
				proj = os.Args[2]
			}
			trace, _ := svc.QuickNote(ctx, "Initialized research context "+proj, proj, "cli")
			fmt.Printf("Initialized RAE-CRL project '%s' (Genesis Trace ID: %s)\n", proj, trace.ID)
			return
		case "add":
			if len(os.Args) < 6 {
				log.Fatalf("Usage: rae-crl add <type> <title> <description> <project>")
			}
			artType := models.ArtifactType(os.Args[2])
			title := os.Args[3]
			desc := os.Args[4]
			proj := os.Args[5]
			trace, _ := svc.QuickNote(ctx, desc, proj, "cli")
			refined, err := svc.RefineArtifact(ctx, trace.ID, artType, title, desc)
			if err != nil {
				log.Fatalf("Failed to add artifact: %v", err)
			}
			b, _ := json.MarshalIndent(refined, "", "  ")
			fmt.Println(string(b))
			return
		case "query":
			proj := ""
			if len(os.Args) > 2 {
				proj = os.Args[2]
			}
			arts, err := svc.ListTraces(ctx, proj, "")
			if err != nil {
				log.Fatalf("Query failed: %v", err)
			}
			b, _ := json.MarshalIndent(arts, "", "  ")
			fmt.Println(string(b))
			return
		case "fork":
			if len(os.Args) < 5 {
				log.Fatalf("Usage: rae-crl fork <original_id> <new_author> <reason>")
			}
			forked, err := svc.ForkArtifact(ctx, os.Args[2], os.Args[3], os.Args[4])
			if err != nil {
				log.Fatalf("Fork failed: %v", err)
			}
			b, _ := json.MarshalIndent(forked, "", "  ")
			fmt.Println(string(b))
			return
		case "seal":
			proj := "default-research"
			sbom := "sbom://cyclonedx-v1.5"
			if len(os.Args) > 2 {
				proj = os.Args[2]
			}
			if len(os.Args) > 3 {
				sbom = os.Args[3]
			}
			pkg, err := svc.SealEvidencePackage(ctx, proj, sbom, "NOT_AFFECTED", []string{"rae-go-v4.1"})
			if err != nil {
				log.Fatalf("Seal failed: %v", err)
			}
			b, _ := json.MarshalIndent(pkg, "", "  ")
			fmt.Println(string(b))
			return
		case "verify":
			if len(os.Args) < 3 {
				log.Fatalf("Usage: rae-crl verify <package.json>")
			}
			raw, err := os.ReadFile(os.Args[2])
			if err != nil {
				log.Fatalf("Read package file failed: %v", err)
			}
			var pkg models.EvidencePackage
			if err := json.Unmarshal(raw, &pkg); err != nil {
				log.Fatalf("Invalid package JSON: %v", err)
			}
			if err := svc.VerifyPackageOffline(&pkg); err != nil {
				log.Fatalf("Verification FAILED: %v", err)
			}
			fmt.Printf("Verified EvidencePackage %s (CID=%s, SBOM=%s)\n", pkg.PackageID, pkg.VersionCID, pkg.SBOMReference)
			return
		}
	}

	svc.StartSyncEngine(ctx, 60*time.Second)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8503"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-CRL (Go Native v5.3.2) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, security.WrapSecureHandler(mux)))
}
