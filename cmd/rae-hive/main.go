package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/hive"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildHandler(ctx context.Context) http.Handler {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-hive")
	svc := hive.NewService()
	svc.StartWatchdogLoop(ctx, 30*time.Second)
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	return security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8013"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-hive", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	handler := buildHandler(ctx)
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

