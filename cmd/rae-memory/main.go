package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/drivers/sqlstore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/memory"
)

func main() {
	guard := security.NewHardFramesGuard("")
	dataDir := os.Getenv("RAE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp/rae-suite-store"
	}

	var baseStore *memorystore.UnifiedStore
	if ps, err := sqlstore.NewPersistentStore(dataDir); err == nil && ps != nil {
		baseStore = ps.UnifiedStore
	} else {
		baseStore = memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	}
	hydrated := baseStore.EnableLiveBackends()
	log.Printf("RAE-Memory live backend initialization completed (hydrated %d records from PostgreSQL)", hydrated)

	svc := memory.NewService(baseStore, baseStore, guard)
	svc.StartBackgroundWorkers(context.Background(), 60*time.Second)

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("RAE_PORT")
	}
	if port == "" {
		port = "8000"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Memory 8-Layer Engine (Go Native v5.3.0) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
