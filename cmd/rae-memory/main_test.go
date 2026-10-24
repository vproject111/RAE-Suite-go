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

func TestRaeMemoryMainAndRoutes(t *testing.T) {
	origListen := listenAndServe
	origExit := exitFunc
	defer func() {
		listenAndServe = origListen
		exitFunc = origExit
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Setenv("RAE_DATA_DIR", t.TempDir())
	h := buildHandler(ctx, nil)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %d", rec.Code)
	}

	// Test fallback when RAE_DATA_DIR is a file instead of a directory
	badDir := filepath.Join(t.TempDir(), "not-a-dir")
	_ = os.WriteFile(badDir, []byte("file"), 0600)
	t.Setenv("RAE_DATA_DIR", badDir)
	_ = buildHandler(ctx, nil)

	// Test default RAE_DATA_DIR and default PORT / RAE_PORT
	_ = os.Unsetenv("RAE_DATA_DIR")
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("RAE_PORT")

	var boundAddr string
	listenAndServe = func(addr string, handler http.Handler) error {
		boundAddr = addr
		return nil
	}
	if err := run(ctx, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundAddr != ":8000" {
		t.Fatalf("expected :8000, got %s", boundAddr)
	}

	t.Setenv("RAE_PORT", "18000")
	main()
	if boundAddr != ":18000" {
		t.Fatalf("expected :18000, got %s", boundAddr)
	}

	exited := 0
	exitFunc = func(code int) { exited = code }
	listenAndServe = func(addr string, handler http.Handler) error {
		return errors.New("bind failed")
	}
	main()
	if exited != 1 {
		t.Fatalf("expected exit code 1, got %d", exited)
	}
}
