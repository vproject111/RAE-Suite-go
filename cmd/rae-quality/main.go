package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/quality"
)

func main() {
	svc := quality.NewService()
	if len(os.Args) > 1 && os.Args[1] == "check" {
		code := ""
		if len(os.Args) > 2 {
			b, err := os.ReadFile(os.Args[2])
			if err == nil {
				code = string(b)
			}
		}
		rep := svc.AuditCode(code, nil, "")
		out, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(out))
		if !rep.Passed {
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 2 && (os.Args[1] == "scan-dir" || os.Args[1] == "-audit" || os.Args[1] == "--audit") {
		res, err := svc.GatewayV2.ScanRepositoryDir(os.Args[2])
		if err != nil {
			log.Fatalf("scan-dir failed: %v", err)
		}
		out, _ := json.MarshalIndent(res, "", "  ")
		fmt.Println(string(out))
		return
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8010"
	}
	mux := http.NewServeMux()
	svc.RegisterRoutes(mux)
	log.Printf("RAE-Quality (Go Native v5.3.2) listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, security.WrapSecureHandler(mux)))
}
