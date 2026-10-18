package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/memory"
)

func main() {
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	svc := memory.NewService(store, store, guard)
	svc.StartBackgroundWorkers(context.Background(), 60*time.Second)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Memory 8-Layer Engine (Go Native v4.0) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
