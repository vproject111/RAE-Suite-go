package main

import (
	"log"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/portal"
)

func main() {
	svc := portal.NewService()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Portal Command Center (Go Native v5.3.2) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, security.WrapSecureHandler(mux)))
}
