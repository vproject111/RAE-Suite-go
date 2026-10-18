package main

import (
	"log"
	"net/http"
	"os"

	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/services/lab"
)

func main() {
	svc := lab.NewService(raemath.NewMathController(models.MathL3))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8011"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Lab (Go Native v4.0) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
