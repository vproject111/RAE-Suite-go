package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestRaeMCPStdioAndHTTP(t *testing.T) {
	origArgs := os.Args
	origStdin := os.Stdin
	origListen := listenAndServe
	origExit := exitFunc
	defer func() {
		os.Args = origArgs
		os.Stdin = origStdin
		listenAndServe = origListen
		exitFunc = origExit
	}()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe failed: %v", err)
	}
	_, _ = w.WriteString(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	_ = w.Close()
	os.Stdin = r

	os.Args = []string{"rae-mcp", "--stdio"}
	main()

	_, h := buildServiceAndHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /health, got %d", rec.Code)
	}

	os.Args = []string{"rae-mcp"}
	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("RAE_MCP_PORT")

	var boundAddr string
	listenAndServe = func(addr string, handler http.Handler) error {
		boundAddr = addr
		return nil
	}
	if err := run(context.Background(), ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if boundAddr != ":8005" {
		t.Fatalf("expected :8005, got %s", boundAddr)
	}

	t.Setenv("RAE_MCP_PORT", "18005")
	main()
	if boundAddr != ":18005" {
		t.Fatalf("expected :18005, got %s", boundAddr)
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
