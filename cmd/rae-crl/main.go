package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
)

func main() {
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), security.NewHardFramesGuard(""))
	svc := crl.NewService(store, store, security.NewHardFramesGuard(""))
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
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8503"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-CRL (Go Native v4.0) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
