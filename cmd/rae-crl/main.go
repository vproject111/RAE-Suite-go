package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildServiceAndHandler(ctx context.Context) (*crl.Service, http.Handler) {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-crl")
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	svc := crl.NewService(store, store, guard)
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	return svc, security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	svc, handler := buildServiceAndHandler(ctx)
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			proj := "default-research"
			if len(os.Args) > 2 {
				proj = os.Args[2]
			}
			trace, _ := svc.QuickNote(ctx, "Initialized research context "+proj, proj, "cli")
			fmt.Printf("Initialized RAE-CRL project '%s' (Genesis Trace ID: %s)\n", proj, trace.ID)
			return nil
		case "add":
			if len(os.Args) < 6 {
				return fmt.Errorf("usage: rae-crl add <type> <title> <description> <project>")
			}
			trace, _ := svc.QuickNote(ctx, os.Args[4], os.Args[5], "cli")
			refined, err := svc.RefineArtifact(ctx, trace.ID, models.ArtifactType(os.Args[2]), os.Args[3], os.Args[4])
			if err != nil {
				return fmt.Errorf("failed to add artifact: %w", err)
			}
			b, _ := json.MarshalIndent(refined, "", "  ")
			fmt.Println(string(b))
			return nil
		case "query":
			proj := ""
			if len(os.Args) > 2 {
				proj = os.Args[2]
			}
			arts, err := svc.ListTraces(ctx, proj, "")
			if err != nil {
				return fmt.Errorf("query failed: %w", err)
			}
			b, _ := json.MarshalIndent(arts, "", "  ")
			fmt.Println(string(b))
			return nil
		case "fork":
			if len(os.Args) < 5 {
				return fmt.Errorf("usage: rae-crl fork <original_id> <new_author> <reason>")
			}
			forked, err := svc.ForkArtifact(ctx, os.Args[2], os.Args[3], os.Args[4])
			if err != nil {
				return fmt.Errorf("fork failed: %w", err)
			}
			b, _ := json.MarshalIndent(forked, "", "  ")
			fmt.Println(string(b))
			return nil
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
				return fmt.Errorf("seal failed: %w", err)
			}
			b, _ := json.MarshalIndent(pkg, "", "  ")
			fmt.Println(string(b))
			return nil
		case "verify":
			if len(os.Args) < 3 {
				return fmt.Errorf("usage: rae-crl verify <package.json>")
			}
			raw, err := os.ReadFile(os.Args[2])
			if err != nil {
				return fmt.Errorf("read package file failed: %w", err)
			}
			var pkg models.EvidencePackage
			if err := json.Unmarshal(raw, &pkg); err != nil {
				return fmt.Errorf("invalid package JSON: %w", err)
			}
			if err := svc.VerifyPackageOffline(&pkg); err != nil {
				return fmt.Errorf("verification FAILED: %w", err)
			}
			fmt.Printf("Verified EvidencePackage %s (CID=%s, SBOM=%s)\n", pkg.PackageID, pkg.VersionCID, pkg.SBOMReference)
			return nil
		}
	}

	svc.StartSyncEngine(ctx, 60*time.Second)
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8503"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-crl", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

