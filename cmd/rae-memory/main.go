package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	"github.com/vproject111/RAE-Suite-go/pkg/drivers/sqlstore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/memory"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildHandler(ctx context.Context, logger *slog.Logger) http.Handler {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-memory")
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
	if logger != nil {
		logger.InfoContext(ctx, "live_backend_initialized", slog.String("event", "storage_hydrated"), slog.Int("hydrated_records", hydrated))
	}

	svc := memory.NewService(baseStore, baseStore, guard)
	svc.StartBackgroundWorkers(ctx, 60*time.Second)

	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	return security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = os.Getenv("RAE_PORT")
		}
		if port == "" {
			port = "8000"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-memory", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	handler := buildHandler(ctx, logger)
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

