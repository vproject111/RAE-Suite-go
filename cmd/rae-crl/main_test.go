package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vproject111/RAE-Suite-go/pkg/drivers/memorystore"
	raemath "github.com/vproject111/RAE-Suite-go/pkg/raecore/math"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/models"
	"github.com/vproject111/RAE-Suite-go/pkg/raecore/security"
	"github.com/vproject111/RAE-Suite-go/services/crl"
)

func TestRaeCRLCLIAndHTTP(t *testing.T) {
	origArgs := os.Args
	origListen := listenAndServe
	origExit := exitFunc
	defer func() {
		os.Args = origArgs
		listenAndServe = origListen
		exitFunc = origExit
	}()

	exited := 0
	exitFunc = func(code int) { exited = code }

	os.Args = []string{"rae-crl", "init", "test-proj"}
	main()

	os.Args = []string{"rae-crl", "add", "HYPOTHESIS", "Title 1", "Description 1", "test-proj"}
	main()

	os.Args = []string{"rae-crl", "add"}
	main()

	os.Args = []string{"rae-crl", "query", "test-proj"}
	main()

	os.Args = []string{"rae-crl", "fork", "nonexistent-id", "author", "reason"}
	main()

	os.Args = []string{"rae-crl", "fork"}
	main()

	os.Args = []string{"rae-crl", "seal", "test-proj", "sbom://test"}
	main()

	// Create a valid sealed EvidencePackage file to verify
	guard := security.NewHardFramesGuard("")
	store := memorystore.NewUnifiedStore(raemath.NewMathController(models.MathL3), guard)
	svc := crl.NewService(store, store, guard)
	_, _ = svc.QuickNote(context.Background(), "note", "test-proj", "author")
	pkg, err := svc.SealEvidencePackage(context.Background(), "test-proj", "sbom://test", "NOT_AFFECTED", []string{"v1"})
	if err != nil {
		t.Fatalf("SealEvidencePackage failed: %v", err)
	}
	raw, _ := json.Marshal(pkg)
	tmpFile := filepath.Join(t.TempDir(), "pkg.json")
	_ = os.WriteFile(tmpFile, raw, 0600)

	os.Args = []string{"rae-crl", "verify", tmpFile}
	main()

	os.Args = []string{"rae-crl", "verify"}
	main()

	os.Args = []string{"rae-crl", "verify", filepath.Join(t.TempDir(), "missing.json")}
	main()

	badJSON := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(badJSON, []byte("{invalid"), 0600)
	os.Args = []string{"rae-crl", "verify", badJSON}
	main()

	tampered := *pkg
	tampered.VersionCID = "tampered-cid"
	rawTampered, _ := json.Marshal(tampered)
	tamperedFile := filepath.Join(t.TempDir(), "tampered.json")
	_ = os.WriteFile(tamperedFile, rawTampered, 0600)
	os.Args = []string{"rae-crl", "verify", tamperedFile}
	main()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, h := buildServiceAndHandler(ctx)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %d", rec.Code)
	}

	os.Args = []string{"rae-crl"}
	_ = os.Unsetenv("PORT")
	var boundAddr string
	listenAndServe = func(addr string, handler http.Handler) error {
		boundAddr = addr
		return nil
	}
	if err := run(ctx, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundAddr != ":8503" {
		t.Fatalf("expected :8503, got %s", boundAddr)
	}

	t.Setenv("PORT", "18503")
	main()
	if boundAddr != ":18503" {
		t.Fatalf("expected :18503, got %s", boundAddr)
	}

	exited = 0
	listenAndServe = func(addr string, handler http.Handler) error {
		return errors.New("bind failed")
	}
	main()
	if exited != 1 {
		t.Fatalf("expected exit code 1, got %d", exited)
	}
}
