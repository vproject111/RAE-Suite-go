package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/ops"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

var (
	listenAndServe = http.ListenAndServe
	exitFunc       = os.Exit
)

func buildHandler() (*quality.Service, http.Handler) {
	_ = os.Setenv("RAE_SERVICE_NAME", "rae-quality")
	svc := quality.NewService()
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	return svc, security.WrapSecureHandler(mux)
}

func run(ctx context.Context, addr string) error {
	svc, handler := buildHandler()
	if len(os.Args) > 1 && os.Args[1] == "check" {
		code := ""
		if len(os.Args) > 2 {
			if b, err := os.ReadFile(os.Args[2]); err == nil {
				code = string(b)
			}
		}
		rep := svc.AuditCode(code, nil, "")
		out, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(out))
		if !rep.Passed {
			exitFunc(1)
		}
		return nil
	}
	if len(os.Args) > 2 && (os.Args[1] == "scan-dir" || os.Args[1] == "-audit" || os.Args[1] == "--audit") {
		res, err := svc.GatewayV2.ScanRepositoryDir(os.Args[2])
		if err != nil {
			return fmt.Errorf("scan-dir failed: %w", err)
		}
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8010"
		}
		addr = ":" + port
	}
	logger := ops.NewCanonicalLogger("rae-quality", os.Getenv("RAE_ENV"), "5.3.3", os.Stdout, slog.LevelInfo)
	logger.InfoContext(ctx, "service_listening", slog.String("event", "service_startup"), slog.String("addr", addr))
	return listenAndServe(addr, handler)
}

func main() {
	if err := run(context.Background(), ""); err != nil {
		exitFunc(1)
	}
}

