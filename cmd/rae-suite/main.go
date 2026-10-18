package main

import (
	"log"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/orchestrator"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func main() {
	guard := security.NewHardFramesGuard("")
	mc := raemath.NewMathController(models.MathL3)
	store := memorystore.NewUnifiedStore(mc, guard)
	crlSvc := crl.NewService(store, store, guard)
	qualSvc := quality.NewService()
	svc := orchestrator.NewService(mc, crlSvc, qualSvc, guard)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8009"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Suite Orchestrator (Go Native v4.0) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
