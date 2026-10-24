package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/mcp"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildServiceAndHandler() (*mcp.Service, http.Handler) {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-mcp")
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	crlSvc := crl.NewService(store, store, guard)
	mcpSvc := mcp.NewService(store, crlSvc)

	mux := http.NewServeMux()
	mcpSvc.RegisterRoutes(mux)
	return mcpSvc, security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	mcpSvc, handler := buildServiceAndHandler()
	if len(os.Args) > 1 && (os.Args[1] == "--stdio" || os.Args[1] == "stdio") {
		return mcpSvc.ServeStdio(ctx, os.Stdin, os.Stdout)
	}

	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = os.Getenv("RAE_MCP_PORT")
		}
		if port == "" {
			port = "8005"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-mcp", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

