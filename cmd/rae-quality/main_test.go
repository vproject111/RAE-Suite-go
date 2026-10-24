package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRaeQualityCLIAndHTTP(t *testing.T) {
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

	tmpDir := t.TempDir()
	cleanFile := filepath.Join(tmpDir, "clean.go")
	_ = os.WriteFile(cleanFile, []byte("package sample\n\nfunc Add(a, b int) int { return a + b }\n"), 0600)

	os.Args = []string{"rae-quality", "check", cleanFile}
	main()

	// Test failing check (eval violation)
	badFile := filepath.Join(tmpDir, "bad.py")
	_ = os.WriteFile(badFile, []byte("eval('1+1')\n"), 0600)
	os.Args = []string{"rae-quality", "check", badFile}
	main()
	if exited != 1 {
		t.Fatalf("expected exit code 1 on failing audit, got %d", exited)
	}

	os.Args = []string{"rae-quality", "scan-dir", tmpDir}
	main()

	_, h := buildHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %d", rec.Code)
	}

	os.Args = []string{"rae-quality"}
	_ = os.Unsetenv("PORT")
	var boundAddr string
	listenAndServe = func(addr string, handler http.Handler) error {
		boundAddr = addr
		return nil
	}
	if err := run(context.Background(), ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundAddr != ":8010" {
		t.Fatalf("expected :8010, got %s", boundAddr)
	}

	t.Setenv("PORT", "18010")
	main()
	if boundAddr != ":18010" {
		t.Fatalf("expected :18010, got %s", boundAddr)
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
