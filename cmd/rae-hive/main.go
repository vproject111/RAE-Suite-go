package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vproject111/RAE-Suite-go/services/hive"
)

func main() {
	svc := hive.NewService()
	svc.StartWatchdogLoop(context.Background(), 30*time.Second)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8013"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Hive (Go Native v5.3.0) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
