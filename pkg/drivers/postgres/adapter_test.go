package postgres

import "testing"

func TestEnterpriseCatalog12TablesAndTasks(t *testing.T) {
	if len(Canonical12Tables()) != 12 {
		t.Fatalf("expected 12 canonical tables, got %d", len(Canonical12Tables()))
	}
	cat := NewEnterpriseCatalog()
	if len(cat.ListTenants()) == 0 || len(cat.ListComputeNodes()) < 3 {
		t.Fatalf("expected default tenant and 3 compute nodes")
	}
	task := cat.CreateTask("node1-lumina", "benchmark", map[string]interface{}{"dataset": "academic_lite"})
	done, ok := cat.CompleteTask(task.TaskID, map[string]interface{}{"mrr": 1.0})
	if !ok || done.Status != "COMPLETED" {
		t.Fatalf("CompleteTask failed: %+v", done)
	}
	savings := cat.TokenSavingsSummary()
	if savings["total_tokens_saved"].(int) <= 0 {
		t.Fatalf("expected positive token savings: %+v", savings)
	}
}
