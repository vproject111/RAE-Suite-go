package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/phoenix"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildHandler() (*phoenix.Service, http.Handler) {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-phoenix")
	svc := phoenix.NewService()
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	return svc, security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8012"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-phoenix", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	svc, handler := buildHandler()
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr), slog.Int("max_attempts", svc.MaxAttempts))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

