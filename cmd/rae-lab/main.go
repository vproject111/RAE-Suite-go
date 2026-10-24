package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/lab"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildHandler() http.Handler {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-lab")
	svc := lab.NewService(raemath.NewMathController(models.MathL3))
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	return security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8011"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-lab", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	handler := buildHandler()
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

