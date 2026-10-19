package main

import (
	"log"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/services/phoenix"
)

func main() {
	svc := phoenix.NewService()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8012"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Phoenix (Go Native v5.3.0, max_attempts=%d) listening on :%s", svc.MaxAttempts, port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
