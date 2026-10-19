package postgres

import (
	"testing"
)

func TestPGWireClientAndEnterpriseCatalog(t *testing.T) {
	cat := NewEnterpriseCatalog()
	rows, err := cat.ExecuteSQLQuery("SELECT 1 AS alive;")
	if err != nil {
		t.Fatalf("ExecuteSQLQuery error: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected non-empty result from ExecuteSQLQuery")
	}
}
