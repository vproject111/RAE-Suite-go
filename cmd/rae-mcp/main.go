package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
	"github.com/vproject111/RAE-Suite-go/services/mcp"
)

func main() {
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	crlSvc := crl.NewService(store, store, guard)
	mcpSvc := mcp.NewService(store, crlSvc)

	if len(os.Args) > 1 && (os.Args[1] == "--stdio" || os.Args[1] == "stdio") {
		if err := mcpSvc.ServeStdio(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio error: %v", err)
		}
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("RAE_MCP_PORT")
	}
	if port == "" {
		port = "8005"
	}
	mux := http.NewServeMux()
	mcpSvc.RegisterRoutes(mux)
	log.Printf("RAE-MCP Unified Server (Go Native v5.3.2) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, security.WrapSecureHandler(mux)))
}
