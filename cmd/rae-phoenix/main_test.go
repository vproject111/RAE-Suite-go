package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRaePhoenixMainAndRoutes(t *testing.T) {
	origListen := listenAndServe
	origExit := exitFunc
	defer func() {
		listenAndServe = origListen
		exitFunc = origExit
	}()

	_, h := buildHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %d", rec.Code)
	}

	var boundAddr string
	listenAndServe = func(addr string, handler http.Handler) error {
		boundAddr = addr
		return nil
	}
	_ = os.Unsetenv("PORT")
	if err := run(context.Background(), ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundAddr != ":8012" {
		t.Fatalf("expected :8012, got %s", boundAddr)
	}

	t.Setenv("PORT", "18012")
	main()
	if boundAddr != ":18012" {
		t.Fatalf("expected :18012, got %s", boundAddr)
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
